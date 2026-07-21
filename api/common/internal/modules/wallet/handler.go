package wallet

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/services/paystack"
	"github.com/pulse/api/internal/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// GET /api/wallet
func handleGetWallet(c *gin.Context) {
	userID := middleware.GetUserID(c)
	w, txs, err := getWallet(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrWalletNotFound) {
			utils.Fail(c, http.StatusNotFound, "Wallet not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch wallet")
		return
	}

	recentTx := make([]TxResponse, 0, len(txs))
	for i := range txs {
		recentTx = append(recentTx, toTxResponse(&txs[i]))
	}

	utils.OK(c, http.StatusOK, "", WalletResponse{
		ID:               w.ID.Hex(),
		Role:             w.Role,
		AvailableBalance: w.AvailableBalance,
		PendingBalance:   w.PendingBalance,
		TotalEarned:      w.TotalEarned,
		TotalSpent:       w.TotalSpent,
		Currency:         config.App.PaystackCurrency,
		RecentTx:         recentTx,
		UpdatedAt:        w.UpdatedAt,
	})
}

// GET /api/wallet/transactions
func handleGetTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	userID := middleware.GetUserID(c)
	txs, total, err := getTransactions(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch transactions")
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	resp := make([]TxResponse, 0, len(txs))
	for i := range txs {
		resp = append(resp, toTxResponse(&txs[i]))
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, TransactionListMeta{
		Total: total,
		Page:  page,
		Limit: limit,
		Pages: int64(math.Ceil(float64(total) / float64(limit))),
	})
}

// POST /api/wallet/topup
func handleCreateTopup(c *gin.Context) {
	var req TopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	resp, err := createTopup(c.Request.Context(), userID, req.Amount)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to initiate payment")
		return
	}

	utils.OK(c, http.StatusOK, "Payment initiated", resp)
}

// GET /api/wallet/topup/verify?reference=xxx
func handleVerifyTopup(c *gin.Context) {
	reference := c.Query("reference")
	if reference == "" {
		utils.Fail(c, http.StatusBadRequest, "reference is required")
		return
	}

	if err := verifyTopup(c.Request.Context(), reference); err != nil {
		switch {
		case errors.Is(err, ErrPaystackNotConfigured):
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, ErrPaymentNotSuccessful):
			utils.Fail(c, http.StatusBadRequest, "Payment was not completed")
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to verify payment")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Wallet credited successfully", nil)
}

// POST /api/wallet/topup/webhook  (no auth — Paystack calls this for both
// charge and transfer events; the route name is legacy but left unchanged
// so the URL already configured in the Paystack dashboard keeps working)
func handlePaystackWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.Fail(c, http.StatusBadRequest, "Could not read request body")
		return
	}

	sig := c.GetHeader("X-Paystack-Signature")
	if !paystack.ValidateWebhookSignature(config.App.PaystackSecretKey, sig, body) {
		utils.Fail(c, http.StatusBadRequest, "Invalid webhook signature")
		return
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference    string         `json:"reference"`
			TransferCode string         `json:"transfer_code"`
			Status       string         `json:"status"`
			Amount       int64          `json:"amount"`
			Reason       string         `json:"reason"`
			Metadata     map[string]any `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		utils.Fail(c, http.StatusBadRequest, "Could not parse webhook payload")
		return
	}

	switch event.Event {
	case "charge.success":
		userID, _ := event.Data.Metadata["userID"].(string)
		eventType, _ := event.Data.Metadata["type"].(string)
		if userID == "" || eventType != "wallet_topup" {
			break
		}
		amount := float64(event.Data.Amount) / 100.0
		if err := creditWallet(c.Request.Context(), userID, amount, event.Data.Reference); err != nil {
			utils.Fail(c, http.StatusInternalServerError, "Failed to credit wallet")
			return
		}

	case "transfer.success":
		if withdrawalID, err := bson.ObjectIDFromHex(event.Data.Reference); err == nil {
			_ = CompleteWithdrawal(c.Request.Context(), withdrawalID)
		}

	case "transfer.failed", "transfer.reversed":
		if withdrawalID, err := bson.ObjectIDFromHex(event.Data.Reference); err == nil {
			reason := event.Data.Reason
			if reason == "" {
				reason = "transfer " + event.Event
			}
			_ = RefundAndFailWithdrawal(c.Request.Context(), withdrawalID, reason)
		}
	}

	c.Status(http.StatusOK)
}

// POST /api/wallet/withdraw
func handleWithdraw(c *gin.Context) {
	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	w, err := requestWithdrawal(c.Request.Context(), userID, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, ErrBelowMinimum), errors.Is(err, ErrInsufficientBalance), errors.Is(err, ErrNoBankAccount):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to process withdrawal")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Withdrawal requested — admin will review and process it", toWithdrawalResponse(w))
}

// GET /api/wallet/withdrawals
func handleGetWithdrawals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	userID := middleware.GetUserID(c)
	withdrawals, total, err := getWithdrawals(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch withdrawals")
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	resp := make([]WithdrawalResponse, 0, len(withdrawals))
	for i := range withdrawals {
		resp = append(resp, toWithdrawalResponse(&withdrawals[i]))
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, TransactionListMeta{
		Total: total,
		Page:  page,
		Limit: limit,
		Pages: int64(math.Ceil(float64(total) / float64(limit))),
	})
}

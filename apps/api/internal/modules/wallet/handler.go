package wallet

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"

	stripe "github.com/stripe/stripe-go/v78"
	stripewebhook "github.com/stripe/stripe-go/v78/webhook"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/utils"
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
		Currency:         w.Currency,
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
		if errors.Is(err, ErrStripeNotConfigured) {
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to create payment intent")
		return
	}

	utils.OK(c, http.StatusOK, "Payment intent created — complete payment on the client", resp)
}

// POST /api/wallet/topup/webhook  (no auth — Stripe calls this)
func handleTopupWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.Fail(c, http.StatusBadRequest, "Could not read request body")
		return
	}

	sig := c.GetHeader("Stripe-Signature")
	event, err := stripewebhook.ConstructEvent(payload, sig, config.App.StripeWebhookSecret)
	if err != nil {
		utils.Fail(c, http.StatusBadRequest, "Invalid webhook signature")
		return
	}

	if event.Type != "payment_intent.succeeded" {
		// Acknowledge but ignore other event types.
		c.Status(http.StatusOK)
		return
	}

	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		utils.Fail(c, http.StatusBadRequest, "Could not parse payment intent")
		return
	}

	userID, ok := pi.Metadata["userID"]
	if !ok || userID == "" {
		utils.Fail(c, http.StatusBadRequest, "Missing userID in metadata")
		return
	}
	eventType, _ := pi.Metadata["type"]
	if eventType != "wallet_topup" {
		c.Status(http.StatusOK)
		return
	}

	amount := float64(pi.Amount) / 100.0
	if err := creditWallet(c.Request.Context(), userID, amount, pi.ID); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to credit wallet")
		return
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
		case errors.Is(err, ErrBelowMinimum):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInsufficientBalance):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoConnectAccount), errors.Is(err, ErrConnectNotActive):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrStripeNotConfigured):
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to process withdrawal")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Withdrawal initiated", toWithdrawalResponse(w))
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

// POST /api/wallet/connect
func handleCreateConnect(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := createConnectAccount(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrStripeNotConfigured) {
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to create Connect account")
		return
	}

	utils.OK(c, http.StatusOK, "Complete onboarding at the provided URL", resp)
}

// GET /api/wallet/connect/status
func handleGetConnectStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := getConnectStatus(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoConnectAccount):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrStripeNotConfigured):
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to fetch Connect status")
		}
		return
	}

	utils.OK(c, http.StatusOK, "", resp)
}

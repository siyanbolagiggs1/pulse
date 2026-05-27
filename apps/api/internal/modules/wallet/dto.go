package wallet

import (
	"time"

	"github.com/pulse/api/internal/models"
)

// ── Requests ─────────────────────────────────────────────────

type TopupRequest struct {
	Amount float64 `json:"amount" binding:"required,min=5"`
}

type WithdrawRequest struct {
	Amount float64 `json:"amount" binding:"required,min=10"`
}

// ── Responses ────────────────────────────────────────────────

type WalletResponse struct {
	ID               string          `json:"id"`
	Role             models.Role     `json:"role"`
	AvailableBalance float64         `json:"availableBalance"`
	PendingBalance   float64         `json:"pendingBalance"`
	TotalEarned      float64         `json:"totalEarned"`
	TotalSpent       float64         `json:"totalSpent"`
	Currency         string          `json:"currency"`
	RecentTx         []TxResponse    `json:"recentTransactions"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

type TxResponse struct {
	ID           string                `json:"id"`
	Type         models.TransactionType `json:"type"`
	Amount       float64               `json:"amount"`
	Fee          float64               `json:"fee"`
	BalanceAfter float64               `json:"balanceAfter"`
	ReferenceID  string                `json:"referenceId,omitempty"`
	Description  string                `json:"description"`
	CreatedAt    time.Time             `json:"createdAt"`
}

type TopupResponse struct {
	ClientSecret    string  `json:"clientSecret"`
	PaymentIntentID string  `json:"paymentIntentId"`
	Amount          float64 `json:"amount"`
}

type WithdrawalResponse struct {
	ID             string                  `json:"id"`
	Amount         float64                 `json:"amount"`
	Fee            float64                 `json:"fee"`
	NetAmount      float64                 `json:"netAmount"`
	Status         models.WithdrawalStatus `json:"status"`
	StripePayoutID string                  `json:"stripePayoutId,omitempty"`
	RequestedAt    time.Time               `json:"requestedAt"`
	ProcessedAt    time.Time               `json:"processedAt,omitempty"`
}

type ConnectOnboardingResponse struct {
	URL               string `json:"url"`
	ConnectAccountID  string `json:"connectAccountId"`
}

type ConnectStatusResponse struct {
	AccountID       string `json:"accountId"`
	ChargesEnabled  bool   `json:"chargesEnabled"`
	PayoutsEnabled  bool   `json:"payoutsEnabled"`
	Status          string `json:"status"` // pending | active | restricted
}

type TransactionListMeta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int64 `json:"pages"`
}

// ── Mappers ──────────────────────────────────────────────────

func toTxResponse(t *models.Transaction) TxResponse {
	return TxResponse{
		ID:           t.ID.Hex(),
		Type:         t.Type,
		Amount:       t.Amount,
		Fee:          t.Fee,
		BalanceAfter: t.BalanceAfter,
		ReferenceID:  t.ReferenceID,
		Description:  t.Description,
		CreatedAt:    t.CreatedAt,
	}
}

func toWithdrawalResponse(w *models.Withdrawal) WithdrawalResponse {
	return WithdrawalResponse{
		ID:             w.ID.Hex(),
		Amount:         w.Amount,
		Fee:            w.Fee,
		NetAmount:      w.NetAmount,
		Status:         w.Status,
		StripePayoutID: w.StripePayoutID,
		RequestedAt:    w.RequestedAt,
		ProcessedAt:    w.ProcessedAt,
	}
}

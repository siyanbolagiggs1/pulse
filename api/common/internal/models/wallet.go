package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Wallet struct {
	ID     bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID bson.ObjectID `bson:"userId"        json:"userId"`
	Role   Role          `bson:"role"          json:"role"`

	AvailableBalance float64 `bson:"availableBalance" json:"availableBalance"`
	PendingBalance   float64 `bson:"pendingBalance"   json:"pendingBalance"`

	TotalEarned float64 `bson:"totalEarned" json:"totalEarned"`
	TotalSpent  float64 `bson:"totalSpent"  json:"totalSpent"`

	Currency  string    `bson:"currency"  json:"currency"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

const WalletsCollection = "wallets"

// ────────────────────────────────────────────────────────────
// Transaction
// ────────────────────────────────────────────────────────────

type TransactionType string

const (
	TxTopup          TransactionType = "topup"
	TxCampaignLock   TransactionType = "campaign_lock"
	TxPayoutPending  TransactionType = "payout_pending"
	TxPayoutReleased TransactionType = "payout_released"
	TxCommission     TransactionType = "commission"
	TxWithdrawal     TransactionType = "withdrawal"
	TxRefund         TransactionType = "refund"
)

type Transaction struct {
	ID           bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	WalletID     bson.ObjectID   `bson:"walletId"      json:"walletId"`
	UserID       bson.ObjectID   `bson:"userId"        json:"userId"`
	Type         TransactionType `bson:"type"          json:"type"`
	Amount       float64         `bson:"amount"        json:"amount"`
	Fee          float64         `bson:"fee"           json:"fee"`
	BalanceAfter float64         `bson:"balanceAfter"  json:"balanceAfter"`
	ReferenceID  string          `bson:"referenceId"   json:"referenceId"`
	Description  string          `bson:"description"   json:"description"`
	CreatedAt    time.Time       `bson:"createdAt"     json:"createdAt"`
}

const TransactionsCollection = "transactions"

// ────────────────────────────────────────────────────────────
// Withdrawal
// ────────────────────────────────────────────────────────────

type WithdrawalStatus string

const (
	WithdrawalPending    WithdrawalStatus = "pending"
	WithdrawalProcessing WithdrawalStatus = "processing"
	WithdrawalCompleted  WithdrawalStatus = "completed"
	WithdrawalFailed     WithdrawalStatus = "failed"
)

type Withdrawal struct {
	ID             bson.ObjectID    `bson:"_id,omitempty"  json:"id"`
	UserID         bson.ObjectID    `bson:"userId"         json:"userId"`
	Amount         float64          `bson:"amount"         json:"amount"`
	Fee            float64          `bson:"fee"            json:"fee"`
	NetAmount      float64          `bson:"netAmount"      json:"netAmount"`
	PayoutID       string           `bson:"payoutId"       json:"payoutId,omitempty"`
	Status         WithdrawalStatus `bson:"status"         json:"status"`
	RequestedAt    time.Time        `bson:"requestedAt"    json:"requestedAt"`
	ProcessedAt    time.Time        `bson:"processedAt"    json:"processedAt,omitempty"`
	CreatedAt      time.Time        `bson:"createdAt"      json:"createdAt"`
}

const WithdrawalsCollection = "withdrawals"

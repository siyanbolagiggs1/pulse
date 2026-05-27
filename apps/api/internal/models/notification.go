package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type NotificationType string

const (
	NotifSubmissionApproved  NotificationType = "submission_approved"
	NotifSubmissionRejected  NotificationType = "submission_rejected"
	NotifWithdrawalProcessed NotificationType = "withdrawal_processed"
	NotifCampaignUpdated     NotificationType = "campaign_updated"
	NotifWalletTopup         NotificationType = "wallet_topup"
	NotifFraudFlag           NotificationType = "fraud_flag"
)

type Notification struct {
	ID        bson.ObjectID          `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID          `bson:"userId"        json:"userId"`
	Type      NotificationType       `bson:"type"          json:"type"`
	Title     string                 `bson:"title"         json:"title"`
	Message   string                 `bson:"message"       json:"message"`
	IsRead    bool                   `bson:"isRead"        json:"isRead"`
	Metadata  map[string]interface{} `bson:"metadata"      json:"metadata,omitempty"`
	CreatedAt time.Time              `bson:"createdAt"     json:"createdAt"`
}

const NotificationsCollection = "notifications"

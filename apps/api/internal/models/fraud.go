package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type FraudFlagReason string

const (
	FraudLowFollowerRatio   FraudFlagReason = "low_follower_ratio"
	FraudSuspiciousGrowth   FraudFlagReason = "suspicious_follower_growth"
	FraudAbnormalEngagement FraudFlagReason = "abnormal_engagement_rate"
	FraudDuplicateRepost    FraudFlagReason = "duplicate_repost_url"
	FraudRateLimit          FraudFlagReason = "submission_rate_limit"
	FraudDuplicateIP        FraudFlagReason = "duplicate_ip_address"
	FraudNewAccount         FraudFlagReason = "account_too_new"
)

type FraudFlag struct {
	ID         bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID     bson.ObjectID   `bson:"userId"        json:"userId"`
	Reason     FraudFlagReason `bson:"reason"        json:"reason"`
	Details    string          `bson:"details"       json:"details"`
	Resolved   bool            `bson:"resolved"      json:"resolved"`
	ResolvedBy bson.ObjectID   `bson:"resolvedBy"    json:"resolvedBy,omitempty"`
	CreatedAt  time.Time       `bson:"createdAt"     json:"createdAt"`
	UpdatedAt  time.Time       `bson:"updatedAt"     json:"updatedAt"`
}

const FraudFlagsCollection = "fraud_flags"

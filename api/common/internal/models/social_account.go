package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Platform string

const (
	PlatformInstagram Platform = "instagram"
	PlatformTwitter   Platform = "twitter"
	PlatformTikTok    Platform = "tiktok"
)

type SocialAccountStatus string

const (
	SocialAccountPending  SocialAccountStatus = "pending_review"
	SocialAccountActive   SocialAccountStatus = "active"
	SocialAccountRejected SocialAccountStatus = "rejected"
)

type SocialAccount struct {
	ID              bson.ObjectID          `bson:"_id,omitempty" json:"id"`
	UserID          bson.ObjectID          `bson:"userId"        json:"userId"`
	Platform        Platform               `bson:"platform"      json:"platform"`
	PlatformUserID  string                 `bson:"platformUserId" json:"platformUserId"`
	Username        string                 `bson:"username"      json:"username"`
	ProfileURL      string                 `bson:"profileUrl"    json:"profileUrl"`
	FollowerCount   int64                  `bson:"followerCount" json:"followerCount"`
	Tier            int                    `bson:"tier"          json:"tier"`
	InfluenceScore  float64                `bson:"influenceScore" json:"influenceScore"`
	IsVerified      bool                   `bson:"isVerified"    json:"isVerified"`
	Status          SocialAccountStatus    `bson:"status"        json:"status"`
	RejectedReason  string                 `bson:"rejectedReason,omitempty" json:"rejectedReason,omitempty"`
	DisconnectedAt  *time.Time             `bson:"disconnectedAt,omitempty" json:"-"`
	LastVerifiedAt  *time.Time             `bson:"lastVerifiedAt,omitempty" json:"-"`
	FollowerHistory []FollowerHistoryEntry `bson:"followerHistory,omitempty" json:"-"`
	LastSyncedAt    time.Time              `bson:"lastSyncedAt"  json:"lastSyncedAt"`
	CreatedAt       time.Time              `bson:"createdAt"     json:"createdAt"`
}

// FollowerHistoryEntry records a follower-count snapshot each time an admin
// approves/re-verifies a social account. Admin-only — never serialized to
// the promoter-facing SocialAccountResponse — so admins can spot suspicious
// jumps (e.g. bought followers) when reviewing a reconnect request.
type FollowerHistoryEntry struct {
	FollowerCount int64     `bson:"followerCount" json:"followerCount"`
	Tier          int       `bson:"tier"          json:"tier"`
	RecordedAt    time.Time `bson:"recordedAt"    json:"recordedAt"`
}

const SocialAccountsCollection = "social_accounts"

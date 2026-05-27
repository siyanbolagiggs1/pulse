package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Platform string

const (
	PlatformInstagram Platform = "instagram"
	PlatformTwitter   Platform = "twitter"
)

type SocialAccount struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         bson.ObjectID `bson:"userId"        json:"userId"`
	Platform       Platform      `bson:"platform"      json:"platform"`
	PlatformUserID string        `bson:"platformUserId" json:"platformUserId"`
	Username       string        `bson:"username"      json:"username"`
	ProfileURL     string        `bson:"profileUrl"    json:"profileUrl"`
	FollowerCount  int64         `bson:"followerCount" json:"followerCount"`
	FollowingCount int64         `bson:"followingCount" json:"followingCount"`
	EngagementRate float64       `bson:"engagementRate" json:"engagementRate"`
	AccountAge     int           `bson:"accountAgeDays" json:"accountAgeDays"`
	InfluenceScore float64       `bson:"influenceScore" json:"influenceScore"`
	IsVerified     bool          `bson:"isVerified"    json:"isVerified"`
	LastSyncedAt   time.Time     `bson:"lastSyncedAt"  json:"lastSyncedAt"`
	CreatedAt      time.Time     `bson:"createdAt"     json:"createdAt"`
}

const SocialAccountsCollection = "social_accounts"

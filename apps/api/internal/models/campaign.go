package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusCancelled CampaignStatus = "cancelled"
)

type Campaign struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	BusinessID bson.ObjectID `bson:"businessId"    json:"businessId"`

	Title       string   `bson:"title"       json:"title"`
	Description string   `bson:"description" json:"description"`
	TargetURL   string   `bson:"targetUrl"   json:"targetUrl"`
	MediaAssets []string `bson:"mediaAssets" json:"mediaAssets"`
	Platform    Platform `bson:"platform"    json:"platform"`

	Budget          float64 `bson:"budget"          json:"budget"`
	RemainingBudget float64 `bson:"remainingBudget" json:"remainingBudget"`
	BaseRepostRate  float64 `bson:"baseRepostRate"  json:"baseRepostRate"`

	MinFollowers      int64   `bson:"minFollowers"      json:"minFollowers"`
	MinEngagementRate float64 `bson:"minEngagementRate" json:"minEngagementRate"`
	MinInfluenceScore float64 `bson:"minInfluenceScore" json:"minInfluenceScore"`

	MaxParticipants     int `bson:"maxParticipants"     json:"maxParticipants"`
	CurrentParticipants int `bson:"currentParticipants" json:"currentParticipants"`

	Status    CampaignStatus `bson:"status"    json:"status"`
	StartDate time.Time      `bson:"startDate" json:"startDate"`
	EndDate   time.Time      `bson:"endDate"   json:"endDate"`
	CreatedAt time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
}

const CampaignsCollection = "campaigns"

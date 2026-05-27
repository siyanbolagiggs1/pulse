package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type SubmissionStatus string

const (
	SubmissionStatusPending  SubmissionStatus = "pending"
	SubmissionStatusApproved SubmissionStatus = "approved"
	SubmissionStatusRejected SubmissionStatus = "rejected"
)

type CampaignSubmission struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CampaignID bson.ObjectID `bson:"campaignId"    json:"campaignId"`
	PromoterID bson.ObjectID `bson:"promoterId"    json:"promoterId"`
	BusinessID bson.ObjectID `bson:"businessId"    json:"businessId"`

	RepostURL     string `bson:"repostUrl"     json:"repostUrl"`
	ScreenshotURL string `bson:"screenshotUrl" json:"screenshotUrl"`

	Status          SubmissionStatus `bson:"status"          json:"status"`
	RejectionReason string           `bson:"rejectionReason" json:"rejectionReason,omitempty"`
	ReviewedBy      bson.ObjectID    `bson:"reviewedBy"      json:"reviewedBy,omitempty"`

	BaseAmount          float64 `bson:"baseAmount"          json:"baseAmount"`
	InfluenceMultiplier float64 `bson:"influenceMultiplier" json:"influenceMultiplier"`
	FinalAmount         float64 `bson:"finalAmount"         json:"finalAmount"`
	PlatformFee         float64 `bson:"platformFee"         json:"platformFee"`
	PromoterEarning     float64 `bson:"promoterEarning"     json:"promoterEarning"`

	SubmittedAt      time.Time `bson:"submittedAt"       json:"submittedAt"`
	ReviewedAt       time.Time `bson:"reviewedAt"        json:"reviewedAt,omitempty"`
	PayoutReleasedAt time.Time `bson:"payoutReleasedAt"  json:"payoutReleasedAt,omitempty"`
	PayoutReleased   bool      `bson:"payoutReleased"    json:"payoutReleased"`
	CreatedAt        time.Time `bson:"createdAt"         json:"createdAt"`
	UpdatedAt        time.Time `bson:"updatedAt"         json:"updatedAt"`
}

const SubmissionsCollection = "submissions"

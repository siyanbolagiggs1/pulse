package submissions

import (
	"time"

	"github.com/pulse/api/internal/models"
)

// ── Requests ─────────────────────────────────────────────────

type CreateSubmissionRequest struct {
	CampaignID      string `json:"campaignId"      binding:"required"`
	SocialAccountID string `json:"socialAccountId" binding:"required"`
	RepostURL       string `json:"repostUrl"       binding:"required"`
	ScreenshotURL   string `json:"screenshotUrl"   binding:"required"`
}

type RejectRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

// ── Responses ────────────────────────────────────────────────

type SubmissionResponse struct {
	ID              string                  `json:"id"`
	CampaignID      string                  `json:"campaignId"`
	PromoterID      string                  `json:"promoterId"`
	BusinessID      string                  `json:"businessId"`
	RepostURL       string                  `json:"repostUrl"`
	ScreenshotURL   string                  `json:"screenshotUrl"`
	Status          models.SubmissionStatus `json:"status"`
	RejectionReason string                  `json:"rejectionReason,omitempty"`
	ReviewedBy      string                  `json:"reviewedBy,omitempty"`

	BaseAmount          float64 `json:"baseAmount"`
	InfluenceMultiplier float64 `json:"influenceMultiplier"`
	FinalAmount         float64 `json:"finalAmount"`
	PlatformFee         float64 `json:"platformFee"`
	PromoterEarning     float64 `json:"promoterEarning"`

	SubmittedAt      time.Time `json:"submittedAt"`
	ReviewedAt       time.Time `json:"reviewedAt,omitempty"`
	PayoutReleasedAt time.Time `json:"payoutReleasedAt,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SubmissionListQuery struct {
	CampaignID string `form:"campaignId"`
	PromoterID string `form:"promoterId"`
	Status     string `form:"status"`
	// View scopes non-admin results: "mine" (submissions I made) or
	// "incoming" (submissions to campaigns I own). Ignored for admins.
	View  string `form:"view"`
	Page  int    `form:"page"`
	Limit int    `form:"limit"`
}

type SubmissionListMeta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int64 `json:"pages"`
}

type UploadResponse struct {
	URL string `json:"url"`
}

// ── Mapper ───────────────────────────────────────────────────

func toSubmissionResponse(s *models.CampaignSubmission) SubmissionResponse {
	return SubmissionResponse{
		ID:              s.ID.Hex(),
		CampaignID:      s.CampaignID.Hex(),
		PromoterID:      s.PromoterID.Hex(),
		BusinessID:      s.BusinessID.Hex(),
		RepostURL:       s.RepostURL,
		ScreenshotURL:   s.ScreenshotURL,
		Status:          s.Status,
		RejectionReason: s.RejectionReason,
		ReviewedBy:      s.ReviewedBy.Hex(),

		BaseAmount:          s.BaseAmount,
		InfluenceMultiplier: s.InfluenceMultiplier,
		FinalAmount:         s.FinalAmount,
		PlatformFee:         s.PlatformFee,
		PromoterEarning:     s.PromoterEarning,

		SubmittedAt:      s.SubmittedAt,
		ReviewedAt:       s.ReviewedAt,
		PayoutReleasedAt: s.PayoutReleasedAt,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

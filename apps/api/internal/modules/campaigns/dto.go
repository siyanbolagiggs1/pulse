package campaigns

import (
	"time"

	"github.com/pulse/api/internal/models"
)

// ── Requests ─────────────────────────────────────────────────

type CreateCampaignRequest struct {
	Title             string          `json:"title"             binding:"required,min=3,max=200"`
	Description       string          `json:"description"       binding:"required,min=10,max=2000"`
	TargetURL         string          `json:"targetUrl"         binding:"required"`
	MediaAssets       []string        `json:"mediaAssets"`
	Platform          models.Platform `json:"platform"          binding:"required,oneof=instagram twitter"`
	Budget            float64         `json:"budget"            binding:"required,min=10"`
	BaseRepostRate    float64         `json:"baseRepostRate"    binding:"required,min=0.01"`
	MinFollowers      int64           `json:"minFollowers"      binding:"min=0"`
	MinEngagementRate float64         `json:"minEngagementRate" binding:"min=0"`
	MinInfluenceScore float64         `json:"minInfluenceScore" binding:"min=0,max=100"`
	MaxParticipants   int             `json:"maxParticipants"   binding:"min=0"`
	StartDate         time.Time       `json:"startDate"         binding:"required"`
	EndDate           time.Time       `json:"endDate"           binding:"required"`
}

type UpdateCampaignRequest struct {
	Title             string     `json:"title"             binding:"omitempty,min=3,max=200"`
	Description       string     `json:"description"       binding:"omitempty,min=10,max=2000"`
	MediaAssets       []string   `json:"mediaAssets"`
	MinFollowers      *int64     `json:"minFollowers"`
	MinEngagementRate *float64   `json:"minEngagementRate"`
	MinInfluenceScore *float64   `json:"minInfluenceScore"`
	EndDate           *time.Time `json:"endDate"`
	Status            string     `json:"status" binding:"omitempty,oneof=active paused"`
}

type CampaignListQuery struct {
	Platform string `form:"platform" binding:"omitempty,oneof=instagram twitter"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	Sort     string `form:"sort"`
}

// ── Responses ────────────────────────────────────────────────

type CampaignResponse struct {
	ID                  string                `json:"id"`
	BusinessID          string                `json:"businessId"`
	Title               string                `json:"title"`
	Description         string                `json:"description"`
	TargetURL           string                `json:"targetUrl"`
	MediaAssets         []string              `json:"mediaAssets"`
	Platform            models.Platform       `json:"platform"`
	Budget              float64               `json:"budget"`
	RemainingBudget     float64               `json:"remainingBudget"`
	BaseRepostRate      float64               `json:"baseRepostRate"`
	MinFollowers        int64                 `json:"minFollowers"`
	MinEngagementRate   float64               `json:"minEngagementRate"`
	MinInfluenceScore   float64               `json:"minInfluenceScore"`
	MaxParticipants     int                   `json:"maxParticipants"`
	CurrentParticipants int                   `json:"currentParticipants"`
	Status              models.CampaignStatus `json:"status"`
	StartDate           time.Time             `json:"startDate"`
	EndDate             time.Time             `json:"endDate"`
	CreatedAt           time.Time             `json:"createdAt"`
	UpdatedAt           time.Time             `json:"updatedAt"`
}

type CampaignListMeta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int64 `json:"pages"`
}

// ── Mapper ───────────────────────────────────────────────────

func toCampaignResponse(c *models.Campaign) CampaignResponse {
	assets := c.MediaAssets
	if assets == nil {
		assets = []string{}
	}
	return CampaignResponse{
		ID:                  c.ID.Hex(),
		BusinessID:          c.BusinessID.Hex(),
		Title:               c.Title,
		Description:         c.Description,
		TargetURL:           c.TargetURL,
		MediaAssets:         assets,
		Platform:            c.Platform,
		Budget:              c.Budget,
		RemainingBudget:     c.RemainingBudget,
		BaseRepostRate:      c.BaseRepostRate,
		MinFollowers:        c.MinFollowers,
		MinEngagementRate:   c.MinEngagementRate,
		MinInfluenceScore:   c.MinInfluenceScore,
		MaxParticipants:     c.MaxParticipants,
		CurrentParticipants: c.CurrentParticipants,
		Status:              c.Status,
		StartDate:           c.StartDate,
		EndDate:             c.EndDate,
		CreatedAt:           c.CreatedAt,
		UpdatedAt:           c.UpdatedAt,
	}
}

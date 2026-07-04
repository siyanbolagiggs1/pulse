package admin

import (
	"time"

	"github.com/pulse/api/internal/models"
)

// ── Requests ─────────────────────────────────────────────────

type SuspendRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

type RejectWithdrawalRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

type ApproveSocialAccountRequest struct {
	FollowerCount int64 `json:"followerCount" binding:"required,min=100"`
}

type RejectSocialAccountRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

type PendingSocialAccountResponse struct {
	ID              string                        `json:"id"`
	UserID          string                        `json:"userId"`
	Platform        models.Platform               `json:"platform"`
	Username        string                        `json:"username"`
	ProfileURL      string                        `json:"profileUrl"`
	FollowerCount   int64                         `json:"followerCount"`
	Tier            int                           `json:"tier"`
	FollowerHistory []models.FollowerHistoryEntry `json:"followerHistory"`
	CreatedAt       time.Time                     `json:"createdAt"`
}

// ── Query params ─────────────────────────────────────────────

type UserListQuery struct {
	Role      string `form:"role"`
	Suspended string `form:"suspended"` // "true" | "false" | ""
	Search    string `form:"search"`    // email or name prefix
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
}

type FraudFlagQuery struct {
	UserID   string `form:"userId"`
	Resolved string `form:"resolved"` // "true" | "false" | ""
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
}

type WithdrawalQuery struct {
	UserID string `form:"userId"`
	Status string `form:"status"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

// ── Responses ────────────────────────────────────────────────

type PlatformStats struct {
	Users       UserStats       `json:"users"`
	Campaigns   CampaignStats   `json:"campaigns"`
	Submissions SubmissionStats `json:"submissions"`
	Financials  FinancialStats  `json:"financials"`
}

type UserStats struct {
	Total      int64 `json:"total"`
	Businesses int64 `json:"businesses"`
	Promoters  int64 `json:"promoters"`
	Suspended  int64 `json:"suspended"`
}

type CampaignStats struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Draft     int64 `json:"draft"`
	Completed int64 `json:"completed"`
}

type SubmissionStats struct {
	Total    int64 `json:"total"`
	Pending  int64 `json:"pending"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
}

type FinancialStats struct {
	TotalPendingWithdrawals float64 `json:"totalPendingWithdrawals"`
	TotalWithdrawn          float64 `json:"totalWithdrawn"`
	TotalPromoterPending    float64 `json:"totalPromoterPending"`
}

type AdminUserResponse struct {
	ID                  string                     `json:"id"`
	Email               string                     `json:"email"`
	Name                string                     `json:"name"`
	Role                models.Role                `json:"role"`
	Avatar              string                     `json:"avatar,omitempty"`
	IsEmailVerified     bool                       `json:"isEmailVerified"`
	IsSuspended         bool                       `json:"isSuspended"`
	TrustScore          float64                    `json:"trustScore"`
	Badges              []models.VerificationBadge `json:"badges"`
	CreatedAt           time.Time                  `json:"createdAt"`
}

type FraudFlagResponse struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"userId"`
	Reason     models.FraudFlagReason `json:"reason"`
	Details    string                 `json:"details"`
	Resolved   bool                   `json:"resolved"`
	ResolvedBy string                 `json:"resolvedBy,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type WithdrawalAdminResponse struct {
	ID             string                  `json:"id"`
	UserID         string                  `json:"userId"`
	Amount         float64                 `json:"amount"`
	Fee            float64                 `json:"fee"`
	NetAmount      float64                 `json:"netAmount"`
	Status         models.WithdrawalStatus `json:"status"`
	PayoutID       string                  `json:"payoutId,omitempty"`
	RequestedAt    time.Time               `json:"requestedAt"`
	ProcessedAt    time.Time               `json:"processedAt,omitempty"`
}

type ListMeta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int64 `json:"pages"`
}

// ── Mappers ──────────────────────────────────────────────────

func toAdminUserResponse(u *models.User) AdminUserResponse {
	badges := u.Badges
	if badges == nil {
		badges = []models.VerificationBadge{}
	}
	return AdminUserResponse{
		ID:                  u.ID.Hex(),
		Email:               u.Email,
		Name:                u.Name,
		Role:                u.Role,
		Avatar:              u.Avatar,
		IsEmailVerified:     u.IsEmailVerified,
		IsSuspended:         u.IsSuspended,
		TrustScore:          u.TrustScore,
		Badges:    badges,
		CreatedAt: u.CreatedAt,
	}
}

func toFraudFlagResponse(f *models.FraudFlag) FraudFlagResponse {
	return FraudFlagResponse{
		ID:         f.ID.Hex(),
		UserID:     f.UserID.Hex(),
		Reason:     f.Reason,
		Details:    f.Details,
		Resolved:   f.Resolved,
		ResolvedBy: f.ResolvedBy.Hex(),
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
	}
}

func toWithdrawalAdminResponse(w *models.Withdrawal) WithdrawalAdminResponse {
	return WithdrawalAdminResponse{
		ID:             w.ID.Hex(),
		UserID:         w.UserID.Hex(),
		Amount:         w.Amount,
		Fee:            w.Fee,
		NetAmount:      w.NetAmount,
		Status:         w.Status,
		PayoutID:       w.PayoutID,
		RequestedAt:    w.RequestedAt,
		ProcessedAt:    w.ProcessedAt,
	}
}

package users

import (
	"time"

	"github.com/pulse/api/internal/models"
)

// ── Requests ─────────────────────────────────────────────────

type UpdateProfileRequest struct {
	Name   string `json:"name"   binding:"omitempty,min=2,max=100"`
	Avatar string `json:"avatar" binding:"omitempty"`
}

type ConnectSocialAccountRequest struct {
	Platform       models.Platform `json:"platform"       binding:"required,oneof=instagram twitter"`
	PlatformUserID string          `json:"platformUserId" binding:"required"`
	Username       string          `json:"username"       binding:"required"`
	ProfileURL     string          `json:"profileUrl"     binding:"required"`
	FollowerCount  int64           `json:"followerCount"  binding:"min=0"`
	FollowingCount int64           `json:"followingCount" binding:"min=0"`
	EngagementRate float64         `json:"engagementRate" binding:"min=0"`
	AccountAgeDays int             `json:"accountAgeDays" binding:"min=0"`
	IsVerified     bool            `json:"isVerified"`
}

// ── Responses ────────────────────────────────────────────────

type SocialAccountResponse struct {
	ID             string          `json:"id"`
	Platform       models.Platform `json:"platform"`
	PlatformUserID string          `json:"platformUserId"`
	Username       string          `json:"username"`
	ProfileURL     string          `json:"profileUrl"`
	FollowerCount  int64           `json:"followerCount"`
	FollowingCount int64           `json:"followingCount"`
	EngagementRate float64         `json:"engagementRate"`
	AccountAgeDays int             `json:"accountAgeDays"`
	InfluenceScore float64         `json:"influenceScore"`
	IsVerified     bool            `json:"isVerified"`
	LastSyncedAt   time.Time       `json:"lastSyncedAt"`
}

type UserResponse struct {
	ID              string                     `json:"id"`
	Email           string                     `json:"email"`
	Name            string                     `json:"name"`
	Role            models.Role                `json:"role"`
	Avatar          string                     `json:"avatar,omitempty"`
	IsEmailVerified bool                       `json:"isEmailVerified"`
	IsSuspended     bool                       `json:"isSuspended"`
	TrustScore      float64                    `json:"trustScore"`
	Badges          []models.VerificationBadge `json:"badges"`
	SocialAccounts  []SocialAccountResponse    `json:"socialAccounts"`
	CreatedAt       time.Time                  `json:"createdAt"`
}

type AccountInfluenceScore struct {
	AccountID         string          `json:"accountId"`
	Platform          models.Platform `json:"platform"`
	Username          string          `json:"username"`
	OverallScore      float64         `json:"overallScore"`
	FollowerScore     float64         `json:"followerScore"`
	EngagementScore   float64         `json:"engagementScore"`
	AccountAgeScore   float64         `json:"accountAgeScore"`
	CompletionScore   float64         `json:"completionScore"`
	AudienceQualScore float64         `json:"audienceQualityScore"`
}

type InfluenceScoreResponse struct {
	Accounts  []AccountInfluenceScore `json:"accounts"`
	BestScore float64                 `json:"bestScore"`
}

// ── Mappers ──────────────────────────────────────────────────

func toUserResponse(u *models.User, accounts []models.SocialAccount) UserResponse {
	badges := u.Badges
	if badges == nil {
		badges = []models.VerificationBadge{}
	}

	socialResps := make([]SocialAccountResponse, 0, len(accounts))
	for i := range accounts {
		socialResps = append(socialResps, toSocialAccountResponse(&accounts[i]))
	}

	return UserResponse{
		ID:              u.ID.Hex(),
		Email:           u.Email,
		Name:            u.Name,
		Role:            u.Role,
		Avatar:          u.Avatar,
		IsEmailVerified: u.IsEmailVerified,
		IsSuspended:     u.IsSuspended,
		TrustScore:      u.TrustScore,
		Badges:          badges,
		SocialAccounts:  socialResps,
		CreatedAt:       u.CreatedAt,
	}
}

func toSocialAccountResponse(a *models.SocialAccount) SocialAccountResponse {
	return SocialAccountResponse{
		ID:             a.ID.Hex(),
		Platform:       a.Platform,
		PlatformUserID: a.PlatformUserID,
		Username:       a.Username,
		ProfileURL:     a.ProfileURL,
		FollowerCount:  a.FollowerCount,
		FollowingCount: a.FollowingCount,
		EngagementRate: a.EngagementRate,
		AccountAgeDays: a.AccountAge,
		InfluenceScore: a.InfluenceScore,
		IsVerified:     a.IsVerified,
		LastSyncedAt:   a.LastSyncedAt,
	}
}

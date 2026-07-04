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
	Platform   models.Platform `json:"platform"   binding:"required,oneof=instagram twitter tiktok"`
	Username   string          `json:"username"   binding:"omitempty"`
	ProfileURL string          `json:"profileUrl" binding:"required,url"`
}

// ── Responses ────────────────────────────────────────────────

type SocialAccountResponse struct {
	ID             string                     `json:"id"`
	Platform       models.Platform            `json:"platform"`
	PlatformUserID string                     `json:"platformUserId"`
	Username       string                     `json:"username"`
	ProfileURL     string                     `json:"profileUrl"`
	Tier           int                        `json:"tier"`
	IsVerified     bool                       `json:"isVerified"`
	Status         models.SocialAccountStatus `json:"status"`
	RejectedReason string                     `json:"rejectedReason,omitempty"`
	LastSyncedAt   time.Time                  `json:"lastSyncedAt"`
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

// SearchUserResponse is the minimal shape returned by the recipient-search
// endpoint — no email/trustScore/etc., just enough to pick a chat recipient.
type SearchUserResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

func toSearchUserResponse(u *models.User) SearchUserResponse {
	return SearchUserResponse{ID: u.ID.Hex(), Name: u.Name, Avatar: u.Avatar}
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
		Tier:           a.Tier,
		IsVerified:     a.IsVerified,
		Status:         a.Status,
		RejectedReason: a.RejectedReason,
		LastSyncedAt:   a.LastSyncedAt,
	}
}

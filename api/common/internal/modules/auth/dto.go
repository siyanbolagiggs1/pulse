package auth

import "github.com/pulse/api/internal/models"

// ── Requests ─────────────────────────────────────────────────

type RegisterRequest struct {
	Name     string `json:"name"     binding:"required,min=2,max=80"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type GoogleSignInRequest struct {
	Credential string `json:"credential" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// ── Responses ────────────────────────────────────────────────

type BankAccountResponse struct {
	BankCode      string `json:"bankCode"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
}

func toBankAccountResponse(b *models.BankAccount) *BankAccountResponse {
	if b == nil {
		return nil
	}
	return &BankAccountResponse{
		BankCode:      b.BankCode,
		BankName:      b.BankName,
		AccountNumber: b.AccountNumber,
		AccountName:   b.AccountName,
	}
}

type UserResponse struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Email           string            `json:"email"`
	Role            models.Role       `json:"role"`
	Avatar          string            `json:"avatar,omitempty"`
	IsEmailVerified bool              `json:"isEmailVerified"`
	TrustScore      float64           `json:"trustScore"`
	Badges          []models.VerificationBadge `json:"badges"`
	BankAccount     *BankAccountResponse        `json:"bankAccount,omitempty"`
	CreatedAt       string            `json:"createdAt"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken,omitempty"`
}

func toUserResponse(u *models.User) UserResponse {
	badges := u.Badges
	if badges == nil {
		badges = []models.VerificationBadge{}
	}
	return UserResponse{
		ID:              u.ID.Hex(),
		Name:            u.Name,
		Email:           u.Email,
		Role:            u.Role,
		Avatar:          u.Avatar,
		IsEmailVerified: u.IsEmailVerified,
		TrustScore:      u.TrustScore,
		Badges:          badges,
		BankAccount:     toBankAccountResponse(u.BankAccount),
		CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

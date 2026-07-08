package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleBusiness Role = "business"
	RolePromoter Role = "promoter"
)

type VerificationBadge string

const (
	BadgeVerifiedIdentity VerificationBadge = "verified_identity"
	BadgeTrustedPromoter  VerificationBadge = "trusted_promoter"
	BadgeTopCreator       VerificationBadge = "top_creator"
)

// BankAccount is a verified Paystack payout destination. AccountName is
// resolved from the bank via Paystack, never taken from user input, so a
// payout can't be silently misdirected to a name that doesn't match the
// account. RecipientCode caches the Paystack transfer recipient id and must
// be cleared whenever the underlying bank details change.
type BankAccount struct {
	BankCode      string `bson:"bankCode"                json:"bankCode"`
	BankName      string `bson:"bankName"                json:"bankName"`
	AccountNumber string `bson:"accountNumber"            json:"accountNumber"`
	AccountName   string `bson:"accountName"              json:"accountName"`
	RecipientCode string `bson:"recipientCode,omitempty"  json:"-"`
}

type User struct {
	ID              bson.ObjectID       `bson:"_id,omitempty"       json:"id"`
	Email           string              `bson:"email"               json:"email"`
	Password        string              `bson:"password"            json:"-"`
	Role            Role                `bson:"role"                json:"role"`
	Name            string              `bson:"name"                json:"name"`
	Avatar          string              `bson:"avatar"              json:"avatar,omitempty"`
	IsEmailVerified bool                `bson:"isEmailVerified"     json:"isEmailVerified"`
	IsSuspended     bool                `bson:"isSuspended"         json:"isSuspended"`
	TrustScore      float64             `bson:"trustScore"          json:"trustScore"`
	Badges          []VerificationBadge `bson:"badges"              json:"badges"`
	BankAccount     *BankAccount        `bson:"bankAccount,omitempty" json:"bankAccount,omitempty"`

	RefreshToken        string    `bson:"refreshToken"         json:"-"`
	EmailVerifyToken    string    `bson:"emailVerifyToken"     json:"-"`
	PasswordResetToken  string    `bson:"passwordResetToken"   json:"-"`
	PasswordResetExpiry time.Time `bson:"passwordResetExpiry"  json:"-"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

const UsersCollection = "users"

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

	RefreshToken        string    `bson:"refreshToken"         json:"-"`
	EmailVerifyToken    string    `bson:"emailVerifyToken"     json:"-"`
	PasswordResetToken  string    `bson:"passwordResetToken"   json:"-"`
	PasswordResetExpiry time.Time `bson:"passwordResetExpiry"  json:"-"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

const UsersCollection = "users"

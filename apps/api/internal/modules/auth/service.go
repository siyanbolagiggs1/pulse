package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services"
	"github.com/pulse/api/internal/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrEmailTaken        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailNotVerified  = errors.New("please verify your email before logging in")
	ErrAccountSuspended  = errors.New("account suspended — contact support")
	ErrInvalidToken      = errors.New("invalid or expired token")
	ErrTokenBlacklisted  = errors.New("token has been revoked")
)

// register creates a new user, wallet, and sends a verification email.
func register(ctx context.Context, req RegisterRequest) (*models.User, string, error) {
	col := database.GetCollection(models.UsersCollection)

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	// Check email uniqueness
	var existing models.User
	err := col.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		return nil, "", ErrEmailTaken
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, "", err
	}

	hashedPw, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, "", err
	}

	verifyToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	user := &models.User{
		Name:             req.Name,
		Email:            req.Email,
		Password:         hashedPw,
		Role:             req.Role,
		IsEmailVerified:  false,
		IsSuspended:      false,
		TrustScore:       50,
		Badges:           []models.VerificationBadge{},
		EmailVerifyToken: verifyToken,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	result, err := col.InsertOne(ctx, user)
	if err != nil {
		return nil, "", err
	}
	// Re-fetch to get the assigned ObjectID cleanly.
	if err := col.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(user); err != nil {
		return nil, "", err
	}

	// Create wallet
	if err := createWallet(ctx, user); err != nil {
		return nil, "", err
	}

	// Send verification email (non-blocking — log failures, don't fail registration)
	go func() {
		fmt.Printf("\n[DEV] Verification link: %s/verify-email/%s\n", config.App.ClientURL, verifyToken)
		if err := services.SendVerificationEmail(user.Email, user.Name, verifyToken); err != nil {
			fmt.Printf("Warning: could not send verification email to %s: %v\n", user.Email, err)
		}
	}()

	accessToken, err := utils.GenerateAccessToken(user.ID.Hex(), string(user.Role))
	if err != nil {
		return nil, "", err
	}

	return user, accessToken, nil
}

// login verifies credentials and returns tokens.
func login(ctx context.Context, req LoginRequest) (*models.User, string, string, error) {
	col := database.GetCollection(models.UsersCollection)

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	if err := col.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", err
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		return nil, "", "", ErrInvalidCredentials
	}
	if !user.IsEmailVerified {
		return nil, "", "", ErrEmailNotVerified
	}
	if user.IsSuspended {
		return nil, "", "", ErrAccountSuspended
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.Hex(), string(user.Role))
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.Hex(), string(user.Role))
	if err != nil {
		return nil, "", "", err
	}

	// Store refresh token on user document
	_, err = col.UpdateOne(ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"refreshToken": refreshToken, "updatedAt": time.Now().UTC()}},
	)
	if err != nil {
		return nil, "", "", err
	}

	return &user, accessToken, refreshToken, nil
}

// logout blacklists the refresh token in Redis.
func logout(ctx context.Context, refreshToken string) error {
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil // already invalid, nothing to do
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}

	if database.Redis == nil {
		return nil
	}
	key := fmt.Sprintf("blacklist:%s", refreshToken)
	return database.Redis.Set(ctx, key, "1", ttl).Err()
}

// refresh validates a refresh token and returns a new access token.
func refresh(ctx context.Context, refreshToken string) (string, error) {
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", ErrInvalidToken
	}

	// Check Redis blacklist (skipped if Redis unavailable)
	if database.Redis != nil {
		key := fmt.Sprintf("blacklist:%s", refreshToken)
		exists, err := database.Redis.Exists(ctx, key).Result()
		if err != nil {
			return "", err
		}
		if exists > 0 {
			return "", ErrTokenBlacklisted
		}
	}

	// Verify token still matches what's stored on the user
	col := database.GetCollection(models.UsersCollection)
	objID, err := bson.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return "", ErrInvalidToken
	}

	var user models.User
	if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return "", ErrInvalidToken
	}
	if user.RefreshToken != refreshToken {
		return "", ErrInvalidToken
	}
	if user.IsSuspended {
		return "", ErrAccountSuspended
	}

	return utils.GenerateAccessToken(claims.UserID, claims.Role)
}

// verifyEmail finds the user by token and marks them verified.
func verifyEmail(ctx context.Context, token string) error {
	col := database.GetCollection(models.UsersCollection)

	result, err := col.UpdateOne(ctx,
		bson.M{"emailVerifyToken": token, "isEmailVerified": false},
		bson.M{"$set": bson.M{
			"isEmailVerified":  true,
			"emailVerifyToken": "",
			"updatedAt":        time.Now().UTC(),
		}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrInvalidToken
	}
	return nil
}

// forgotPassword generates a reset token and emails it.
func forgotPassword(ctx context.Context, email string) error {
	col := database.GetCollection(models.UsersCollection)

	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := col.FindOne(ctx, bson.M{"email": email}).Decode(&user); err != nil {
		// Return nil even if not found — don't expose whether email exists
		return nil
	}

	token, err := utils.GenerateSecureToken(32)
	if err != nil {
		return err
	}

	expiry := time.Now().UTC().Add(1 * time.Hour)
	_, err = col.UpdateOne(ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"passwordResetToken":  token,
			"passwordResetExpiry": expiry,
			"updatedAt":           time.Now().UTC(),
		}},
	)
	if err != nil {
		return err
	}

	go func() {
		fmt.Printf("\n[DEV] Password reset link: %s/reset-password/%s\n", config.App.ClientURL, token)
		if err := services.SendPasswordResetEmail(user.Email, user.Name, token); err != nil {
			fmt.Printf("Warning: could not send reset email to %s: %v\n", user.Email, err)
		}
	}()

	return nil
}

// resetPassword validates the token and updates the password.
func resetPassword(ctx context.Context, token, newPassword string) error {
	col := database.GetCollection(models.UsersCollection)

	var user models.User
	err := col.FindOne(ctx, bson.M{
		"passwordResetToken":  token,
		"passwordResetExpiry": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrInvalidToken
		}
		return err
	}

	hashed, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Blacklist the current refresh token if any
	if user.RefreshToken != "" {
		_ = logout(ctx, user.RefreshToken)
	}

	_, err = col.UpdateOne(ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"password":            hashed,
			"passwordResetToken":  "",
			"passwordResetExpiry": time.Time{},
			"refreshToken":        "",
			"updatedAt":           time.Now().UTC(),
		}},
	)
	return err
}

// googleSignIn verifies a Google ID token and signs in or creates a user.
func googleSignIn(ctx context.Context, credential, role string) (*models.User, string, error) {
	if config.App.GoogleClientID == "" {
		return nil, "", errors.New("Google sign-in is not configured")
	}

	// Verify the Google ID token.
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + credential)
	if err != nil {
		return nil, "", errors.New("could not verify Google token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("invalid Google token")
	}

	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Aud           string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, "", errors.New("could not parse Google token")
	}
	if info.Aud != config.App.GoogleClientID {
		return nil, "", errors.New("invalid Google token audience")
	}
	if info.Email == "" {
		return nil, "", errors.New("no email returned from Google")
	}

	col := database.GetCollection(models.UsersCollection)
	var user models.User
	err = col.FindOne(ctx, bson.M{"email": info.Email}).Decode(&user)

	if errors.Is(err, mongo.ErrNoDocuments) {
		// New user — create account.
		if role == "" {
			role = "promoter"
		}
		now := time.Now().UTC()
		user = models.User{
			Name:            info.Name,
			Email:           info.Email,
			Password:        "", // no password for Google-auth users
			Role:            models.Role(role),
			IsEmailVerified: true,
			IsSuspended:     false,
			TrustScore:      50,
			Badges:          []models.VerificationBadge{},
			Avatar:          info.Picture,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		result, err := col.InsertOne(ctx, user)
		if err != nil {
			return nil, "", err
		}
		if err := col.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&user); err != nil {
			return nil, "", err
		}
		if err := createWallet(ctx, &user); err != nil {
			return nil, "", err
		}
	} else if err != nil {
		return nil, "", err
	}

	if user.IsSuspended {
		return nil, "", ErrAccountSuspended
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.Hex(), string(user.Role))
	if err != nil {
		return nil, "", err
	}

	return &user, accessToken, nil
}

// createWallet initialises a zero-balance wallet for a new user.
func createWallet(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	wallet := models.Wallet{
		UserID:           user.ID,
		Role:             user.Role,
		AvailableBalance: 0,
		PendingBalance:   0,
		TotalEarned:      0,
		TotalSpent:       0,
		Currency:         "USD",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_, err := database.GetCollection(models.WalletsCollection).InsertOne(ctx, wallet)
	return err
}

// me returns the authenticated user by ID.
func me(ctx context.Context, userID string) (*models.User, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var user models.User
	err = database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// refreshTokenTTL returns remaining duration from a JWT's expiry claim.
func refreshTokenTTL(token string) time.Duration {
	cfg := config.App
	_ = cfg // used indirectly via ValidateRefreshToken
	claims, err := utils.ValidateRefreshToken(token)
	if err != nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}

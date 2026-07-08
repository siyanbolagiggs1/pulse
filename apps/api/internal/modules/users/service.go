package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/paystack"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const reconnectCooldown = 30 * 24 * time.Hour

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrAccountNotFound        = errors.New("social account not found")
	ErrDuplicatePlatform      = errors.New("a social account for this platform is already connected")
	ErrDuplicateSocialAccount = errors.New("this social account is already connected to another user")
	ErrCooldownActive         = errors.New("you can request re-verification once every 30 days")
	ErrPaystackNotConfigured  = errors.New("payment processing is not configured — add PAYSTACK_SECRET_KEY")
	ErrInvalidBankAccount     = errors.New("could not verify this bank account — check the account number and bank")
)

func getMe(ctx context.Context, userID string) (*models.User, []models.SocialAccount, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	cursor, err := database.GetCollection(models.SocialAccountsCollection).
		Find(ctx, bson.M{"userId": objID})
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	var accounts []models.SocialAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, nil, err
	}

	return &user, accounts, nil
}

func updateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*models.User, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	fields := bson.M{"updatedAt": time.Now().UTC()}
	if req.Name != "" {
		fields["name"] = req.Name
	}
	if req.Avatar != "" {
		fields["avatar"] = req.Avatar
	}

	var user models.User
	err = database.GetCollection(models.UsersCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$set": fields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func connectSocialAccount(ctx context.Context, userID string, req ConnectSocialAccountRequest) (*models.SocialAccount, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	col := database.GetCollection(models.SocialAccountsCollection)

	// Normalise URL: lowercase + strip trailing slash.
	profileURL := strings.ToLower(strings.TrimRight(strings.TrimSpace(req.ProfileURL), "/"))

	// Derive username from the URL path's last segment (strip leading @).
	username := req.Username
	if username == "" {
		parts := strings.Split(profileURL, "/")
		username = strings.TrimPrefix(parts[len(parts)-1], "@")
	}
	username = strings.ToLower(strings.TrimSpace(username))

	var existing models.SocialAccount

	// Check: this user already has an account on this platform (active or
	// soft-disconnected — the unique userId+platform index guarantees at
	// most one document either way).
	err = col.FindOne(ctx, bson.M{"userId": objID, "platform": req.Platform}).Decode(&existing)
	if err == nil {
		if existing.DisconnectedAt == nil {
			return nil, ErrDuplicatePlatform
		}
		return reconnectSocialAccount(ctx, col, &existing, profileURL, username)
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	// Check: another user already linked this profile URL.
	// Case-insensitive + optional trailing slash to catch old records stored with different formatting.
	if err := findProfileURLCollision(ctx, col, profileURL, nil); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	acc := models.SocialAccount{
		UserID:         objID,
		Platform:       req.Platform,
		Username:       username,
		PlatformUserID: username,
		ProfileURL:     profileURL,
		Status:         models.SocialAccountPending,
		LastSyncedAt:   now,
		CreatedAt:      now,
	}

	result, err := col.InsertOne(ctx, acc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateSocialAccount
		}
		return nil, err
	}
	_ = col.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&acc)

	return &acc, nil
}

// findProfileURLCollision returns ErrDuplicateSocialAccount if profileURL is
// already claimed by a different document (excludeID, if non-nil, is skipped
// so a reconnecting account doesn't collide with its own unchanged URL).
func findProfileURLCollision(ctx context.Context, col *mongo.Collection, profileURL string, excludeID *bson.ObjectID) error {
	urlPattern := "^" + regexp.QuoteMeta(profileURL) + "/?$"
	filter := bson.M{"profileUrl": bson.Regex{Pattern: urlPattern, Options: "i"}}
	if excludeID != nil {
		filter["_id"] = bson.M{"$ne": *excludeID}
	}

	var existing models.SocialAccount
	err := col.FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		return ErrDuplicateSocialAccount
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}
	return nil
}

// reconnectSocialAccount handles a connect request for a social account the
// user previously soft-disconnected. Enforces a cooldown since the last
// admin-verified follower count, then reuses the same document (preserving
// FollowerCount/Tier/InfluenceScore/FollowerHistory) rather than inserting a
// new one, sending it back into the admin review queue.
func reconnectSocialAccount(ctx context.Context, col *mongo.Collection, existing *models.SocialAccount, profileURL, username string) (*models.SocialAccount, error) {
	if existing.LastVerifiedAt != nil {
		elapsed := time.Since(*existing.LastVerifiedAt)
		if elapsed < reconnectCooldown {
			nextEligible := existing.LastVerifiedAt.Add(reconnectCooldown)
			return nil, fmt.Errorf("%w — next eligible %s", ErrCooldownActive, nextEligible.Format("2006-01-02"))
		}
	}

	if err := findProfileURLCollision(ctx, col, profileURL, &existing.ID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	_, err := col.UpdateOne(ctx,
		bson.M{"_id": existing.ID},
		bson.M{"$set": bson.M{
			"status":         models.SocialAccountPending,
			"username":       username,
			"platformUserId": username,
			"profileUrl":     profileURL,
			"disconnectedAt": nil,
			"lastSyncedAt":   now,
		}},
	)
	if err != nil {
		return nil, err
	}

	_ = col.FindOne(ctx, bson.M{"_id": existing.ID}).Decode(existing)
	return existing, nil
}

// searchUsers powers the chat recipient picker. It always forces the
// returned role to be the opposite of the caller's own role — a business can
// only ever find promoters and vice versa — regardless of anything the
// caller might pass in. Callers of any other role (i.e. admin, who doesn't
// participate in chat) get an empty result.
func searchUsers(ctx context.Context, callerRole, query string, limit int) ([]models.User, error) {
	if limit < 1 || limit > 20 {
		limit = 20
	}

	var targetRole models.Role
	switch callerRole {
	case string(models.RoleBusiness):
		targetRole = models.RolePromoter
	case string(models.RolePromoter):
		targetRole = models.RoleBusiness
	default:
		return []models.User{}, nil
	}

	filter := bson.M{"role": targetRole, "isSuspended": false}
	if query != "" {
		// QuoteMeta escapes regex metacharacters — an unescaped user string
		// here is a NoSQL regex-injection / ReDoS vector, and this endpoint
		// is reachable by any authenticated business/promoter.
		filter["name"] = bson.M{"$regex": regexp.QuoteMeta(query), "$options": "i"}
	}

	cursor, err := database.GetCollection(models.UsersCollection).Find(ctx, filter,
		options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// listBanks proxies Paystack's bank list for the platform's configured
// currency so the frontend can offer a bank picker for bank-account setup.
func listBanks(ctx context.Context) ([]paystack.Bank, error) {
	if config.App.PaystackSecretKey == "" {
		return nil, ErrPaystackNotConfigured
	}
	return paystack.ListBanks(config.App.PaystackSecretKey, config.App.PaystackCurrency)
}

// setBankAccount verifies the account via Paystack (so a payout destination
// is always a real, named account — never an unverified user-typed string)
// and persists it. Any previously cached Paystack transfer recipient is
// dropped since it belongs to the old account details.
func setBankAccount(ctx context.Context, userID string, req SetBankAccountRequest) (*models.BankAccount, error) {
	if config.App.PaystackSecretKey == "" {
		return nil, ErrPaystackNotConfigured
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	resolved, err := paystack.ResolveAccount(config.App.PaystackSecretKey, req.AccountNumber, req.BankCode)
	if err != nil {
		return nil, ErrInvalidBankAccount
	}

	banks, err := paystack.ListBanks(config.App.PaystackSecretKey, config.App.PaystackCurrency)
	if err != nil {
		return nil, err
	}
	bankName := req.BankCode
	for _, b := range banks {
		if b.Code == req.BankCode {
			bankName = b.Name
			break
		}
	}

	bankAccount := models.BankAccount{
		BankCode:      req.BankCode,
		BankName:      bankName,
		AccountNumber: resolved.AccountNumber,
		AccountName:   resolved.AccountName,
	}

	_, err = database.GetCollection(models.UsersCollection).UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{
			"$set": bson.M{"bankAccount": bankAccount, "updatedAt": time.Now().UTC()},
		},
	)
	if err != nil {
		return nil, err
	}

	return &bankAccount, nil
}

func deleteSocialAccount(ctx context.Context, userID, accountID string) error {
	userObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrAccountNotFound
	}
	accObjID, err := bson.ObjectIDFromHex(accountID)
	if err != nil {
		return ErrAccountNotFound
	}

	now := time.Now().UTC()
	result, err := database.GetCollection(models.SocialAccountsCollection).
		UpdateOne(ctx,
			bson.M{"_id": accObjID, "userId": userObjID},
			bson.M{"$set": bson.M{"disconnectedAt": now}},
		)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrAccountNotFound
	}
	return nil
}

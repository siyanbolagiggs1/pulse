package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/scoring"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrAccountNotFound        = errors.New("social account not found")
	ErrAccountAgeTooLow       = errors.New("account must be at least 30 days old to be eligible")
	ErrDuplicatePlatform      = errors.New("a social account for this platform is already connected")
	ErrDuplicateSocialAccount = errors.New("this social account is already connected to another user")
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

	username := strings.ToLower(strings.TrimSpace(req.Username))

	var existing models.SocialAccount
	err = col.FindOne(ctx, bson.M{"userId": objID, "platform": req.Platform}).Decode(&existing)
	if err == nil {
		return nil, ErrDuplicatePlatform
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	// All platforms use admin review — Twitter auto-lookup requires paid API access.
	if req.ProfileURL == "" {
		return nil, fmt.Errorf("profile URL is required")
	}

	now := time.Now().UTC()
	acc := models.SocialAccount{
		UserID:         objID,
		Platform:       req.Platform,
		Username:       username,
		PlatformUserID: username,
		ProfileURL:     req.ProfileURL,
		Status:         models.SocialAccountPending,
		LastSyncedAt:   now,
		CreatedAt:      now,
	}

	result, err := col.InsertOne(ctx, acc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			errStr := err.Error()
			if strings.Contains(errStr, "platform_username_unique") {
				return nil, ErrDuplicateSocialAccount
			}
			return nil, ErrDuplicatePlatform
		}
		return nil, err
	}
	_ = col.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&acc)

	return &acc, nil
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

	result, err := database.GetCollection(models.SocialAccountsCollection).
		DeleteOne(ctx, bson.M{"_id": accObjID, "userId": userObjID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func getInfluenceScore(ctx context.Context, userID string) (*InfluenceScoreResponse, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	cursor, err := database.GetCollection(models.SocialAccountsCollection).
		Find(ctx, bson.M{"userId": objID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accounts []models.SocialAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}

	resp := &InfluenceScoreResponse{
		Accounts: make([]AccountInfluenceScore, 0, len(accounts)),
	}

	// Completion score is per-promoter, not per-account — compute once.
	cs := scoring.ComputeCompletionScore(ctx, objID)

	for _, acc := range accounts {
		fs := scoring.ScoreFollowers(acc.FollowerCount)
		es := scoring.ScoreEngagement(acc.EngagementRate)
		as := scoring.ScoreAge(acc.AccountAge)
		qs := scoring.ScoreAudienceQuality(acc.FollowerCount, acc.FollowingCount)

		overall := scoring.Round2(fs + es + as + cs + qs)
		resp.Accounts = append(resp.Accounts, AccountInfluenceScore{
			AccountID:         acc.ID.Hex(),
			Platform:          acc.Platform,
			Username:          acc.Username,
			OverallScore:      overall,
			FollowerScore:     scoring.Round2(fs),
			EngagementScore:   scoring.Round2(es),
			AccountAgeScore:   scoring.Round2(as),
			CompletionScore:   scoring.Round2(cs),
			AudienceQualScore: scoring.Round2(qs),
		})
		if overall > resp.BestScore {
			resp.BestScore = overall
		}
	}

	return resp, nil
}

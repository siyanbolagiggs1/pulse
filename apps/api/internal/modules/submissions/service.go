package submissions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/modules/notifications"
	"github.com/pulse/api/internal/services/fraud"
	"github.com/pulse/api/internal/services/scoring"
	"github.com/pulse/api/internal/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrCampaignNotActive  = errors.New("campaign is not accepting submissions")
	ErrCampaignFull       = errors.New("campaign has reached its maximum participants")
	ErrCampaignExpired    = errors.New("campaign has ended")
	ErrPlatformMismatch   = errors.New("social account platform does not match campaign platform")
	ErrEligibility        = errors.New("social account does not meet campaign eligibility requirements")
	ErrAlreadySubmitted   = errors.New("you have already submitted to this campaign")
	ErrDuplicateRepostURL = errors.New("this repost URL has already been submitted by another promoter")
	ErrRateLimited        = errors.New("submission rate limit exceeded — maximum 3 submissions per hour")
	ErrNotReviewable      = errors.New("submission is not in a reviewable state")
	ErrAccountSuspended   = errors.New("your account is suspended — contact support")
)

// ── Submission creation ──────────────────────────────────────

func createSubmission(ctx context.Context, promoterID string, req CreateSubmissionRequest) (*models.CampaignSubmission, error) {
	promoterObjID, err := bson.ObjectIDFromHex(promoterID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}

	// Check promoter is not suspended.
	var promoter models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": promoterObjID}).Decode(&promoter); err != nil {
		return nil, ErrSubmissionNotFound
	}
	if promoter.IsSuspended {
		return nil, ErrAccountSuspended
	}
	if promoter.TrustScore < 20 {
		return nil, ErrAccountSuspended
	}

	// Rate limit: max 3 submissions per hour per promoter.
	if err := checkRateLimit(ctx, promoterID); err != nil {
		return nil, err
	}

	// Load campaign.
	campObjID, err := bson.ObjectIDFromHex(req.CampaignID)
	if err != nil {
		return nil, ErrCampaignNotActive
	}
	var campaign models.Campaign
	if err := database.GetCollection(models.CampaignsCollection).
		FindOne(ctx, bson.M{"_id": campObjID}).Decode(&campaign); err != nil {
		return nil, ErrCampaignNotActive
	}
	if campaign.Status != models.CampaignStatusActive {
		return nil, ErrCampaignNotActive
	}
	if time.Now().UTC().After(campaign.EndDate) {
		return nil, ErrCampaignExpired
	}
	if campaign.CurrentParticipants >= campaign.MaxParticipants {
		return nil, ErrCampaignFull
	}

	// Load social account and verify it belongs to this promoter.
	accObjID, err := bson.ObjectIDFromHex(req.SocialAccountID)
	if err != nil {
		return nil, ErrEligibility
	}
	var account models.SocialAccount
	if err := database.GetCollection(models.SocialAccountsCollection).
		FindOne(ctx, bson.M{"_id": accObjID, "userId": promoterObjID}).Decode(&account); err != nil {
		return nil, ErrEligibility
	}

	// Platform must match campaign.
	if account.Platform != campaign.Platform {
		return nil, ErrPlatformMismatch
	}

	// Eligibility requirements.
	if account.FollowerCount < campaign.MinFollowers ||
		account.InfluenceScore < campaign.MinInfluenceScore {
		return nil, ErrEligibility
	}

	// Fraud analysis (async flags — does not block the submission).
	fraud.CheckSubmission(context.Background(), promoterObjID, &account)

	// Payout calculation.
	influenceMultiplier := 0.5 + (account.InfluenceScore / 100)
	finalAmount := campaign.BaseRepostRate * influenceMultiplier
	platformFee := finalAmount * config.App.PlatformCommission
	promoterEarning := finalAmount - platformFee

	now := time.Now().UTC()
	submission := &models.CampaignSubmission{
		CampaignID:          campObjID,
		PromoterID:          promoterObjID,
		BusinessID:          campaign.BusinessID,
		RepostURL:           req.RepostURL,
		ScreenshotURL:       req.ScreenshotURL,
		Status:              models.SubmissionStatusPending,
		BaseAmount:          campaign.BaseRepostRate,
		InfluenceMultiplier: scoring.Round2(influenceMultiplier),
		FinalAmount:         scoring.Round2(finalAmount),
		PlatformFee:         scoring.Round2(platformFee),
		PromoterEarning:     scoring.Round2(promoterEarning),
		SubmittedAt:         now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	col := database.GetCollection(models.SubmissionsCollection)
	result, err := col.InsertOne(ctx, submission)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			errStr := err.Error()
			if containsIndex(errStr, "campaign_promoter_unique") {
				return nil, ErrAlreadySubmitted
			}
			return nil, ErrDuplicateRepostURL
		}
		return nil, err
	}

	// Increment campaign participants.
	_, _ = database.GetCollection(models.CampaignsCollection).UpdateOne(ctx,
		bson.M{"_id": campObjID},
		bson.M{"$inc": bson.M{"currentParticipants": 1}, "$set": bson.M{"updatedAt": now}},
	)

	// Re-fetch.
	if err := col.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(submission); err != nil {
		return nil, err
	}

	return submission, nil
}

// ── Listing ──────────────────────────────────────────────────

func getSubmissions(ctx context.Context, userID, role string, q SubmissionListQuery) ([]models.CampaignSubmission, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 50 {
		q.Limit = 20
	}

	filter := bson.M{}

	// Scope by role: promoters see only their own; businesses see only theirs.
	switch role {
	case string(models.RolePromoter):
		objID, _ := bson.ObjectIDFromHex(userID)
		filter["promoterId"] = objID
	case string(models.RoleBusiness):
		objID, _ := bson.ObjectIDFromHex(userID)
		filter["businessId"] = objID
	}
	// admin sees all

	if q.CampaignID != "" {
		if id, err := bson.ObjectIDFromHex(q.CampaignID); err == nil {
			filter["campaignId"] = id
		}
	}
	if q.PromoterID != "" && role == string(models.RoleAdmin) {
		if id, err := bson.ObjectIDFromHex(q.PromoterID); err == nil {
			filter["promoterId"] = id
		}
	}
	if q.Status != "" {
		filter["status"] = q.Status
	}

	col := database.GetCollection(models.SubmissionsCollection)
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((q.Page - 1) * q.Limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "submittedAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(q.Limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var submissions []models.CampaignSubmission
	if err := cursor.All(ctx, &submissions); err != nil {
		return nil, 0, err
	}

	return submissions, total, nil
}

func getSubmission(ctx context.Context, id, userID, role string) (*models.CampaignSubmission, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}

	filter := bson.M{"_id": objID}

	// Scope access.
	userObjID, _ := bson.ObjectIDFromHex(userID)
	switch role {
	case string(models.RolePromoter):
		filter["promoterId"] = userObjID
	case string(models.RoleBusiness):
		filter["businessId"] = userObjID
	}

	var s models.CampaignSubmission
	if err := database.GetCollection(models.SubmissionsCollection).
		FindOne(ctx, filter).Decode(&s); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	return &s, nil
}

// ── Review ───────────────────────────────────────────────────

func approveSubmission(ctx context.Context, adminID, submissionID string) (*models.CampaignSubmission, error) {
	adminObjID, _ := bson.ObjectIDFromHex(adminID)
	subObjID, err := bson.ObjectIDFromHex(submissionID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}

	var s models.CampaignSubmission
	if err := database.GetCollection(models.SubmissionsCollection).
		FindOne(ctx, bson.M{"_id": subObjID}).Decode(&s); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	if s.Status != models.SubmissionStatusPending {
		return nil, ErrNotReviewable
	}

	now := time.Now().UTC()
	payoutRelease := now.Add(48 * time.Hour)

	// Update submission.
	if _, err := database.GetCollection(models.SubmissionsCollection).UpdateOne(ctx,
		bson.M{"_id": subObjID},
		bson.M{"$set": bson.M{
			"status":           models.SubmissionStatusApproved,
			"reviewedBy":       adminObjID,
			"reviewedAt":       now,
			"payoutReleasedAt": payoutRelease,
			"updatedAt":        now,
		}},
	); err != nil {
		return nil, err
	}

	// Business: pendingBalance -= finalAmount (paid out).
	_, _ = database.GetCollection(models.WalletsCollection).UpdateOne(ctx,
		bson.M{"userId": s.BusinessID},
		bson.M{
			"$inc": bson.M{"pendingBalance": -s.FinalAmount},
			"$set": bson.M{"updatedAt": now},
		},
	)

	// Campaign: remainingBudget -= finalAmount.
	_, _ = database.GetCollection(models.CampaignsCollection).UpdateOne(ctx,
		bson.M{"_id": s.CampaignID},
		bson.M{
			"$inc": bson.M{"remainingBudget": -s.FinalAmount},
			"$set": bson.M{"updatedAt": now},
		},
	)

	// Promoter: pendingBalance += promoterEarning (holds for 48h).
	_, _ = database.GetCollection(models.WalletsCollection).UpdateOne(ctx,
		bson.M{"userId": s.PromoterID},
		bson.M{
			"$inc": bson.M{"pendingBalance": s.PromoterEarning},
			"$set": bson.M{"updatedAt": now},
		},
	)

	// Record promoter payout transaction.
	var promoterWallet models.Wallet
	if err := database.GetCollection(models.WalletsCollection).
		FindOne(ctx, bson.M{"userId": s.PromoterID}).Decode(&promoterWallet); err == nil {
		tx := models.Transaction{
			WalletID:     promoterWallet.ID,
			UserID:       s.PromoterID,
			Type:         models.TxPayoutPending,
			Amount:       s.PromoterEarning,
			BalanceAfter: promoterWallet.PendingBalance + s.PromoterEarning,
			ReferenceID:  s.ID.Hex(),
			Description:  fmt.Sprintf("Payout pending (releases %s)", payoutRelease.Format("2006-01-02")),
			CreatedAt:    now,
		}
		_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)
	}

	// Promoter trust score +5.
	_, _ = database.GetCollection(models.UsersCollection).UpdateOne(ctx,
		bson.M{"_id": s.PromoterID},
		bson.M{"$inc": bson.M{"trustScore": 5}, "$set": bson.M{"updatedAt": now}},
	)

	// Refresh promoter's influence scores to reflect the updated completion rate.
	go scoring.RefreshAllAccounts(context.Background(), s.PromoterID)

	go notifications.Send(context.Background(), s.PromoterID, models.NotifSubmissionApproved,
		"Submission Approved",
		fmt.Sprintf("Your submission was approved. %.2f USD is pending release in 48h.", s.PromoterEarning),
		map[string]interface{}{"submissionId": s.ID.Hex(), "amount": s.PromoterEarning})

	// Re-fetch updated submission.
	if err := database.GetCollection(models.SubmissionsCollection).
		FindOne(ctx, bson.M{"_id": subObjID}).Decode(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

func rejectSubmission(ctx context.Context, adminID, submissionID, reason string) (*models.CampaignSubmission, error) {
	adminObjID, _ := bson.ObjectIDFromHex(adminID)
	subObjID, err := bson.ObjectIDFromHex(submissionID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}

	var s models.CampaignSubmission
	if err := database.GetCollection(models.SubmissionsCollection).
		FindOne(ctx, bson.M{"_id": subObjID}).Decode(&s); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	if s.Status != models.SubmissionStatusPending {
		return nil, ErrNotReviewable
	}

	now := time.Now().UTC()

	// Update submission.
	if _, err := database.GetCollection(models.SubmissionsCollection).UpdateOne(ctx,
		bson.M{"_id": subObjID},
		bson.M{"$set": bson.M{
			"status":          models.SubmissionStatusRejected,
			"rejectionReason": reason,
			"reviewedBy":      adminObjID,
			"reviewedAt":      now,
			"updatedAt":       now,
		}},
	); err != nil {
		return nil, err
	}

	// Re-open campaign slot.
	_, _ = database.GetCollection(models.CampaignsCollection).UpdateOne(ctx,
		bson.M{"_id": s.CampaignID},
		bson.M{
			"$inc": bson.M{"currentParticipants": -1},
			"$set": bson.M{"updatedAt": now},
		},
	)

	// Promoter trust score -15; auto-suspend at < 20.
	var promoterUser models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": s.PromoterID}).Decode(&promoterUser); err == nil {
		newScore := promoterUser.TrustScore - 15
		update := bson.M{"$set": bson.M{"trustScore": newScore, "updatedAt": now}}
		if newScore < 20 {
			update["$set"].(bson.M)["isSuspended"] = true
		}
		_, _ = database.GetCollection(models.UsersCollection).UpdateOne(ctx,
			bson.M{"_id": s.PromoterID}, update)
	}

	// Refresh promoter's influence scores to reflect the updated completion rate.
	go scoring.RefreshAllAccounts(context.Background(), s.PromoterID)

	go notifications.Send(context.Background(), s.PromoterID, models.NotifSubmissionRejected,
		"Submission Rejected",
		fmt.Sprintf("Your submission was rejected: %s", reason),
		map[string]interface{}{"submissionId": s.ID.Hex(), "reason": reason})

	// Re-fetch updated submission.
	if err := database.GetCollection(models.SubmissionsCollection).
		FindOne(ctx, bson.M{"_id": subObjID}).Decode(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

// ── File upload ──────────────────────────────────────────────

func saveScreenshot(file multipart.File, header *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}

	token, err := utils.GenerateSecureToken(16)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), token, ext)
	dir := filepath.Join(config.App.UploadDir, "screenshots")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	dest := filepath.Join(dir, filename)
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	return "/uploads/screenshots/" + filename, nil
}

// ── Helpers ──────────────────────────────────────────────────

func checkRateLimit(ctx context.Context, promoterID string) error {
	if database.Redis == nil {
		return nil // rate limiting disabled when Redis unavailable
	}
	key := fmt.Sprintf("rate:submissions:%s", promoterID)
	count, err := database.Redis.Incr(ctx, key).Result()
	if err != nil {
		return nil // fail open on Redis error
	}
	if count == 1 {
		database.Redis.Expire(ctx, key, time.Hour)
	}
	if count > 3 {
		return ErrRateLimited
	}
	return nil
}

func containsIndex(errMsg, indexName string) bool {
	return len(errMsg) >= len(indexName) && containsStr(errMsg, indexName)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

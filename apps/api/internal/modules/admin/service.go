package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/modules/notifications"
	"github.com/pulse/api/internal/modules/wallet"
	"github.com/pulse/api/internal/services/scoring"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrNotReviewable = errors.New("resource is not in a reviewable state")
)

// translateWithdrawalErr maps wallet package sentinel errors onto admin's
// own so admin/handler.go's existing error switches don't need to know
// about the wallet package's internals.
func translateWithdrawalErr(err error) error {
	switch {
	case errors.Is(err, wallet.ErrWithdrawalNotFound):
		return ErrNotFound
	case errors.Is(err, wallet.ErrNotReviewable):
		return ErrNotReviewable
	default:
		return err
	}
}

// ── Platform stats ───────────────────────────────────────────

func getPlatformStats(ctx context.Context) (*PlatformStats, error) {
	usersCol := database.GetCollection(models.UsersCollection)
	campaignsCol := database.GetCollection(models.CampaignsCollection)
	subsCol := database.GetCollection(models.SubmissionsCollection)
	withdrawalsCol := database.GetCollection(models.WithdrawalsCollection)
	walletsCol := database.GetCollection(models.WalletsCollection)

	totalUsers, _ := usersCol.CountDocuments(ctx, bson.M{})
	totalBiz, _ := usersCol.CountDocuments(ctx, bson.M{"role": models.RoleBusiness})
	totalPromoters, _ := usersCol.CountDocuments(ctx, bson.M{"role": models.RolePromoter})
	suspended, _ := usersCol.CountDocuments(ctx, bson.M{"isSuspended": true})

	totalCampaigns, _ := campaignsCol.CountDocuments(ctx, bson.M{})
	activeCampaigns, _ := campaignsCol.CountDocuments(ctx, bson.M{"status": models.CampaignStatusActive})
	draftCampaigns, _ := campaignsCol.CountDocuments(ctx, bson.M{"status": models.CampaignStatusDraft})
	completedCampaigns, _ := campaignsCol.CountDocuments(ctx, bson.M{"status": models.CampaignStatusCompleted})

	totalSubs, _ := subsCol.CountDocuments(ctx, bson.M{})
	pendingSubs, _ := subsCol.CountDocuments(ctx, bson.M{"status": models.SubmissionStatusPending})
	approvedSubs, _ := subsCol.CountDocuments(ctx, bson.M{"status": models.SubmissionStatusApproved})
	rejectedSubs, _ := subsCol.CountDocuments(ctx, bson.M{"status": models.SubmissionStatusRejected})

	// Sum pending withdrawals.
	pendingWithdrawalPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"status": models.WithdrawalPending}}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$amount"}}}},
	}
	var pendingWResult []struct{ Total float64 `bson:"total"` }
	if cur, err := withdrawalsCol.Aggregate(ctx, pendingWithdrawalPipeline); err == nil {
		_ = cur.All(ctx, &pendingWResult)
	}
	var pendingWithdrawals float64
	if len(pendingWResult) > 0 {
		pendingWithdrawals = pendingWResult[0].Total
	}

	// Sum completed withdrawals.
	completedWithdrawalPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"status": models.WithdrawalCompleted}}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$netAmount"}}}},
	}
	var completedWResult []struct{ Total float64 `bson:"total"` }
	if cur, err := withdrawalsCol.Aggregate(ctx, completedWithdrawalPipeline); err == nil {
		_ = cur.All(ctx, &completedWResult)
	}
	var totalWithdrawn float64
	if len(completedWResult) > 0 {
		totalWithdrawn = completedWResult[0].Total
	}

	// Sum promoter pending balances.
	promoterPendingPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"role": models.RolePromoter}}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$pendingBalance"}}}},
	}
	var promoterPendingResult []struct{ Total float64 `bson:"total"` }
	if cur, err := walletsCol.Aggregate(ctx, promoterPendingPipeline); err == nil {
		_ = cur.All(ctx, &promoterPendingResult)
	}
	var totalPromoterPending float64
	if len(promoterPendingResult) > 0 {
		totalPromoterPending = promoterPendingResult[0].Total
	}

	return &PlatformStats{
		Users: UserStats{
			Total:      totalUsers,
			Businesses: totalBiz,
			Promoters:  totalPromoters,
			Suspended:  suspended,
		},
		Campaigns: CampaignStats{
			Total:     totalCampaigns,
			Active:    activeCampaigns,
			Draft:     draftCampaigns,
			Completed: completedCampaigns,
		},
		Submissions: SubmissionStats{
			Total:    totalSubs,
			Pending:  pendingSubs,
			Approved: approvedSubs,
			Rejected: rejectedSubs,
		},
		Financials: FinancialStats{
			TotalPendingWithdrawals: pendingWithdrawals,
			TotalWithdrawn:          totalWithdrawn,
			TotalPromoterPending:    totalPromoterPending,
		},
	}, nil
}

// ── User management ──────────────────────────────────────────

func listUsers(ctx context.Context, q UserListQuery) ([]models.User, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	filter := bson.M{}
	if q.Role != "" {
		filter["role"] = q.Role
	}
	if q.Suspended == "true" {
		filter["isSuspended"] = true
	} else if q.Suspended == "false" {
		filter["isSuspended"] = false
	}
	if q.Search != "" {
		// QuoteMeta escapes regex metacharacters so the search term is
		// matched literally — an unescaped user string here is a NoSQL
		// regex-injection / ReDoS vector (e.g. "(a+)+$").
		pattern := regexp.QuoteMeta(q.Search)
		filter["$or"] = bson.A{
			bson.M{"email": bson.M{"$regex": pattern, "$options": "i"}},
			bson.M{"name": bson.M{"$regex": pattern, "$options": "i"}},
		}
	}

	col := database.GetCollection(models.UsersCollection)
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((q.Page - 1) * q.Limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(q.Limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func getUser(ctx context.Context, userID string) (*models.User, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrNotFound
	}

	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func suspendUser(ctx context.Context, adminID, userID, reason string) error {
	adminObjID, _ := bson.ObjectIDFromHex(adminID)
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrNotFound
	}

	now := time.Now().UTC()
	result, err := database.GetCollection(models.UsersCollection).UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"isSuspended": true, "updatedAt": now}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	// Record a fraud flag for audit trail.
	flag := models.FraudFlag{
		UserID:    objID,
		Reason:    models.FraudFlagReason("admin_suspension"),
		Details:   reason,
		Resolved:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = adminObjID // referenced via resolvedBy if needed later
	_, _ = database.GetCollection(models.FraudFlagsCollection).InsertOne(ctx, flag)

	return nil
}

func unsuspendUser(ctx context.Context, userID string) error {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrNotFound
	}

	now := time.Now().UTC()
	result, err := database.GetCollection(models.UsersCollection).UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{
			"isSuspended": false,
			"trustScore":  50, // reset to neutral on reinstatement
			"updatedAt":   now,
		}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

// ── Fraud flags ──────────────────────────────────────────────

func listFraudFlags(ctx context.Context, q FraudFlagQuery) ([]models.FraudFlag, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	filter := bson.M{}
	if q.UserID != "" {
		if id, err := bson.ObjectIDFromHex(q.UserID); err == nil {
			filter["userId"] = id
		}
	}
	if q.Resolved == "true" {
		filter["resolved"] = true
	} else if q.Resolved == "false" {
		filter["resolved"] = false
	}

	col := database.GetCollection(models.FraudFlagsCollection)
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((q.Page - 1) * q.Limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(q.Limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var flags []models.FraudFlag
	if err := cursor.All(ctx, &flags); err != nil {
		return nil, 0, err
	}

	return flags, total, nil
}

func resolveFraudFlag(ctx context.Context, adminID, flagID string) error {
	adminObjID, err := bson.ObjectIDFromHex(adminID)
	if err != nil {
		return ErrNotFound
	}
	flagObjID, err := bson.ObjectIDFromHex(flagID)
	if err != nil {
		return ErrNotFound
	}

	now := time.Now().UTC()
	result, err := database.GetCollection(models.FraudFlagsCollection).UpdateOne(ctx,
		bson.M{"_id": flagObjID, "resolved": false},
		bson.M{"$set": bson.M{
			"resolved":   true,
			"resolvedBy": adminObjID,
			"updatedAt":  now,
		}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

// ── Withdrawal approval ──────────────────────────────────────

func listWithdrawals(ctx context.Context, q WithdrawalQuery) ([]models.Withdrawal, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	filter := bson.M{}
	if q.UserID != "" {
		if id, err := bson.ObjectIDFromHex(q.UserID); err == nil {
			filter["userId"] = id
		}
	}
	if q.Status != "" {
		filter["status"] = q.Status
	}

	col := database.GetCollection(models.WithdrawalsCollection)
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((q.Page - 1) * q.Limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "requestedAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(q.Limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var withdrawals []models.Withdrawal
	if err := cursor.All(ctx, &withdrawals); err != nil {
		return nil, 0, err
	}

	return withdrawals, total, nil
}

func approveWithdrawal(ctx context.Context, withdrawalID string) (*models.Withdrawal, error) {
	w, err := wallet.AdminApproveWithdrawal(ctx, withdrawalID)
	if err != nil {
		return nil, translateWithdrawalErr(err)
	}
	return w, nil
}

func rejectWithdrawal(ctx context.Context, withdrawalID, reason string) (*models.Withdrawal, error) {
	w, err := wallet.AdminRejectWithdrawal(ctx, withdrawalID, reason)
	if err != nil {
		return nil, translateWithdrawalErr(err)
	}
	return w, nil
}

// ── Social account review ────────────────────────────────────

func listPendingSocialAccounts(ctx context.Context, page, limit int) ([]models.SocialAccount, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	col := database.GetCollection(models.SocialAccountsCollection)
	filter := bson.M{"status": models.SocialAccountPending}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: 1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var accounts []models.SocialAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

func approveSocialAccount(ctx context.Context, accountID string, req ApproveSocialAccountRequest) (*models.SocialAccount, error) {
	accObjID, err := bson.ObjectIDFromHex(accountID)
	if err != nil {
		return nil, ErrNotFound
	}

	col := database.GetCollection(models.SocialAccountsCollection)
	var acc models.SocialAccount
	if err := col.FindOne(ctx, bson.M{"_id": accObjID, "status": models.SocialAccountPending}).Decode(&acc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	tier := scoring.FollowerTier(req.FollowerCount)

	acc.FollowerCount = req.FollowerCount
	acc.Tier = tier
	acc.Status = models.SocialAccountActive
	acc.InfluenceScore = scoring.ComputeFullScore(ctx, &acc, acc.UserID)

	now := time.Now().UTC()
	historyEntry := models.FollowerHistoryEntry{
		FollowerCount: req.FollowerCount,
		Tier:          tier,
		RecordedAt:    now,
	}
	acc.LastVerifiedAt = &now
	acc.DisconnectedAt = nil
	acc.FollowerHistory = append(acc.FollowerHistory, historyEntry)

	_, err = col.UpdateOne(ctx,
		bson.M{"_id": accObjID},
		bson.M{
			"$set": bson.M{
				"status":         models.SocialAccountActive,
				"followerCount":  req.FollowerCount,
				"tier":           tier,
				"influenceScore": acc.InfluenceScore,
				"lastVerifiedAt": now,
				"lastSyncedAt":   now,
				"disconnectedAt": nil,
			},
			"$push": bson.M{"followerHistory": historyEntry},
		},
	)
	if err != nil {
		return nil, err
	}

	_ = col.FindOne(ctx, bson.M{"_id": accObjID}).Decode(&acc)

	go notifications.Send(ctx, acc.UserID, models.NotifSocialAccountApproved,
		"Social account verified",
		fmt.Sprintf("Your %s account @%s has been verified and is now active.", acc.Platform, acc.Username),
		map[string]interface{}{"accountId": acc.ID.Hex(), "platform": string(acc.Platform)},
	)

	return &acc, nil
}

func rejectSocialAccount(ctx context.Context, accountID, reason string) error {
	accObjID, err := bson.ObjectIDFromHex(accountID)
	if err != nil {
		return ErrNotFound
	}

	col := database.GetCollection(models.SocialAccountsCollection)

	var acc models.SocialAccount
	if err := col.FindOne(ctx, bson.M{"_id": accObjID, "status": models.SocialAccountPending}).Decode(&acc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	result, err := col.UpdateOne(ctx,
		bson.M{"_id": accObjID},
		bson.M{"$set": bson.M{
			"status":         models.SocialAccountRejected,
			"rejectedReason": reason,
		}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	go notifications.Send(ctx, acc.UserID, models.NotifSocialAccountRejected,
		"Social account not verified",
		fmt.Sprintf("Your %s account @%s could not be verified. Reason: %s", acc.Platform, acc.Username, reason),
		map[string]interface{}{"accountId": acc.ID.Hex(), "platform": string(acc.Platform), "reason": reason},
	)

	return nil
}

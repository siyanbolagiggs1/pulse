package admin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	stripe "github.com/stripe/stripe-go/v78"
	stripetransfer "github.com/stripe/stripe-go/v78/transfer"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/modules/notifications"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrNotReviewable     = errors.New("resource is not in a reviewable state")
	ErrNoConnectAccount  = errors.New("promoter has no active Stripe Connect account")
	ErrStripeNotConfigured = errors.New("payment processing is not configured")
)

func initStripe() {
	stripe.Key = config.App.StripeSecretKey
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
		filter["$or"] = bson.A{
			bson.M{"email": bson.M{"$regex": q.Search, "$options": "i"}},
			bson.M{"name": bson.M{"$regex": q.Search, "$options": "i"}},
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
	wObjID, err := bson.ObjectIDFromHex(withdrawalID)
	if err != nil {
		return nil, ErrNotFound
	}

	withdrawalCol := database.GetCollection(models.WithdrawalsCollection)
	var w models.Withdrawal
	if err := withdrawalCol.FindOne(ctx, bson.M{"_id": wObjID}).Decode(&w); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if w.Status != models.WithdrawalPending {
		return nil, ErrNotReviewable
	}

	// Fetch the promoter to get their Connect account.
	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": w.UserID}).Decode(&user); err != nil {
		return nil, ErrNotFound
	}
	if user.StripeConnectAccountID == "" || user.StripeConnectStatus != "active" {
		return nil, ErrNoConnectAccount
	}

	now := time.Now().UTC()
	status := models.WithdrawalProcessing
	stripePayoutID := ""

	if config.App.StripeSecretKey != "" {
		initStripe()
		cents := int64(math.Round(w.NetAmount * 100))
		transferParams := &stripe.TransferParams{
			Amount:      stripe.Int64(cents),
			Currency:    stripe.String("usd"),
			Destination: stripe.String(user.StripeConnectAccountID),
			Metadata:    map[string]string{"withdrawalId": w.ID.Hex()},
		}
		t, stripeErr := stripetransfer.New(transferParams)
		if stripeErr != nil {
			return nil, fmt.Errorf("stripe transfer failed: %w", stripeErr)
		}
		stripePayoutID = t.ID
	}

	_, _ = withdrawalCol.UpdateOne(ctx,
		bson.M{"_id": wObjID},
		bson.M{"$set": bson.M{
			"status":         status,
			"stripePayoutId": stripePayoutID,
			"processedAt":    now,
		}},
	)

	w.Status = status
	w.StripePayoutID = stripePayoutID
	w.ProcessedAt = now

	go notifications.Send(context.Background(), w.UserID, models.NotifWithdrawalProcessed,
		"Withdrawal Approved",
		fmt.Sprintf("Your withdrawal of %.2f USD has been approved and is being transferred.", w.NetAmount),
		map[string]interface{}{"withdrawalId": w.ID.Hex(), "amount": w.NetAmount})

	return &w, nil
}

func rejectWithdrawal(ctx context.Context, withdrawalID, reason string) (*models.Withdrawal, error) {
	wObjID, err := bson.ObjectIDFromHex(withdrawalID)
	if err != nil {
		return nil, ErrNotFound
	}

	withdrawalCol := database.GetCollection(models.WithdrawalsCollection)
	var w models.Withdrawal
	if err := withdrawalCol.FindOne(ctx, bson.M{"_id": wObjID}).Decode(&w); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if w.Status != models.WithdrawalPending {
		return nil, ErrNotReviewable
	}

	now := time.Now().UTC()

	// Refund to promoter's available balance.
	_, _ = database.GetCollection(models.WalletsCollection).UpdateOne(ctx,
		bson.M{"userId": w.UserID},
		bson.M{
			"$inc": bson.M{"availableBalance": w.Amount},
			"$set": bson.M{"updatedAt": now},
		},
	)

	// Record refund transaction.
	var wallet models.Wallet
	if err := database.GetCollection(models.WalletsCollection).
		FindOne(ctx, bson.M{"userId": w.UserID}).Decode(&wallet); err == nil {
		tx := models.Transaction{
			WalletID:     wallet.ID,
			UserID:       w.UserID,
			Type:         models.TxRefund,
			Amount:       w.Amount,
			BalanceAfter: wallet.AvailableBalance + w.Amount,
			ReferenceID:  w.ID.Hex(),
			Description:  fmt.Sprintf("Withdrawal rejected: %s", reason),
			CreatedAt:    now,
		}
		_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)
	}

	_, _ = withdrawalCol.UpdateOne(ctx,
		bson.M{"_id": wObjID},
		bson.M{"$set": bson.M{"status": models.WithdrawalFailed, "processedAt": now}},
	)

	w.Status = models.WithdrawalFailed
	w.ProcessedAt = now

	go notifications.Send(context.Background(), w.UserID, models.NotifWithdrawalProcessed,
		"Withdrawal Rejected",
		fmt.Sprintf("Your withdrawal of %.2f USD was rejected and returned to your wallet: %s", w.Amount, reason),
		map[string]interface{}{"withdrawalId": w.ID.Hex(), "amount": w.Amount, "reason": reason})

	return &w, nil
}

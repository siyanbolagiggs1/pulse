package wallet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	stripe "github.com/stripe/stripe-go/v78"
	stripeaccount "github.com/stripe/stripe-go/v78/account"
	stripeaccountlink "github.com/stripe/stripe-go/v78/accountlink"
	stripepaymentintent "github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/modules/notifications"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrInsufficientBalance  = errors.New("insufficient available balance")
	ErrBelowMinimum         = errors.New("minimum withdrawal amount is $10.00")
	ErrNoConnectAccount     = errors.New("Stripe Connect account not set up — complete onboarding first")
	ErrConnectNotActive     = errors.New("Stripe Connect account is not yet active — complete onboarding")
	ErrStripeNotConfigured  = errors.New("payment processing is not configured")
)

func initStripe() {
	stripe.Key = config.App.StripeSecretKey
}

// ── Wallet read ──────────────────────────────────────────────

func getWallet(ctx context.Context, userID string) (*models.Wallet, []models.Transaction, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, nil, ErrWalletNotFound
	}

	// Release any matured pending payouts before returning balance.
	_ = releaseMaturePending(ctx, objID)

	var w models.Wallet
	if err := database.GetCollection(models.WalletsCollection).
		FindOne(ctx, bson.M{"userId": objID}).Decode(&w); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, ErrWalletNotFound
		}
		return nil, nil, err
	}

	cursor, err := database.GetCollection(models.TransactionsCollection).Find(ctx,
		bson.M{"userId": objID},
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetLimit(10),
	)
	if err != nil {
		return &w, nil, nil
	}
	defer cursor.Close(ctx)

	var txs []models.Transaction
	_ = cursor.All(ctx, &txs)

	return &w, txs, nil
}

func getTransactions(ctx context.Context, userID string, page, limit int) ([]models.Transaction, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, ErrWalletNotFound
	}

	col := database.GetCollection(models.TransactionsCollection)
	filter := bson.M{"userId": objID}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var txs []models.Transaction
	if err := cursor.All(ctx, &txs); err != nil {
		return nil, 0, err
	}

	return txs, total, nil
}

// ── Top-up (business) ────────────────────────────────────────

func createTopup(ctx context.Context, userID string, amount float64) (*TopupResponse, error) {
	if config.App.StripeSecretKey == "" {
		return nil, ErrStripeNotConfigured
	}
	initStripe()

	cents := int64(math.Round(amount * 100))
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(cents),
		Currency: stripe.String("usd"),
		Metadata: map[string]string{
			"userID": userID,
			"type":   "wallet_topup",
		},
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	pi, err := stripepaymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", err)
	}

	return &TopupResponse{
		ClientSecret:    pi.ClientSecret,
		PaymentIntentID: pi.ID,
		Amount:          amount,
	}, nil
}

// creditWallet is called from the webhook when Stripe confirms payment.
func creditWallet(ctx context.Context, userID string, amount float64, stripePaymentIntentID string) error {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrWalletNotFound
	}

	now := time.Now().UTC()
	walletCol := database.GetCollection(models.WalletsCollection)

	var w models.Wallet
	if err := walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&w); err != nil {
		return ErrWalletNotFound
	}

	_, err = walletCol.UpdateOne(ctx,
		bson.M{"userId": objID},
		bson.M{
			"$inc": bson.M{"availableBalance": amount},
			"$set": bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		return err
	}

	tx := models.Transaction{
		WalletID:     w.ID,
		UserID:       objID,
		Type:         models.TxTopup,
		Amount:       amount,
		BalanceAfter: w.AvailableBalance + amount,
		ReferenceID:  stripePaymentIntentID,
		Description:  fmt.Sprintf("Wallet top-up via Stripe (%.2f USD)", amount),
		CreatedAt:    now,
	}
	_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)

	go notifications.Send(context.Background(), objID, models.NotifWalletTopup,
		"Wallet Topped Up",
		fmt.Sprintf("%.2f USD has been added to your wallet.", amount),
		map[string]interface{}{"amount": amount, "paymentIntentId": stripePaymentIntentID})

	return nil
}

// ── Withdrawal (promoter) ────────────────────────────────────

func requestWithdrawal(ctx context.Context, userID string, amount float64) (*models.Withdrawal, error) {
	if amount < 10 {
		return nil, ErrBelowMinimum
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrWalletNotFound
	}

	// Fetch user to get Stripe Connect account.
	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return nil, ErrWalletNotFound
	}
	if user.StripeConnectAccountID == "" {
		return nil, ErrNoConnectAccount
	}
	if user.StripeConnectStatus != "active" {
		return nil, ErrConnectNotActive
	}

	// Check balance.
	walletCol := database.GetCollection(models.WalletsCollection)
	var w models.Wallet
	if err := walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&w); err != nil {
		return nil, ErrWalletNotFound
	}
	if w.AvailableBalance < amount {
		return nil, ErrInsufficientBalance
	}

	now := time.Now().UTC()

	// Deduct from available balance immediately.
	_, err = walletCol.UpdateOne(ctx,
		bson.M{"userId": objID},
		bson.M{
			"$inc": bson.M{"availableBalance": -amount},
			"$set": bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		return nil, err
	}

	// Create withdrawal record.
	withdrawal := &models.Withdrawal{
		UserID:      objID,
		Amount:      amount,
		Fee:         0,
		NetAmount:   amount,
		Status:      models.WithdrawalPending,
		RequestedAt: now,
		CreatedAt:   now,
	}
	result, err := database.GetCollection(models.WithdrawalsCollection).InsertOne(ctx, withdrawal)
	if err != nil {
		// Rollback balance.
		_, _ = walletCol.UpdateOne(ctx, bson.M{"userId": objID},
			bson.M{"$inc": bson.M{"availableBalance": amount}, "$set": bson.M{"updatedAt": now}})
		return nil, err
	}

	// Record transaction.
	tx := models.Transaction{
		WalletID:     w.ID,
		UserID:       objID,
		Type:         models.TxWithdrawal,
		Amount:       amount,
		BalanceAfter: w.AvailableBalance - amount,
		ReferenceID:  result.InsertedID.(bson.ObjectID).Hex(),
		Description:  fmt.Sprintf("Withdrawal request (%.2f USD)", amount),
		CreatedAt:    now,
	}
	_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)

	// Re-fetch to get assigned ID.
	_ = database.GetCollection(models.WithdrawalsCollection).
		FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(withdrawal)

	return withdrawal, nil
}

func getWithdrawals(ctx context.Context, userID string, page, limit int) ([]models.Withdrawal, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, ErrWalletNotFound
	}

	col := database.GetCollection(models.WithdrawalsCollection)
	filter := bson.M{"userId": objID}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "requestedAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
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

// ── Stripe Connect ───────────────────────────────────────────

func createConnectAccount(ctx context.Context, userID string) (*ConnectOnboardingResponse, error) {
	if config.App.StripeSecretKey == "" {
		return nil, ErrStripeNotConfigured
	}
	initStripe()

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrWalletNotFound
	}

	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return nil, ErrWalletNotFound
	}

	connectAccountID := user.StripeConnectAccountID

	// Create a new Connect account if one doesn't exist yet.
	if connectAccountID == "" {
		acctParams := &stripe.AccountParams{
			Type:  stripe.String("express"),
			Email: stripe.String(user.Email),
			Capabilities: &stripe.AccountCapabilitiesParams{
				Transfers: &stripe.AccountCapabilitiesTransfersParams{
					Requested: stripe.Bool(true),
				},
			},
		}
		acct, err := stripeaccount.New(acctParams)
		if err != nil {
			return nil, fmt.Errorf("stripe: %w", err)
		}
		connectAccountID = acct.ID

		_, _ = database.GetCollection(models.UsersCollection).UpdateOne(ctx,
			bson.M{"_id": objID},
			bson.M{"$set": bson.M{
				"stripeConnectAccountId": connectAccountID,
				"stripeConnectStatus":    "pending",
				"updatedAt":              time.Now().UTC(),
			}},
		)
	}

	// Generate onboarding link.
	linkParams := &stripe.AccountLinkParams{
		Account:    stripe.String(connectAccountID),
		RefreshURL: stripe.String(config.App.ClientURL + "/dashboard/wallet/connect/refresh"),
		ReturnURL:  stripe.String(config.App.ClientURL + "/dashboard/wallet/connect/complete"),
		Type:       stripe.String("account_onboarding"),
	}
	link, err := stripeaccountlink.New(linkParams)
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", err)
	}

	return &ConnectOnboardingResponse{
		URL:              link.URL,
		ConnectAccountID: connectAccountID,
	}, nil
}

func getConnectStatus(ctx context.Context, userID string) (*ConnectStatusResponse, error) {
	if config.App.StripeSecretKey == "" {
		return nil, ErrStripeNotConfigured
	}
	initStripe()

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrWalletNotFound
	}

	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return nil, ErrWalletNotFound
	}
	if user.StripeConnectAccountID == "" {
		return nil, ErrNoConnectAccount
	}

	acct, err := stripeaccount.GetByID(user.StripeConnectAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", err)
	}

	status := "pending"
	if acct.ChargesEnabled && acct.PayoutsEnabled {
		status = "active"
	} else if len(acct.Requirements.Errors) > 0 {
		status = "restricted"
	}

	// Sync status to DB.
	if user.StripeConnectStatus != status {
		_, _ = database.GetCollection(models.UsersCollection).UpdateOne(ctx,
			bson.M{"_id": objID},
			bson.M{"$set": bson.M{"stripeConnectStatus": status, "updatedAt": time.Now().UTC()}},
		)
	}

	return &ConnectStatusResponse{
		AccountID:      acct.ID,
		ChargesEnabled: acct.ChargesEnabled,
		PayoutsEnabled: acct.PayoutsEnabled,
		Status:         status,
	}, nil
}

// ── 48h payout release ───────────────────────────────────────

// releaseMaturePending moves promoter earnings from pending to available
// once the 48h hold window has passed.
func releaseMaturePending(ctx context.Context, promoterObjID bson.ObjectID) error {
	now := time.Now().UTC()

	cursor, err := database.GetCollection(models.SubmissionsCollection).Find(ctx, bson.M{
		"promoterId":     promoterObjID,
		"status":         models.SubmissionStatusApproved,
		"payoutReleased": false,
		"payoutReleasedAt": bson.M{"$lte": now},
	})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var matured []models.CampaignSubmission
	if err := cursor.All(ctx, &matured); err != nil {
		return err
	}
	if len(matured) == 0 {
		return nil
	}

	var totalRelease float64
	for _, sub := range matured {
		totalRelease += sub.PromoterEarning
	}

	walletCol := database.GetCollection(models.WalletsCollection)
	var w models.Wallet
	if err := walletCol.FindOne(ctx, bson.M{"userId": promoterObjID}).Decode(&w); err != nil {
		return err
	}

	// Move from pending to available.
	_, err = walletCol.UpdateOne(ctx,
		bson.M{"userId": promoterObjID},
		bson.M{
			"$inc": bson.M{
				"availableBalance": totalRelease,
				"pendingBalance":   -totalRelease,
				"totalEarned":      totalRelease,
			},
			"$set": bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		return err
	}

	// Mark each submission as released.
	ids := make([]bson.ObjectID, len(matured))
	for i, sub := range matured {
		ids[i] = sub.ID
	}
	_, _ = database.GetCollection(models.SubmissionsCollection).UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"payoutReleased": true, "updatedAt": now}},
	)

	// Record a single release transaction.
	tx := models.Transaction{
		WalletID:     w.ID,
		UserID:       promoterObjID,
		Type:         models.TxPayoutReleased,
		Amount:       totalRelease,
		BalanceAfter: w.AvailableBalance + totalRelease,
		Description:  fmt.Sprintf("%.2f USD released from 48h hold (%d payout(s))", totalRelease, len(matured)),
		CreatedAt:    now,
	}
	_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)

	return nil
}

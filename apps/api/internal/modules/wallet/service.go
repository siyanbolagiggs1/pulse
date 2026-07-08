package wallet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/modules/notifications"
	"github.com/pulse/api/internal/services/paystack"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient available balance")
	ErrBelowMinimum        = errors.New("minimum withdrawal amount is 0.10")
	ErrPaystackNotConfigured = errors.New("payment processing is not configured — add PAYSTACK_SECRET_KEY")
	ErrPaymentNotSuccessful  = errors.New("payment was not completed successfully")
	ErrNoBankAccount         = errors.New("add a payout bank account before requesting a withdrawal")
	ErrWithdrawalNotFound    = errors.New("withdrawal not found")
	ErrNotReviewable         = errors.New("withdrawal is not in a reviewable state")
	ErrTransferOTPRequired   = errors.New("Paystack requires manual OTP confirmation for transfers — disable OTP for transfers in the Paystack dashboard to enable automated payouts")
)

// ── Wallet read ──────────────────────────────────────────────

func getWallet(ctx context.Context, userID string) (*models.Wallet, []models.Transaction, error) {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, nil, ErrWalletNotFound
	}

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

// ── Top-up ───────────────────────────────────────────────────

func createTopup(ctx context.Context, userID string, amount float64) (*TopupResponse, error) {
	// Dev mode: no Paystack key — credit wallet directly.
	if config.App.PaystackSecretKey == "" {
		if err := creditWallet(ctx, userID, amount, "dev_topup"); err != nil {
			return nil, err
		}
		return &TopupResponse{Amount: amount}, nil
	}

	// Look up user email for Paystack.
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrWalletNotFound
	}
	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return nil, ErrWalletNotFound
	}

	// Paystack amount is in smallest unit (kobo for NGN, cents for USD).
	smallestUnit := int64(math.Round(amount * 100))
	ref := fmt.Sprintf("pulse_%s_%d", userID[len(userID)-6:], time.Now().UnixNano())

	data, err := paystack.Initialize(config.App.PaystackSecretKey, paystack.InitRequest{
		Email:       user.Email,
		Amount:      smallestUnit,
		Reference:   ref,
		Currency:    config.App.PaystackCurrency,
		CallbackURL: config.App.ClientURL + "/dashboard/wallet/topup/callback",
		Metadata: map[string]any{
			"userID": userID,
			"type":   "wallet_topup",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("paystack: %w", err)
	}

	return &TopupResponse{
		AuthorizationURL: data.AuthorizationURL,
		Reference:        data.Reference,
		Amount:           amount,
	}, nil
}

func verifyTopup(ctx context.Context, reference string) error {
	if config.App.PaystackSecretKey == "" {
		return ErrPaystackNotConfigured
	}

	data, err := paystack.Verify(config.App.PaystackSecretKey, reference)
	if err != nil {
		return fmt.Errorf("paystack: %w", err)
	}
	if data.Status != "success" {
		return ErrPaymentNotSuccessful
	}

	userID, _ := data.Metadata["userID"].(string)
	if userID == "" {
		return errors.New("missing userID in payment metadata")
	}

	amount := float64(data.Amount) / 100.0
	return creditWallet(ctx, userID, amount, reference)
}

// creditWallet adds funds to a user's available balance and records the transaction.
func creditWallet(ctx context.Context, userID string, amount float64, reference string) error {
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
		ReferenceID:  reference,
		Description:  fmt.Sprintf("Wallet top-up (%.2f %s)", amount, config.App.PaystackCurrency),
		CreatedAt:    now,
	}
	_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)

	go notifications.Send(context.Background(), objID, models.NotifWalletTopup,
		"Wallet Topped Up",
		fmt.Sprintf("%.2f %s has been added to your wallet.", amount, config.App.PaystackCurrency),
		map[string]any{"amount": amount, "reference": reference})

	return nil
}

// ── Withdrawal ───────────────────────────────────────────────

func requestWithdrawal(ctx context.Context, userID string, amount float64) (*models.Withdrawal, error) {
	if amount < 0.1 {
		return nil, ErrBelowMinimum
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrWalletNotFound
	}

	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		return nil, ErrWalletNotFound
	}
	if user.BankAccount == nil {
		return nil, ErrNoBankAccount
	}

	walletCol := database.GetCollection(models.WalletsCollection)
	var w models.Wallet
	if err := walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&w); err != nil {
		return nil, ErrWalletNotFound
	}
	if w.AvailableBalance < amount {
		return nil, ErrInsufficientBalance
	}

	now := time.Now().UTC()

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
		_, _ = walletCol.UpdateOne(ctx, bson.M{"userId": objID},
			bson.M{"$inc": bson.M{"availableBalance": amount}, "$set": bson.M{"updatedAt": now}})
		return nil, err
	}

	tx := models.Transaction{
		WalletID:     w.ID,
		UserID:       objID,
		Type:         models.TxWithdrawal,
		Amount:       amount,
		BalanceAfter: w.AvailableBalance - amount,
		ReferenceID:  result.InsertedID.(bson.ObjectID).Hex(),
		Description:  fmt.Sprintf("Withdrawal request (%.2f %s)", amount, config.App.PaystackCurrency),
		CreatedAt:    now,
	}
	_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)

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

// ── 48h payout release ───────────────────────────────────────

func releaseMaturePending(ctx context.Context, promoterObjID bson.ObjectID) error {
	now := time.Now().UTC()

	cursor, err := database.GetCollection(models.SubmissionsCollection).Find(ctx, bson.M{
		"promoterId":       promoterObjID,
		"status":           models.SubmissionStatusApproved,
		"payoutReleased":   false,
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

	ids := make([]bson.ObjectID, len(matured))
	for i, sub := range matured {
		ids[i] = sub.ID
	}
	_, _ = database.GetCollection(models.SubmissionsCollection).UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"payoutReleased": true, "updatedAt": now}},
	)

	tx := models.Transaction{
		WalletID:     w.ID,
		UserID:       promoterObjID,
		Type:         models.TxPayoutReleased,
		Amount:       totalRelease,
		BalanceAfter: w.AvailableBalance + totalRelease,
		Description:  fmt.Sprintf("%.2f %s released from 48h hold (%d payout(s))", totalRelease, config.App.PaystackCurrency, len(matured)),
		CreatedAt:    now,
	}
	_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)

	return nil
}

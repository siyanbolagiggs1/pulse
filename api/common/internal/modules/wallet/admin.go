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
)

// AdminApproveWithdrawal moves real money: it creates (or reuses) a Paystack
// transfer recipient for the promoter's bank account and initiates a
// transfer. Completion is asynchronous — a "success" response here means
// Paystack accepted the transfer, not that funds have landed; final state
// mostly arrives via the transfer.success/transfer.failed webhook.
func AdminApproveWithdrawal(ctx context.Context, withdrawalID string) (*models.Withdrawal, error) {
	if config.App.PaystackSecretKey == "" {
		return nil, ErrPaystackNotConfigured
	}

	wObjID, err := bson.ObjectIDFromHex(withdrawalID)
	if err != nil {
		return nil, ErrWithdrawalNotFound
	}

	withdrawalCol := database.GetCollection(models.WithdrawalsCollection)
	var w models.Withdrawal
	if err := withdrawalCol.FindOne(ctx, bson.M{"_id": wObjID}).Decode(&w); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	if w.Status != models.WithdrawalPending {
		return nil, ErrNotReviewable
	}

	usersCol := database.GetCollection(models.UsersCollection)
	var user models.User
	if err := usersCol.FindOne(ctx, bson.M{"_id": w.UserID}).Decode(&user); err != nil {
		return nil, err
	}
	if user.BankAccount == nil {
		return nil, ErrNoBankAccount
	}

	recipientCode := user.BankAccount.RecipientCode
	if recipientCode == "" {
		recipient, err := paystack.CreateTransferRecipient(config.App.PaystackSecretKey, paystack.RecipientRequest{
			Type:          "nuban",
			Name:          user.BankAccount.AccountName,
			AccountNumber: user.BankAccount.AccountNumber,
			BankCode:      user.BankAccount.BankCode,
			Currency:      config.App.PaystackCurrency,
		})
		if err != nil {
			return nil, fmt.Errorf("paystack: %w", err)
		}
		recipientCode = recipient.RecipientCode

		_, _ = usersCol.UpdateOne(ctx,
			bson.M{"_id": user.ID},
			bson.M{"$set": bson.M{"bankAccount.recipientCode": recipientCode}},
		)
	}

	transfer, err := paystack.InitiateTransfer(config.App.PaystackSecretKey, paystack.TransferRequest{
		Source:    "balance",
		Amount:    int64(math.Round(w.NetAmount * 100)),
		Recipient: recipientCode,
		Reason:    "Pulse payout",
		Reference: w.ID.Hex(),
	})
	if err != nil {
		return nil, fmt.Errorf("paystack: %w", err)
	}

	if transfer.Status == "otp" {
		return nil, ErrTransferOTPRequired
	}

	now := time.Now().UTC()
	status := models.WithdrawalProcessing
	if transfer.Status == "success" {
		status = models.WithdrawalCompleted
	}

	_, _ = withdrawalCol.UpdateOne(ctx,
		bson.M{"_id": wObjID},
		bson.M{"$set": bson.M{
			"status":      status,
			"payoutId":    transfer.TransferCode,
			"processedAt": now,
		}},
	)
	w.Status = status
	w.PayoutID = transfer.TransferCode
	w.ProcessedAt = now

	go notifications.Send(context.Background(), w.UserID, models.NotifWithdrawalProcessed,
		"Withdrawal Approved",
		fmt.Sprintf("Your withdrawal of %.2f %s has been approved and is being transferred.", w.NetAmount, config.App.PaystackCurrency),
		map[string]interface{}{"withdrawalId": w.ID.Hex(), "amount": w.NetAmount})

	return &w, nil
}

// AdminRejectWithdrawal refunds a pending withdrawal back to the requester's
// available balance without ever contacting Paystack.
func AdminRejectWithdrawal(ctx context.Context, withdrawalID, reason string) (*models.Withdrawal, error) {
	wObjID, err := bson.ObjectIDFromHex(withdrawalID)
	if err != nil {
		return nil, ErrWithdrawalNotFound
	}

	withdrawalCol := database.GetCollection(models.WithdrawalsCollection)
	var w models.Withdrawal
	if err := withdrawalCol.FindOne(ctx, bson.M{"_id": wObjID}).Decode(&w); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	if w.Status != models.WithdrawalPending {
		return nil, ErrNotReviewable
	}

	if err := refundWithdrawal(ctx, &w, reason); err != nil {
		return nil, err
	}

	go notifications.Send(context.Background(), w.UserID, models.NotifWithdrawalProcessed,
		"Withdrawal Rejected",
		fmt.Sprintf("Your withdrawal of %.2f %s was rejected and returned to your wallet: %s", w.Amount, config.App.PaystackCurrency, reason),
		map[string]interface{}{"withdrawalId": w.ID.Hex(), "amount": w.Amount, "reason": reason})

	return &w, nil
}

// RefundAndFailWithdrawal handles a withdrawal that failed after Paystack
// already accepted the transfer (transfer.failed / transfer.reversed
// webhooks) — same balance-restoring effect as an admin rejection, just
// triggered asynchronously instead of by an admin click.
func RefundAndFailWithdrawal(ctx context.Context, withdrawalID bson.ObjectID, reason string) error {
	withdrawalCol := database.GetCollection(models.WithdrawalsCollection)
	var w models.Withdrawal
	if err := withdrawalCol.FindOne(ctx, bson.M{"_id": withdrawalID}).Decode(&w); err != nil {
		return err
	}
	if w.Status == models.WithdrawalCompleted || w.Status == models.WithdrawalFailed {
		return nil // already terminal — avoid double-refunding
	}

	if err := refundWithdrawal(ctx, &w, reason); err != nil {
		return err
	}

	go notifications.Send(context.Background(), w.UserID, models.NotifWithdrawalProcessed,
		"Withdrawal Failed",
		fmt.Sprintf("Your withdrawal of %.2f %s could not be completed and was returned to your wallet: %s", w.Amount, config.App.PaystackCurrency, reason),
		map[string]interface{}{"withdrawalId": w.ID.Hex(), "amount": w.Amount, "reason": reason})

	return nil
}

// CompleteWithdrawal marks a withdrawal as finished after Paystack confirms
// the transfer landed (transfer.success webhook).
func CompleteWithdrawal(ctx context.Context, withdrawalID bson.ObjectID) error {
	now := time.Now().UTC()
	withdrawalCol := database.GetCollection(models.WithdrawalsCollection)

	var w models.Withdrawal
	if err := withdrawalCol.FindOne(ctx, bson.M{"_id": withdrawalID}).Decode(&w); err != nil {
		return err
	}
	if w.Status == models.WithdrawalCompleted || w.Status == models.WithdrawalFailed {
		return nil
	}

	_, err := withdrawalCol.UpdateOne(ctx,
		bson.M{"_id": withdrawalID},
		bson.M{"$set": bson.M{"status": models.WithdrawalCompleted, "processedAt": now}},
	)
	if err != nil {
		return err
	}

	go notifications.Send(context.Background(), w.UserID, models.NotifWithdrawalProcessed,
		"Withdrawal Completed",
		fmt.Sprintf("%.2f %s has been sent to your bank account.", w.Amount, config.App.PaystackCurrency),
		map[string]interface{}{"withdrawalId": w.ID.Hex(), "amount": w.Amount})

	return nil
}

// refundWithdrawal restores a withdrawal's amount to the owner's available
// balance, records the refund transaction, and marks the withdrawal failed.
// Shared by admin rejection and webhook-triggered transfer failures.
func refundWithdrawal(ctx context.Context, w *models.Withdrawal, reason string) error {
	now := time.Now().UTC()

	walletCol := database.GetCollection(models.WalletsCollection)
	_, err := walletCol.UpdateOne(ctx,
		bson.M{"userId": w.UserID},
		bson.M{
			"$inc": bson.M{"availableBalance": w.Amount},
			"$set": bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		return err
	}

	var wal models.Wallet
	if err := walletCol.FindOne(ctx, bson.M{"userId": w.UserID}).Decode(&wal); err == nil {
		tx := models.Transaction{
			WalletID:     wal.ID,
			UserID:       w.UserID,
			Type:         models.TxRefund,
			Amount:       w.Amount,
			BalanceAfter: wal.AvailableBalance,
			ReferenceID:  w.ID.Hex(),
			Description:  fmt.Sprintf("Withdrawal failed: %s", reason),
			CreatedAt:    now,
		}
		_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)
	}

	_, err = database.GetCollection(models.WithdrawalsCollection).UpdateOne(ctx,
		bson.M{"_id": w.ID},
		bson.M{"$set": bson.M{"status": models.WithdrawalFailed, "processedAt": now}},
	)
	if err != nil {
		return err
	}

	w.Status = models.WithdrawalFailed
	w.ProcessedAt = now
	return nil
}

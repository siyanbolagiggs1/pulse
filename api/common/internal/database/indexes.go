package database

import (
	"context"
	"log"
	"time"

	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CreateIndexes sets up all MongoDB indexes at startup.
// Safe to call multiple times — MongoDB skips existing indexes.
func CreateIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createUserIndexes(ctx)
	createCampaignIndexes(ctx)
	createSubmissionIndexes(ctx)
	createWalletIndexes(ctx)
	createNotificationIndexes(ctx)
	createFraudFlagIndexes(ctx)
	createSocialAccountIndexes(ctx)
	createChatIndexes(ctx)

	log.Println("MongoDB indexes ensured")
}

func createUserIndexes(ctx context.Context) {
	col := GetCollection(models.UsersCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("email_unique"),
		},
		{
			Keys:    bson.D{{Key: "emailVerifyToken", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("email_verify_token"),
		},
		{
			Keys:    bson.D{{Key: "passwordResetToken", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("password_reset_token"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "users")
}

func createCampaignIndexes(ctx context.Context) {
	col := GetCollection(models.CampaignsCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "businessId", Value: 1}},
			Options: options.Index().SetName("business_id"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status"),
		},
		{
			Keys:    bson.D{{Key: "platform", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("platform_status"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "campaigns")
}

func createSubmissionIndexes(ctx context.Context) {
	col := GetCollection(models.SubmissionsCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "campaignId", Value: 1}},
			Options: options.Index().SetName("campaign_id"),
		},
		{
			Keys:    bson.D{{Key: "promoterId", Value: 1}},
			Options: options.Index().SetName("promoter_id"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status"),
		},
		// Prevent a promoter submitting to the same campaign twice
		{
			Keys:    bson.D{{Key: "campaignId", Value: 1}, {Key: "promoterId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("campaign_promoter_unique"),
		},
		// Prevent duplicate repost URLs across any submission
		{
			Keys:    bson.D{{Key: "repostUrl", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("repost_url_unique"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "submissions")
}

func createWalletIndexes(ctx context.Context) {
	col := GetCollection(models.WalletsCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("user_id_unique"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "wallets")

	txCol := GetCollection(models.TransactionsCollection)
	txIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("user_id"),
		},
		{
			Keys:    bson.D{{Key: "walletId", Value: 1}},
			Options: options.Index().SetName("wallet_id"),
		},
		{
			// Paystack calls both the webhook and the client-triggered verify
			// endpoint for the same charge, so creditWallet can be invoked
			// twice for one payment. This index makes the top-up transaction
			// insert (which creditWallet does before touching the balance)
			// fail the second time with the same reference, so the balance
			// only ever gets credited once per payment.
			Keys: bson.D{{Key: "referenceId", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("topup_reference_unique").
				SetPartialFilterExpression(bson.M{"type": string(models.TxTopup)}),
		},
	}
	mustCreateIndexes(ctx, txCol, txIndexes, "transactions")
}

func createNotificationIndexes(ctx context.Context) {
	col := GetCollection(models.NotificationsCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("user_id_created_at"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "isRead", Value: 1}},
			Options: options.Index().SetName("user_id_is_read"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "notifications")
}

func createFraudFlagIndexes(ctx context.Context) {
	col := GetCollection(models.FraudFlagsCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("user_id"),
		},
		{
			Keys:    bson.D{{Key: "resolved", Value: 1}},
			Options: options.Index().SetName("resolved"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "resolved", Value: 1}},
			Options: options.Index().SetName("user_id_resolved"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "fraud_flags")
}

func createSocialAccountIndexes(ctx context.Context) {
	col := GetCollection(models.SocialAccountsCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("user_id"),
		},
		// One platform per user
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "platform", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("user_platform_unique"),
		},
		// One profile URL across all users — URLs are stored normalised (lowercase, no trailing slash)
		{
			Keys:    bson.D{{Key: "profileUrl", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("profile_url_unique"),
		},
	}
	mustCreateIndexes(ctx, col, indexes, "social_accounts")
}

func createChatIndexes(ctx context.Context) {
	convCol := GetCollection(models.ConversationsCollection)
	convIndexes := []mongo.IndexModel{
		// One conversation per unordered user pair — canonical ordering in
		// chat.canonicalOrder guarantees userAId/userBId is always assigned
		// consistently, so this also powers the upsert-based get-or-create in
		// chat.startOrGetConversation / chat.SendWelcomeMessage.
		{
			Keys:    bson.D{{Key: "userAId", Value: 1}, {Key: "userBId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("user_pair_unique"),
		},
		{
			Keys:    bson.D{{Key: "userAId", Value: 1}, {Key: "lastMessageAt", Value: -1}},
			Options: options.Index().SetName("user_a_last_message"),
		},
		{
			Keys:    bson.D{{Key: "userBId", Value: 1}, {Key: "lastMessageAt", Value: -1}},
			Options: options.Index().SetName("user_b_last_message"),
		},
	}
	mustCreateIndexes(ctx, convCol, convIndexes, "conversations")

	msgCol := GetCollection(models.MessagesCollection)
	msgIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "conversationId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("conversation_created_at"),
		},
	}
	mustCreateIndexes(ctx, msgCol, msgIndexes, "messages")
}

func mustCreateIndexes(ctx context.Context, col *mongo.Collection, indexes []mongo.IndexModel, name string) {
	if _, err := col.Indexes().CreateMany(ctx, indexes); err != nil {
		log.Printf("Warning: could not create indexes for %s: %v", name, err)
	}
}

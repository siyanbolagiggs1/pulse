package fraud

import (
	"context"
	"time"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FlagUser records a fraud flag and penalises the user's trust score by 30 points.
// If the resulting score falls below 20, the account is automatically suspended.
func FlagUser(ctx context.Context, userID bson.ObjectID, reason models.FraudFlagReason, details string) {
	now := time.Now().UTC()
	flag := models.FraudFlag{
		UserID:    userID,
		Reason:    reason,
		Details:   details,
		Resolved:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, _ = database.GetCollection(models.FraudFlagsCollection).InsertOne(ctx, flag)

	var user models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		return
	}

	newScore := user.TrustScore - 30
	update := bson.M{"$set": bson.M{"trustScore": newScore, "updatedAt": now}}
	if newScore < 20 {
		update["$set"].(bson.M)["isSuspended"] = true
	}
	_, _ = database.GetCollection(models.UsersCollection).UpdateOne(ctx,
		bson.M{"_id": userID}, update)
}

// CheckSubmission runs fraud heuristics against the social account used in a submission.
// Currently a no-op placeholder: the follower:following ratio and engagement-rate
// implausibility checks were removed when the platform switched from self-reported
// stats to admin-verified follower-tier scoring (those two signals no longer exist
// on SocialAccount). Retained as a hook for future heuristics — e.g. a "suspicious
// follower jump" check using SocialAccount.FollowerHistory.
func CheckSubmission(ctx context.Context, promoterObjID bson.ObjectID, acc *models.SocialAccount) {
}

// CheckAccount runs the same fraud heuristics when a social account is first connected.
// Called asynchronously from the users service so onboarding is never delayed.
func CheckAccount(ctx context.Context, promoterObjID bson.ObjectID, acc *models.SocialAccount) {
	CheckSubmission(ctx, promoterObjID, acc)
}

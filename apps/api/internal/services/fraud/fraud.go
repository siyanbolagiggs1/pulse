package fraud

import (
	"context"
	"fmt"
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
// Detected violations are flagged asynchronously so this call never blocks the request.
func CheckSubmission(ctx context.Context, promoterObjID bson.ObjectID, acc *models.SocialAccount) {
	// Follower:following ratio below 0.2 — typical of follow-churn / follow-spam accounts.
	if acc.FollowingCount > 0 {
		ratio := float64(acc.FollowerCount) / float64(acc.FollowingCount)
		if ratio < 0.2 {
			go FlagUser(context.Background(), promoterObjID, models.FraudLowFollowerRatio,
				fmt.Sprintf("follower:following ratio %.2f (followers=%d, following=%d)",
					ratio, acc.FollowerCount, acc.FollowingCount))
		}
	}

	// Engagement rate above 50% on an account with more than 10k followers is
	// statistically implausible — almost certainly inflated by bots or pods.
	if acc.FollowerCount > 10_000 && acc.EngagementRate > 50 {
		go FlagUser(context.Background(), promoterObjID, models.FraudAbnormalEngagement,
			fmt.Sprintf("engagement rate %.1f%% with %d followers exceeds plausible ceiling",
				acc.EngagementRate, acc.FollowerCount))
	}
}

// CheckAccount runs the same fraud heuristics when a social account is first connected.
// Called asynchronously from the users service so onboarding is never delayed.
func CheckAccount(ctx context.Context, promoterObjID bson.ObjectID, acc *models.SocialAccount) {
	CheckSubmission(ctx, promoterObjID, acc)
}

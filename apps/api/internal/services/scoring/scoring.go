package scoring

import (
	"context"
	"math"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── Score sub-components ─────────────────────────────────────
// Each component has a defined ceiling; together they total 100.
// followers(30) + engagement(25) + age(15) + completion(20) + audience(10)

func ScoreFollowers(count int64) float64 {
	if count <= 0 {
		return 0
	}
	return math.Min(30, (math.Log10(float64(count)+1)/6)*30)
}

func ScoreEngagement(rate float64) float64 {
	return math.Min(25, (rate/5.0)*25)
}

func ScoreAge(days int) float64 {
	return math.Min(15, (float64(days)/365.0)*15)
}

func ScoreAudienceQuality(followers, following int64) float64 {
	if following == 0 {
		if followers > 0 {
			return 10
		}
		return 0
	}
	return math.Min(10, (float64(followers)/float64(following))*10)
}

func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ── Completion score ─────────────────────────────────────────

// ComputeCompletionScore returns 0–20 based on the promoter's submission track record.
// No completed submissions → 10 (neutral). 100% approval rate → 20. 100% rejection → 0.
func ComputeCompletionScore(ctx context.Context, promoterObjID bson.ObjectID) float64 {
	col := database.GetCollection(models.SubmissionsCollection)

	approved, _ := col.CountDocuments(ctx, bson.M{
		"promoterId": promoterObjID,
		"status":     models.SubmissionStatusApproved,
	})
	rejected, _ := col.CountDocuments(ctx, bson.M{
		"promoterId": promoterObjID,
		"status":     models.SubmissionStatusRejected,
	})

	total := approved + rejected
	if total == 0 {
		return 10
	}

	return Round2((float64(approved) / float64(total)) * 20)
}

// ── Full score ───────────────────────────────────────────────

// ComputeFullScore calculates the complete influence score for a social account,
// including the dynamic completion score derived from the promoter's submission history.
func ComputeFullScore(ctx context.Context, acc *models.SocialAccount, promoterObjID bson.ObjectID) float64 {
	cs := ComputeCompletionScore(ctx, promoterObjID)
	return Round2(
		ScoreFollowers(acc.FollowerCount) +
			ScoreEngagement(acc.EngagementRate) +
			ScoreAge(acc.AccountAge) +
			ScoreAudienceQuality(acc.FollowerCount, acc.FollowingCount) +
			cs,
	)
}

// ── Score persistence ────────────────────────────────────────

// RefreshAllAccounts recomputes and persists the influence score for every
// social account belonging to the given promoter. Designed to be called as a
// goroutine after submission approval or rejection.
func RefreshAllAccounts(ctx context.Context, promoterObjID bson.ObjectID) {
	cs := ComputeCompletionScore(ctx, promoterObjID)

	col := database.GetCollection(models.SocialAccountsCollection)
	cursor, err := col.Find(ctx, bson.M{"userId": promoterObjID})
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	var accounts []models.SocialAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return
	}

	for _, acc := range accounts {
		score := Round2(
			ScoreFollowers(acc.FollowerCount) +
				ScoreEngagement(acc.EngagementRate) +
				ScoreAge(acc.AccountAge) +
				ScoreAudienceQuality(acc.FollowerCount, acc.FollowingCount) +
				cs,
		)
		_, _ = col.UpdateOne(ctx,
			bson.M{"_id": acc.ID},
			bson.M{"$set": bson.M{"influenceScore": score}},
		)
	}
}

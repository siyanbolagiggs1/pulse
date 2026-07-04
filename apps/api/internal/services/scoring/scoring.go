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
// followerTier(80) + completion(20)

// FollowerTier maps a raw follower count to a tier number: 100–500 followers
// is tier 1, 501–1000 is tier 2, then +1 tier per +500 followers thereafter.
// Below 100 followers there is no tier (0) — admin approval requires >=100.
func FollowerTier(followerCount int64) int {
	if followerCount < 100 {
		return 0
	}
	if followerCount <= 500 {
		return 1
	}
	return 2 + int((followerCount-501)/500)
}

// ScoreFollowerTier converts a tier into the 0–80 point follower component.
// Uses a log curve (diminishing returns per tier, mirroring the previous
// per-follower log10 curve's shape) that saturates at the 80-point cap
// around tier 20 (~10k followers).
func ScoreFollowerTier(tier int) float64 {
	if tier <= 0 {
		return 0
	}
	return math.Min(80, (math.Log10(float64(tier)+1)/math.Log10(21))*80)
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
	return Round2(ScoreFollowerTier(acc.Tier) + cs)
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
		score := Round2(ScoreFollowerTier(acc.Tier) + cs)
		_, _ = col.UpdateOne(ctx,
			bson.M{"_id": acc.ID},
			bson.M{"$set": bson.M{"influenceScore": score}},
		)
	}
}

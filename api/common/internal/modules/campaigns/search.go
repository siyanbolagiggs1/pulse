package campaigns

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/ai"
	"github.com/pulse/api/internal/services/search"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Marketplace search: hybrid keyword + meaning search over the campaigns a
// promoter can take right now.
//
//  1. Live filter (MongoDB): active, not ended, not full, and with budget
//     left for at least one base payout; optionally only campaigns one of
//     the promoter's active social accounts qualifies for. Same rules as
//     submissions.createSubmission, so nothing shown is refused later.
//  2. Keyword ranking: Atlas Search BM25 on title (boosted) and description,
//     then the live filter.
//  3. Meaning ranking: exact cosine between the query embedding and each
//     live campaign's embedding. The live set is small, so an exact scan
//     after the filter is both accurate and cheap, and it saves the vector
//     index the smallest Atlas tiers don't have room for.
//  4. Reciprocal rank fusion. Keyword hits always count; meaning-only hits
//     must reach searchMinSimilarity, so a query never returns "everything".
//
// Without Atlas Search the keyword ranking is a simple term match in
// process; without an embeddings key it is keyword-only.
const (
	searchLimit         = 50
	searchMinSimilarity = 0.60
	liveCandidateCap    = 2000
)

// liveFilter matches campaigns that can accept a submission now.
func liveFilter(now time.Time, platform string) bson.M {
	f := bson.M{
		"status":  models.CampaignStatusActive,
		"endDate": bson.M{"$gt": now},
		"$expr": bson.M{"$and": bson.A{
			bson.M{"$lt": bson.A{"$currentParticipants", "$maxParticipants"}},
			bson.M{"$gte": bson.A{"$remainingBudget", "$baseRepostRate"}},
		}},
	}
	if platform != "" {
		f["platform"] = platform
	}
	return f
}

// eligibilityFilter matches campaigns the user can take with at least one of
// their active social accounts, excluding their own. ok is false when they
// have no active account, so nothing can match.
func eligibilityFilter(ctx context.Context, userID string) (filter bson.M, ok bool, err error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, false, nil
	}
	cursor, err := database.GetCollection(models.SocialAccountsCollection).Find(ctx,
		bson.M{"userId": uid, "status": models.SocialAccountActive})
	if err != nil {
		return nil, false, err
	}
	var accounts []models.SocialAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, false, err
	}
	if len(accounts) == 0 {
		return nil, false, nil
	}
	anyAccount := bson.A{}
	for _, a := range accounts {
		anyAccount = append(anyAccount, bson.M{
			"platform":          a.Platform,
			"minFollowers":      bson.M{"$lte": a.FollowerCount},
			"minInfluenceScore": bson.M{"$lte": a.InfluenceScore},
		})
	}
	return bson.M{"businessId": bson.M{"$ne": uid}, "$or": anyAccount}, true, nil
}

// searchCampaigns returns up to searchLimit live campaigns, best first.
func searchCampaigns(ctx context.Context, userID string, q CampaignListQuery) ([]models.Campaign, error) {
	filter := liveFilter(time.Now().UTC(), q.Platform)
	if q.Eligible {
		eligible, ok, err := eligibilityFilter(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return []models.Campaign{}, nil
		}
		filter = bson.M{"$and": bson.A{filter, eligible}}
	}
	col := database.GetCollection(models.CampaignsCollection)

	query := strings.TrimSpace(q.Q)
	if query == "" {
		cursor, err := col.Find(ctx, filter, options.Find().
			SetProjection(bson.M{"embedding": 0}).
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetLimit(searchLimit))
		if err != nil {
			return nil, err
		}
		campaigns := []models.Campaign{}
		return campaigns, cursor.All(ctx, &campaigns)
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetLimit(liveCandidateCap))
	if err != nil {
		return nil, err
	}
	var live []models.Campaign
	if err := cursor.All(ctx, &live); err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]models.Campaign, len(live))
	for _, c := range live {
		byID[c.ID] = c
	}

	meaning := meaningRanking(ctx, query, live)
	keywords, err := atlasKeywordRanking(ctx, col, query, filter)
	if err != nil {
		log.Printf("[Search] Atlas campaign search failed, using an in-process keyword match: %v", err)
	}
	if keywords == nil {
		keywords = termRanking(query, live)
	}

	out := []models.Campaign{}
	for _, f := range search.RRF(meaning, keywords) {
		c, ok := byID[f.ID]
		if !ok {
			continue // ended or filled between the two queries
		}
		c.Embedding = nil
		out = append(out, c)
		if len(out) == searchLimit {
			break
		}
	}
	return out, nil
}

// meaningRanking ranks live campaigns by cosine with the query, keeping
// those at or above searchMinSimilarity. Nil without an embeddings key or if
// the embedding call fails (search still works on keywords).
func meaningRanking(ctx context.Context, query string, live []models.Campaign) []bson.ObjectID {
	if config.App.GeminiAPIKey == "" {
		return nil
	}
	embedding, err := ai.EmbedFor(ctx, query, ai.TaskQuery)
	if err != nil {
		log.Printf("[Search] embedding the campaign query failed, searching keywords only: %v", err)
		return nil
	}
	version := campaignEmbeddingVersion()
	type scored struct {
		id bson.ObjectID
		s  float64
	}
	var hits []scored
	for _, c := range live {
		if c.EmbeddingModel != version {
			continue
		}
		if s := ai.CosineSimilarity(embedding, c.Embedding); s >= searchMinSimilarity {
			hits = append(hits, scored{c.ID, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].s > hits[j].s })
	ids := make([]bson.ObjectID, len(hits))
	for i, h := range hits {
		ids[i] = h.id
	}
	return ids
}

// atlasKeywordRanking runs the BM25 query on the campaigns_text index. Nil,
// nil when the index isn't ready.
func atlasKeywordRanking(ctx context.Context, col *mongo.Collection, query string, filter bson.M) ([]bson.ObjectID, error) {
	if !database.CampaignSearchReady() {
		return nil, nil
	}
	text := func(path string, boost float64) bson.D {
		return bson.D{{Key: "text", Value: bson.D{
			{Key: "query", Value: query},
			{Key: "path", Value: path},
			{Key: "fuzzy", Value: bson.D{{Key: "maxEdits", Value: 1}, {Key: "prefixLength", Value: 2}}},
			{Key: "score", Value: bson.D{{Key: "boost", Value: bson.D{{Key: "value", Value: boost}}}}},
		}}}
	}
	cursor, err := col.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$search", Value: bson.D{
			{Key: "index", Value: database.CampaignTextIndex},
			{Key: "compound", Value: bson.D{{Key: "should", Value: bson.A{
				text("title", 3),
				text("description", 1),
			}}}},
		}}},
		{{Key: "$match", Value: filter}},
		{{Key: "$limit", Value: searchLimit}},
		{{Key: "$project", Value: bson.D{{Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var docs []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]bson.ObjectID, len(docs))
	for i, d := range docs {
		ids[i] = d.ID
	}
	return ids, nil
}

var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "for": true, "i": true, "in": true,
	"is": true, "me": true, "my": true, "of": true, "on": true, "or": true, "the": true,
	"to": true, "with": true,
}

// termRanking is the keyword fallback without Atlas Search: campaigns that
// contain query words, title matches weighted 3× as in the Atlas query.
func termRanking(query string, live []models.Campaign) []bson.ObjectID {
	var terms []string
	for _, t := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if !stopWords[t] {
			terms = append(terms, t)
		}
	}
	type scored struct {
		id bson.ObjectID
		s  int
	}
	var hits []scored
	for _, c := range live {
		title, desc := strings.ToLower(c.Title), strings.ToLower(c.Description)
		s := 0
		for _, t := range terms {
			if strings.Contains(title, t) {
				s += 3
			}
			if strings.Contains(desc, t) {
				s++
			}
		}
		if s > 0 {
			hits = append(hits, scored{c.ID, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].s > hits[j].s })
	ids := make([]bson.ObjectID, len(hits))
	for i, h := range hits {
		ids[i] = h.id
	}
	return ids
}

// campaignEmbeddingVersion marks campaign vectors as document-side
// embeddings, so a model, size or task change triggers a re-embed.
func campaignEmbeddingVersion() string {
	return ai.EmbeddingVersion() + "/" + ai.TaskDocument
}

func campaignText(title, description string) string {
	return title + "\n\n" + description
}

// embedCampaign stores the embedding for a campaign's title and description.
// Called in the background after a create or a title/description change; a
// failure only means the campaign is found by keywords until the next start.
func embedCampaign(id bson.ObjectID, title, description string) {
	if config.App.GeminiAPIKey == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	embedding, err := ai.EmbedFor(ctx, campaignText(title, description), ai.TaskDocument)
	if err != nil {
		log.Printf("[Search] could not embed campaign %s: %v", id.Hex(), err)
		return
	}
	if _, err := database.GetCollection(models.CampaignsCollection).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"embedding": embedding, "embeddingModel": campaignEmbeddingVersion()}}); err != nil {
		log.Printf("[Search] could not store campaign %s embedding: %v", id.Hex(), err)
	}
}

// BackfillCampaignEmbeddings embeds every active or paused campaign without a
// current embedding (created before search, or with an older model). Run once
// at start; a no-op when everything is current.
func BackfillCampaignEmbeddings(ctx context.Context) {
	if config.App.GeminiAPIKey == "" {
		return
	}
	version := campaignEmbeddingVersion()
	col := database.GetCollection(models.CampaignsCollection)
	cursor, err := col.Find(ctx, bson.M{
		"embeddingModel": bson.M{"$ne": version},
		"status":         bson.M{"$in": bson.A{models.CampaignStatusActive, models.CampaignStatusPaused}},
	}, options.Find().SetProjection(bson.M{"title": 1, "description": 1}))
	if err != nil {
		log.Printf("[Search] campaign backfill: %v", err)
		return
	}
	var campaigns []models.Campaign
	if err := cursor.All(ctx, &campaigns); err != nil {
		log.Printf("[Search] campaign backfill: %v", err)
		return
	}
	done := 0
	for _, c := range campaigns {
		embedding, err := ai.EmbedFor(ctx, campaignText(c.Title, c.Description), ai.TaskDocument)
		if err != nil {
			log.Printf("[Search] campaign backfill stopped after %d of %d: %v", done, len(campaigns), err)
			return
		}
		if _, err := col.UpdateOne(ctx, bson.M{"_id": c.ID}, bson.M{"$set": bson.M{
			"embedding": embedding, "embeddingModel": version,
		}}); err != nil {
			log.Printf("[Search] campaign backfill stopped after %d of %d: %v", done, len(campaigns), err)
			return
		}
		done++
		time.Sleep(700 * time.Millisecond) // stay under the embeddings per-minute limit
	}
	if done > 0 {
		log.Printf("[Search] campaign backfill embedded %d campaigns with %s", done, version)
	}
}

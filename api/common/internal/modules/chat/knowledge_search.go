package chat

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/ai"
	"github.com/pulse/api/internal/services/search"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Retrieval for the support assistant's learned knowledge base.
//
// Two rankings are fused with reciprocal rank fusion (RRF): Atlas Vector
// Search over the question embedding (meaning) and Atlas Search BM25 over the
// question and answer text (exact words such as "tiktok" or "paystack").
// Fusion decides the order; whether an entry is shown to the model at all is
// decided by its cosine similarity, with a lower bar for the entry that is
// first on meaning and also near the top on keywords. Short queries ("tiktok
// support?") rank the right entry first but score below the plain threshold;
// keyword agreement is what lets them through without letting loosely
// related entries in.
//
// Without Atlas Search (plain MongoDB, or while the indexes build) the
// vector ranking comes from an in-process cosine scan and there is no text
// ranking, which is exactly the original behaviour.
const (
	kbSimilarityThreshold = 0.80 // accept on meaning alone
	kbAgreementThreshold  = 0.73 // accept when first on meaning and top kbAgreementRank on keywords
	kbAgreementRank       = 3
	kbMaxMatches          = 3
	kbCandidates          = 10
)

type knowledgeHit struct {
	entry      models.KnowledgeEntry
	similarity float64 // cosine with the question
	vectorRank int     // 1-based; 0 = not in the vector ranking
	textRank   int     // 1-based; 0 = not in the text ranking
	fused      float64
}

type knowledgeResult struct {
	mode    string // "hybrid" or "scan"
	matches []knowledgeHit
}

// retrieveKnowledge returns up to kbMaxMatches entries for question, best
// first. embedding is the question's embedding.
func retrieveKnowledge(ctx context.Context, question string, embedding []float32) (knowledgeResult, error) {
	if database.KnowledgeSearchReady() {
		vector, text, err := atlasCandidates(ctx, question, embedding)
		if err == nil {
			return knowledgeResult{"hybrid", selectKnowledge(fuse(vector, text))}, nil
		}
		log.Printf("[AISupport] Atlas knowledge search failed, falling back to a scan: %v", err)
	}
	vector, err := scanCandidates(ctx, embedding)
	if err != nil {
		return knowledgeResult{mode: "scan"}, err
	}
	return knowledgeResult{"scan", selectKnowledge(fuse(vector, nil))}, nil
}

// knowledgeFields are the fields both searches return. The embedding comes
// back so every candidate gets an exact cosine, whichever ranking found it.
var knowledgeFields = bson.D{
	{Key: "question", Value: 1}, {Key: "answer", Value: 1},
	{Key: "embedding", Value: 1}, {Key: "embeddingModel", Value: 1},
}

func atlasCandidates(ctx context.Context, question string, embedding []float32) (vector, text []knowledgeHit, err error) {
	col := database.GetCollection(models.KnowledgeCollection)

	vector, err = runKnowledgeSearch(ctx, col, embedding, mongo.Pipeline{
		{{Key: "$vectorSearch", Value: bson.D{
			{Key: "index", Value: database.KnowledgeVectorIndex},
			{Key: "path", Value: "embedding"},
			{Key: "queryVector", Value: embedding},
			{Key: "numCandidates", Value: kbCandidates * 10},
			{Key: "limit", Value: kbCandidates},
		}}},
		{{Key: "$project", Value: knowledgeFields}},
	})
	if err != nil {
		return nil, nil, err
	}

	textQuery := func(path string, boost float64) bson.D {
		return bson.D{{Key: "text", Value: bson.D{
			{Key: "query", Value: question},
			{Key: "path", Value: path},
			{Key: "fuzzy", Value: bson.D{{Key: "maxEdits", Value: 1}, {Key: "prefixLength", Value: 2}}},
			{Key: "score", Value: bson.D{{Key: "boost", Value: bson.D{{Key: "value", Value: boost}}}}},
		}}}
	}
	text, err = runKnowledgeSearch(ctx, col, embedding, mongo.Pipeline{
		{{Key: "$search", Value: bson.D{
			{Key: "index", Value: database.KnowledgeTextIndex},
			{Key: "compound", Value: bson.D{{Key: "should", Value: bson.A{
				textQuery("question", 2),
				textQuery("answer", 1),
			}}}},
		}}},
		{{Key: "$limit", Value: kbCandidates}},
		{{Key: "$project", Value: knowledgeFields}},
	})
	if err != nil {
		return nil, nil, err
	}
	return vector, text, nil
}

func runKnowledgeSearch(ctx context.Context, col *mongo.Collection, embedding []float32, pipeline mongo.Pipeline) ([]knowledgeHit, error) {
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var entries []models.KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return rankedHits(entries, embedding), nil
}

// scanCandidates ranks every entry by cosine in-process.
func scanCandidates(ctx context.Context, embedding []float32) ([]knowledgeHit, error) {
	cursor, err := database.GetCollection(models.KnowledgeCollection).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var entries []models.KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	hits := rankedHits(entries, embedding)
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].similarity > hits[j].similarity })
	if len(hits) > kbCandidates {
		hits = hits[:kbCandidates]
	}
	return hits, nil
}

// rankedHits scores entries in the order given. An embedding from another
// model can't be compared, so it scores 0 until the backfill re-embeds it.
func rankedHits(entries []models.KnowledgeEntry, embedding []float32) []knowledgeHit {
	version := ai.EmbeddingVersion()
	hits := make([]knowledgeHit, 0, len(entries))
	for _, e := range entries {
		h := knowledgeHit{entry: e}
		if e.EmbeddingModel == version {
			h.similarity = ai.CosineSimilarity(embedding, e.Embedding)
		}
		hits = append(hits, h)
	}
	return hits
}

// fuse merges the two rankings by reciprocal rank fusion, best first.
func fuse(vector, text []knowledgeHit) []knowledgeHit {
	byID := map[bson.ObjectID]knowledgeHit{}
	ids := func(list []knowledgeHit) []bson.ObjectID {
		out := make([]bson.ObjectID, len(list))
		for i, h := range list {
			out[i] = h.entry.ID
			byID[h.entry.ID] = h
		}
		return out
	}
	fused := search.RRF(ids(vector), ids(text))
	out := make([]knowledgeHit, 0, len(fused))
	for _, f := range fused {
		h := byID[f.ID]
		h.vectorRank, h.textRank, h.fused = f.Ranks[0], f.Ranks[1], f.Score
		out = append(out, h)
	}
	return out
}

// selectKnowledge keeps the fused hits close enough to show the model.
func selectKnowledge(fused []knowledgeHit) []knowledgeHit {
	var out []knowledgeHit
	for _, h := range fused {
		agree := h.vectorRank == 1 && h.textRank >= 1 && h.textRank <= kbAgreementRank
		if h.similarity >= kbSimilarityThreshold || (agree && h.similarity >= kbAgreementThreshold) {
			out = append(out, h)
			if len(out) == kbMaxMatches {
				break
			}
		}
	}
	return out
}

// BackfillKnowledgeEmbeddings re-embeds every learned answer whose embedding
// came from a different model or size than the one configured, so it can be
// found again. Run once at start; a no-op when everything is current.
func BackfillKnowledgeEmbeddings(ctx context.Context) {
	if config.App.GeminiAPIKey == "" {
		return
	}
	version := ai.EmbeddingVersion()
	col := database.GetCollection(models.KnowledgeCollection)
	cursor, err := col.Find(ctx, bson.M{"embeddingModel": bson.M{"$ne": version}})
	if err != nil {
		log.Printf("[AISupport] knowledge backfill: %v", err)
		return
	}
	var entries []models.KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		log.Printf("[AISupport] knowledge backfill: %v", err)
		return
	}
	done := 0
	for _, e := range entries {
		embedding, err := ai.Embed(ctx, e.Question)
		if err != nil {
			log.Printf("[AISupport] knowledge backfill stopped after %d of %d: %v", done, len(entries), err)
			return
		}
		if _, err := col.UpdateOne(ctx, bson.M{"_id": e.ID}, bson.M{"$set": bson.M{
			"embedding": embedding, "embeddingModel": version,
		}}); err != nil {
			log.Printf("[AISupport] knowledge backfill stopped after %d of %d: %v", done, len(entries), err)
			return
		}
		done++
		time.Sleep(700 * time.Millisecond) // stay under the embeddings per-minute limit
	}
	if done > 0 {
		log.Printf("[AISupport] knowledge backfill re-embedded %d entries with %s", done, version)
	}
}

package database

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Atlas Search indexes. Three in total, which is the most the smallest Atlas
// tiers allow:
//   - support_knowledge: a vector index on the question embedding and a
//     full-text (BM25) index on the question and answer;
//   - campaigns: a full-text index on the title and description. Campaign
//     meaning is ranked by exact cosine over the live, filtered campaigns
//     instead of a fourth (vector) index.
const (
	KnowledgeVectorIndex = "support_knowledge_vector"
	KnowledgeTextIndex   = "support_knowledge_text"
	CampaignTextIndex    = "campaigns_text"
)

// Until an index is queryable (or on a MongoDB without Atlas Search, such as
// the plain mongo image in docker-compose) its search falls back to an
// in-process ranking.
var knowledgeSearchReady, campaignSearchReady atomic.Bool

func KnowledgeSearchReady() bool { return knowledgeSearchReady.Load() }
func CampaignSearchReady() bool  { return campaignSearchReady.Load() }

type searchIndexSpec struct {
	name, typ  string
	definition bson.D
	dims       int // vector indexes only
}

// EnsureSearchIndexes creates the search indexes that are missing, then
// waits in the background until Atlas reports them queryable. Never fatal:
// the API works without them.
func EnsureSearchIndexes(dims int) {
	textIndex := func(name string, fields ...string) searchIndexSpec {
		mapped := bson.D{}
		for _, f := range fields {
			mapped = append(mapped, bson.E{Key: f, Value: bson.D{{Key: "type", Value: "string"}}})
		}
		return searchIndexSpec{name: name, typ: "search", definition: bson.D{
			{Key: "analyzer", Value: "lucene.english"},
			{Key: "mappings", Value: bson.D{{Key: "dynamic", Value: false}, {Key: "fields", Value: mapped}}},
		}}
	}

	ensureSearchIndexes(models.KnowledgeCollection, &knowledgeSearchReady, []searchIndexSpec{
		{name: KnowledgeVectorIndex, typ: "vectorSearch", dims: dims, definition: bson.D{{Key: "fields", Value: bson.A{
			bson.D{
				{Key: "type", Value: "vector"},
				{Key: "path", Value: "embedding"},
				{Key: "numDimensions", Value: dims},
				{Key: "similarity", Value: "cosine"},
			},
		}}}},
		textIndex(KnowledgeTextIndex, "question", "answer"),
	})
	ensureSearchIndexes(models.CampaignsCollection, &campaignSearchReady, []searchIndexSpec{
		textIndex(CampaignTextIndex, "title", "description"),
	})
}

func ensureSearchIndexes(collection string, ready *atomic.Bool, specs []searchIndexSpec) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// A search index can't be created on a collection that doesn't exist
	// yet (a fresh database, or no learned answers so far).
	if err := DB.CreateCollection(ctx, collection); err != nil && !mongo.IsDuplicateKeyError(err) {
		var cmdErr mongo.CommandError
		if !errors.As(err, &cmdErr) || cmdErr.Name != "NamespaceExists" {
			log.Printf("[Search] could not create collection %s: %v", collection, err)
		}
	}

	view := GetCollection(collection).SearchIndexes()
	existing, err := listSearchIndexes(ctx, view)
	if err != nil {
		log.Printf("[Search] Atlas Search unavailable for %s, using in-process ranking: %v", collection, err)
		return
	}

	for _, ix := range specs {
		current, ok := existing[ix.name]
		switch {
		case !ok:
			model := mongo.SearchIndexModel{
				Definition: ix.definition,
				Options:    options.SearchIndexes().SetName(ix.name).SetType(ix.typ),
			}
			if _, err := view.CreateOne(ctx, model); err != nil {
				log.Printf("[Search] could not create search index %s, using in-process ranking: %v", ix.name, err)
				return
			}
			log.Printf("[Search] created search index %s", ix.name)
		case ix.dims != 0 && current.dims != ix.dims:
			// The embedding size changed (GEMINI_EMBEDDING_DIMS); the vector
			// index must match it or every query fails.
			if err := view.UpdateOne(ctx, ix.name, ix.definition); err != nil {
				log.Printf("[Search] could not update search index %s, using in-process ranking: %v", ix.name, err)
				return
			}
			log.Printf("[Search] updated search index %s to %d dimensions", ix.name, ix.dims)
		}
	}

	go waitUntilQueryable(collection, view, ready, specs)
}

// waitUntilQueryable polls until every index can serve queries. A new index
// on a small collection is usually ready within a minute.
func waitUntilQueryable(collection string, view mongo.SearchIndexView, ready *atomic.Bool, specs []searchIndexSpec) {
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		indexes, err := listSearchIndexes(ctx, view)
		cancel()
		if err == nil && allReady(indexes, specs) {
			ready.Store(true)
			log.Printf("[Search] %s search indexes are queryable", collection)
			return
		}
		time.Sleep(10 * time.Second)
	}
	log.Printf("[Search] %s search indexes not queryable after 10 minutes, using in-process ranking", collection)
}

func allReady(indexes map[string]searchIndexState, specs []searchIndexSpec) bool {
	for _, ix := range specs {
		st := indexes[ix.name]
		if !st.ready() || (ix.dims != 0 && st.dims != ix.dims) {
			return false
		}
	}
	return true
}

type searchIndexState struct {
	status    string // PENDING, BUILDING, READY, FAILED, …
	queryable bool
	dims      int // vector indexes only
}

// ready is true once the latest definition is built, not just an older one.
func (s searchIndexState) ready() bool { return s.queryable && s.status == "READY" }

// listSearchIndexes returns each search index's state by name. It reads the
// latest definition, so an update still building already shows its new size.
func listSearchIndexes(ctx context.Context, view mongo.SearchIndexView) (map[string]searchIndexState, error) {
	cursor, err := view.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	var docs []struct {
		Name             string `bson:"name"`
		Status           string `bson:"status"`
		Queryable        bool   `bson:"queryable"`
		LatestDefinition struct {
			Fields []struct {
				NumDimensions int `bson:"numDimensions"`
			} `bson:"fields"`
		} `bson:"latestDefinition"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make(map[string]searchIndexState, len(docs))
	for _, d := range docs {
		st := searchIndexState{status: d.Status, queryable: d.Queryable}
		if len(d.LatestDefinition.Fields) > 0 {
			st.dims = d.LatestDefinition.Fields[0].NumDimensions
		}
		out[d.Name] = st
	}
	return out, nil
}

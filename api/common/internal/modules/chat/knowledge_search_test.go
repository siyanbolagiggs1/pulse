package chat

import (
	"testing"

	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func hit(id bson.ObjectID, similarity float64) knowledgeHit {
	return knowledgeHit{entry: models.KnowledgeEntry{ID: id}, similarity: similarity}
}

func TestFuseRanksAgreementFirst(t *testing.T) {
	a, b, c := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	// b is second on meaning but first on keywords; a is only found by meaning.
	vector := []knowledgeHit{hit(a, 0.78), hit(b, 0.75), hit(c, 0.60)}
	text := []knowledgeHit{hit(b, 0.75)}

	got := fuse(vector, text)
	if len(got) != 3 {
		t.Fatalf("got %d hits, want 3", len(got))
	}
	if got[0].entry.ID != b || got[0].vectorRank != 2 || got[0].textRank != 1 {
		t.Fatalf("first = %+v, want b with vectorRank 2, textRank 1", got[0])
	}
	if got[1].entry.ID != a || got[1].textRank != 0 {
		t.Fatalf("second = %+v, want a found by meaning only", got[1])
	}
}

func TestSelectKnowledge(t *testing.T) {
	a, b, c, d := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	e := bson.NewObjectID()
	vector := []knowledgeHit{hit(a, 0.74), hit(b, 0.82), hit(c, 0.72), hit(d, 0.65), hit(e, 0.76)}
	text := []knowledgeHit{hit(a, 0.74), hit(d, 0.65), hit(e, 0.76)}

	got := selectKnowledge(fuse(vector, text))
	ids := map[bson.ObjectID]bool{}
	for _, h := range got {
		ids[h.entry.ID] = true
	}
	switch {
	case !ids[a]:
		t.Error("a: below the plain threshold but first on meaning and on keywords; want it kept")
	case !ids[b]:
		t.Error("b: above the plain threshold; want it kept")
	case ids[c]:
		t.Error("c: below the plain threshold and no keyword match; want it dropped")
	case ids[d]:
		t.Error("d: below the agreement threshold; want it dropped")
	case ids[e]:
		t.Error("e: a keyword match but not first on meaning; want it dropped")
	}
}

func TestSelectKnowledgeCapsMatches(t *testing.T) {
	var vector []knowledgeHit
	for i := 0; i < 5; i++ {
		vector = append(vector, hit(bson.NewObjectID(), 0.9))
	}
	if got := selectKnowledge(fuse(vector, nil)); len(got) != kbMaxMatches {
		t.Fatalf("got %d matches, want %d", len(got), kbMaxMatches)
	}
}

// Without a text ranking (the scan fallback) the rule is the original one:
// cosine at or above kbSimilarityThreshold.
func TestSelectKnowledgeScanMatchesOriginalRule(t *testing.T) {
	a, b := bson.NewObjectID(), bson.NewObjectID()
	got := selectKnowledge(fuse([]knowledgeHit{hit(a, 0.79), hit(b, 0.80)}, nil))
	if len(got) != 1 || got[0].entry.ID != b {
		t.Fatalf("got %+v, want only b", got)
	}
}

package search

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestRRF(t *testing.T) {
	a, b, c := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	got := RRF([]bson.ObjectID{a, b, c}, []bson.ObjectID{b, c})

	if len(got) != 3 {
		t.Fatalf("got %d results, want 3", len(got))
	}
	// b: 1/62 + 1/61 beats a: 1/61 alone.
	if got[0].ID != b || got[0].Ranks[0] != 2 || got[0].Ranks[1] != 1 {
		t.Fatalf("first = %+v, want b ranked 2 and 1", got[0])
	}
	if got[2].ID != a || got[2].Ranks[1] != 0 {
		t.Fatalf("last = %+v, want a, absent from the second ranking", got[2])
	}
}

func TestRRFIgnoresDuplicatesWithinARanking(t *testing.T) {
	a := bson.NewObjectID()
	got := RRF([]bson.ObjectID{a, a})
	if len(got) != 1 || got[0].Ranks[0] != 1 || got[0].Score != 1.0/61 {
		t.Fatalf("got %+v, want one result ranked 1 with one contribution", got)
	}
}

func TestRRFEmpty(t *testing.T) {
	if got := RRF(nil, nil); len(got) != 0 {
		t.Fatalf("got %d results, want 0", len(got))
	}
}

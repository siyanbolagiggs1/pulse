// Package search holds the ranking helpers shared by Pulse's hybrid searches
// (support knowledge and marketplace campaigns).
package search

import (
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// RRFK is the standard reciprocal rank fusion constant: it damps the gap
// between neighbouring ranks so no single ranking dominates.
const RRFK = 60

// Fused is one document after fusion. Ranks holds its 1-based rank in each
// input ranking, in the order the rankings were given; 0 means absent.
type Fused struct {
	ID    bson.ObjectID
	Score float64
	Ranks []int
}

// RRF merges rankings (each a list of IDs, best first) by reciprocal rank
// fusion: score = Σ 1/(RRFK + rank). Best first; ties keep first-seen order.
func RRF(rankings ...[]bson.ObjectID) []Fused {
	byID := map[bson.ObjectID]*Fused{}
	var order []bson.ObjectID
	for r, ranking := range rankings {
		for i, id := range ranking {
			f, ok := byID[id]
			if !ok {
				f = &Fused{ID: id, Ranks: make([]int, len(rankings))}
				byID[id] = f
				order = append(order, id)
			}
			if f.Ranks[r] != 0 {
				continue // duplicate within one ranking: keep the best rank
			}
			f.Ranks[r] = i + 1
			f.Score += 1 / float64(RRFK+i+1)
		}
	}
	out := make([]Fused, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

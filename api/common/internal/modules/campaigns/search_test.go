package campaigns

import (
	"testing"

	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTermRankingWeightsTitleMatches(t *testing.T) {
	food := models.Campaign{ID: bson.NewObjectID(), Title: "Lekki Food Festival", Description: "Repost our weekend event."}
	mention := models.Campaign{ID: bson.NewObjectID(), Title: "New gym in Yaba", Description: "Healthy food bar inside."}
	other := models.Campaign{ID: bson.NewObjectID(), Title: "Phone sale", Description: "Discounts on all phones."}

	got := termRanking("food in Lekki", []models.Campaign{mention, other, food})
	if len(got) != 2 || got[0] != food.ID || got[1] != mention.ID {
		t.Fatalf("got %v, want [food, mention]", got)
	}
}

func TestTermRankingIgnoresStopWords(t *testing.T) {
	c := models.Campaign{ID: bson.NewObjectID(), Title: "The best of the city", Description: "for you"}
	if got := termRanking("the for", []models.Campaign{c}); len(got) != 0 {
		t.Fatalf("got %v, want no matches on stop words only", got)
	}
}

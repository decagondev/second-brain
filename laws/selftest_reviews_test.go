package laws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/reviews"
)

func TestLAW16_AccessReviews(t *testing.T) {
	reviewProposals = 0
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16Reviews(r, RealReviews) })
	if reviewProposals == 0 {
		t.Fatal("no removal ever became a manifest proposal: the law test checked nothing")
	}
}

// The review law test catches a review that misses people, an alert a day late, and
// removals that never become requests.
func TestSelf_ReviewMutationsAreCaught(t *testing.T) {
	firstGroup := RealReviews
	firstGroup.Open = func(ctx context.Context, tn manifest.Tenant, now time.Time, ro reviews.Roster, last reviews.LastUse, have func(string, string) bool, newID func(string) string) ([]reviews.Review, error) {
		cut := tn
		cut.Spaces = map[string]manifest.Space{}
		for k, v := range tn.Spaces {
			v.Members = v.Members[:1] // reads only the first member group
			cut.Spaces[k] = v
		}
		return reviews.OpenAll(ctx, cut, now, ro, last, have, newID)
	}
	if !violates(runs, func(r *rand.Rand) error { return law16Reviews(r, firstGroup) }) {
		t.Error("LAW-16 (reviews): a review that lists only the first member group's people was not caught")
	}
	late := RealReviews
	late.Overdue = func(rv reviews.Review, now time.Time) bool {
		return reviews.Overdue(rv, now.Add(-24*time.Hour)) // 15 days
	}
	if !violates(runs, func(r *rand.Rand) error { return law16Reviews(r, late) }) {
		t.Error("LAW-16 (reviews): an overdue alert a day late was not caught")
	}
	lost := RealReviews
	lost.Requests = func(tn manifest.Tenant, rv reviews.Review) (manifest.Tenant, bool, []reviews.Request) {
		return tn, false, []reviews.Request{} // removals vanish
	}
	if !violates(runs, func(r *rand.Rand) error { return law16Reviews(r, lost) }) {
		t.Error("LAW-16 (reviews): removals that never become change requests were not caught")
	}
}

package ask

import (
	"context"
	"math"
	"time"
)

// Repeats and old questions (sb-ask(3), Repeats and Old questions; SB-3402, SB-3404).
const (
	LikeWindow   = 90 * 24 * time.Hour
	LikeAt       = 0.85 // the cosine a repeat needs
	MaxLikeScan  = 200  // the newest questions of the queue compared
	OverdueAfter = 5    // working days
	RerouteAfter = 10   // working days
)

// WorkingDays counts the Mondays to Fridays in zone after from's day, up to and including
// to's day. A question sent on a Monday is 5 working days old the next Monday.
func WorkingDays(from, to time.Time, zone *time.Location) int {
	if zone == nil {
		zone = time.UTC
	}
	f, t := from.In(zone), to.In(zone)
	d := time.Date(f.Year(), f.Month(), f.Day(), 12, 0, 0, 0, zone)
	end := time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, zone)
	n := 0
	for d = d.AddDate(0, 0, 1); !d.After(end); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n++
		}
	}
	return n
}

// repeat is the id of the question of sq's queue that sq looks like: the closest of the
// newest MaxLikeScan in the last LikeWindow, at LikeAt or more. No embedder, or one that
// fails, links nothing: a repeat is a hint to the stewards, never needed to send.
func (x Stewards) repeat(ctx context.Context, sq StewardQuestion) string {
	if x.Agent.Embedder == nil {
		return ""
	}
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, sq.Space)
	if err != nil {
		return ""
	}
	var cand []StewardQuestion
	for _, q := range qs { // newest first
		if q.ID != sq.ID && sq.At.Sub(q.At) <= LikeWindow && len(cand) < MaxLikeScan {
			cand = append(cand, q)
		}
	}
	if len(cand) == 0 {
		return ""
	}
	texts := []string{sq.Question}
	for _, q := range cand {
		texts = append(texts, q.Question)
	}
	vs, err := x.Agent.Embedder.Embed(ctx, texts)
	if err != nil || len(vs) != len(texts) {
		return ""
	}
	best, at := "", LikeAt
	for i, q := range cand {
		if c := cosine(vs[0], vs[i+1]); c >= at {
			best, at = q.ID, c
		}
	}
	return best
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var ab, aa, bb float64
	for i := range a {
		ab += float64(a[i]) * float64(b[i])
		aa += float64(a[i]) * float64(a[i])
		bb += float64(b[i]) * float64(b[i])
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return ab / math.Sqrt(aa*bb)
}

// partner is the question q looks like in qs: its own like, or the open question that
// looks like it. "" when there is none.
func partner(q StewardQuestion, qs []StewardQuestion) string {
	if q.Like != "" {
		for _, o := range qs {
			if o.ID == q.Like {
				return o.ID
			}
		}
	}
	for _, o := range qs {
		if o.Like == q.ID && o.Reply == nil {
			return o.ID
		}
	}
	return ""
}

// waitedFor reports whether a question in space shows on the company-wide queue because it
// waited there (SB-3404).
func waitedFor(q StewardQuestion, space string) bool {
	return space == "company" && q.Also == "company"
}

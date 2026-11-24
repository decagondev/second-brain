package ask

import (
	"context"
	"fmt"
	"sort"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// MaxGapTerms is the most glossary terms found_nothing names, and MaxFeedbackDocs the most
// documents the stewards' feedback lists (SB-3101, SB-3104).
const (
	MaxGapTerms     = 10
	MaxFeedbackDocs = 10
)

// FoundNothing counts a space's answers that found nothing this week and last, and the
// glossary terms their questions used (sb-ask(3), Questions that found nothing; SB-3101).
type FoundNothing struct {
	ThisWeek int      `json:"this_week"`
	LastWeek int      `json:"last_week"`
	Terms    []string `json:"terms"`
}

// foundNothing reads traces only: an ask in space whose notice is no_sources or
// most_uncited, by week back from now. It keeps no question text.
func foundNothing(traces []Trace, space string, now time.Time) FoundNothing {
	out := FoundNothing{Terms: []string{}}
	uses := map[string]int{}
	week := now.Add(-7 * 24 * time.Hour)
	for _, t := range traces {
		if t.Space != space || !nothing(t) || t.At.Before(week.Add(-7*24*time.Hour)) || t.At.After(now) {
			continue
		}
		if t.At.Before(week) {
			out.LastWeek++
		} else {
			out.ThisWeek++
		}
		for _, term := range t.Response.Terms {
			uses[term.Term]++
		}
	}
	for term := range uses {
		out.Terms = append(out.Terms, term)
	}
	sort.Slice(out.Terms, func(i, j int) bool {
		if uses[out.Terms[i]] != uses[out.Terms[j]] {
			return uses[out.Terms[i]] > uses[out.Terms[j]]
		}
		return out.Terms[i] < out.Terms[j]
	})
	if len(out.Terms) > MaxGapTerms {
		out.Terms = out.Terms[:MaxGapTerms]
	}
	return out
}

// nothing is an ask that found nothing: no sources, or mostly uncited.
func nothing(t Trace) bool {
	if t.Response.Agent != "" && t.Response.Agent != "ask" || t.Response.Notice == nil {
		return false
	}
	return *t.Response.Notice == NoticeNoSources || *t.Response.Notice == NoticeMostUncited
}

// FoundNothingBySpace counts, per space (spc_NAME or company), the asks between from and
// to that found nothing, for the stewards' digest line (SB-3101).
func FoundNothingBySpace(traces []Trace, from, to time.Time) map[string]int {
	out := map[string]int{}
	for _, t := range traces {
		if nothing(t) && !t.At.Before(from) && t.At.Before(to) {
			out[t.Space]++
		}
	}
	return out
}

// Ratings are a month's Helpful and Not helpful ratings.
type Ratings struct {
	Helpful    int `json:"helpful"`
	NotHelpful int `json:"not_helpful"`
}

// StewardFeedback is a space's ratings this month and last, and the documents answers rated
// Not helpful this month cited most, checked as the steward (SB-3104). Never a comment.
type StewardFeedback struct {
	ThisMonth       Ratings  `json:"this_month"`
	LastMonth       Ratings  `json:"last_month"`
	Documents       []Result `json:"documents"`
	DocumentsHidden int      `json:"documents_hidden"`
}

// stewardFeedback joins the months' feedback to their traces by id: only answers asked in
// space count, each rating once (the newest, as the store keeps it).
func (x Stewards) stewardFeedback(ctx context.Context, s tenancy.Scope, space string, traces []Trace, fbs []Feedback, now time.Time) (StewardFeedback, error) {
	out := StewardFeedback{Documents: []Result{}}
	byID := map[string]Trace{}
	for _, t := range traces {
		byID[t.ID] = t
	}
	this, last := MonthStart(now), MonthStart(MonthStart(now).Add(-time.Hour))
	cited := map[string]int{}
	var order []string
	for _, f := range fbs {
		t, ok := byID[f.Trace]
		if !ok || t.Space != space || f.At.Before(last) || f.At.After(now) {
			continue
		}
		r := &out.LastMonth
		if !f.At.Before(this) {
			r = &out.ThisMonth
		}
		switch f.Rating {
		case "up":
			r.Helpful++
		case "down":
			r.NotHelpful++
			if r != &out.ThisMonth {
				continue
			}
			for _, c := range t.Response.Answer.Claims {
				for _, ci := range c.Cites {
					if cited[ci.Cite] == 0 {
						order = append(order, ci.Cite)
					}
					cited[ci.Cite]++
				}
			}
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return cited[order[i]] > cited[order[j]] })
	if len(order) > MaxFeedbackDocs {
		order = order[:MaxFeedbackDocs]
	}
	docs, hidden, err := x.Agent.CheckCites(ctx, s, order)
	if err != nil {
		return out, fmt.Errorf("feedback documents: %w", err)
	}
	out.Documents, out.DocumentsHidden = docs, hidden
	return out, nil
}

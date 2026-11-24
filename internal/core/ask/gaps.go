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

// MaxConflicts is the most pairs of disagreeing documents the review list names.
const MaxConflicts = 10

// Conflict is a pair of documents claims marked Sources disagree cited together (SB-3605).
type Conflict struct {
	A       Result `json:"a"`
	B       Result `json:"b"`
	Answers int    `json:"answers"`
}

// conflicts are the pairs of documents that disagreeing claims of answers asked in space
// this calendar month cited, by how many answers did, checked as the steward: a pair is
// named only when they can open both, and hidden counts the rest. Counts only (LAW-07).
func (x Stewards) conflicts(ctx context.Context, s tenancy.Scope, space string, traces []Trace, now time.Time) ([]Conflict, int, error) {
	type pair struct{ a, b string } // cites, a's document before b's
	count := map[[2]string]int{}
	cites := map[[2]string]pair{}
	var order [][2]string
	for _, t := range traces {
		if t.Space != space || t.At.Before(MonthStart(now)) || t.At.After(now) {
			continue
		}
		seen := map[[2]string]bool{}
		for _, c := range t.Response.Answer.Claims {
			if !c.Disagree {
				continue
			}
			docs := map[string]string{} // document -> its first cite in the claim
			var keys []string
			for _, ci := range c.Cites {
				if k := docKey(ci.Cite); docs[k] == "" {
					docs[k] = ci.Cite
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			for i := range keys {
				for j := i + 1; j < len(keys); j++ {
					k := [2]string{keys[i], keys[j]}
					if seen[k] {
						continue
					}
					seen[k] = true
					if count[k] == 0 {
						order = append(order, k)
						cites[k] = pair{docs[keys[i]], docs[keys[j]]}
					}
					count[k]++
				}
			}
		}
	}
	if len(order) == 0 {
		return nil, 0, nil
	}
	sort.SliceStable(order, func(i, j int) bool { return count[order[i]] > count[order[j]] })
	var all []string
	asked := map[string]bool{}
	for _, k := range order {
		for _, c := range []string{cites[k].a, cites[k].b} {
			if !asked[c] {
				asked[c] = true
				all = append(all, c)
			}
		}
	}
	docs, _, err := x.Agent.CheckCites(ctx, s, all)
	if err != nil {
		return nil, 0, fmt.Errorf("conflicts: %w", err)
	}
	open := map[string]Result{}
	for _, r := range docs {
		open[r.Cite] = r
	}
	var out []Conflict
	hidden := 0
	for _, k := range order {
		a, okA := open[cites[k].a]
		b, okB := open[cites[k].b]
		switch {
		case !okA || !okB:
			hidden++
		case len(out) < MaxConflicts:
			out = append(out, Conflict{A: a, B: b, Answers: count[k]})
		}
	}
	return out, hidden, nil
}

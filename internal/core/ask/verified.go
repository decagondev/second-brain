package ask

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Verified only and mark intervals (sb-ask(3); SB-3602, SB-3603, SB-3604).

// MarkDays are the intervals a review mark may take instead of review_after_days.
var MarkDays = map[int]bool{30: true, 90: true, 180: true, 365: true}

// UnhelpfulAfterMark is how many Not helpful ratings since a mark list the document again.
const UnhelpfulAfterMark = 3

type verifiedKey struct{}

func withVerified(ctx context.Context, on bool) context.Context {
	return context.WithValue(ctx, verifiedKey{}, on)
}

func verifiedOf(ctx context.Context) bool {
	v, _ := ctx.Value(verifiedKey{}).(bool)
	return v
}

// markAfter is how many days mark m lasts: its own interval, else after.
func markAfter(m ReviewMark, after int) int {
	if m.Days > 0 {
		return m.Days
	}
	return after
}

var notePathRe = regexp.MustCompile(`^notes/[^/]+/qs_[^/]+\.md$`)

// isNote reports whether c is a steward's published note (SB-1705).
func isNote(c ports.Chunk) bool {
	return c.Kind == "docs" && notePathRe.MatchString(c.SourceID)
}

// onlyVerified keeps the chunks of notes and of documents with a live mark at their
// version: younger than its interval and not past the document's own review-by date.
func (a Agent) onlyVerified(ctx context.Context, cs []ports.Chunk) []ports.Chunk {
	marks := &marksCache{store: a.Marks, by: map[string]map[string]ReviewMark{}}
	now := a.Now()
	out := cs[:0:0]
	for _, c := range cs {
		if isNote(c) {
			out = append(out, c)
			continue
		}
		m, ok := marks.reviewed(ctx, a.Tenant, c)
		if ok && now.Sub(m.At) <= time.Duration(markAfter(m, a.reviewAfter(c.Spaces)))*24*time.Hour && !pastReviewBy(c, now) {
			out = append(out, c)
		}
	}
	return out
}

// unhelpfulSince counts, by document, the Not helpful ratings after its mark of answers
// citing it at the marked version.
func unhelpfulSince(traces []Trace, fbs []Feedback, marks map[string]ReviewMark) map[string]int {
	cites := map[string]map[string]int{} // trace -> document -> version
	for _, t := range traces {
		for _, c := range t.Response.Answer.Claims {
			for _, ci := range c.Cites {
				k := docKey(ci.Cite)
				if _, ok := marks[k]; !ok {
					continue
				}
				if cites[t.ID] == nil {
					cites[t.ID] = map[string]int{}
				}
				cites[t.ID][k] = versionOf(ci.Cite)
			}
		}
	}
	out := map[string]int{}
	for _, f := range fbs {
		if f.Rating != "down" {
			continue
		}
		for k, v := range cites[f.Trace] {
			if m := marks[k]; v == m.Version && f.At.After(m.At) {
				out[k]++
			}
		}
	}
	return out
}

// versionOf is a cite's version, 0 when it has none.
func versionOf(cite string) int {
	_, v, ok := strings.Cut(cite, "@")
	if !ok {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

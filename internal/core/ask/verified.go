package ask

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
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

var notePathRe = regexp.MustCompile(`^notes/[^/]+/(?:updates/)?qs_[^/]+\.md$`)

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
		if a.checked(ctx, marks, c, now) {
			out = append(out, c)
		}
	}
	return out
}

// checked says whether chunk c is a note or has a live mark at its version: younger than
// its interval, and the document not past its own review-by date.
func (a Agent) checked(ctx context.Context, marks *marksCache, c ports.Chunk, now time.Time) bool {
	if isNote(c) {
		return true
	}
	m, ok := marks.reviewed(ctx, a.Tenant, c)
	return ok && now.Sub(m.At) <= time.Duration(markAfter(m, a.reviewAfter(c.Spaces)))*24*time.Hour && !pastReviewBy(c, now)
}

// VerifiedCount is how many of an answer's cited documents someone has checked (SB-3607).
type VerifiedCount struct {
	Of     int `json:"of"`
	Marked int `json:"marked"`
}

// verifiedCount counts the distinct documents claims cite, and those checked at the cited
// version; nil when no claim has a cite.
func (a Agent) verifiedCount(ctx context.Context, claims []answer.Claim, chunks []ports.Chunk) *VerifiedCount {
	byCite := map[string]ports.Chunk{}
	for _, c := range chunks {
		byCite[c.Cite] = c
	}
	marks := &marksCache{store: a.Marks, by: map[string]map[string]ReviewMark{}}
	now := a.Now()
	seen := map[string]bool{}
	var v VerifiedCount
	for _, cl := range claims {
		for _, ci := range cl.Cites {
			k := docKey(ci.Cite)
			if seen[k] || strings.HasPrefix(ci.Cite, FileKind+":") {
				continue // the asker's own file is no source anyone checks
			}
			seen[k] = true
			v.Of++
			if c, ok := byCite[ci.Cite]; ok && a.checked(ctx, marks, c, now) {
				v.Marked++
			}
		}
	}
	if v.Of == 0 {
		return nil
	}
	return &v
}

// VerifiedLine is the words under an answer for its count (SB-3607).
func VerifiedLine(v VerifiedCount) string {
	return fmt.Sprintf("%d of %d sources verified.", v.Marked, v.Of)
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

// MaxDecisions is the most decisions a search without a query lists (SB-3702).
const MaxDecisions = 50

var decisionHeadRe = regexp.MustCompile(`(?mi)^#{1,3} +decisions? *$`)

var frontDecisionRe = regexp.MustCompile(`(?s)\A(?:[^\n]*\n\n)?---\n(?:[^\n]*\n){0,30}?type:\s*decision\s*\n(?:[^\n]*\n){0,30}?---`)

// isDecision reports whether chunk c is a decision (SB-3702): it is under or holds a heading
// Decision or Decisions, or it opens a document whose front matter says type: decision.
func isDecision(c ports.Chunk) bool { return IsDecision(c) }

// IsDecision is isDecision, for the events out (SB-4404).
func IsDecision(c ports.Chunk) bool {
	if s := strings.ToLower(strings.TrimSpace(c.Section)); s == "decision" || s == "decisions" || decisionHeadRe.MatchString(c.Text) {
		return true
	}
	return c.Start == 0 && frontDecisionRe.MatchString(c.Text)
}

// wantsDecisions reports whether types asks for decisions.
func wantsDecisions(types []string) bool {
	for _, t := range types {
		if t == "decision" {
			return true
		}
	}
	return false
}

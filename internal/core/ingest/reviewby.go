package ingest

import (
	"regexp"
	"strings"
	"time"
)

// Review-by dates (sb-ingest(7), Review-by dates; SB-2705): a document can say when it
// should be reviewed or when it expires. The date is data: it can only make a cite stale
// sooner than its age would, never fresher (LAW-09).

// maxFrontMatter is how far into a document its front matter may run.
const maxFrontMatter = 4096

// FrontMatterDate is the review-by or expires value of a Markdown document's front matter,
// as written, or "" when it has none. review-by wins over expires.
func FrontMatterDate(text string) string {
	fm := frontMatter(text)
	if v, ok := fm["review-by"]; ok {
		return v
	}
	if v, ok := fm["review_by"]; ok {
		return v
	}
	return fm["expires"]
}

// frontMatter is a Markdown document's front matter as key: value, keys lower case, the
// first of each key; nil for none.
func frontMatter(text string) map[string]string {
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return nil
	}
	head := text[strings.Index(text, "\n")+1:]
	if len(head) > maxFrontMatter {
		head = head[:maxFrontMatter]
	}
	end := strings.Index(head, "\n---")
	if end < 0 {
		return nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(head[:end], "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		if _, had := out[k]; !had {
			out[k] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return out
}

// Mark is what a document's front matter says an agent wrote (SB-4706, LAW-28).
type Mark struct {
	GeneratedBy, ApprovedBy string
	Edited                  bool
}

// markShape is AGENT@VERSION, or AGENT alone.
var markShape = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}(@[A-Za-z0-9._-]{1,80})?$`)

// MarkOf reads a document's generated_by, approved_by and edited. The mark is data and only
// ever adds that an agent wrote it: approved_by is kept only when it is the document's own
// owner at the source, and a generated_by of any other shape is dropped.
func MarkOf(text, owner string) Mark {
	fm := frontMatter(text)
	gb := fm["generated_by"]
	if !markShape.MatchString(gb) {
		return Mark{}
	}
	m := Mark{GeneratedBy: gb, Edited: fm["edited"] == "true"}
	if ab := strings.ToLower(fm["approved_by"]); ab != "" && strings.EqualFold(ab, strings.TrimSpace(owner)) {
		m.ApprovedBy = ab
	}
	return m
}

// ParseReviewBy reads a date written as YYYY-MM-DD or RFC 3339, in UTC; ok is false for a
// value that is neither.
func ParseReviewBy(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

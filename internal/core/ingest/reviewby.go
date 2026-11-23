package ingest

import (
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
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return ""
	}
	head := text[strings.Index(text, "\n")+1:]
	if len(head) > maxFrontMatter {
		head = head[:maxFrontMatter]
	}
	end := strings.Index(head, "\n---")
	if end < 0 {
		return ""
	}
	var expires string
	for _, line := range strings.Split(head[:end], "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "review-by", "review_by":
			return v
		case "expires":
			expires = v
		}
	}
	return expires
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

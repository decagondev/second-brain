package ingest

import "strings"

// Labels, least to most sensitive (sb-spaces(7), Labels): route.Ladder's, kept here so the
// core reads them without the router.
var Labels = []string{"public", "internal", "confidential", "restricted"}

// LabelRestricted is LAW-20's label.
const LabelRestricted = "restricted"

// Rank is a label's place on the ladder, -1 for "" or one that isn't on it.
func Rank(label string) int {
	for i, l := range Labels {
		if l == label {
			return i
		}
	}
	return -1
}

// Sensitivity is a Markdown document's front matter sensitivity (SB-5405), lower case: one
// of Labels, restricted for a value that isn't one (unknown never widens), "" for none.
func Sensitivity(text string) string {
	v, ok := frontMatter(text)["sensitivity"]
	if !ok {
		return ""
	}
	v = strings.ToLower(strings.TrimSpace(v))
	if Rank(v) < 0 {
		return LabelRestricted
	}
	return v
}

// Highest is the most sensitive of labels; "" when none is on the ladder.
func Highest(labels ...string) string {
	best := ""
	for _, l := range labels {
		if Rank(l) > Rank(best) {
			best = l
		}
	}
	return best
}

// PersonOnly says content with this label goes only to a person, never to a chat channel
// (LAW-32): confidential and above.
func PersonOnly(label string) bool { return Rank(label) >= Rank("confidential") }

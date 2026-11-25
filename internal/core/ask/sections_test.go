package ask_test

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

// SB-4902: a draft splits into headings, list items and paragraphs, and joins back exactly.
func TestDraftSections(t *testing.T) {
	text := "# Week\nIntro line\nmore\n\n- one [A]\n- two [B]\n  wrapped\n\nLast paragraph.\n"
	got := ask.DraftSections(text)
	want := []string{"# Week\nIntro line\nmore\n\n", "- one [A]\n", "- two [B]\n  wrapped\n\n", "Last paragraph.\n"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("sections %q", got)
	}
	for _, s := range []string{"", "x", "a\n\n\nb", "- a\n- b", "\n\n# h\n"} {
		if j := strings.Join(ask.DraftSections(s), ""); j != s {
			t.Errorf("%q joins to %q", s, j)
		}
	}
}

func FuzzDraftSections(f *testing.F) {
	f.Add("# a\n- b\n\nc\n")
	f.Fuzz(func(t *testing.T, s string) {
		if j := strings.Join(ask.DraftSections(s), ""); j != s {
			t.Fatalf("%q joins to %q", s, j)
		}
	})
}

package ingest

import (
	"strings"
	"testing"
)

// SB-1701: the indexed title is the sanitised text's first line, bounded.
func TestTitle(t *testing.T) {
	if got := Title("Refund policy", "Refund policy\n\nRefunds over 500 need approval."); got != "Refund policy" {
		t.Errorf("got %q", got)
	}
	if got := Title("", "No title here\n\ntext"); got != "" {
		t.Errorf("a document without a title got %q", got)
	}
	if got := Title("x", strings.Repeat("é", MaxTitle+50)); len([]rune(got)) != MaxTitle {
		t.Errorf("title of %d characters", len([]rune(got)))
	}
}

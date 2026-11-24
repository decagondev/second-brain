package ingest_test

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ingest"
)

// SB-3002: a chunk's section is the nearest heading of level 1 to 3 at or above it.
func TestSection(t *testing.T) {
	text := ingest.Normalise("Intro text.\n\n# Leave\n\nBasics.\n\n## Carry-over\n\nUp to five days.\n\n#### Too deep\n\nStill carry-over.\n\n#no-space heading")
	at := func(s string) int {
		i := strings.Index(text, s)
		if i < 0 {
			t.Fatalf("no %q", s)
		}
		return len([]rune(text[:i]))
	}
	for _, c := range []struct{ at, want string }{
		{"Intro", ""}, {"# Leave", "Leave"}, {"Basics", "Leave"}, {"Up to", "Carry-over"}, {"Still", "Carry-over"}, {"#no-space", "Carry-over"},
	} {
		if got := ingest.Section(text, at(c.at)); got != c.want {
			t.Errorf("at %q: %q, want %q", c.at, got, c.want)
		}
	}
	long := "# " + strings.Repeat("x", 200)
	if got := ingest.Section(long, 0); len([]rune(got)) != ingest.MaxSection {
		t.Fatalf("%d", len([]rune(got)))
	}
}

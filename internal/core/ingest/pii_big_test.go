package ingest

import (
	"strings"
	"testing"
	"time"
)

// A long run of short digit groups stays linear (a card has at most 19 groups).
func TestPIIScanIsBounded(t *testing.T) {
	text := strings.Repeat("1 ", 200000)
	start := time.Now()
	_ = PIISpans(text)
	if d := time.Since(start); d > 20*time.Second {
		t.Fatalf("took %v", d)
	}
}

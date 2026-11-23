package mmapp

import (
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
)

// SB-2701, SB-2704: the stale and sync warnings come before the answer, escaped.
func TestWarningsComeFirst(t *testing.T) {
	at := time.Date(2027, 3, 10, 14, 5, 0, 0, time.UTC)
	r := ask.Response{Trace: "tr_1", NoticeStale: &ask.StaleNotice{Stale: 2, Of: 3, OldestDays: 400},
		NoticeSync: []ask.SyncNotice{{Kind: "docs", LastSuccess: &at}},
		Answer:     ask.Answer{Claims: []answer.Claim{{Text: "The cutoff is 3pm.", Cites: []answer.Cite{{Cite: "docs:a@1"}}}}}}
	text := Render("cutoff?", r).Text
	stale, sync, claim := strings.Index(text, "the oldest is 400 days old"), strings.Index(text, `docs last synced at 14\:05 UTC`), strings.Index(text, "The cutoff")
	if stale < 0 || sync < 0 || stale >= claim || sync >= claim {
		t.Fatalf("%q", text)
	}
}

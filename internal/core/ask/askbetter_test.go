package ask

import (
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
)

// SB-3301: See also is the allowed chunks' documents no claim cites, at most three, each once.
func TestSeeAlso(t *testing.T) {
	now := time.Date(2027, 3, 10, 9, 0, 0, 0, time.UTC)
	chunks := []ports.Chunk{
		{Cite: "docs:a@1", Title: "A", Start: 0, End: 10, Seen: now.AddDate(0, 0, -4)},
		{Cite: "docs:b@2", Title: "B", Start: 0, End: 10},
		{Cite: "docs:b@2", Title: "B", Start: 10, End: 20},
		{Cite: "docs:c@1", Title: "C"}, {Cite: "docs:d@1", Title: "D"}, {Cite: "docs:e@1", Title: "E"},
	}
	claims := []answer.Claim{{Text: "x", Cites: []answer.Cite{{Cite: "docs:a@1"}}}}
	got := seeAlso(claims, chunks, now)
	var names []string
	for _, s := range got {
		names = append(names, s.Cite)
	}
	if strings.Join(names, ",") != "docs:b@2,docs:c@1,docs:d@1" {
		t.Fatalf("see also %v", names)
	}
	if got[0].Span != [2]int{0, 10} || got[0].AgeDays != nil {
		t.Fatalf("first %+v", got[0])
	}
	if r := (Response{SeeAlso: got}); func() bool { _, ok := spanOf(r, "docs:c@1", -1); return !ok }() {
		t.Fatal("a See also document's passage opens like a cite's")
	}
	if len(seeAlso(claims, chunks[:1], now)) != 0 {
		t.Fatal("nothing left over is no See also")
	}
}

// SB-3303: the short line goes only on a short ask.
func TestShortLine(t *testing.T) {
	if shortLine(withShort(t.Context(), true)) != ShortLine || shortLine(withShort(t.Context(), false)) != "" || shortLine(t.Context()) != "" {
		t.Fatal("short line")
	}
}

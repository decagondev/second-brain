package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// SB-4406: 12 weeks of counts, 1 to 4 shown as "fewer than 5", and a source whose flags
// doubled named.
func TestGuardrails(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	last, before := now.AddDate(0, 0, -7), now.AddDate(0, 0, -14)
	var cs []ports.Chunk
	add := func(kind string, n int, at time.Time) {
		for i := 0; i < n; i++ {
			cs = append(cs, ports.Chunk{ID: kind + at.Format("0102") + string(rune('a'+i)), Kind: kind, Seen: at, Flags: []string{ingest.FlagInstruction}, Text: "SECRET TEXT"})
		}
	}
	add("docs", 6, last)
	add("docs", 3, before)
	add("chat", 5, last)
	add("chat", 4, before)
	cs = append(cs, ports.Chunk{ID: "clean", Kind: "docs", Seen: now})
	traces := []ask.Trace{{At: now, FlaggedFiles: 1}, {At: now, Response: ask.Response{Dropped: 2}}, {At: now.AddDate(0, 0, -200), FlaggedFiles: 1}}
	rows := []audit.Row{
		{At: now.Format(time.RFC3339Nano), Kind: "approval", State: "Rejected"},
		{At: now.Format(time.RFC3339Nano), Kind: "decision", State: "deny", Reason: "call_tool: no permit"},
		{At: now.Format(time.RFC3339Nano), Kind: "decision", State: "deny", Reason: "ask: no permit"},
	}
	g := BuildGuardrails(now, map[string][]ports.Chunk{"ops": cs}, traces, rows)
	if len(g.Weeks) != GuardrailWeeks || g.Weeks[0].Week != "2026-W48" || g.Weeks[1].Week != "2026-W47" {
		t.Fatalf("weeks: %+v", g.Weeks)
	}
	if strings.Join(g.Doubled, ",") != "docs" {
		t.Fatalf("doubled: %v", g.Doubled)
	}
	b, _ := json.Marshal(g)
	s := string(b)
	for _, want := range []string{`"flags":[{"source":"chat","space":"ops","count":5},{"source":"docs","space":"ops","count":6}]`,
		`"week":"2026-W48","flags":[],"flagged_uploads":"fewer than 5","refused_tools":"fewer than 5","denied_tools":"fewer than 5","dropped_uncited":"fewer than 5"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("no %s in %s", want, s)
		}
	}
	if strings.Contains(s, "SECRET") {
		t.Fatal("chunk text in the counts")
	}
}

package ask

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// SB-5706: shares add up from traces by space and agent; a space whose prompt carries a
// timestamp, so its prefix changes on every ask, is named with the part that changed; the
// report holds no question, passage or prompt text.
func TestCacheHitsFromTraces(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	at := from.Add(48 * time.Hour)
	stable := map[string]string{PrefixPrompt: prefixHash("You answer."), PrefixLanguage: prefixHash("")}
	var traces []Trace
	for i := 0; i < 3; i++ { // ops: the same prefix, mostly cached
		traces = append(traces, Trace{Space: "spc_ops", At: at.Add(time.Duration(i) * time.Hour), InputTokens: 1000, CachedTokens: 800, Prefix: stable,
			Kept: []string{"QUESTION-TEXT"}})
	}
	for i := 0; i < 3; i++ { // finance: a timestamp in the prompt, never cached
		p := map[string]string{PrefixPrompt: prefixHash("You answer. Today is " + at.Add(time.Duration(i)*time.Minute).String()), PrefixLanguage: prefixHash("")}
		traces = append(traces, Trace{Space: "spc_finance", At: at.Add(time.Duration(i) * time.Minute), InputTokens: 1000, Prefix: p,
			Response: Response{Agent: "finance"}})
	}
	traces = append(traces, Trace{Space: "spc_ops", At: to.Add(time.Hour), InputTokens: 5000}) // next month
	c := BuildCacheHits(traces, "2026-11", from, to)
	if c.Total.Input != 6000 || c.Total.Cached != 2400 || c.Total.Asks != 6 {
		t.Fatalf("total %+v", c.Total)
	}
	var fin, ops CacheRow
	for _, r := range c.Spaces {
		switch r.Name {
		case "finance":
			fin = r
		case "ops":
			ops = r
		}
	}
	if ops.Share != 0.8 || ops.Low || !fin.Low || fin.Share != 0 || strings.Join(fin.Changed, ",") != PrefixPrompt {
		t.Fatalf("ops %+v finance %+v", ops, fin)
	}
	lines := strings.Join(c.Lines(), "\n")
	if !strings.Contains(lines, "The finance space read 0% of its input from the prompt cache over 3 asks; its prompt changed between asks") {
		t.Fatalf("lines %q", lines)
	}
	raw, _ := json.Marshal(c)
	if strings.Contains(string(raw), "QUESTION-TEXT") || strings.Contains(string(raw), "You answer") || strings.Contains(string(raw), "Today is") {
		t.Fatalf("text in the report: %s", raw)
	}
}

// SB-5706: an ask records a hash of each prefix part, never the text.
func TestPrefixIsHashed(t *testing.T) {
	ctx, r := withPrefix(context.Background())
	notePrefix(ctx, map[string]string{PrefixPrompt: "You answer.", PrefixSkill: ""})
	notePrefix(ctx, map[string]string{PrefixPrompt: "a retry"})
	got := r.get()
	if got[PrefixPrompt] != prefixHash("You answer.") || len(got[PrefixPrompt]) != 8 || strings.Contains(got[PrefixPrompt], "answer") {
		t.Fatalf("prefix %v", got)
	}
}

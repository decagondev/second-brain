package report

import (
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// SB-5103: guardrail rows, refused and denied tool calls become events, oldest first, since
// a time, with ids and counts only; CSV has the same columns.
func TestGuardrailEvents(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339Nano) }
	rows := []audit.Row{
		{At: at(-30 * 24 * time.Hour), Kind: "guardrail", Subject: "suspect:docs:old@1", State: "skipped", Space: "spc_ops"},
		{At: at(-2 * time.Hour), Kind: "guardrail", Subject: "suspect:docs:a@3", State: "skipped", Reason: "agent=ask", Space: "spc_finance"},
		{At: at(-time.Hour), Kind: "guardrail", Subject: "masked:api", State: "masked", Reason: "strings=3"},
		{At: at(-50 * time.Minute), Kind: "approval", State: "Rejected", Subject: "apr_1"},
		{At: at(-40 * time.Minute), Kind: "decision", State: "deny", Reason: "call_tool ticket.update"},
		{At: at(-30 * time.Minute), Kind: "decision", State: "allow", Reason: "call_tool ticket.update"},
		{At: at(-20 * time.Minute), Kind: "config", Subject: "manifest"},
	}
	es := GuardrailEvents(rows, now.Add(-7*24*time.Hour))
	if len(es) != 4 {
		t.Fatalf("events %+v", es)
	}
	if e := es[0]; e.Kind != "suspect" || e.Document != "docs:a@3" || e.Agent != "ask" || e.Space != "finance" || e.Count != 1 {
		t.Fatalf("suspect %+v", e)
	}
	if e := es[1]; e.Kind != "masked" || e.Count != 3 || e.Space != "" {
		t.Fatalf("masked %+v", e)
	}
	if es[2].Kind != "refused" || es[3].Kind != "denied" {
		t.Fatalf("tools %+v", es[2:])
	}
	csv := string(GuardrailCSV(es))
	if !strings.HasPrefix(csv, "at,kind,space,agent,document,count\n") || strings.Count(csv, "\n") != 5 {
		t.Fatalf("csv %q", csv)
	}
}

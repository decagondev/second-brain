package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// Guardrail insights (sb-observability(7), Guardrail insights; SB-4406): counts only.

// GuardrailWeeks is how many weeks the insights show.
const GuardrailWeeks = 12

// Count is a count shown as "fewer than 5" from 1 to 4.
type Count int

// MarshalJSON implements json.Marshaler.
func (c Count) MarshalJSON() ([]byte, error) {
	if c > 0 && c < 5 {
		return []byte(`"fewer than 5"`), nil
	}
	return json.Marshal(int(c))
}

// FlagCount is one source and space's flagged chunks in a week.
type FlagCount struct {
	Source string `json:"source"`
	Space  string `json:"space"`
	Count  Count  `json:"count"`
}

// GuardrailWeek is one week's counts.
type GuardrailWeek struct {
	Week           string      `json:"week"`
	Flags          []FlagCount `json:"flags"`
	FlaggedUploads Count       `json:"flagged_uploads"`
	RefusedTools   Count       `json:"refused_tools"`
	DeniedTools    Count       `json:"denied_tools"`
	DroppedUncited Count       `json:"dropped_uncited"`
}

// Guardrails are the insights now.
type Guardrails struct {
	Weeks   []GuardrailWeek `json:"weeks"`
	Doubled []string        `json:"doubled"`
}

// isoWeek is t's ISO week, as 2027-W10.
func isoWeek(t time.Time) string {
	y, w := t.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func flagged(flags []string) bool {
	for _, f := range flags {
		if f == ingest.FlagInstruction || f == ingest.FlagHidden {
			return true
		}
	}
	return false
}

// BuildGuardrails counts, for the 12 weeks to now: chunks by space name, traces and audit rows.
func BuildGuardrails(now time.Time, chunks map[string][]ports.Chunk, traces []ask.Trace, rows []audit.Row) Guardrails {
	weeks := map[string]*GuardrailWeek{}
	var order []string
	for i := 0; i < GuardrailWeeks; i++ {
		w := isoWeek(now.AddDate(0, 0, -7*i))
		if weeks[w] == nil {
			weeks[w] = &GuardrailWeek{Week: w, Flags: []FlagCount{}}
			order = append(order, w)
		}
	}
	type fk struct{ week, source, space string }
	flags := map[fk]int{}
	bySource := map[string]map[string]int{} // source → week → count
	for space, cs := range chunks {
		seen := map[string]bool{}
		for _, c := range cs {
			w := isoWeek(c.Seen)
			if c.Seen.IsZero() || weeks[w] == nil || !flagged(c.Flags) || seen[c.ID] {
				continue
			}
			seen[c.ID] = true
			flags[fk{w, c.Kind, space}]++
			if bySource[c.Kind] == nil {
				bySource[c.Kind] = map[string]int{}
			}
			bySource[c.Kind][w]++
		}
	}
	for k, n := range flags {
		weeks[k.week].Flags = append(weeks[k.week].Flags, FlagCount{Source: k.source, Space: k.space, Count: Count(n)})
	}
	for _, t := range traces {
		g := weeks[isoWeek(t.At)]
		if g == nil {
			continue
		}
		if t.FlaggedFiles > 0 {
			g.FlaggedUploads++
		}
		if t.Response.Dropped > 0 {
			g.DroppedUncited++
		}
	}
	for _, r := range rows {
		at, err := time.Parse(time.RFC3339Nano, r.At)
		g := weeks[isoWeek(at)]
		if err != nil || g == nil {
			continue
		}
		switch {
		case r.Kind == "approval" && r.State == "Rejected":
			g.RefusedTools++
		case r.Kind == "decision" && r.State != "allow" && (strings.HasPrefix(r.Reason, "call_tool") || strings.HasPrefix(r.Reason, "propose")):
			g.DeniedTools++
		}
	}
	out := Guardrails{Doubled: Doubled(bySource, now)}
	for _, w := range order {
		g := weeks[w]
		sort.Slice(g.Flags, func(i, j int) bool {
			if g.Flags[i].Source != g.Flags[j].Source {
				return g.Flags[i].Source < g.Flags[j].Source
			}
			return g.Flags[i].Space < g.Flags[j].Space
		})
		out.Weeks = append(out.Weeks, *g)
	}
	return out
}

// Doubled are the sources whose flags in the last full week are at least 5 and at least
// twice the week before's.
func Doubled(bySource map[string]map[string]int, now time.Time) []string {
	last, before := isoWeek(now.AddDate(0, 0, -7)), isoWeek(now.AddDate(0, 0, -14))
	out := []string{}
	for src, ws := range bySource {
		if ws[last] >= 5 && ws[last] >= 2*ws[before] {
			out = append(out, src)
		}
	}
	sort.Strings(out)
	return out
}

// Guardrail events (sb-observability(7), Guardrail events; SB-5103): ids and counts only.

// MaxGuardrailEvents is the most events one export lists.
const MaxGuardrailEvents = 10000

// GuardrailEvent is one event: a suspect passage, a masked output, a tool call an approver
// refused or the policy kernel denied.
type GuardrailEvent struct {
	At       string `json:"at"`
	Kind     string `json:"kind"`
	Space    string `json:"space,omitempty"`
	Agent    string `json:"agent,omitempty"`
	Document string `json:"document,omitempty"`
	Count    int    `json:"count"`
}

// reasonValue is key's value in a row's reason, "k=v k2=v2" or "k=v".
func reasonValue(reason, key string) string {
	for _, f := range strings.FieldsFunc(reason, func(r rune) bool { return r == ' ' || r == ',' }) {
		if k, v, ok := strings.Cut(f, "="); ok && k == key {
			return v
		}
	}
	return ""
}

// GuardrailEvents are the rows at or after since as events, oldest first, at most
// MaxGuardrailEvents.
func GuardrailEvents(rows []audit.Row, since time.Time) []GuardrailEvent {
	out := []GuardrailEvent{}
	for _, r := range rows {
		at, err := time.Parse(time.RFC3339Nano, r.At)
		if err != nil || at.Before(since) {
			continue
		}
		e := GuardrailEvent{At: r.At, Space: strings.TrimPrefix(r.Space, "spc_"), Count: 1}
		switch {
		case r.Kind == "guardrail" && strings.HasPrefix(r.Subject, "suspect:"):
			e.Kind, e.Document, e.Agent = "suspect", strings.TrimPrefix(r.Subject, "suspect:"), reasonValue(r.Reason, "agent")
		case r.Kind == "guardrail" && strings.HasPrefix(r.Subject, "masked:"):
			e.Kind = "masked"
			if n, err := strconv.Atoi(reasonValue(r.Reason, "strings")); err == nil && n > 0 {
				e.Count = n
			}
		case r.Kind == "approval" && r.State == "Rejected":
			e.Kind = "refused"
		case r.Kind == "decision" && r.State != "allow" && (strings.HasPrefix(r.Reason, "call_tool") || strings.HasPrefix(r.Reason, "propose")):
			e.Kind = "denied"
		default:
			continue
		}
		if e.Space == "company" {
			e.Space = ""
		}
		out = append(out, e)
		if len(out) == MaxGuardrailEvents {
			break
		}
	}
	return out
}

// GuardrailCSV is events as CSV, with a header row.
func GuardrailCSV(es []GuardrailEvent) []byte {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"at", "kind", "space", "agent", "document", "count"})
	for _, e := range es {
		_ = w.Write([]string{e.At, e.Kind, e.Space, e.Agent, e.Document, strconv.Itoa(e.Count)})
	}
	w.Flush()
	return []byte(b.String())
}

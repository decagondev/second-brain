package ask

import (
	"testing"
	"time"
)

// SB-3506: one row per space; a space with fewer than 5 asks shows only that, the rows' asks
// add up to the asks, and searches and refusals aren't asks.
func TestUsageBySpace(t *testing.T) {
	none := NoticeNoSources
	var traces []Trace
	for i := 0; i < 6; i++ {
		tr := Trace{ID: string(rune('a' + i)), Space: "spc_finance"}
		if i < 2 {
			tr.Response.Notice = &none
		}
		traces = append(traces, tr)
	}
	traces = append(traces, Trace{ID: "x", Space: "spc_hr"}, Trace{ID: "y"}, Trace{ID: "s", Space: "spc_hr", Response: Response{Agent: AgentSearch}},
		Trace{ID: "l", Space: "spc_hr", Response: Response{Agent: AgentLimited}})
	now := time.Now()
	helpful := Helpful([]Feedback{{Trace: "c", Rating: "up", At: now}, {Trace: "d", Rating: "up", At: now.Add(time.Minute)}, {Trace: "d", Rating: "down", At: now}, {Trace: "e", Rating: "up", At: now}, {Trace: "e", Rating: "down", At: now.Add(time.Minute)}})
	rows := UsageBySpace(traces, helpful)
	if len(rows) != 3 || rows[0].Space != "company" || rows[1].Space != "finance" || rows[2].Space != "hr" {
		t.Fatalf("rows %+v", rows)
	}
	f := rows[1]
	if f.FewerThan5 || *f.Asks != 6 || *f.FoundNothing != 0.333 || *f.Helpful != 0.333 {
		t.Fatalf("finance %+v %v %v", f, *f.FoundNothing, *f.Helpful)
	}
	for _, r := range []SpaceUsage{rows[0], rows[2]} {
		if !r.FewerThan5 || r.Asks != nil || r.FoundNothing != nil || r.Helpful != nil {
			t.Fatalf("a small space shows more than that: %+v", r)
		}
	}
	if u := ownUsage(traces, nil, "hr", now); u.Space != "hr" || !u.FewerThan5 {
		t.Fatalf("own usage %+v", u)
	}
	if u := ownUsage(traces, nil, "nowhere", now); u.Space != "nowhere" || !u.FewerThan5 {
		t.Fatalf("an empty space %+v", u)
	}
}

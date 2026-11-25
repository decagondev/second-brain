package ask

import (
	"testing"
	"time"
)

func draftQ(agent, space string, at time.Time, text string) StewardQuestion {
	return StewardQuestion{ID: "qs", Space: "spc_" + space, At: at, Thread: &SavedThread{Agent: agent, Kind: DraftBrief, Text: text}}
}

// SB-4701: drafts, publishes as is or edited, discards, rounds, hours, helpful rate and
// cost, per agent and space; counts only.
func TestAgentValueCounts(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	from := now.AddDate(0, 0, -84)
	as := draftQ("case-brief", "support", now.Add(-10*time.Hour), "- A [Doc]")
	as.Reply = &StewardReply{Text: " - A [Doc] ", At: now.Add(-8 * time.Hour)}
	edited := draftQ("case-brief", "support", now.Add(-10*time.Hour), "- A [Doc]")
	edited.Reply = &StewardReply{Text: "- A, said better [Doc]", At: now.Add(-6 * time.Hour)}
	edited.Thread.Rounds = 2
	gone := draftQ("case-brief", "support", now.Add(-5*time.Hour), "")
	gone.Thread.Discarded, gone.Thread.Rounds = true, 1
	open := draftQ("case-brief", "support", now.Add(-time.Hour), "- B [Doc]")
	old := draftQ("case-brief", "support", from.Add(-time.Hour), "- C [Doc]")
	other := draftQ("voc", "product", now.Add(-time.Hour), "- D [Doc]")
	chat := StewardQuestion{Space: "spc_support", At: now, Thread: &SavedThread{Text: "a saved chat thread"}}
	plain := StewardQuestion{Space: "spc_support", At: now}
	traces := []Trace{
		{ID: "tr_1", Space: "spc_support", At: now, CostUSD: 0.5, Response: Response{Agent: "case-brief"}},
		{ID: "tr_2", Space: "spc_support", At: now, CostUSD: 0.25, Response: Response{Agent: "case-brief"}},
		{ID: "tr_3", Space: "spc_support", At: now, CostUSD: 9, Response: Response{Agent: AgentSearch}},
		{ID: "tr_4", Space: "spc_support", At: now, CostUSD: 0.1},
	}
	fb := []Feedback{{Trace: "tr_1", Rating: "up", At: now}, {Trace: "tr_2", Rating: "down", At: now}, {Trace: "tr_4", Rating: "up", At: now}}
	rows := AgentValue([]StewardQuestion{as, edited, gone, open, old, other, chat, plain}, traces, fb, from, nil)
	if len(rows) != 3 || rows[0].Agent != "ask" || rows[1].Agent != "case-brief" || rows[2].Agent != "voc" {
		t.Fatalf("rows: %+v", rows)
	}
	r := rows[1]
	if r.Space != "support" || r.Drafts != 4 || r.Published != 1 || r.PublishedEdited != 1 || r.Discarded != 1 || r.Rounds != 3 {
		t.Fatalf("counts: %+v", r)
	}
	if r.MedianHours == nil || *r.MedianHours != 3 || r.Rated != 2 || r.HelpfulRate == nil || *r.HelpfulRate != 0.5 {
		t.Fatalf("hours and helpful: %+v", r)
	}
	if r.CostUSD != 0.75 || r.CostPerPublish == nil || *r.CostPerPublish != 0.375 {
		t.Fatalf("cost: %+v", r)
	}
	if rows := AgentValue([]StewardQuestion{as, other}, traces, fb, from, []string{"voc"}); len(rows) != 1 || rows[0].Agent != "voc" || rows[0].MedianHours != nil || rows[0].CostPerPublish != nil {
		t.Fatalf("only voc: %+v", rows)
	}
	if rows := AgentValue(nil, nil, nil, from, []string{}); len(rows) != 0 {
		t.Fatalf("an owner of nothing: %+v", rows)
	}
}

// A doubled discard share, with at least 4 drafts, is told; less is not.
func TestDoubledDiscards(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC) // a Wednesday
	var qs []StewardQuestion
	add := func(agent string, daysAgo int, n, discarded int) {
		for i := 0; i < n; i++ {
			q := draftQ(agent, "support", now.AddDate(0, 0, -daysAgo), "x")
			q.Thread.Discarded = i < discarded
			qs = append(qs, q)
		}
	}
	add("case-brief", 7, 4, 2)  // last week: 2 of 4
	add("case-brief", 14, 4, 1) // the week before: 1 of 4
	add("voc", 7, 3, 3)         // only 3 drafts
	add("voc", 14, 3, 1)
	add("sop-steps", 7, 5, 2) // 2 of 5 against 1 of 3: not doubled
	add("sop-steps", 14, 3, 1)
	got := DoubledDiscards(qs, now, time.UTC)
	if len(got) != 1 || got[0].Agent != "case-brief" || got[0].Discarded != 2 || got[0].Drafts != 4 || got[0].Week != "2026-W47" {
		t.Fatalf("doubled: %+v", got)
	}
}

// SB-4703: the month's spend by agent, against its cap.
func TestMonthSpend(t *testing.T) {
	at := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	traces := []Trace{
		{At: at, CostUSD: 3, Response: Response{Agent: "case-brief"}},
		{At: at, CostUSD: 2, Response: Response{Agent: "case-brief"}},
		{At: at.AddDate(0, -1, 0), CostUSD: 50, Response: Response{Agent: "case-brief"}},
		{At: at, CostUSD: 1},
	}
	got := MonthSpend(traces, "2026-11", map[string]float64{"case-brief": 5, "voc": 10}, nil)
	if len(got) != 3 || got[0].Agent != "ask" || got[0].MonthlyUSD != nil || got[1].SpentUSD != 5 || !got[1].Capped || got[2].Agent != "voc" || got[2].Capped || got[2].SpentUSD != 0 {
		t.Fatalf("spend: %+v", got)
	}
}

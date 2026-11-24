package notify

import (
	"context"
	"testing"
	"time"
)

// SB-4006: a welcome brief goes once to its person, off-able; a transcript's decisions are
// drafted once.
func TestTriggers(t *testing.T) {
	now := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	j, s, c := world(now)
	j.Questions = questions{}
	c.weekly["weekly"] = Monday(now, nil)
	waiting := []Welcome{{ID: "w1", User: "usr_ann", Space: "finance", Text: "Welcome to finance. Start here: Close calendar"},
		{ID: "w2", User: "usr_bob", Space: "finance", Text: ""}, {ID: "w3", User: "usr_cat", Space: "finance", Text: "Welcome to finance. Start here: X"}}
	done := map[string]bool{}
	j.Welcomes = func(context.Context) ([]Welcome, error) {
		var out []Welcome
		for _, w := range waiting {
			if !done[w.ID] {
				out = append(out, w)
			}
		}
		return out, nil
	}
	j.Welcomed = func(_ context.Context, w Welcome) error { done[w.ID] = true; return nil }
	j.Prefs = prefs{"usr_cat": Prefs{Off: []string{KindWelcome}}}
	drafted := map[string]int{}
	j.Transcripts = func(context.Context, time.Time, time.Time) ([]NewDoc, error) {
		return []NewDoc{{Space: "spc_eng", Ref: "docs:standup.vtt", Title: "standup.vtt"}, {Space: "spc_ops", Ref: "docs:standup.vtt"}}, nil
	}
	j.RunDecisions = func(_ context.Context, d NewDoc) (string, error) { drafted[d.Space+" "+d.Ref]++; return "", nil }
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.got) != 1 || s.got[0].to.ID != "usr_ann" || s.got[0].msg.Kind != KindWelcome || s.got[0].msg.Link != "https://brain.acme.example/?tab=answer" {
		t.Fatalf("sent %+v", s.got)
	}
	if !done["w1"] || !done["w2"] || !done["w3"] {
		t.Fatalf("done %v", done)
	}
	if drafted["spc_eng docs:standup.vtt"] != 1 || drafted["spc_ops docs:standup.vtt"] != 1 {
		t.Fatalf("drafted %v", drafted)
	}
	s.got = nil
	j.Now = func() time.Time { return now.Add(time.Hour) }
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 0 || drafted["spc_eng docs:standup.vtt"] != 1 {
		t.Fatalf("again: sent %v drafted %v %v", s.got, drafted, err)
	}
}

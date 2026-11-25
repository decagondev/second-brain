package ask

import (
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
)

// SB-4504: a wait is overdue after the end of the working day after its date, and goes to
// the lowest space above both sides.
func TestWaitOverdueAndCommonParent(t *testing.T) {
	fri := "2026-11-20" // a Friday: due by the end of Monday
	if waitOverdue(fri, time.Date(2026, 11, 23, 22, 0, 0, 0, time.UTC), nil) {
		t.Fatal("overdue on Monday")
	}
	if !waitOverdue(fri, time.Date(2026, 11, 24, 0, 30, 0, 0, time.UTC), nil) {
		t.Fatal("not overdue on Tuesday")
	}
	parents := map[string]string{"ap": "finance", "ar": "finance", "finance": "", "legal": ""}
	x := Stewards{Parent: func(n string) string { return parents[n] }}
	if p := x.commonParent("ap", "ar"); p != "finance" {
		t.Fatalf("ap and ar: %q", p)
	}
	if p := x.commonParent("ap", "legal"); p != "" {
		t.Fatalf("ap and legal: %q", p)
	}
	if p := x.commonParent("ap", "finance"); p != "" {
		t.Fatalf("a team and its own department: %q", p) // nothing above both
	}
	if w := goalWait(map[string]string{"waiting_on": "Legal", "waiting_reason": "the contract\nline two", "waiting_until": "not a date"}); w == nil || w.On != "legal" || w.Until != "" || strings.Contains(w.Reason, "two") {
		t.Fatalf("goal wait: %+v", w)
	}
}

// SB-4505: supersedes names a decision by KIND:ID or title; a trail reads oldest first, each
// after the one it replaces.
func TestSupersedeLinksAndOrder(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	ch := func(id, title, front string, seen time.Time) ports.Chunk {
		return ports.Chunk{ID: id, Kind: "docs", SourceID: id, Title: title, Cite: "docs:" + id + "@1", Seen: seen,
			Text: "---\ntype: decision\n" + front + "---\nDecision."}
	}
	first := map[string]ports.Chunk{
		"docs:a": ch("a", "Dock hours", "", day(1)),
		"docs:b": ch("b", "Dock hours from December", "supersedes: dock hours\n", day(3)),
		"docs:c": ch("c", "Dock in January", "supersedes: docs:b@4\n", day(2)), // seen before b, still after it
		"docs:d": ch("d", "Itself", "supersedes: docs:d\n", day(4)),
	}
	links := supersedeLinks(first)
	if links["docs:a"].SourceID != "b" || links["docs:b"].SourceID != "c" || len(links) != 2 {
		t.Fatalf("links: %+v", links)
	}
	var cs []ports.Chunk
	for _, k := range []string{"docs:c", "docs:b", "docs:a"} {
		cs = append(cs, first[k])
	}
	var got []string
	for _, c := range inOrder(cs, links) {
		got = append(got, c.SourceID)
	}
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("order: %v", got)
	}
	claims := []answer.Claim{{Disagree: true, Cites: []answer.Cite{{Cite: "docs:t@1"}, {Cite: "docs:a@2", Title: "Dock hours"}}}, {Cites: []answer.Cite{{Cite: "docs:x@1"}}}}
	if s := supersedeSuggestions(claims, "docs:t"); len(s) != 1 || s[0].Cite != "docs:a" {
		t.Fatalf("suggestions: %+v", s)
	}
}

// SB-4506: catch-up is today back to 90 days.
func TestCatchUpSince(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	if _, d, err := catchUpSince("", now, nil); err != nil || d != "2026-11-18" {
		t.Fatalf("default: %q %v", d, err)
	}
	for _, bad := range []string{"2026-11-26", "2026-08-26", "soon"} {
		if _, _, err := catchUpSince(bad, now, nil); err == nil {
			t.Fatalf("%s accepted", bad)
		}
	}
	if _, _, err := catchUpSince("2026-08-27", now, nil); err != nil {
		t.Fatal(err)
	}
	cs := []ports.Chunk{{Kind: "docs", SourceID: "a", Seen: now}, {Kind: "docs", SourceID: "b", Seen: now.AddDate(0, 0, -10)}, {Kind: "docs", SourceID: "c", Seen: now, Rebuilt: true}}
	if got := changedDocs(cs, now.AddDate(0, 0, -1)); strings.Join(got, ",") != "docs:a" {
		t.Fatalf("changed: %v", got)
	}
}

// SB-4507: a row under 5 shows nothing but the mask; medians and the late share are from
// the records.
func TestServiceRows(t *testing.T) {
	at := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	var qs []StewardQuestion
	for i := 0; i < 5; i++ {
		q := StewardQuestion{At: at, Request: &RequestMeta{Category: "Access", Due: at.Add(48 * time.Hour)}}
		if i < 4 {
			q.Reply = &StewardReply{At: at.Add(time.Duration(i+1) * 24 * time.Hour)} // 24, 48, 72, 96 hours
			q.FirstClaim = at.Add(time.Hour)
		}
		qs = append(qs, q)
	}
	r := serviceRow("Access", "2026-W48", qs, nil)
	if r.Count != 5 || *r.FirstReply != 1 || *r.Close != 60 || *r.LateShare != 0.5 {
		t.Fatalf("row: %+v %v %v %v", r, *r.FirstReply, *r.Close, *r.LateShare)
	}
	if m := serviceRow("Access", "2026-W48", qs[:4], nil); !m.FewerThan5 || m.Count != 0 || m.Close != nil {
		t.Fatalf("masked: %+v", m)
	}
	weeks, from := serviceWeeks(at, nil)
	if len(weeks) != ServiceWeeks || weeks[11] != "2026-W48" || from.Weekday() != time.Monday {
		t.Fatalf("weeks: %v %v", weeks, from)
	}
}

package report

import (
	"fmt"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

// SB-3804: people who asked, of the seats, this week and last, and by space only where 5
// or more asked; searches and limited asks don't count; no one is named.
func TestAdoption(t *testing.T) {
	tr := func(user, space, agent string) ask.Trace {
		return ask.Trace{User: user, Space: space, Response: ask.Response{Agent: agent}}
	}
	var week []ask.Trace
	for i := 0; i < 6; i++ {
		week = append(week, tr(fmt.Sprintf("usr_f%d", i), "spc_finance", "ask"), tr(fmt.Sprintf("usr_f%d", i), "spc_finance", "ask"))
	}
	for i := 0; i < 4; i++ {
		week = append(week, tr(fmt.Sprintf("usr_o%d", i), "spc_ops", "ask"))
	}
	week = append(week, tr("usr_s", "", ask.AgentSearch), tr("usr_l", "", ask.AgentLimited), tr("usr_f0", "", "ask"))
	before := []ask.Trace{tr("usr_a", "", "ask"), tr("usr_b", "", "ask"), tr("usr_a", "", "ask")}
	a := AdoptionOf(week, before, 120)
	if a.Asked != 10 || a.AskedLastWeek != 2 || a.Seats != 120 || len(a.Spaces) != 1 || a.Spaces["finance"] != 6 {
		t.Fatalf("%+v", a)
	}
	line := a.line()
	if line != "10 of 120 seats asked this week (2 last week); by space: finance 6" || strings.Contains(line, "usr_") {
		t.Fatal(line)
	}
	r := Report{Adoption: a}
	if !strings.Contains(r.Markdown(), "| Adoption | 10 of 120 seats asked this week (2 last week); by space: finance 6 |") {
		t.Fatal(r.Markdown())
	}
	if got := (Adoption{Asked: 1, Seats: 3}).line(); got != "1 of 3 seats asked this week (0 last week)" {
		t.Fatal(got)
	}
}

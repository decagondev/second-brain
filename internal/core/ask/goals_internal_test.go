package ask

import "testing"

// SB-4101: a goal's front matter, under its title line or not; and nesting by parent_goal.
func TestParseAndNestGoals(t *testing.T) {
	g, ok := ParseGoal("Close faster\n\n---\ntype: goal\nowner: Ann@acme.example\nquarter: 2027-q1\nstatus: At Risk\nparent_goal: Faster finance\n---\nbody")
	if !ok || g.Owner != "ann@acme.example" || g.Quarter != "2027-Q1" || g.Status != "at risk" || g.Parent != "Faster finance" {
		t.Fatalf("goal %+v %v", g, ok)
	}
	if g, ok := ParseGoal("---\ntype: goal\nquarter: soon\nstatus: great\n---\n"); !ok || g.Quarter != "" || g.Status != "" {
		t.Fatalf("bad values kept %+v", g)
	}
	for _, text := range []string{"---\ntype: decision\n---\n", "no front matter", "a\nb\n\n---\ntype: goal\n---\n"} {
		if _, ok := ParseGoal(text); ok {
			t.Fatalf("%q is a goal", text)
		}
	}
	top := &Goal{Title: "Faster finance", Quarter: "2027-Q1", key: "docs:Finance/Goals/faster.md"}
	kid := &Goal{Title: "Close faster", Quarter: "2027-Q1", parent: "Faster finance", key: "docs:Finance/AP/close.md"}
	orphan := &Goal{Title: "Cut costs", Quarter: "2026-Q4", parent: "A goal nobody here can open", key: "docs:x"}
	none := &Goal{Title: "Someday", key: "docs:y"}
	qs := nestGoals([]*Goal{kid, orphan, top, none})
	if len(qs) != 3 || qs[0].Quarter != "2027-Q1" || qs[1].Quarter != "2026-Q4" || qs[2].Quarter != "" {
		t.Fatalf("quarters %+v", qs)
	}
	if len(qs[0].Goals) != 1 || qs[0].Goals[0].Title != "Faster finance" || len(qs[0].Goals[0].Goals) != 1 || qs[0].Goals[0].Goals[0].Title != "Close faster" {
		t.Fatalf("nesting %+v", qs[0])
	}
	if qs[1].Goals[0].Title != "Cut costs" {
		t.Fatalf("an orphan is shown without its parent: %+v", qs[1])
	}
}

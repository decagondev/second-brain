package manifest

import (
	"strings"
	"testing"
	"time"
)

// SB-4005: a space's brief is one agent of its own on one weekday, and needs stewards.
func TestBrief(t *testing.T) {
	src := strings.Replace(treeHead, "    review_after_days: 90\n", "    review_after_days: 90\n    brief: { agent: close-checklist, weekly: tuesday }\n", 1)
	m, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	b := m.Spaces["finance"].Brief
	if b == nil || b.Agent != "close-checklist" || b.Weekly != "tuesday" || b.Weekday() != time.Tuesday {
		t.Fatalf("brief = %+v", b)
	}
	if (Brief{Weekly: "sunday"}).Weekday() != time.Sunday {
		t.Fatal("sunday")
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || again.Spaces["finance"].Brief == nil || *again.Spaces["finance"].Brief != *b {
		t.Fatalf("format round trip: %v", ds)
	}
	cases := map[string]string{
		"not a day":                 "{ agent: close-checklist, weekly: someday }",
		"isn't one of this space's": "{ agent: incident-summary, weekly: monday }",
		"drafts replies":            "{ agent: draft-reply, weekly: monday }",
		"needs an agent":            "{ weekly: monday }",
		"unknown field in a brief":  "{ agent: ask, weekly: monday, daily: true }",
	}
	for want, brief := range cases {
		src := strings.Replace(treeHead, "    review_after_days: 90\n", "    review_after_days: 90\n    brief: "+brief+"\n", 1)
		if _, ds := Check([]byte(src)); !strings.Contains(format("m.yaml", ds), want) {
			t.Errorf("%s: got %s", want, format("m.yaml", ds))
		}
	}
	up := strings.Replace(treeHead, "    review_after_days: 90\n", "    review_after_days: 90\n    updates: true\n", 1)
	if m, ds := Check([]byte(up)); len(ds) > 0 || !m.Spaces["finance"].Updates {
		t.Fatalf("updates: %v", ds)
	} else if again, ds := Check(Format(m)); len(ds) > 0 || !again.Spaces["finance"].Updates {
		t.Fatalf("updates round trip: %v", ds)
	}
	if _, ds := Check([]byte(strings.Replace(up, "updates: true", "updates: weekly", 1))); !strings.Contains(format("m.yaml", ds), "must be true or false") {
		t.Errorf("updates: weekly: %s", format("m.yaml", ds))
	}
	noStewards := strings.Replace(strings.Replace(treeHead, "    stewards: [group:finance-leads]\n", "", 1), "    label: confidential\n", "    label: internal\n    brief: { agent: ask, weekly: monday }\n", 1)
	if _, ds := Check([]byte(noStewards)); !strings.Contains(format("m.yaml", ds), "needs stewards") {
		t.Errorf("no stewards: %s", format("m.yaml", ds))
	}
}

// SB-4104, SB-4105: a space's request categories and its budget, which a team keeps
// within its department's.
func TestRequestsAndBudget(t *testing.T) {
	src := strings.Replace(treeHead, "    review_after_days: 90\n", "    review_after_days: 90\n    requests: { categories: [Access, \"IT: laptop\"], due_days: 3 }\n    budget_usd_month: 40\n", 1)
	m, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	f := m.Spaces["finance"]
	if f.Requests == nil || len(f.Requests.Categories) != 2 || f.Requests.Categories[1] != "IT: laptop" || f.Requests.DueDays != 3 || f.BudgetUSDMonth != 40 {
		t.Fatalf("finance = %+v %+v", f.Requests, f.BudgetUSDMonth)
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || again.Spaces["finance"].Requests == nil || again.Spaces["finance"].Requests.DueDays != 3 || again.Spaces["finance"].BudgetUSDMonth != 40 {
		t.Fatalf("format round trip: %v", ds)
	}
	if d := strings.Replace(src, ", due_days: 3", "", 1); true {
		if m, ds := Check([]byte(d)); len(ds) > 0 || m.Spaces["finance"].Requests.DueDays != DefaultDueDays {
			t.Fatalf("default due_days: %v", ds)
		}
	}
	cases := map[string]string{
		"listed twice":              "    requests: { categories: [Access, access] }\n",
		"at least one category":     "    requests: { categories: [] }\n",
		"unknown field in requests": "    requests: { categories: [Access], sla: 3 }\n",
		"must be 1 to":              "    requests: { categories: [\"<b>\"] }\n",
		"above the tenant's":        "    budget_usd_month: 100000000\n",
	}
	for want, line := range cases {
		src := strings.Replace(treeHead, "    review_after_days: 90\n", "    review_after_days: 90\n"+line, 1)
		if _, ds := Check([]byte(src)); !strings.Contains(format("m.yaml", ds), want) {
			t.Errorf("%s: got %s", want, format("m.yaml", ds))
		}
	}
	team := src + "  ap:\n    parent: finance\n    members: [group:ap]\n    budget_usd_month: 50\n    scope: { docs: [\"/Finance/AP/**\"] }\n"
	if _, ds := Check([]byte(team)); !strings.Contains(format("m.yaml", ds), "above its parent finance's") {
		t.Errorf("team over its department: %s", format("m.yaml", ds))
	}
}

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
	noStewards := strings.Replace(strings.Replace(treeHead, "    stewards: [group:finance-leads]\n", "", 1), "    label: confidential\n", "    label: internal\n    brief: { agent: ask, weekly: monday }\n", 1)
	if _, ds := Check([]byte(noStewards)); !strings.Contains(format("m.yaml", ds), "needs stewards") {
		t.Errorf("no stewards: %s", format("m.yaml", ds))
	}
}

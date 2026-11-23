package retention

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// SB-1905: the plan's counts are what the sweep then deletes, the space passes included,
// and the plan deletes nothing.
func TestPlanMatchesTheSweep(t *testing.T) {
	day := 24 * time.Hour
	now := time.Date(2027, 6, 1, 3, 0, 0, 0, time.UTC)
	recs := []srec{
		{"spc_finance", now.Add(-100 * day)}, // finance keeps 90: goes
		{"spc_finance", now.Add(-400 * day)}, // older than the tenant's 365: goes, counted once
		{"spc_people", now.Add(-100 * day)},  // kept
		{"spc_people", now.Add(-400 * day)},  // goes
		{"spc_held", now.Add(-400 * day)},    // a hold keeps it
		{"", now.Add(-10 * day)},
	}
	st := &spaced{recs: append([]srec(nil), recs...)}
	j := Job{Tenant: "acme-logistics", Days: 365, Stores: []Store{st}, Audit: &auditor{}, Now: func() time.Time { return now },
		SpaceDays: map[string]int{"spc_finance": 90}, Held: func(space, _ string) bool { return space == "spc_held" }}
	plan, err := j.Plan(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.recs) != len(recs) {
		t.Fatalf("the plan deleted: %d left", len(st.recs))
	}
	if plan.Counts["traces"] != 3 || plan.Held != 1 {
		t.Fatalf("plan %+v", plan)
	}
	rep, err := j.Sweep(context.Background())
	if err != nil || rep.Counts["traces"] != plan.Counts["traces"] || rep.Held != plan.Held {
		t.Fatalf("sweep %+v %v, plan %+v", rep, err, plan)
	}
	// In 300 days, the records now 100 days old in people are past 365 too.
	st.recs = append([]srec(nil), recs...)
	later, err := j.Plan(context.Background(), 300*day)
	if err != nil || later.Counts["traces"] != 4 {
		t.Fatalf("in 300 days: %+v %v", later, err)
	}
	if _, err := j.Plan(context.Background(), -day); !errors.Is(err, ErrInvalid) {
		t.Fatalf("negative: %v", err)
	}
}

func TestPlanNamesAFailedStore(t *testing.T) {
	j, a := job(&fake{name: "traces"}, &fake{name: "logs", fail: true})
	rep, err := j.Plan(context.Background(), 0)
	if !errors.Is(err, ErrStore) || len(rep.Failed) != 1 || rep.Failed[0] != "logs" || len(a.rows) != 0 {
		t.Fatalf("%+v %v %+v", rep, err, a.rows)
	}
}

// SB-1903: an export is audited first, holds what erase removes, and after the erase is empty.
func TestExportThenEraseThenExport(t *testing.T) {
	var order []string
	tr := &fake{name: "traces", recs: []rec{{"usr_a", now}, {"usr_b", now}, {"usr_a", now.Add(-time.Hour)}}, log: &order}
	fb := &fake{name: "feedback", recs: []rec{{"usr_a", now}}}
	j, a := job(tr, fb)
	ex, err := j.Export(context.Background(), "usr_dpo", "usr_a", "subject access request 7")
	if err != nil || ex.Count() != 3 || len(ex.Records["traces"]) != 2 || len(ex.Records["feedback"]) != 1 || len(ex.SHA256) != 64 {
		t.Fatalf("export %+v %v", ex, err)
	}
	if len(a.rows) != 2 || a.rows[0].State != "requested" || a.rows[0].Subject != "export:usr_a" || !strings.Contains(a.rows[1].Reason, "sha256="+ex.SHA256) {
		t.Fatalf("audit %+v", a.rows)
	}
	if strings.Contains(a.rows[1].Reason, `"user"`) {
		t.Fatalf("the audit row holds the export's text: %q", a.rows[1].Reason)
	}
	rep, err := j.Erase(context.Background(), "usr_dpo", "usr_a", "erasure request 7")
	if err != nil || rep.Counts["traces"]+rep.Counts["feedback"] != ex.Count() {
		t.Fatalf("erase %+v %v", rep, err)
	}
	again, err := j.Export(context.Background(), "usr_dpo", "usr_a", "check")
	if err != nil || again.Count() != 0 || again.Records["traces"] == nil {
		t.Fatalf("after erase: %+v %v", again, err)
	}
}

func TestExportRefusals(t *testing.T) {
	j, a := job(&fake{name: "traces"})
	for _, c := range [][2]string{{"ann", "r"}, {"usr_a", " "}} {
		if _, err := j.Export(context.Background(), "usr_dpo", c[0], c[1]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%v: %v", c, err)
		}
	}
	if len(a.rows) != 0 {
		t.Fatalf("audited a refused request: %+v", a.rows)
	}
	a.fail = true
	if _, err := j.Export(context.Background(), "usr_dpo", "usr_a", "r"); !errors.Is(err, ErrAudit) {
		t.Fatalf("audit down: %v", err)
	}
	j2, a2 := job(&fake{name: "traces", recs: []rec{{"usr_a", now}}}, &fake{name: "logs", fail: true})
	ex, err := j2.Export(context.Background(), "usr_dpo", "usr_a", "r")
	if !errors.Is(err, ErrStore) || ex.Records != nil || a2.rows[1].State != "failed" {
		t.Fatalf("a failed store: %+v %v %+v", ex, err, a2.rows)
	}
}

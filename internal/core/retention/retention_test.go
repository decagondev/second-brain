package retention

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

type rec struct {
	user string
	at   time.Time
}

type fake struct {
	name string
	recs []rec
	fail bool
	log  *[]string
}

func (f *fake) Name() string { return f.name }
func (f *fake) Expire(_ context.Context, _ string, before time.Time, held Held) (int, int, error) {
	n, err := f.drop(func(r rec) bool { return r.at.Before(before) && !held.Is("", r.user) })
	return n, 0, err
}
func (f *fake) Erase(_ context.Context, _ string, user string) (int, error) {
	if f.log != nil {
		*f.log = append(*f.log, "erase "+f.name)
	}
	return f.drop(func(r rec) bool { return r.user == user })
}
func (f *fake) Plan(_ context.Context, _ string, before time.Time, held Held) (int, int, error) {
	if f.fail {
		return 0, 0, errors.New("down")
	}
	n := 0
	for _, r := range f.recs {
		if r.at.Before(before) && !held.Is("", r.user) {
			n++
		}
	}
	return n, 0, nil
}
func (f *fake) Export(_ context.Context, _ string, user string) ([]json.RawMessage, error) {
	if f.fail {
		return nil, errors.New("down")
	}
	var out []json.RawMessage
	for _, r := range f.recs {
		if r.user == user {
			out = append(out, json.RawMessage(`{"user":"`+r.user+`"}`))
		}
	}
	return out, nil
}
func (f *fake) drop(m func(rec) bool) (int, error) {
	if f.fail {
		return 0, errors.New("down")
	}
	var keep []rec
	n := 0
	for _, r := range f.recs {
		if m(r) {
			n++
		} else {
			keep = append(keep, r)
		}
	}
	f.recs = keep
	return n, nil
}

type auditor struct {
	rows []audit.Event
	fail bool
	log  *[]string
}

func (a *auditor) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	if a.fail {
		return audit.Row{}, errors.New("audit down")
	}
	a.rows = append(a.rows, e)
	if a.log != nil {
		*a.log = append(*a.log, "audit "+e.State)
	}
	return audit.Row{ID: "aud_" + e.State}, nil
}

var now = time.Date(2027, 3, 10, 3, 0, 0, 0, time.UTC)

func job(stores ...Store) (Job, *auditor) {
	a := &auditor{}
	return Job{Tenant: "acme-logistics", Days: 30, Stores: stores, Audit: a, Now: func() time.Time { return now }}, a
}

func TestSweepKeepsExactlyDaysOld(t *testing.T) {
	cut := now.Add(-30 * 24 * time.Hour)
	tr := &fake{name: "traces", recs: []rec{{"usr_a", cut}, {"usr_a", cut.Add(-time.Millisecond)}, {"usr_b", now}}}
	j, a := job(tr)
	rep, err := j.Sweep(context.Background())
	if err != nil || rep.Counts["traces"] != 1 || len(tr.recs) != 2 {
		t.Fatalf("sweep: %+v %v, left %v", rep, err, tr.recs)
	}
	if len(a.rows) != 1 || a.rows[0].Kind != "retention" || a.rows[0].Subject != "sweep" || !strings.Contains(a.rows[0].Reason, "traces=1") {
		t.Fatalf("audit: %+v", a.rows)
	}
}

func TestSweepRefusesBadDays(t *testing.T) {
	tr := &fake{name: "traces", recs: []rec{{"usr_a", now.Add(-24 * time.Hour)}}}
	for _, d := range []int{0, -1, 29, 3651} {
		j, a := job(tr)
		j.Days = d
		if _, err := j.Sweep(context.Background()); !errors.Is(err, ErrDays) || len(tr.recs) != 1 || len(a.rows) != 0 {
			t.Errorf("days %d: %v, left %d", d, err, len(tr.recs))
		}
	}
}

func TestSweepRecordsAFailedStore(t *testing.T) {
	ok := &fake{name: "feedback", recs: []rec{{"usr_a", now.Add(-400 * 24 * time.Hour)}}}
	bad := &fake{name: "logs", fail: true}
	j, a := job(ok, bad)
	rep, err := j.Sweep(context.Background())
	if !errors.Is(err, ErrStore) || rep.Counts["feedback"] != 1 || a.rows[0].State != "failed" || !strings.Contains(a.rows[0].Reason, "failed=logs") {
		t.Fatalf("%+v %v %+v", rep, err, a.rows)
	}
}

func TestEraseAuditsFirst(t *testing.T) {
	var order []string
	tr := &fake{name: "traces", recs: []rec{{"usr_a", now}, {"usr_b", now}}, log: &order}
	j, a := job(tr)
	a.log = &order
	rep, err := j.Erase(context.Background(), "usr_op", "usr_a", "subject request 42")
	if err != nil || rep.Counts["traces"] != 1 || len(tr.recs) != 1 || len(rep.Audit) != 2 {
		t.Fatalf("%+v %v", rep, err)
	}
	if strings.Join(order, ",") != "audit requested,erase traces,audit done" {
		t.Fatalf("order: %v", order)
	}
	if a.rows[0].Subject != "erase:usr_a" || a.rows[0].Actor != "usr_op" {
		t.Fatalf("rows: %+v", a.rows)
	}
}

func TestEraseWithoutAuditErasesNothing(t *testing.T) {
	tr := &fake{name: "traces", recs: []rec{{"usr_a", now}}}
	j, a := job(tr)
	a.fail = true
	if _, err := j.Erase(context.Background(), "usr_op", "usr_a", "r"); !errors.Is(err, ErrAudit) || len(tr.recs) != 1 {
		t.Fatalf("%v, left %d", err, len(tr.recs))
	}
}

func TestEraseValidates(t *testing.T) {
	j, a := job()
	for _, u := range []string{"", "ann@acme.example", "usr_", "usr_A", "usr_a/../b", "*"} {
		if _, err := j.Erase(context.Background(), "usr_op", u, "r"); !errors.Is(err, ErrInvalid) {
			t.Errorf("user %q: %v", u, err)
		}
	}
	if _, err := j.Erase(context.Background(), "usr_op", "usr_a", " "); !errors.Is(err, ErrInvalid) {
		t.Errorf("no reason: %v", err)
	}
	if len(a.rows) != 0 {
		t.Fatal("an invalid request was audited as requested")
	}
}

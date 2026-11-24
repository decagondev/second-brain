package retention

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type forgetStore struct {
	name string
	recs []struct{ user, space string }
}

func (f *forgetStore) Name() string { return f.name }
func (f *forgetStore) sel(held Held, del bool) (int, int) {
	n, k := 0, 0
	var keep []struct{ user, space string }
	for _, r := range f.recs {
		if held.Is(r.space, r.user) {
			k++
			keep = append(keep, r)
			continue
		}
		n++
		if !del {
			keep = append(keep, r)
		}
	}
	f.recs = keep
	return n, k
}
func (f *forgetStore) Expire(_ context.Context, _ string, _ time.Time, held Held) (int, int, error) {
	n, k := f.sel(held, true)
	return n, k, nil
}
func (f *forgetStore) Plan(_ context.Context, _ string, _ time.Time, held Held) (int, int, error) {
	n, k := f.sel(held, false)
	return n, k, nil
}
func (f *forgetStore) Erase(context.Context, string, string) (int, error) { return 0, nil }
func (f *forgetStore) Export(context.Context, string, string) ([]json.RawMessage, error) {
	return nil, nil
}

// SB-3502: a person forgets their own records of the forgettable stores, whatever their age;
// a held one is kept and counted, nobody else's is touched, and the audit holds counts only.
func TestForget(t *testing.T) {
	recs := func() []struct{ user, space string } {
		return []struct{ user, space string }{{"usr_ann", "spc_ops"}, {"usr_ann", "spc_legal"}, {"usr_bob", "spc_ops"}}
	}
	traces, evals := &forgetStore{name: "traces", recs: recs()}, &forgetStore{name: "eval cases", recs: recs()}
	j, a := job(traces, evals)
	j.Held = func(space, _ string) bool { return space == "spc_legal" }
	if _, err := j.Forget(context.Background(), "bob"); err == nil {
		t.Fatal("not a usr_ id")
	}
	rep, err := j.Forget(context.Background(), "usr_ann")
	if err != nil || rep.Counts["traces"] != 1 || rep.Held != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	if len(traces.recs) != 2 || traces.recs[0].space != "spc_legal" || traces.recs[1].user != "usr_bob" || len(evals.recs) != 3 {
		t.Fatalf("left %+v %+v", traces.recs, evals.recs)
	}
	if len(a.rows) != 2 || a.rows[0].Subject != "forget:usr_ann" || a.rows[1].State != "done" || !strings.Contains(a.rows[1].Reason, "held=1") {
		t.Fatalf("audit %+v", a.rows)
	}
}

package apply

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/ports"
)

// world is an in-memory control plane and account. Every write counts as one op, and
// the op numbered crashAt fails, simulating a crash at that point.
type world struct {
	reg       map[string][]ports.Resource
	resources map[string]string // kind/name -> id
	journal   []ports.JournalEntry
	cells     map[string]ports.CellRecord
	ops       int
	crashAt   int
	writes    int
}

var errCrash = errors.New("crash")

func (w *world) op() error {
	w.ops++
	if w.ops == w.crashAt {
		return errCrash
	}
	w.writes++
	return nil
}

func (w *world) Applied(_ context.Context, tenant string) ([]ports.Resource, bool, error) {
	rs, ok := w.reg[tenant]
	return append([]ports.Resource(nil), rs...), ok, nil
}
func (w *world) Put(_ context.Context, tenant string, rs []ports.Resource) error {
	if err := w.op(); err != nil {
		return err
	}
	w.reg[tenant] = append([]ports.Resource(nil), rs...)
	return nil
}
func (w *world) Account() string { return "mem://acct" }
func (w *world) Ensure(_ context.Context, r ports.CellResource) (string, error) {
	k := r.Tenant + "/" + r.Kind + "/" + r.Name
	if id, ok := w.resources[k]; ok {
		return id, nil // finding is not a write
	}
	if err := w.op(); err != nil {
		return "", err
	}
	w.resources[k] = fmt.Sprintf("id-%d", len(w.resources)+1)
	return w.resources[k], nil
}
func (w *world) EnsureRootKey(ctx context.Context, r ports.CellResource) (string, error) {
	return w.Ensure(ctx, r)
}
func (w *world) Wrap(context.Context, string, []byte) ([]byte, error)   { return nil, nil }
func (w *world) Unwrap(context.Context, string, []byte) ([]byte, error) { return nil, nil }
func (w *world) PutCell(_ context.Context, c ports.CellRecord) error {
	if err := w.op(); err != nil {
		return err
	}
	w.cells[c.Tenant] = c
	return nil
}
func (w *world) Record(_ context.Context, e ports.JournalEntry) error {
	if err := w.op(); err != nil {
		return err
	}
	w.journal = append(w.journal, e)
	return nil
}

func newWorld() *world {
	return &world{reg: map[string][]ports.Resource{}, resources: map[string]string{}, cells: map[string]ports.CellRecord{}}
}

func (w *world) deps() Deps {
	return Deps{Registry: w, Writer: w, Placement: w, Keys: w, Journal: w, Cells: w, BaseDomain: "sb.example", NewID: func() string { return "apply-1" }}
}

func tenant(t *testing.T, src string) manifest.Tenant {
	t.Helper()
	m, ds := manifest.Check([]byte(src))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	return m
}

const pilot = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack }
agents: [ask]
`

func TestNewCellThenNoChanges(t *testing.T) {
	w := newWorld()
	m := tenant(t, pilot)
	var lines []string
	p, err := Run(context.Background(), w.deps(), m, func(c plan.Change) { lines = append(lines, c.Line()) })
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != len(p.Changes) || len(w.resources) != 5 {
		t.Fatalf("lines %d of %d, resources %d", len(lines), len(p.Changes), len(w.resources))
	}
	if !reflect.DeepEqual(w.reg[m.Tenant], plan.Desired(m)) {
		t.Fatal("registry does not hold the desired state")
	}
	want := ports.CellRecord{Tenant: "acme-logistics", Region: "eu", Endpoint: "mem://acct/cells/acme-logistics",
		Hosts: []string{"acme-logistics.sb.example"}, Custody: "factory", Placement: "factory-account", KeyCustodian: "factory"}
	if !reflect.DeepEqual(w.cells[m.Tenant], want) {
		t.Fatalf("cell record %+v", w.cells[m.Tenant])
	}
	before := w.writes
	p, err = Run(context.Background(), w.deps(), m, nil)
	if err != nil || len(p.Changes) != 0 || w.writes != before {
		t.Fatalf("second apply: %v changes, %d writes, err %v; want none", len(p.Changes), w.writes-before, err)
	}
}

// A crash at any write, then a rerun, ends exactly where one clean run ends: same
// registry record, no duplicate resource, and a journal where every step that says
// done also said start first.
func TestCrashAtEveryStepResumes(t *testing.T) {
	m := tenant(t, pilot)
	later := tenant(t, strings.Replace(pilot, "agents: [ask]", "agents: [ask, draft-reply]\nbudget_usd_month: 50", 1))
	clean := newWorld()
	for _, x := range []manifest.Tenant{m, later} {
		if _, err := Run(context.Background(), clean.deps(), x, nil); err != nil {
			t.Fatal(err)
		}
	}
	total := clean.ops
	for k := 1; k <= total; k++ {
		w := newWorld()
		w.crashAt = k
		for _, x := range []manifest.Tenant{m, later} {
			for attempt := 0; ; attempt++ {
				_, err := Run(context.Background(), w.deps(), x, nil)
				if err == nil {
					break
				}
				var se *StepError
				if !errors.As(err, &se) || !errors.Is(err, errCrash) || attempt > 0 {
					t.Fatalf("crash at op %d: %v", k, err)
				}
			}
		}
		if !reflect.DeepEqual(w.cells, clean.cells) {
			t.Fatalf("crash at op %d: cells %v\nwant %v", k, w.cells, clean.cells)
		}
		if !reflect.DeepEqual(w.reg, clean.reg) {
			t.Fatalf("crash at op %d: registry %v\nwant %v", k, w.reg, clean.reg)
		}
		if len(w.resources) != 5 {
			t.Fatalf("crash at op %d: %d resources, want 5 (no duplicates)", k, len(w.resources))
		}
		started := map[string]bool{}
		for _, e := range w.journal {
			key := fmt.Sprintf("%s/%d", e.ApplyID, e.Step)
			if e.Phase == "start" {
				started[key] = true
			} else if !started[key] {
				t.Fatalf("crash at op %d: step %s done without start", k, key)
			}
		}
		if p, err := Run(context.Background(), w.deps(), later, nil); err != nil || len(p.Changes) != 0 {
			t.Fatalf("crash at op %d: after resume, plan is not empty: %v %v", k, p.Changes, err)
		}
	}
}

// The registry can't answer: apply fails and writes nothing (never "no cell").
func TestRegistryUnavailable(t *testing.T) {
	w := newWorld()
	d := w.deps()
	d.Registry = broken{}
	_, err := Run(context.Background(), d, tenant(t, pilot), nil)
	var ue *UnavailableError
	if !errors.As(err, &ue) || w.writes != 0 {
		t.Fatalf("err %v, writes %d", err, w.writes)
	}
}

type broken struct{}

func (broken) Applied(context.Context, string) ([]ports.Resource, bool, error) {
	return nil, false, errors.New("timeout")
}

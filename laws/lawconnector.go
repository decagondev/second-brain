package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/core/ports"
)

// Connector-level laws (SB-201): what a connector itself must guarantee, checked on the
// fault-injecting mock. LAW-03: under any fault, ACL is never Allow; with no fault it
// tells the truth. LAW-05: Act returns a target-issued receipt or an error, and the
// same key twice has one effect.

// ConnectorWorld drives one connector over a source the law controls.
type ConnectorWorld interface {
	// ACL asks for user's verdicts on n documents; access[i] is the truth; fault is
	// "", "500", "timeout" or "404" for the call.
	ACL(access []bool, fault string) (verdicts []string, err error)
	// Act runs action key against a target that does or doesn't issue ids; it returns
	// the receipt id ("" if none) and the error.
	Act(key string, targetIssuesID bool) (receipt string, err error)
	// Effects is how many changes the target has applied.
	Effects() int
}

// NewConnectorWorld builds the system under test: the mock source and its writable
// connector (SB-201).
var NewConnectorWorld = func() ConnectorWorld {
	src := mock.NewSource("tickets")
	return &connWorld{src: src, c: mock.New(src, true)}
}

type connWorld struct {
	src *mock.Source
	c   *mock.Connector
	n   int
}

func (w *connWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		id := fmt.Sprintf("d%d_%d", w.n, i)
		if ok {
			w.src.Put(id, "t", "x", "ann@acme.example")
		} else {
			w.src.Put(id, "t", "x", "bob@acme.example")
		}
		refs[i] = ports.SourceRef{Kind: "tickets", ID: id}
	}
	w.src.InjectFault(mock.Fault(fault))
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *connWorld) Act(key string, issues bool) (string, error) {
	w.src.IssueIDs(issues)
	r, err := w.c.Act(context.Background(), ports.Action{Target: "tickets:OPS-1", Payload: []byte(`{}`)}, ports.IdempotencyKey(key))
	return r.TargetID, err
}

func (w *connWorld) Effects() int { return w.src.Effects() }

func law03Connector(r *rand.Rand, w ConnectorWorld) error {
	for i := 0; i < 6; i++ {
		access := make([]bool, 1+r.Intn(8))
		for j := range access {
			access[j] = r.Intn(2) == 0
		}
		fault := []string{"", "", "500", "timeout", "404"}[r.Intn(5)]
		if fault == "timeout" && r.Intn(4) != 0 {
			fault = "500" // keep the slow case rare; it waits out the 800 ms deadline
		}
		got, err := w.ACL(access, fault)
		if err != nil {
			return err
		}
		if len(got) != len(access) {
			return fmt.Errorf("asked %d refs, got %d verdicts", len(access), len(got))
		}
		for j, v := range got {
			want := "deny"
			if access[j] {
				want = "allow"
			}
			if fault != "" {
				want = "unknown"
			}
			if v != want {
				return fmt.Errorf("fault %q, ref %d (access %v): verdict %s, want %s", fault, j, access[j], v, want)
			}
		}
	}
	return nil
}

func law05Connector(r *rand.Rand, w ConnectorWorld) error {
	receipts := map[string]string{}
	effects := 0
	for i := 0; i < 8; i++ {
		key := fmt.Sprintf("apr_%d", r.Intn(4))
		issues := r.Intn(3) != 0
		got, err := w.Act(key, issues)
		if errors.Is(err, ErrPending) {
			return err
		}
		prev, seen := receipts[key]
		switch {
		case err == nil && got == "":
			return fmt.Errorf("Act %s succeeded without a receipt", key)
		case err == nil && seen && got != prev:
			return fmt.Errorf("Act %s twice gave receipts %s and %s", key, prev, got)
		case err == nil && !seen:
			receipts[key] = got
			effects++
		}
		if w.Effects() != effects {
			return fmt.Errorf("after %d distinct successful keys the target shows %d effects", effects, w.Effects())
		}
	}
	return nil
}

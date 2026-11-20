package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/quickbooks"
	"example.com/second-brain/internal/adapters/connectors/quickbooks/qbotest"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/obs"
)

// The QuickBooks Online connector under the laws (SB-1105). LAW-03: anything QuickBooks
// can't answer is Unknown. LAW-01: the readers list decides now. LAW-07: through the real
// pipeline, no account or routing number is ever stored, and every record is restricted.

func newQBO(readers *[]string) (*qbotest.Fake, *quickbooks.Connector) {
	f := qbotest.New()
	c := &quickbooks.Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return f.CredsJSON(*readers...), nil },
		Now:   func() time.Time { return time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC) }}
	return f, c
}

// NewQBOWorld is the connector-level world: access means the budget exists; "script"
// makes the first ref an id QuickBooks can't place.
var NewQBOWorld = func() ConnectorWorld {
	readers := []string{"ann@acme.example"}
	f, c := newQBO(&readers)
	return &qboWorld{f: f, c: c}
}

type qboWorld struct {
	f *qbotest.Fake
	c *quickbooks.Connector
	n int
}

func (w *qboWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		id := fmt.Sprintf("b%d_%d", w.n, i)
		if ok {
			w.f.PutBudget(qbotest.Budget{ID: id, Name: "B", Start: "2027-01-01", End: "2027-12-31"})
		}
		refs[i] = ports.SourceRef{Kind: "accounting", ID: "budgets/" + id}
	}
	switch fault {
	case "script":
		refs[0] = ports.SourceRef{Kind: "accounting", ID: "payroll/" + refs[0].ID}
	case "":
	default:
		w.f.FailNext(fault)
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *qboWorld) Act(key string, _ bool) (string, error) {
	r, err := w.c.Act(context.Background(), ports.Action{Target: "accounting:x"}, ports.IdempotencyKey(key))
	return r.TargetID, err
}

func (w *qboWorld) Effects() int { return 0 }

// law01QBO: adding and removing a reader changes the verdict at the next check.
func law01QBO(r *rand.Rand) error {
	readers := []string{}
	f, c := newQBO(&readers)
	f.PutBudget(qbotest.Budget{ID: "b1", Name: "FY27", Start: "2027-01-01", End: "2027-12-31"})
	ref := []ports.SourceRef{{Kind: "accounting", ID: "budgets/b1"}}
	for i := 0; i < 6; i++ {
		in := r.Intn(2) == 0
		readers = readers[:0]
		if in {
			readers = append(readers, "ann@acme.example")
		}
		vs, err := c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, ref)
		if err != nil {
			return err
		}
		if want := map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[in]; vs[0] != want {
			return fmt.Errorf("reader %v: the check said %v", in, vs[0])
		}
	}
	return nil
}

// QBOConnector wraps the connector under test (the identity in the real system).
type QBOConnector func(c *quickbooks.Connector, f *qbotest.Fake) ports.Connector

func law07QBO(r *rand.Rand, wrap QBOConnector) error {
	ctx := context.Background()
	readers := []string{"fin@acme.example"}
	f, c := newQBO(&readers)
	all := []ports.Space{{ID: spaces.ID("finance"), Name: "finance", Label: "restricted",
		Scope: map[string][]string{"accounting": {"**"}}}}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test", Connector: wrap(c, f), Embedder: hashing.Embedder{}, Index: idx,
		Spaces: func(context.Context) ([]ports.Space, error) { return all, nil }}
	for i := 0; i < 1+r.Intn(4); i++ {
		f.PutAccount(qbotest.Account{ID: fmt.Sprintf("a%d", i), Name: fmt.Sprintf("Account %d", i), Type: "Bank", Classification: "Asset",
			Balance: float64(1000 * (i + 1)), AcctNum: fmt.Sprintf("NUM-ACCT-%d-%x", i, r.Int63()), BankNum: fmt.Sprintf("NUM-BANK-%d-%x", i, r.Int63())})
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	chunks, err := idx.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return fmt.Errorf("nothing was indexed")
	}
	for _, ch := range chunks {
		if strings.Contains(ch.Text, "NUM-") {
			return fmt.Errorf("%s holds an account or routing number", ch.SourceID)
		}
		if ch.Label != "restricted" {
			return fmt.Errorf("%s in the finance space is labelled %q", ch.SourceID, ch.Label)
		}
	}
	return nil
}

func asIs(c *quickbooks.Connector, _ *qbotest.Fake) ports.Connector { return c }

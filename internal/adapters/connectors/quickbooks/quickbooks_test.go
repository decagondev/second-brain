package quickbooks

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/quickbooks/qbotest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

func setup(t *testing.T) (*qbotest.Fake, *Connector, *[]string) {
	t.Helper()
	f := qbotest.New()
	t.Cleanup(f.Server.Close)
	readers := &[]string{"fin@acme.example"}
	return f, &Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return f.CredsJSON(*readers...), nil },
		Now:   func() time.Time { return time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC) }}, readers
}

type sandbox struct {
	f       *qbotest.Fake
	c       *Connector
	readers *[]string
}

func (s sandbox) Connector() ports.Connector { return s.c }
func (s sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "accounting", ID: "budgets/b1"} }
func (s sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_fin", Email: "fin@acme.example"}
}
func (s sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "accounting", ID: fmt.Sprintf("budgets/b%d", i+1)}
	}
	return out
}
func (s sandbox) InjectACLFault(kind string) { s.f.FailNext(kind) }
func (s sandbox) Edit(ports.SourceRef) {
	s.f.PutBudget(qbotest.Budget{ID: "b1", Name: "FY27 opex", Start: "2027-01-01", End: "2027-12-31", Lines: map[string]float64{"Travel": 1200}})
}
func (s sandbox) ChangePermissions(ports.SourceRef) {
	*s.readers = append(*s.readers, "cfo@acme.example")
}
func (s sandbox) Action() (ports.Action, bool) { return ports.Action{}, false }
func (s sandbox) Effects() int                 { return 0 }

func TestKit(t *testing.T) {
	f, c, readers := setup(t)
	for i := 1; i <= 200; i++ {
		f.PutBudget(qbotest.Budget{ID: fmt.Sprintf("b%d", i), Name: fmt.Sprintf("Budget %d", i), Start: "2027-01-01", End: "2027-12-31",
			Lines: map[string]float64{"Travel": 1000}})
	}
	connectorkit.Run(t, sandbox{f, c, readers})
}

// SB-1105: account numbers never leave the connector; reports cover the last 12 months.
func TestNumbersNeverRead(t *testing.T) {
	f, c, _ := setup(t)
	ctx := context.Background()
	f.PutAccount(qbotest.Account{ID: "a1", Name: "Operating account", Type: "Bank", Classification: "Asset", Balance: 125000.5,
		AcctNum: "ACCTNUM-7788", BankNum: "BANKNUM-0011"})
	f.PutAccount(qbotest.Account{ID: "a2", Name: "Sales", Type: "Income", Classification: "Revenue", Balance: 98000})
	d, err := c.Fetch(ctx, ports.SourceRef{Kind: "accounting", ID: "accounts/a1"})
	if err != nil || d.Fields["Name"] != "Operating account" || d.Location != "accounts/Asset/Operating account" {
		t.Fatalf("%+v %v", d, err)
	}
	if strings.Contains(fmt.Sprint(d), "ACCTNUM") || strings.Contains(fmt.Sprint(d), "BANKNUM") {
		t.Fatalf("an account number left the connector: %+v", d)
	}
	set, _, err := c.Changes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprint(set)
	if !strings.Contains(got, "reports/ProfitAndLoss/2026-11") || !strings.Contains(got, "reports/BalanceSheet/2025-12") || strings.Contains(got, "2025-11") {
		t.Fatalf("report months: %s", got)
	}
	r, err := c.Fetch(ctx, ports.SourceRef{Kind: "accounting", ID: "reports/ProfitAndLoss/2026-11"})
	if err != nil || !strings.Contains(r.Text, "Sales: 98000") || strings.Contains(r.Text, "Operating account") {
		t.Fatalf("P&L: %q %v", r.Text, err)
	}
	if _, err := c.Fetch(ctx, ports.SourceRef{Kind: "accounting", ID: "reports/ProfitAndLoss/2024-01"}); err == nil {
		t.Fatal("a report outside the 12 months was fetched")
	}
	if _, err := c.Fetch(ctx, ports.SourceRef{Kind: "accounting", ID: "payroll/x"}); err == nil {
		t.Fatal("payroll was fetched")
	}
}

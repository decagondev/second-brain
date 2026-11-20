package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/quickbooks"
	"example.com/second-brain/internal/adapters/connectors/quickbooks/qbotest"
	"example.com/second-brain/internal/core/ports"
)

func TestLAW03_QuickBooksUnknownIsNeverAllow(t *testing.T) {
	w := NewQBOWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW01_QuickBooksReadersAreLive(t *testing.T) {
	Check(t, "LAW-01", 20, law01QBO)
}

func TestLAW07_QuickBooksNumbersNeverStored(t *testing.T) {
	Check(t, "LAW-07", 40, func(r *rand.Rand) error { return law07QBO(r, asIs) })
}

// leaky is the mistake LAW-07's QuickBooks test exists for: a connector that passes the
// account number through as an ordinary field.
type leaky struct {
	*quickbooks.Connector
	f *qbotest.Fake
}

func (l leaky) Fields() ([]ports.Field, bool) {
	fs, ok := l.Connector.Fields()
	return append(fs, ports.Field{Name: "AcctNum"}), ok
}

func (l leaky) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	d, err := l.Connector.Fetch(ctx, ref)
	if err == nil && d.Fields != nil {
		d.Fields["AcctNum"] = "NUM-ACCT-leaked"
	}
	return d, err
}

func TestSelf_QuickBooksMutationIsCaught(t *testing.T) {
	wrap := func(c *quickbooks.Connector, f *qbotest.Fake) ports.Connector { return leaky{c, f} }
	if !violates(40, func(r *rand.Rand) error { return law07QBO(r, wrap) }) {
		t.Error("LAW-07 (quickbooks): a connector that passes account numbers through was not caught")
	}
}

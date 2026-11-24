package localfs

import (
	"context"
	"os"
	"testing"
	"time"
)

type counts map[string]float64

func (c counts) Add(name, _ string, _ map[string]string, v float64) { c[name] += v }

func TestLedger(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	next := counts{}
	l := &Ledger{Root: t.TempDir(), Tenant: "acme-logistics", Next: next, Now: func() time.Time { return now }}
	l.Add("sb_model_tokens_total", "", map[string]string{"direction": "input", "provider": "anthropic", "model": "m1"}, 120)
	l.Add("sb_model_tokens_total", "", map[string]string{"direction": "output", "provider": "anthropic", "model": "m1"}, 30)
	if err := l.Spend().Add(context.Background(), "acme-logistics", "2027-03", 1.25); err != nil {
		t.Fatal(err)
	}
	l.Add("sb_drafts_total", "", map[string]string{"outcome": "used"}, 1)
	l.Add("sb_asks_total", "", nil, 1) // forwarded, not kept
	now = now.Add(8 * 24 * time.Hour)
	l.Add("sb_drafts_total", "", map[string]string{"outcome": "discarded"}, 1)
	if next["sb_model_tokens_total"] != 150 || next["sb_asks_total"] != 1 {
		t.Fatalf("not forwarded: %v", next)
	}
	ctx := context.Background()
	from, to := time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC), time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	in, out, err := l.Tokens(ctx, "acme-logistics", from, to)
	if err != nil || in != 120 || out != 30 {
		t.Fatalf("tokens %d %d %v", in, out, err)
	}
	used, discarded, err := l.Drafts(ctx, "acme-logistics", from, to)
	if err != nil || used != 1 || discarded != 0 {
		t.Fatalf("drafts %d %d %v", used, discarded, err)
	}
	us, err := l.Usage(ctx, "acme-logistics", from, to)
	if err != nil || len(us) != 1 || us[0].Provider != "anthropic" || us[0].TokensIn != 120 || us[0].TokensOut != 30 {
		t.Fatalf("usage %+v %v", us, err)
	}
	if usd, err := l.Spend().Spent(ctx, "acme-logistics", "2027-03"); err != nil || usd != 1.25 {
		t.Fatalf("spent %v %v", usd, err)
	}
	if in, _, _ := l.Tokens(ctx, "beta-freight", from, to); in != 0 {
		t.Fatal("another tenant's usage counted")
	}
	f, _ := os.OpenFile(l.file(), os.O_APPEND|os.O_WRONLY, 0o600)
	_, _ = f.WriteString("not json\n")
	_ = f.Close()
	if _, _, err := l.Tokens(ctx, "acme-logistics", from, to); err == nil {
		t.Fatal("a damaged ledger must be an error, never a short count")
	}
}

// SB-1608: only the first claim of a key wins, across ledgers over the same directory.
func TestFirstClaimWins(t *testing.T) {
	dir := t.TempDir()
	a, b := &Ledger{Root: dir}, &Ledger{Root: dir}
	ctx := context.Background()
	if first, err := a.First(ctx, "budget-warning-acme-logistics-2027-03"); err != nil || !first {
		t.Fatalf("first claim: %v %v", first, err)
	}
	if first, _ := b.First(ctx, "budget-warning-acme-logistics-2027-03"); first {
		t.Fatal("a second ledger won the same claim")
	}
	if _, err := a.First(ctx, "../x"); err == nil {
		t.Fatal("an unsafe key was claimed")
	}
}

// SB-4105: spend by space is kept apart from the tenant's, by month.
func TestSpendBySpace(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	l := &Ledger{Root: t.TempDir(), Tenant: "acme-logistics", Now: func() time.Time { return now }}
	ctx := context.Background()
	for _, e := range []struct {
		space, month string
		usd          float64
	}{{"spc_finance", "2027-03", 1}, {"spc_finance", "2027-03", 0.5}, {"spc_ap", "2027-03", 2}, {"spc_ap", "2027-02", 9}} {
		if err := l.Spend().AddSpace(ctx, "acme-logistics", e.space, e.month, e.usd); err != nil {
			t.Fatal(err)
		}
	}
	by, err := l.Spend().SpentBySpace(ctx, "acme-logistics", "2027-03")
	if err != nil || len(by) != 2 || by["spc_finance"] != 1.5 || by["spc_ap"] != 2 {
		t.Fatalf("by space %v %v", by, err)
	}
	if usd, _ := l.Spend().Spent(ctx, "acme-logistics", "2027-03"); usd != 0 {
		t.Fatalf("space spend counted as the tenant's: %v", usd)
	}
}

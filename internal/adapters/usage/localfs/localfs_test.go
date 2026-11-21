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

package obs

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// The dashboard queries only catalogued metrics (SB-607).
func TestDashboardUsesTheCatalog(t *testing.T) {
	b, err := os.ReadFile("../../ops/dashboards/tenant.json")
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Panels []struct {
			Targets []struct{ Expr string } `json:"targets"`
		} `json:"panels"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`\b((?:sb|law)_[a-z_]+)`)
	n := 0
	for _, p := range d.Panels {
		for _, tg := range p.Targets {
			for _, m := range re.FindAllString(tg.Expr, -1) {
				n++
				name := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(m, "_bucket"), "_sum"), "_count")
				if _, ok := Catalog[name]; !ok {
					t.Errorf("dashboard queries %s, which no cell emits", m)
				}
			}
		}
	}
	if n < 7 {
		t.Fatalf("only %d metric references", n)
	}
}

func TestObserve(t *testing.T) {
	m := NewMetrics()
	m.Observe("sb_ask_seconds", "latency", map[string]string{"tenant": "acme-logistics"}, 0.3)
	var b strings.Builder
	_ = m.Write(&b)
	out := b.String()
	for _, want := range []string{`sb_ask_seconds_bucket{le="0.5",tenant="acme-logistics"} 1`, `sb_ask_seconds_bucket{le="0.25",tenant="acme-logistics"}`, `sb_ask_seconds_count{tenant="acme-logistics"} 1`} {
		if want == `sb_ask_seconds_bucket{le="0.25",tenant="acme-logistics"}` {
			if strings.Contains(out, want) {
				t.Errorf("0.3 counted in the 0.25 bucket:\n%s", out)
			}
			continue
		}
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
}

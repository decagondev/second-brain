package obs

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFreshnessMetrics(t *testing.T) {
	m := NewMetrics()
	f := &Freshness{M: m}
	at := time.Unix(1000, 0)
	f.SyncSucceeded("acme-logistics", "docs", at)
	f.SyncFailed("acme-logistics", "chat", nil)
	f.Update(1090)
	labels := map[string]string{"tenant": "acme-logistics", "connector": "docs"}
	if v, ok := m.Get("sb_ingest_last_success_age_seconds", labels); !ok || v != 90 {
		t.Fatalf("age %v %v", v, ok)
	}
	var b bytes.Buffer
	if err := m.Write(&b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"# TYPE sb_ingest_last_success_age_seconds gauge", `sb_ingest_sync_failures_total{connector="chat",tenant="acme-logistics"} 1`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

// Label values can't carry content (sb-observability(7), Telemetry labels; SB-607).
func TestLabelsCantCarryContent(t *testing.T) {
	m := NewMetrics()
	m.Add("sb_asks_total", "asks", map[string]string{"tenant": "acme-logistics", "q": "what is the CEO's salary?"}, 1)
	var b strings.Builder
	_ = m.Write(&b)
	out := b.String()
	if strings.Contains(out, "salary") || !strings.Contains(out, `q="invalid"`) || !strings.Contains(out, "sb_telemetry_rejected_total 1") {
		t.Fatalf("%s", out)
	}
	m.Set("sb_release_lag_max", "lag", map[string]string{"tenant": "acme-logistics", "cite": "docs:a@1"}, 1)
	if v, ok := m.Get("sb_release_lag_max", map[string]string{"tenant": "acme-logistics", "cite": "docs:a@1"}); !ok || v != 1 {
		t.Fatal("an id-shaped label was rejected")
	}
}

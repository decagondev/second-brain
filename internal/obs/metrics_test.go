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
	f.SyncFailed("acme-logistics", "chat")
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

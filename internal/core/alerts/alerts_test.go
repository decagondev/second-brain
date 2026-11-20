package alerts

import (
	"context"
	"example.com/second-brain/internal/core/canary"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Every law with a Watch line in sb-laws(7) has a rule, and every rule has a runbook.
func TestEveryWatchedLawHasARuleAndARunbook(t *testing.T) {
	src, err := os.ReadFile("../../../manual/sb-laws.7.md")
	if err != nil {
		t.Fatal(err)
	}
	sections := regexp.MustCompile(`(?m)^### (LAW-\d+):`).FindAllStringSubmatchIndex(string(src), -1)
	for i, m := range sections {
		end := len(src)
		if i+1 < len(sections) {
			end = sections[i+1][0]
		}
		law := string(src[m[2]:m[3]])
		body := string(src[m[0]:end])
		if !strings.Contains(body, "Watch:") || strings.Contains(body, "Planned for W7") {
			continue
		}
		if _, ok := ForLaw(law); !ok {
			t.Errorf("%s has a production check but no alert rule", law)
		}
	}
	for _, r := range Rules {
		if _, err := os.Stat("../../../" + r.Runbook); err != nil {
			t.Errorf("%s: no runbook at %s", r.Law, r.Runbook)
		}
	}
}

// Each rule fires on its own signal and only then (the staging alert test, in process).
func TestEachRuleFires(t *testing.T) {
	for _, r := range Rules {
		prev := map[string]float64{r.Metric: 0}
		cur := map[string]float64{r.Metric: r.Above + 1}
		got := Evaluate("acme-logistics", prev, cur)
		if len(got) != 1 || got[0].Law != r.Law || got[0].Severity != "sev1" || got[0].Runbook != r.Runbook {
			t.Errorf("%s: %+v", r.Law, got)
		}
		if quiet := Evaluate("acme-logistics", cur, cur); r.Delta && len(quiet) != 0 {
			t.Errorf("%s fired without an increase", r.Law)
		}
	}
	if len(Evaluate("acme-logistics", nil, map[string]float64{"sb_unrelated": 9})) != 0 {
		t.Error("an unrelated series fired")
	}
}

// Every alert's metric is one a cell emits (obs.Catalog is checked against the dashboard).
func TestRulesUseKnownMetrics(t *testing.T) {
	known := []string{"law_violation_total", "sb_audit_write_failures_total", "sb_attestation_findings_total", "sb_secret_findings_total",
		"sb_release_lag_max", "sb_audit_verify_failures_total", "sb_deactivated_requests_total", "sb_drift_findings_total", "sb_unwrappable_keys", "sb_unlisted_access_paths"}
	for _, r := range Rules {
		name := strings.SplitN(r.Metric, "{", 2)[0]
		found := false
		for _, k := range known {
			found = found || k == name
		}
		if !found {
			t.Errorf("%s alerts on %s, which isn't catalogued", r.Law, name)
		}
	}
}

type sent struct{ got []canary.Alert }

func (s *sent) Alert(_ context.Context, a canary.Alert) error { s.got = append(s.got, a); return nil }

// SB-905: firing rules go to the alert sender as Sev-1, with law and runbook, never content.
func TestDispatch(t *testing.T) {
	s := &sent{}
	fired, err := Dispatch(context.Background(), "acme-logistics", map[string]float64{}, map[string]float64{"sb_unlisted_access_paths": 2, "sb_unrelated": 9}, s)
	if err != nil || len(fired) != 1 || len(s.got) != 1 || s.got[0].Severity != "sev1" || s.got[0].Ref != "LAW-17" || s.got[0].Where != "docs/runbooks/LAW-17.md" {
		t.Fatalf("%+v %+v %v", fired, s.got, err)
	}
}

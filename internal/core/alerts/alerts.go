// Package alerts is the Sev-1 rule for every law with a production check (sb-observability(7),
// Law alerts, exactly; SB-608). Rules read metric snapshots; they never see content.
package alerts

import (
	"sort"
	"strings"
)

// Rule is one law's alert.
type Rule struct {
	Law     string
	Metric  string // the series, as written by obs.Metrics ("name" or `name{k="v"}`)
	Above   float64
	Delta   bool // true: fires when the value increases; false: when it is above Above
	Runbook string
}

// Rules is the table in sb-observability(7), Law alerts, exactly.
var Rules = []Rule{
	{Law: "LAW-01", Metric: `law_violation_total{law="LAW-01"}`, Delta: true},
	{Law: "LAW-02", Metric: `law_violation_total{law="LAW-02"}`, Delta: true},
	{Law: "LAW-04", Metric: "sb_audit_write_failures_total", Delta: true},
	{Law: "LAW-06", Metric: "sb_attestation_findings_total", Delta: true},
	{Law: "LAW-07", Metric: "sb_secret_findings_total", Delta: true},
	{Law: "LAW-08", Metric: "sb_release_lag_max", Above: 1},
	{Law: "LAW-11", Metric: "sb_audit_verify_failures_total", Delta: true},
	{Law: "LAW-12", Metric: "sb_deactivated_requests_total", Delta: true},
	{Law: "LAW-16", Metric: "sb_drift_findings_total", Delta: true},
	{Law: "LAW-18", Metric: "sb_unwrappable_keys", Above: 0},
}

func init() {
	for i := range Rules {
		Rules[i].Runbook = "docs/runbooks/" + Rules[i].Law + ".md"
	}
}

// Alert is one firing: Sev-1, ids only.
type Alert struct {
	Severity string  `json:"severity"`
	Law      string  `json:"law"`
	Tenant   string  `json:"tenant"`
	Metric   string  `json:"metric"`
	Value    float64 `json:"value"`
	Runbook  string  `json:"runbook"`
}

// Evaluate compares two snapshots (series -> value) and returns every rule that fires.
func Evaluate(tenant string, prev, cur map[string]float64) []Alert {
	var out []Alert
	for _, r := range Rules {
		v, seen := cur[r.Metric]
		fire := false
		if r.Delta {
			fire = seen && v > prev[r.Metric]
		} else {
			fire = seen && v > r.Above
		}
		if fire {
			out = append(out, Alert{Severity: "sev1", Law: r.Law, Tenant: tenant, Metric: r.Metric, Value: v, Runbook: r.Runbook})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Law < out[j].Law })
	return out
}

// ForLaw returns a law's rule.
func ForLaw(law string) (Rule, bool) {
	for _, r := range Rules {
		if strings.EqualFold(r.Law, law) {
			return r, true
		}
	}
	return Rule{}, false
}

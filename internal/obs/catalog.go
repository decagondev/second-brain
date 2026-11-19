package obs

// Catalog is every metric a cell emits (sb-observability(7), Per-tenant dashboards). The
// dashboard and the law alerts may use only these, and a test checks both.
var Catalog = map[string]string{
	"sb_ask_seconds":                     "Ask latency, a histogram by tenant",
	"sb_asks_total":                      "Answers by tenant and outcome (ok, refused, error code)",
	"sb_acl_verdicts_total":              "Live ACL verdicts by tenant, connector and verdict",
	"sb_ingest_last_success_age_seconds": "Seconds since a connector's last good sync (freshness)",
	"sb_ingest_sync_failures_total":      "Failed syncs by connector",
	"sb_model_tokens_total":              "Model tokens by tenant, provider, model and direction",
	"sb_model_refusals_total":            "Model requests refused by the gateway, by reason",
	"sb_budget_refusals_total":           "Model calls refused by budget or rate limit",
	"sb_drafts_total":                    "Drafts by outcome",
	"sb_telemetry_rejected_total":        "Label values replaced because they could carry content",
	"law_violation_total":                "Production law violations by law",
	"sb_audit_write_failures_total":      "Audit writes that failed (LAW-04)",
	"sb_attestation_findings_total":      "Attestation findings (LAW-06)",
	"sb_secret_findings_total":           "Secret scanner findings (LAW-07)",
	"sb_release_lag_max":                 "Largest release lag among stable tenants (LAW-08)",
	"sb_audit_verify_failures_total":     "Failed audit chain verifications (LAW-11)",
	"sb_deactivated_requests_total":      "Requests from deactivated users (LAW-12)",
	"sb_drift_findings_total":            "Manifest drift findings (LAW-16)",
	"sb_unwrappable_keys":                "Data keys the root key can't unwrap (LAW-18)",
	"sb_unlisted_access_paths":           "Factory access paths not in the hand-off record, from sb factory access (LAW-17)",
}

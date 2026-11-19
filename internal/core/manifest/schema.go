package manifest

import "encoding/json"

// JSONSchema returns the manifest's JSON Schema (draft 2020-12), built from the same
// vocabulary and patterns the checker uses. It is published as
// examples/tenants/tenant.schema.json; a test fails if that file is stale.
//
// The schema helps editors. It can't express every rule (for example "a workflow needs
// an approval"), so `sb tenant check` stays the authority.
func JSONSchema() []byte {
	str := func(desc string, extra map[string]any) map[string]any {
		m := map[string]any{"type": "string", "description": desc}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	props := map[string]any{
		"tenant":      str("Tenant name: a DNS label. Immutable.", map[string]any{"pattern": dnsLabelRe}),
		"channel":     str("Release channel; versions are never pinned (ADR-0004).", map[string]any{"enum": Channels}),
		"idp":         str("OIDC issuer: oidc://ISSUER-HOST[/path].", map[string]any{"pattern": idpRe}),
		"region":      str("Region pin. Immutable.", map[string]any{"enum": Regions}),
		"pilot_group": str("If set, only this IdP group may sign in.", map[string]any{"pattern": groupRe}),
		"connectors": map[string]any{
			"type": "object", "description": "Connector kind to vendor. Read-only unless named in workflows.",
			"minProperties": 1, "propertyNames": map[string]any{"enum": ConnectorKinds},
			"additionalProperties": map[string]any{"type": "string", "pattern": vendorRe},
		},
		"agents": map[string]any{
			"type": "array", "description": "Enabled agents; ask is required.", "uniqueItems": true,
			"items": map[string]any{"enum": Agents}, "contains": map[string]any{"const": "ask"},
		},
		"workflows": map[string]any{
			"type": "array", "description": "At most one approval-gated workflow in v1.", "uniqueItems": true,
			"maxItems": MaxWorkflows, "items": map[string]any{"enum": keys(Workflows)}, "default": []string{},
		},
		"approval": map[string]any{
			"type": "object", "description": "Workflow to approving IdP group.",
			"propertyNames":        map[string]any{"enum": keys(Workflows)},
			"additionalProperties": map[string]any{"type": "string", "pattern": groupRe},
		},
		"retention_days": map[string]any{
			"type": "integer", "description": "Retention for traces, citations and drafts (not audit).",
			"minimum": MinRetentionDays, "maximum": MaxRetentionDays, "default": DefaultRetentionDays,
		},
		"budget_usd_month": map[string]any{
			"type": "integer", "description": "Monthly model spend cap in USD.",
			"minimum": MinBudgetUSDMonth, "default": DefaultBudgetUSD,
		},
		"exclude": map[string]any{
			"type": "object", "description": "Source paths or channels never ingested, by connector kind.",
			"propertyNames": map[string]any{"enum": ConnectorKinds}, "default": map[string]any{},
			"additionalProperties": map[string]any{"type": "array", "items": map[string]any{"type": "string", "minLength": 1}},
		},
	}
	var required []string
	for _, f := range fields {
		if f.required {
			required = append(required, f.name)
		}
	}
	s := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://example.com/second-brain/tenant.schema.json",
		"title":                "Second Brain tenant manifest (sb-tenant(5))",
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties":           props,
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic("manifest: schema does not marshal: " + err.Error()) // static data; a programmer error
	}
	return append(b, '\n')
}

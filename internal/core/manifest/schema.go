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
		"tenant":       str("Tenant name: a DNS label. Immutable.", map[string]any{"pattern": dnsLabelRe}),
		"channel":      str("Release channel; versions are never pinned (ADR-0004).", map[string]any{"enum": Channels}),
		"idp":          str("Identity provider: oidc://ISSUER-HOST[/path] or identity-platform://PROJECT/IDP_TENANT.", map[string]any{"pattern": idpRe}),
		"region":       str("Region pin. Immutable.", map[string]any{"enum": Regions}),
		"pilot_group":  str("If set, only this IdP group may sign in.", map[string]any{"pattern": groupRe}),
		"questions_to": str("Who answers company-wide questions nothing answered (SB-1704).", map[string]any{"pattern": groupRe}),
		"notes":        map[string]any{"type": "boolean", "default": false, "description": "Stewards may publish replies as notes into the docs: folder share (SB-1705)."},
		"digest":       str("The weekly digest's IANA time zone, or off (SB-2103). Default UTC.", nil),
		"language":     str("The web UI's default language (SB-2301). Default en.", map[string]any{"enum": Languages}),
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
		"duties": map[string]any{
			"type": "object", "description": "Segregation of duties per workflow: a second approver group above a threshold (sb-approvals(7)).",
			"propertyNames": map[string]any{"enum": keys(Workflows)},
			"additionalProperties": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"second", "above"},
				"properties": map[string]any{
					"second": map[string]any{"type": "string", "pattern": groupRe},
					"above":  map[string]any{"type": "integer", "minimum": 0},
				},
			},
		},
		"retention_days": map[string]any{
			"type": "integer", "description": "Retention for traces, citations and drafts (not audit).",
			"minimum": MinRetentionDays, "maximum": MaxRetentionDays, "default": DefaultRetentionDays,
		},
		"budget_usd_month": map[string]any{
			"type": "integer", "description": "Monthly model spend cap in USD.",
			"minimum": MinBudgetUSDMonth, "default": DefaultBudgetUSD,
		},
		"asks_per_minute": map[string]any{
			"type": "integer", "description": "Questions one person may ask in any minute; one more is ERATE (SB-2305).",
			"minimum": 1, "maximum": MaxAsksPerMinute, "default": DefaultAsksPerMinute,
		},
		"groups": map[string]any{
			"type": "object", "description": "Only with an identity-platform idp: group to member emails (SB-907).",
			"maxProperties": MaxGroups, "propertyNames": map[string]any{"pattern": groupRe},
			"additionalProperties": map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"type": "string", "pattern": emailRe}},
		},
		"spaces": map[string]any{
			"type": "object", "description": "Department spaces (sb-space(5)); they only narrow access.",
			"maxProperties": MaxSpaces, "propertyNames": map[string]any{"pattern": spaceNameRe, "not": map[string]any{"const": "company"}},
			"additionalProperties": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"members", "scope"},
				"properties": map[string]any{
					"template":          map[string]any{"enum": keys(Templates)},
					"parent":            map[string]any{"type": "string", "pattern": spaceNameRe},
					"members":           map[string]any{"type": "array", "minItems": 1, "uniqueItems": true, "items": map[string]any{"type": "string", "pattern": groupRe}},
					"stewards":          map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"type": "string", "pattern": groupRe}},
					"label":             map[string]any{"enum": Labels},
					"agents":            map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"enum": Agents}},
					"retention_days":    map[string]any{"type": "integer", "minimum": MinSpaceRetention, "maximum": MaxRetentionDays},
					"fields":            map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "array", "items": map[string]any{"type": "string", "pattern": fieldName.String()}}},
					"channels":          map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"type": "string", "pattern": channelName.String()}},
					"review_after_days": map[string]any{"type": "integer", "minimum": MinRetentionDays, "maximum": MaxRetentionDays, "default": DefaultReviewAfterDays},
					"glossary": map[string]any{"type": "object", "maxProperties": MaxGlossary,
						"propertyNames":        map[string]any{"minLength": 1, "maxLength": MaxTerm},
						"additionalProperties": map[string]any{"type": "string", "minLength": 1, "maxLength": MaxMeaning}},
					"start_here": map[string]any{"type": "array", "maxItems": MaxStartHere, "uniqueItems": true, "items": map[string]any{"type": "string", "pattern": startItem.String()}},
					"scope": map[string]any{
						"type": "object", "minProperties": 1, "propertyNames": map[string]any{"enum": ConnectorKinds},
						"additionalProperties": map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"type": "string", "minLength": 1}},
					},
				},
			},
		},
		"barriers": map[string]any{
			"type": "array", "description": "Information barriers: pairs of groups that never meet (sb-space(5)).", "maxItems": MaxBarriers,
			"items": map[string]any{"type": "array", "minItems": 2, "maxItems": 2, "uniqueItems": true, "items": map[string]any{"type": "string", "pattern": groupRe}},
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

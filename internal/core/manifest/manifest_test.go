package manifest

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite examples/tenants/tenant.schema.json")

const valid = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, tickets: jira }
agents: [ask]
`

func format(file string, ds []Diag) string {
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.Format(file) + "\n")
	}
	return b.String()
}

func TestValidMinimal(t *testing.T) {
	got, ds := Check([]byte(valid))
	if len(ds) > 0 {
		t.Fatalf("valid manifest rejected:\n%s", format("m.yaml", ds))
	}
	want := Tenant{
		Tenant: "acme-logistics", Channel: "stable", IdP: "oidc://login.acme.example", Region: "eu",
		Connectors: map[string]string{"docs": "sharepoint", "tickets": "jira"}, Agents: []string{"ask"},
		RetentionDays: DefaultRetentionDays, BudgetUSDMonth: DefaultBudgetUSD, AsksPerMinute: DefaultAsksPerMinute,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

// Block style, comments, quotes and a leading --- mean the same thing as flow style.
func TestBlockStyleEqualsFlowStyle(t *testing.T) {
	src := `--- # the only document
tenant: "acme-logistics"   # quoted is fine
channel: stable
idp: 'oidc://login.acme.example'
region: eu
connectors:
  docs: sharepoint
  tickets: jira
agents:
- ask
`
	a, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	b, _ := Check([]byte(valid))
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("block %+v != flow %+v", a, b)
	}
}

// Every error message, once, at its exact position. One case per message (golden).
func TestEveryErrorMessage(t *testing.T) {
	cases := []struct{ name, src, want string }{
		// Syntax (the first syntax error stops the check).
		{"empty", "# nothing\n", "m.yaml:1:1: syntax: the manifest is empty"},
		{"not a map", "- a\n", "m.yaml:1:1: syntax: the top level must be fields (`field: value`)"},
		{"tab", "tenant: a\n\tchannel: b\n", "m.yaml:2:1: syntax: tabs are not allowed in indentation"},
		{"two docs", "tenant: a\n---\nchannel: b\n", "m.yaml:2:1: syntax: only one YAML document is allowed"},
		{"directive", "%YAML 1.2\ntenant: a\n", "m.yaml:1:1: syntax: YAML directives are not allowed"},
		{"indented start", "  tenant: a\n", "m.yaml:1:3: syntax: the first field must start at column 1"},
		{"bad indent", "connectors:\n    docs: a\n  chat: b\n", "m.yaml:3:3: syntax: unexpected indentation"},
		{"multi-line", "tenant: a\n  b\n", "m.yaml:2:3: syntax: unexpected indentation; values must fit on one line"},
		{"item among fields", "tenant: a\n- b\n", "m.yaml:2:1: syntax: a list item can't appear among fields; indent it under a field"},
		{"no colon", "tenant\n", "m.yaml:1:1: syntax: expected `field: value`"},
		{"empty item", "agents:\n  -\n", "m.yaml:2:3: syntax: a list item needs a value on the same line"},
		{"item is field", "agents:\n  - a: b\n", "m.yaml:2:5: syntax: list items must be single values, not fields"},
		{"unclosed quote", "tenant: \"abc\n", "m.yaml:1:9: syntax: a quoted value must close on the same line"},
		{"bad escape", `tenant: "a\qb"` + "\n", `m.yaml:1:11: syntax: unknown escape; use \" \\ \/ \n \t or \r`},
		{"anchor", "tenant: &a x\n", "m.yaml:1:9: syntax: anchors, aliases and tags are not allowed"},
		{"block scalar", "tenant: |\n", "m.yaml:1:9: syntax: block scalars are not allowed; put the value on one line in quotes"},
		{"reserved", "tenant: @x\n", "m.yaml:1:9: syntax: values can't start with @; quote the value"},
		{"colon in value", "tenant: a: b\n", `m.yaml:1:10: syntax: a value can't contain ": "; quote the value`},
		{"flow key", "[a]: b\n", "m.yaml:1:1: syntax: field names must be plain words"},
		{"unexpected", "tenant: ]\n", "m.yaml:1:9: syntax: unexpected ]"},
		{"after value", "agents: [ask] x\n", "m.yaml:1:15: syntax: unexpected text after the value"},
		{"unclosed list", "agents: [ask\n", "m.yaml:1:9: syntax: a [ list must close with ] on the same line"},
		{"list sep", "agents: [ask: x]\n", "m.yaml:1:13: syntax: expected , or ] in the list"},
		{"empty list item", "agents: [ask, , x]\n", "m.yaml:1:15: syntax: empty list item"},
		{"unclosed map", "connectors: { docs: a\n", "m.yaml:1:13: syntax: a { map must close with } on the same line"},
		{"map key", "connectors: { : a }\n", "m.yaml:1:15: syntax: expected a field name in the map"},
		{"map colon", "connectors: { docs }\n", "m.yaml:1:20: syntax: expected : after the field name"},
		{"map sep", "connectors: { docs: a b: c }\n", "m.yaml:1:24: syntax: expected , or } in the map"},
		{"not utf8", "tenant: \xff\n", "m.yaml:1:1: syntax: the file is not valid UTF-8"},

		// Field rules.
		{"unknown", valid + "colour: blue\n", "m.yaml:7:1: colour: unknown field; see sb-tenant(5) FIELDS"},
		{"missing", strings.Replace(valid, "agents: [ask]\n", "", 1), "m.yaml:1:1: agents: required field is missing"},
		{"duplicate", valid + "region: us\n", "m.yaml:7:1: region: duplicate field; first set on line 4"},
		{"version", valid + "platform: 2\n", "m.yaml:7:1: platform: " + msgVersion},
		{"secret nested", strings.Replace(valid, "tickets: jira", "token: x", 1), "m.yaml:5:33: connectors.token: " + msgSecret},
		{"custody", valid + "key_custodian: tenant\n", "m.yaml:7:1: key_custodian: " + msgCustody},
		{"empty value", strings.Replace(valid, "region: eu", "region:", 1), "m.yaml:4:1: region: needs a value"},
		{"not scalar", strings.Replace(valid, "region: eu", "region: [eu]", 1), "m.yaml:4:9: region: must be a single value"},
		{"dns", strings.Replace(valid, "acme-logistics", "-acme", 1), `m.yaml:1:9: tenant: "-acme" must be a DNS label ([a-z0-9-], 3-40 chars)`},
		{"channel", strings.Replace(valid, "stable", "beta", 1), `m.yaml:2:10: channel: "beta" must be one of stable, testing`},
		{"region", strings.Replace(valid, "region: eu", "region: mars", 1), `m.yaml:4:9: region: "mars" must be one of eu, us`},
		{"idp", strings.Replace(valid, "oidc://login.acme.example", "https://login.acme.example", 1), `m.yaml:3:6: idp: "https://login.acme.example" must be oidc://ISSUER-HOST[/path] or identity-platform://PROJECT/IDP_TENANT`},
		{"connectors type", strings.Replace(valid, "{ docs: sharepoint, tickets: jira }", "sharepoint", 1), "m.yaml:5:13: connectors: must be a map, e.g. { docs: sharepoint }"},
		{"connectors empty", strings.Replace(valid, "{ docs: sharepoint, tickets: jira }", "{}", 1), "m.yaml:5:13: connectors: needs at least one connector, e.g. { docs: sharepoint }"},
		{"connector kind", strings.Replace(valid, "tickets: jira", "wiki: x", 1), "m.yaml:5:33: connectors.wiki: unknown connector kind; use docs, chat, tickets, crm, code, hris or accounting"},
		{"vendor", strings.Replace(valid, "jira", "Jira", 1), `m.yaml:5:42: connectors.tickets: "Jira" must be a vendor name ([a-z0-9-])`},
		{"agents type", strings.Replace(valid, "[ask]", "ask", 1), "m.yaml:6:9: agents: must be a list, e.g. [ask]"},
		{"agent", strings.Replace(valid, "[ask]", "[ask, chat]", 1), `m.yaml:6:15: agents[1]: "chat" is not an agent; use ask, draft-reply, incident-summary, spec-digest, close-checklist, runbook-answer, policy-answer, clause-finder, account-brief, case-brief, brand-check, sop-steps, design-rationale or exec-brief`},
		{"agent twice", strings.Replace(valid, "[ask]", "[ask, ask]", 1), `m.yaml:6:15: agents[1]: "ask" is listed twice`},
		{"ask", strings.Replace(valid, "[ask]", "[draft-reply]", 1), `m.yaml:6:9: agents: "ask" is required`},
		{"workflows type", valid + "workflows: ticket-update\napproval: { ticket-update: group:a }\n", "m.yaml:7:12: workflows: must be a list, e.g. [ticket-update]"},
		{"workflow count", valid + "workflows: [ticket-update, ticket-update]\napproval: { ticket-update: group:a }\n", "m.yaml:7:12: workflows: at most 1 workflow in v1"},
		{"workflow", valid + "workflows: [send-mail]\n", `m.yaml:7:13: workflows[0]: "send-mail" is not a known workflow; use ticket-update`},
		{"workflow approval", valid + "workflows: [ticket-update]\n", `m.yaml:7:13: workflows[0]: "ticket-update" needs an approval rule; add approval: { ticket-update: group:<name> }`},
		{"workflow connector", strings.Replace(valid, ", tickets: jira", "", 1) + "workflows: [ticket-update]\napproval: { ticket-update: group:a }\n", `m.yaml:7:13: workflows[0]: "ticket-update" needs connector kind "tickets" in connectors`},
		{"approval type", valid + "approval: group:a\n", "m.yaml:7:11: approval: must be a map, e.g. { ticket-update: group:it-leads }"},
		{"approval orphan", valid + "approval: { ticket-update: group:a }\n", `m.yaml:7:13: approval.ticket-update: no workflow "ticket-update" in workflows`},
		{"approval user", valid + "workflows: [ticket-update]\napproval: { ticket-update: user:ann }\n", `m.yaml:8:28: approval.ticket-update: "user:ann" must be a group (group:NAME), never a user`},
		{"int", valid + "retention_days: \"365\"\n", `m.yaml:7:17: retention_days: "365" must be a whole number`},
		{"range", valid + "retention_days: 10\n", "m.yaml:7:17: retention_days: 10 must be between 30 and 3650"},
		{"digest zone", valid + "digest: Mars/Olympus\n", "m.yaml:7:9: digest: must be off or an IANA time zone such as Europe/Berlin"},
		{"digest local", valid + "digest: Local\n", "m.yaml:7:9: digest: must be off or an IANA time zone such as Europe/Berlin"},
		{"min", valid + "budget_usd_month: 0\n", "m.yaml:7:19: budget_usd_month: 0 must be at least 1"},
		{"exclude type", valid + "exclude: [a]\n", `m.yaml:7:10: exclude: must be a map, e.g. { chat: ["#hr-*"] }`},
		{"exclude kind", valid + "exclude: { wiki: [a] }\n", "m.yaml:7:12: exclude.wiki: unknown connector kind; use docs, chat, tickets, crm, code, hris or accounting"},
		{"exclude absent", valid + "exclude: { chat: [a] }\n", `m.yaml:7:12: exclude.chat: no connector kind "chat" in connectors`},
		{"exclude list", valid + "exclude: { docs: a }\n", `m.yaml:7:18: exclude.docs: must be a list, e.g. ["#hr-*"]`},
		{"exclude item", valid + "exclude: { docs: [\"\", x] }\n", "m.yaml:7:19: exclude.docs[0]: needs a value"},
		{"list item type", valid + "exclude: { docs: [[a]] }\n", "m.yaml:7:19: exclude.docs[0]: must be a single value"},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if seen[c.name] {
				t.Fatal("duplicate case name")
			}
			seen[c.name] = true
			got, ds := Check([]byte(c.src))
			out := strings.TrimSuffix(format("m.yaml", ds), "\n")
			if out != c.want {
				t.Fatalf("source:\n%s\ngot:  %s\nwant: %s", c.src, out, c.want)
			}
			if len(ds) > 0 && !reflect.DeepEqual(got, Tenant{}) {
				t.Fatal("an invalid manifest must not yield a Tenant")
			}
		})
	}
}

// examples/tenants: valid files pass, invalid.yaml prints exactly invalid.expected.
func TestExamples(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "examples", "tenants")
	for _, f := range []string{"pilot.yaml", "full.yaml", "spaces.yaml"} {
		src, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		if _, ds := Check(src); len(ds) > 0 {
			t.Errorf("%s rejected:\n%s", f, format(f, ds))
		}
	}
	src, err := os.ReadFile(filepath.Join(dir, "invalid.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(dir, "invalid.expected"))
	if err != nil {
		t.Fatal(err)
	}
	_, ds := Check(src)
	got := format("invalid.yaml", ds) + Count(len(ds)) + "\n"
	if got != string(want) {
		t.Fatalf("invalid.yaml:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// The field table, the Tenant type and the schema have the same fields, in order.
func TestOneSourceForTypesAndSchema(t *testing.T) {
	var tags, names []string
	ty := reflect.TypeOf(Tenant{})
	for i := 0; i < ty.NumField(); i++ {
		if tag := ty.Field(i).Tag.Get("json"); tag != "-" { // "-": read from another field's form (agents' caps)
			tags = append(tags, tag)
		}
	}
	for _, f := range fields {
		names = append(names, f.name)
	}
	if !reflect.DeepEqual(tags, names) {
		t.Fatalf("Tenant json tags %v != field table %v", tags, names)
	}
	var props []string
	for _, line := range strings.Split(string(JSONSchema()), "\n") {
		if strings.HasPrefix(line, `    "`) && strings.HasSuffix(line, "{") {
			props = append(props, strings.SplitN(strings.TrimSpace(line), `"`, 3)[1])
		}
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(props, sorted) {
		t.Fatalf("schema properties %v != fields %v", props, sorted)
	}
}

func TestSchemaPublished(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "tenants", "tenant.schema.json")
	got := JSONSchema()
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run: go test ./internal/core/manifest -run TestSchemaPublished -update)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("examples/tenants/tenant.schema.json is stale; run: go test ./internal/core/manifest -run TestSchemaPublished -update")
	}
}

const ipValid = `tenant: delta-tools
channel: stable
idp: identity-platform://sb-idp-delta/delta-tools-x7k2
region: eu
connectors: { docs: sharepoint }
agents: [ask]
groups:
  "group:finance": [ana@delta-tools.example, Bo@delta-tools.example]
  "group:support": [cy@delta-tools.example]
`

func TestIdentityPlatformAndGroups(t *testing.T) {
	got, ds := Check([]byte(ipValid))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	if got.Groups["group:finance"][1] != "bo@delta-tools.example" || len(got.Groups) != 2 {
		t.Fatalf("groups = %v", got.Groups)
	}
	if back, ds := Check(Format(got)); len(ds) > 0 || !reflect.DeepEqual(back, got) {
		t.Fatalf("round trip: %v\n%s", ds, Format(got))
	}
	p, it, ok := IdentityPlatform(got.IdP)
	if !ok || p != "sb-idp-delta" || it != "delta-tools-x7k2" {
		t.Fatalf("IdentityPlatform = %q %q %v", p, it, ok)
	}
	if _, _, ok := IdentityPlatform("oidc://login.acme.example"); ok {
		t.Fatal("an oidc idp is not identity-platform")
	}
	for _, tc := range []struct{ name, src, want string }{
		{"groups with oidc", strings.Replace(ipValid, "identity-platform://sb-idp-delta/delta-tools-x7k2", "oidc://login.delta.example", 1), "only with an identity-platform:// idp"},
		{"not a group", strings.Replace(ipValid, `"group:support"`, "support", 1), `"support" must be a group`},
		{"not an email", strings.Replace(ipValid, "cy@delta-tools.example", "cy", 1), `"cy" must be an email`},
		{"twice", strings.Replace(ipValid, "Bo@", "ana@", 1), "is listed twice"},
		{"no tenant part", strings.Replace(ipValid, "/delta-tools-x7k2", "", 1), "identity-platform://PROJECT/IDP_TENANT"},
	} {
		_, ds := Check([]byte(tc.src))
		if !strings.Contains(format("m.yaml", ds), tc.want) {
			t.Errorf("%s: want %q, got:\n%s", tc.name, tc.want, format("m.yaml", ds))
		}
	}
	var b strings.Builder
	b.WriteString(strings.SplitN(ipValid, "groups:", 2)[0] + "groups:\n")
	for i := 0; i < MaxGroups+1; i++ {
		fmt.Fprintf(&b, "  \"group:g%d\": [a@d.example]\n", i)
	}
	if _, ds := Check([]byte(b.String())); !strings.Contains(format("m.yaml", ds), "at most 200 groups") {
		t.Errorf("201 groups accepted: %s", format("m.yaml", ds))
	}
}

// SB-2103: the digest's zone.
func TestDigestZone(t *testing.T) {
	for _, v := range []string{"off", "UTC", "Europe/Berlin", "America/New_York"} {
		m, diags := Check([]byte(valid + "digest: " + v + "\n"))
		if len(diags) > 0 || m.Digest != v {
			t.Errorf("%s: %v %q", v, diags, m.Digest)
		}
	}
	if z, _ := DigestZone("off"); z != nil {
		t.Error("off has a zone")
	}
	if z, _ := DigestZone(""); z != time.UTC {
		t.Error("the default isn't UTC")
	}
}

// SB-2301: the web UI's default language.
func TestLanguage(t *testing.T) {
	for _, v := range Languages {
		m, diags := Check([]byte(valid + "language: " + v + "\n"))
		if len(diags) > 0 || m.Language != v {
			t.Errorf("%s: %v %q", v, diags, m.Language)
		}
		if again, ds := Check(Format(m)); len(ds) > 0 || again.Language != v {
			t.Errorf("%s doesn't read back after Format: %v", v, ds)
		}
	}
	for _, v := range []string{"es", "EN", "german"} {
		if _, diags := Check([]byte(valid + "language: " + v + "\n")); !strings.Contains(format("m.yaml", diags), "must be one of en, de, fr") {
			t.Errorf("language %s accepted: %s", v, format("m.yaml", diags))
		}
	}
}

// SB-2305: asks_per_minute is 6 unless the manifest says, from 1 to 600, and reads back.
func TestAsksPerMinute(t *testing.T) {
	m, diags := Check([]byte(valid))
	if len(diags) > 0 || m.AsksPerMinute != DefaultAsksPerMinute {
		t.Fatalf("default: %v %d", diags, m.AsksPerMinute)
	}
	m, diags = Check([]byte(valid + "asks_per_minute: 20\n"))
	if again, ds := Check(Format(m)); len(diags) > 0 || len(ds) > 0 || again.AsksPerMinute != 20 {
		t.Fatalf("20: %v %v %d", diags, ds, again.AsksPerMinute)
	}
	for _, v := range []string{"0", "601", "six", `"6"`} {
		if _, diags := Check([]byte(valid + "asks_per_minute: " + v + "\n")); len(diags) == 0 {
			t.Errorf("asks_per_minute %s accepted", v)
		}
	}
}

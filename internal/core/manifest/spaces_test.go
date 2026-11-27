package manifest

import (
	"reflect"
	"strings"
	"testing"
)

const spacesValid = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack, tickets: jira }
agents: [ask, draft-reply]
retention_days: 730
spaces:
  engineering:
    template: engineering
    members: [group:eng, group:product]
    stewards: [group:eng-leads]
    scope:
      docs: ["/Engineering/**", "/Specs/**"]
      chat: ["#eng-*"]
      tickets: [ENG, OPS]
  people:
    template: people
    members: [group:people-team]
    stewards: [group:people-leads]
    scope: { docs: ["/People/**"] }
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    label: restricted
    retention_days: 365
    fields: { docs: [salary_band] }
    scope:
      docs: ["/Finance/**"]
barriers:
  - [group:deal-team-orion, group:everyone]
`

func TestSpacesValid(t *testing.T) {
	got, ds := Check([]byte(spacesValid))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	want := map[string]Space{
		"engineering": {Template: "engineering", Members: []string{"group:eng", "group:product"}, Stewards: []string{"group:eng-leads"},
			Label: "internal", Scope: map[string][]string{"docs": {"/Engineering/**", "/Specs/**"}, "chat": {"#eng-*"}, "tickets": {"ENG", "OPS"}},
			Agents: []string{"ask"}, RetentionDays: 730, Channels: []string{}, Fields: map[string][]string{}},
		"people": {Template: "people", Members: []string{"group:people-team"}, Stewards: []string{"group:people-leads"},
			Label: "restricted", Scope: map[string][]string{"docs": {"/People/**"}}, Agents: []string{"ask"}, RetentionDays: 365, Channels: []string{}, Fields: map[string][]string{}},
		"finance": {Template: "finance", Members: []string{"group:finance"}, Stewards: []string{"group:finance-leads"},
			Label: "restricted", Scope: map[string][]string{"docs": {"/Finance/**"}}, Agents: []string{"ask"}, RetentionDays: 365, Channels: []string{}, Fields: map[string][]string{"docs": {"salary_band"}}},
	}
	if !reflect.DeepEqual(got.Spaces, want) {
		t.Fatalf("spaces:\n%+v\nwant\n%+v", got.Spaces, want)
	}
	if !reflect.DeepEqual(got.Barriers, [][2]string{{"group:deal-team-orion", "group:everyone"}}) {
		t.Fatalf("barriers = %v", got.Barriers)
	}
	back, ds := Check(Format(got))
	if len(ds) > 0 || !reflect.DeepEqual(back, got) {
		t.Fatalf("round trip: %v\n%s", ds, Format(got))
	}
}

func TestSpacesTemplateDefaults(t *testing.T) {
	src := strings.Replace(valid, "agents: [ask]", "agents: [ask, draft-reply]", 1) +
		"spaces:\n  sales:\n    template: sales\n    members: [group:sales]\n    scope: { docs: [\"/Sales/**\"] }\n" +
		"  misc:\n    members: [group:everyone]\n    scope: { tickets: [OPS] }\n"
	got, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	s := got.Spaces["sales"]
	if s.Label != "confidential" || !reflect.DeepEqual(s.Agents, []string{"ask", "draft-reply"}) || s.RetentionDays != DefaultRetentionDays || len(s.Stewards) != 0 {
		t.Fatalf("sales = %+v", s)
	}
	if m := got.Spaces["misc"]; m.Template != "custom" || m.Label != "internal" {
		t.Fatalf("misc = %+v", m)
	}
}

func TestSpaceRules(t *testing.T) {
	sp := func(body string) string { return valid + "spaces:\n  eng:\n" + body }
	base := "    members: [group:eng]\n    scope: { docs: [\"/Eng/**\"] }\n"
	for _, tc := range []struct{ name, src, want string }{
		{"unknown field", sp(base + "    colour: red\n"), "spaces.eng.colour: unknown field in a space"},
		{"no members", sp("    scope: { docs: [\"/Eng/**\"] }\n"), "spaces.eng.members: required field is missing"},
		{"no scope", sp("    members: [group:eng]\n"), "spaces.eng.scope: required field is missing"},
		{"scope kind without connector", sp("    members: [group:eng]\n    scope: { chat: [\"#eng\"] }\n"), `no connector kind "chat" in connectors`},
		{"unknown scope kind", sp("    members: [group:eng]\n    scope: { wiki: [x] }\n"), "unknown connector kind"},
		{"bad label", sp(base + "    label: secret-ish\n"), `"secret-ish"`},
		{"unknown template", sp(base + "    template: astrology\n"), `"astrology" is not a template`},
		{"retention above tenant", sp(base + "    retention_days: 400\n"), "above the tenant's retention_days (365)"},
		{"retention below 30", sp(base + "    retention_days: 7\n"), "spaces.eng.retention_days"},
		{"company", valid + "spaces:\n  company:\n" + base, `"company" is reserved`},
		{"bad name", valid + "spaces:\n  Eng:\n" + base, "a space name is"},
		{"member is a user", sp("    members: [ana]\n    scope: { docs: [\"/Eng/**\"] }\n"), `"ana" must be a group`},
		{"member twice", sp("    members: [group:eng, group:eng]\n    scope: { docs: [\"/Eng/**\"] }\n"), `"group:eng" is listed twice`},
		{"restricted without stewards", sp(base + "    label: restricted\n"), "a restricted space needs stewards"},
		{"restricted template without stewards", sp(base + "    template: legal\n"), "a restricted space needs stewards"},
		{"agent not enabled", sp(base + "    agents: [draft-reply]\n"), `"draft-reply" is not in the manifest's agents`},
		{"another department's agent", strings.Replace(sp(base+"    template: engineering\n    agents: [close-checklist]\n"), "agents: [ask]", "agents: [ask, close-checklist]", 1),
			`"close-checklist" is another department's agent`},
		{"barrier of one", valid + "barriers:\n  - [group:a]\n", "a barrier is exactly two groups"},
		{"barrier with itself", valid + "barriers:\n  - [group:a, group:a]\n", "walled off from itself"},
		{"barrier twice", valid + "barriers:\n  - [group:a, group:b]\n  - [group:b, group:a]\n", "the same barrier as barriers[0]"},
		{"barrier not a group", valid + "barriers:\n  - [group:a, b]\n", `"b" must be a group`},
		{"fields kind not in scope", sp(base + "    fields: { tickets: [salary] }\n"), `kind "tickets" isn't in this space's scope`},
		{"bad field name", sp(base + "    fields: { docs: [Salary] }\n"), `"Salary" isn't a field name`},
		{"bad channel", sp(base + "    channels: [finance]\n"), `"finance" must be a channel`},
		{"channel in two spaces", valid + "spaces:\n  eng:\n" + base + "    channels: [\"#ask\"]\n  ops:\n" + base + "    channels: [\"#ask\"]\n", `"#ask" already asks in space "eng"`},
		{"barrier inside a space", sp("    members: [group:a, group:b]\n    scope: { docs: [\"/Eng/**\"] }\n") + "barriers:\n  - [group:b, group:a]\n", `space "eng" has both group:a and group:b`},
	} {
		got, ds := Check([]byte(tc.src))
		if out := format("m.yaml", ds); !strings.Contains(out, tc.want) {
			t.Errorf("%s: want %q, got:\n%s", tc.name, tc.want, out)
		}
		if !reflect.DeepEqual(got, Tenant{}) {
			t.Errorf("%s: an invalid manifest returned a tenant", tc.name)
		}
	}
	var b strings.Builder
	b.WriteString(valid + "spaces:\n")
	for i := 0; i < MaxSpaces+1; i++ {
		b.WriteString("  s" + strings.Repeat("x", i%3+1) + itoa(i) + ":\n" + base)
	}
	if _, ds := Check([]byte(b.String())); !strings.Contains(format("m.yaml", ds), "at most 50 spaces") {
		t.Errorf("51 spaces accepted")
	}
}

func itoa(i int) string { return string(rune('a'+i/26)) + string(rune('a'+i%26)) }

// SB-6104: a glossary term's words in other languages read back as written; a word for a
// term the glossary doesn't have, or in a language that isn't one, is an error.
func TestGlossaryIn(t *testing.T) {
	add := func(s string) string {
		return strings.Replace(spacesValid, "    scope: { docs: [\"/People/**\"] }\n", "    scope: { docs: [\"/People/**\"] }\n    glossary: { PO: purchase order }\n"+s, 1)
	}
	got, ds := Check([]byte(add("    glossary_in: { PO: { de: Bestellung, fr: \"bon de commande\" } }\n")))
	if len(ds) > 0 {
		t.Fatal(format("m.yaml", ds))
	}
	gi := got.Spaces["people"].GlossaryIn
	if gi["PO"]["de"] != "Bestellung" || gi["PO"]["fr"] != "bon de commande" {
		t.Fatalf("glossary_in %v", gi)
	}
	back, ds := Check(Format(got))
	if len(ds) > 0 || !reflect.DeepEqual(back.Spaces["people"].GlossaryIn, gi) {
		t.Fatalf("format: %v %v", back.Spaces["people"].GlossaryIn, ds)
	}
	for in, want := range map[string]string{
		"    glossary_in: { SLA: { de: Vereinbarung } }\n": "isn't a term of the glossary",
		"    glossary_in: { PO: { xx: Bestellung } }\n":    "isn't a language code",
	} {
		if _, ds := Check([]byte(add(in))); !strings.Contains(format("m.yaml", ds), want) {
			t.Errorf("%s: want %q, got %s", in, want, format("m.yaml", ds))
		}
	}
}

// SB-2203: a space's glossary reads back as written, and its limits are errors.
func TestGlossary(t *testing.T) {
	src := strings.Replace(spacesValid, "    scope: { docs: [\"/People/**\"] }\n",
		"    scope: { docs: [\"/People/**\"] }\n    glossary: { PO: purchase order, \"SLA\": \"service level agreement, \\\"signed\\\"\", HRIS: Personalsystem für Mitarbeiterdaten }\n", 1)
	got, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatal(format("m.yaml", ds))
	}
	g := got.Spaces["people"].Glossary
	if g["PO"] != "purchase order" || g["SLA"] != `service level agreement, "signed"` || g["HRIS"] != "Personalsystem für Mitarbeiterdaten" {
		t.Fatalf("glossary %v", g)
	}
	back, ds := Check(Format(got))
	if len(ds) > 0 || !reflect.DeepEqual(back.Spaces["people"].Glossary, g) {
		t.Fatalf("format: %v %v\n%s", back.Spaces["people"].Glossary, ds, Format(got))
	}
	for _, tc := range []struct{ entry, want string }{
		{"PO: a, po: b", "listed twice"},
		{"\"" + strings.Repeat("x", MaxTerm+1) + "\": y", "a term is 1 to"},
		{"PO: \"" + strings.Repeat("y", MaxMeaning+1) + "\"", "a meaning is at most"},
		{"PO: [a]", "must be a single value"},
	} {
		bad := strings.Replace(spacesValid, "    scope: { docs: [\"/People/**\"] }\n", "    scope: { docs: [\"/People/**\"] }\n    glossary: { "+tc.entry+" }\n", 1)
		if _, ds := Check([]byte(bad)); !strings.Contains(format("m.yaml", ds), tc.want) {
			t.Errorf("%s: want %q, got %s", tc.entry, tc.want, format("m.yaml", ds))
		}
	}
}

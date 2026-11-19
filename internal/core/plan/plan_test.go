package plan

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

const first = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack }
agents: [ask]
`

const later = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack, tickets: jira }
agents: [ask, draft-reply]
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
budget_usd_month: 1500
`

func tenant(t *testing.T, src string) manifest.Tenant {
	t.Helper()
	m, ds := manifest.Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("manifest: %v", ds)
	}
	return m
}

// The two EXAMPLES in sb-tenant(1), byte for byte.
func TestManualExamples(t *testing.T) {
	a, b := tenant(t, first), tenant(t, later)
	p, err := Make(a, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `+ cell acme-logistics (region eu)
+ setting budget_usd_month = 1000
+ setting channel = stable
+ setting idp = oidc://login.acme.example
+ setting retention_days = 365
+ connector chat (slack, read-only)
+ connector docs (sharepoint, read-only)
+ agent ask
plan: 8 to add, 0 to change, 0 to remove
`
	if p.Text() != want {
		t.Fatalf("new cell:\n%s\nwant:\n%s", p.Text(), want)
	}
	p, err = Make(b, Desired(a), true)
	if err != nil {
		t.Fatal(err)
	}
	want = `~ setting budget_usd_month = 1500 (was 1000)
+ connector tickets (jira, read-write)
+ agent draft-reply
+ workflow ticket-update (approval group:it-leads)
plan: 3 to add, 1 to change, 0 to remove
`
	if p.Text() != want {
		t.Fatalf("later:\n%s\nwant:\n%s", p.Text(), want)
	}
}

func TestEveryLineShape(t *testing.T) {
	src := strings.Replace(later, "agents: [ask, draft-reply]", "agents: [ask]", 1) + "pilot_group: group:sb-pilot\nexclude: { chat: [\"#hr-*\", \"#legal-*\"] }\n"
	b := tenant(t, src)
	applied := Desired(tenant(t, strings.Replace(later, "jira", "zendesk", 1)))
	p, err := Make(b, applied, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `+ setting pilot_group = group:sb-pilot
~ connector tickets (jira, read-write; was zendesk, read-write)
+ exclude chat (#hr-*, #legal-*)
- agent draft-reply
plan: 2 to add, 1 to change, 1 to remove
`
	if p.Text() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", p.Text(), want)
	}
}

func TestNoChanges(t *testing.T) {
	a := tenant(t, later)
	p, err := Make(a, Desired(a), true)
	if err != nil || p.Text() != "plan: no changes\n" {
		t.Fatalf("%q %v", p.Text(), err)
	}
	if b, _ := json.Marshal(p); string(b) != `{"tenant":"acme-logistics","changes":[],"add":0,"change":0,"remove":0}` {
		t.Fatalf("json: %s", b)
	}
}

func TestRegionIsImmutable(t *testing.T) {
	a := tenant(t, first)
	_, err := Make(tenant(t, strings.Replace(first, "region: eu", "region: us", 1)), Desired(a), true)
	var ie *ImmutableError
	if !errors.As(err, &ie) || ie.Field != "region" {
		t.Fatalf("want ImmutableError, got %v", err)
	}
}

// Same inputs, same bytes: map iteration order must never leak into the plan.
func TestStableJSON(t *testing.T) {
	a, b := tenant(t, first), tenant(t, later)
	var outs []string
	for i := 0; i < 50; i++ {
		p, _ := Make(b, Desired(a), true)
		j, _ := json.Marshal(p)
		outs = append(outs, string(j))
	}
	for _, o := range outs[1:] {
		if o != outs[0] {
			t.Fatal("plan JSON is not stable")
		}
	}
	want := `{"tenant":"acme-logistics","changes":[{"op":"change","kind":"setting","name":"budget_usd_month","attrs":{"value":"1500"},"was":{"value":"1000"}},` +
		`{"op":"add","kind":"connector","name":"tickets","attrs":{"mode":"read-write","vendor":"jira"}},{"op":"add","kind":"agent","name":"draft-reply"},` +
		`{"op":"add","kind":"workflow","name":"ticket-update","attrs":{"approval":"group:it-leads"}}],"add":3,"change":1,"remove":0}`
	if outs[0] != want {
		t.Fatalf("json:\n%s\nwant:\n%s", outs[0], want)
	}
}

func TestApplyReachesDesired(t *testing.T) {
	a, b := tenant(t, first), tenant(t, later)
	p, _ := Make(b, Desired(a), true)
	if got := Apply(Desired(a), p); !reflect.DeepEqual(got, Desired(b)) {
		t.Fatalf("apply(plan) = %v\nwant %v", got, Desired(b))
	}
}

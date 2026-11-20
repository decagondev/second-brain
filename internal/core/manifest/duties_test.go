package manifest

import (
	"reflect"
	"strings"
	"testing"
)

const dutyBase = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, tickets: linear }
agents: [ask]
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
`

func TestDutiesParseAndRoundTrip(t *testing.T) {
	for _, src := range []string{
		dutyBase + "duties:\n  ticket-update: { second: group:finance-leads, above: 100000 }\n",
		dutyBase + "duties: { ticket-update: { second: group:finance-leads, above: 100000 } }\n",
	} {
		tn, ds := Check([]byte(src))
		if len(ds) > 0 {
			t.Fatalf("%v\n%s", ds, src)
		}
		want := Duty{Second: "group:finance-leads", Above: 100000}
		if tn.Duties["ticket-update"] != want {
			t.Fatalf("duty %+v", tn.Duties)
		}
		back, ds := Check(Format(tn))
		if len(ds) > 0 || !reflect.DeepEqual(back, tn) {
			t.Fatalf("round trip: %v\n%s", ds, Format(tn))
		}
	}
}

func TestDutyRules(t *testing.T) {
	for name, c := range map[string]struct{ src, want string }{
		"unknown workflow": {"duties:\n  other: { second: group:x, above: 0 }\n", "duties.other"},
		"same group":       {"duties:\n  ticket-update: { second: group:it-leads, above: 0 }\n", "must differ"},
		"a user":           {"duties:\n  ticket-update: { second: ana@acme.example, above: 0 }\n", "must be a group"},
		"negative":         {"duties:\n  ticket-update: { second: group:x, above: -1 }\n", "duties.ticket-update.above"},
		"missing above":    {"duties:\n  ticket-update: { second: group:x }\n", "needs above"},
		"unknown field":    {"duties:\n  ticket-update: { second: group:x, above: 0, third: group:y }\n", "unknown field"},
	} {
		_, ds := Check([]byte(dutyBase + c.src))
		found := false
		for _, d := range ds {
			found = found || strings.Contains(d.Field+" "+d.Msg, c.want)
		}
		if !found {
			t.Errorf("%s: want %q in %v", name, c.want, ds)
		}
	}
}

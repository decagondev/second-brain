package api

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/adapters/connectors/linear"
	"example.com/second-brain/internal/adapters/connectors/linear/lineartest"
	killmem "example.com/second-brain/internal/adapters/kill/memory"
	"example.com/second-brain/internal/adapters/policy/cedar"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/toolbus"
	"example.com/second-brain/internal/core/workflow"
)

// SB-410 end to end: the full example manifest's workflow becomes the bus's registry; a
// ticket comment is proposed over the API, approved by the manifest's approver group
// under the real cedar policies, posted to the Linear fake, and comes back Ran with
// Linear's receipt, with every step in the audit chain.
func TestTicketUpdateEndToEnd(t *testing.T) {
	src, err := os.ReadFile("../../examples/tenants/full.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m, ds := manifest.Check(src)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	tools, err := workflow.Registry(m)
	if err != nil || len(tools) != 1 || tools["ticket-update.comment"].Approvers != "it-leads" {
		t.Fatalf("registry %+v %v", tools, err)
	}
	pol := map[string][]byte{}
	for _, f := range []string{"../../policy/platform/base.cedar"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		pol["base.cedar"] = b
	}
	engine, err := cedar.Load(pol)
	if err != nil {
		t.Fatal(err)
	}

	lf := lineartest.New()
	defer lf.Server.Close()
	lf.AddTeam("TOPS", "OPS", false)
	lf.Put("iss_1", "TOPS", "Disk full on db-2", "Alerts fired at 02:10.")
	conn := &linear.Connector{Endpoint: lf.Endpoint(), HTTP: lf.Server.Client(), Key: func(context.Context) (string, error) { return lf.Key, nil }}

	now := time.Unix(1_800_000_000, 0)
	clock := func() time.Time { return now }
	ch := &kill.Checker{Tenant: m.Tenant, Store: killmem.New(), Now: clock}
	_ = ch.Refresh(context.Background())
	rows := auditmem.New()
	log := &audit.Log{Tenant: m.Tenant, Release: func() string { return "rel_e2e" }, Store: rows, Now: clock}
	n := 0
	bus := &toolbus.Bus{Tenant: m.Tenant, Tools: tools, Kill: ch, Policy: policy.Kernel{Engine: engine, Recorder: log}, Audit: log,
		Actions: actmem.New(), Receipts: actmem.NewReceipts(), Connectors: map[string]ports.Connector{"tickets": conn}, Now: clock,
		NewID: func(p string) string { n++; return p + "e2e" + string(rune('a'+n)) }}
	mux := httpMux(Actions{Bus: bus, Release: "rel_e2e"})

	code, a := call(t, mux, "usr_dana", nil, "POST", "/v1/actions", `{"workflow":"ticket-update","target":"tickets:OPS-1","payload":{"comment":"Root cause: disk full; rotated logs."}}`)
	if code != 200 || a["state"] != "PendingApproval" {
		t.Fatalf("propose: %d %v", code, a)
	}
	if code, _ := call(t, mux, "usr_dana", nil, "POST", "/v1/actions", `{"workflow":"ticket-update","target":"tickets:OPS-1","payload":{"comment":"x","assignee":"me"}}`); code != 400 {
		t.Errorf("extra payload field: %d", code)
	}
	id, hash := a["id"].(string), a["payload_sha256"].(string)
	if code, _ := call(t, mux, "usr_erin", []string{"finance"}, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+hash+`"}`); code != 409 {
		t.Errorf("approver outside it-leads: %d", code)
	}
	code, done := call(t, mux, "usr_erin", []string{"it-leads"}, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+hash+`","note":"ok"}`)
	if code != 200 || done["state"] != "Ran" || !strings.HasPrefix(done["receipt"].(string), "linear:comment/") {
		t.Fatalf("approve: %d %v", code, done)
	}
	if len(lf.Comments) != 1 {
		t.Fatalf("%d comments at Linear", len(lf.Comments))
	}
	for _, c := range lf.Comments {
		if c.Body != "Root cause: disk full; rotated logs." || c.IssueID != "iss_1" {
			t.Fatalf("comment %+v", c)
		}
	}
	all, _ := rows.Range(context.Background(), 1, 1000)
	var states []string
	for _, r := range all {
		if r.Kind == "tool_call" || r.Kind == "approval" {
			states = append(states, r.State)
		}
	}
	if got := strings.Join(states, " "); got != "PendingApproval Approved Executing Ran" {
		t.Fatalf("audit trail %q", got)
	}
}

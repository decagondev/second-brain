package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	killmem "example.com/second-brain/internal/adapters/kill/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

type permit struct{}

func (permit) Evaluate(_ context.Context, r policy.Request) (policy.Result, error) {
	return policy.Result{Decision: policy.Allow, Reason: r.Action + ": permitted by test.cedar#1"}, nil
}

// ticketTarget is a tickets connector that can read the current state and act.
type ticketTarget struct {
	ports.Connector
	acted int
}

func (t *ticketTarget) Current(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(`{"status":"open","comment":""}`), nil
}

func (t *ticketTarget) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	t.acted++
	return ports.Receipt{TargetID: "tickets:comment/9"}, nil
}

func actionsServer(t *testing.T) (*http.ServeMux, *ticketTarget, []toolbus.Notice) {
	now := time.Unix(1_800_000_000, 0)
	clock := func() time.Time { return now }
	ks := killmem.New()
	ch := &kill.Checker{Tenant: "acme-logistics", Store: ks, Now: clock}
	_ = ch.Refresh(context.Background())
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: auditmem.New(), Now: clock}
	tgt := &ticketTarget{}
	var notes []toolbus.Notice
	n := 0
	bus := &toolbus.Bus{Tenant: "acme-logistics", Kill: ch, Policy: policy.Kernel{Engine: permit{}, Recorder: log}, Audit: log,
		Actions: actmem.New(), Connectors: map[string]ports.Connector{"tickets": tgt}, Now: clock,
		NewID:  func(p string) string { n++; return p + strings.Repeat("0", 3) + string(rune('a'+n)) },
		Notify: func(_ context.Context, x toolbus.Notice) { notes = append(notes, x) },
		Tools:  map[string]toolbus.Tool{"update": {Name: "update", Connector: "tickets", Consequential: true, Workflow: "ticket-update", Approvers: "grp_it_leads"}}}
	mux := http.NewServeMux()
	Actions{Bus: bus, Release: "rel_x"}.Register(mux)
	return mux, tgt, notes
}

func call(t *testing.T, mux *http.ServeMux, user string, groups []string, method, path, body string) (int, map[string]any) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", user, groups)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(tenancy.WithScope(req.Context(), s))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestActionsFlow(t *testing.T) {
	mux, tgt, _ := actionsServer(t)
	code, a := call(t, mux, "usr_dana", nil, "POST", "/v1/actions", `{"workflow":"ticket-update","target":"tickets:OPS-1182","payload":{"comment":"Root cause: DNS"}}`)
	if code != 200 || a["state"] != "PendingApproval" || a["approvers"] != "group:grp_it_leads" || a["policy_reason"] == "" || a["expires_at"] == "" {
		t.Fatalf("propose: %d %v", code, a)
	}
	id, hash := a["id"].(string), a["payload_sha256"].(string)
	if code, _ := call(t, mux, "usr_dana", nil, "POST", "/v1/actions", `{"workflow":"payroll-run","target":"x","payload":{}}`); code != 403 {
		t.Errorf("disabled workflow: %d", code)
	}
	if code, _ := call(t, mux, "usr_mallory", nil, "GET", "/v1/actions/"+id, ""); code != 403 {
		t.Errorf("outsider read: %d", code)
	}
	code, v := call(t, mux, "usr_erin", []string{"grp_it_leads"}, "GET", "/v1/actions/"+id, "")
	diff, _ := v["diff"].([]any)
	if code != 200 || len(diff) != 2 { // comment changes, status disappears
		t.Fatalf("approver read: %d %v", code, v)
	}
	if code, _ := call(t, mux, "usr_dana", nil, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+hash+`"}`); code != 409 {
		t.Errorf("self-approval: %d", code)
	}
	if code, _ := call(t, mux, "usr_erin", []string{"grp_it_leads"}, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+strings.Repeat("a", 64)+`"}`); code != 409 || tgt.acted != 0 {
		t.Errorf("wrong hash: %d (acted %d)", code, tgt.acted)
	}
	code, done := call(t, mux, "usr_erin", []string{"grp_it_leads"}, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+hash+`","note":"ok"}`)
	if code != 200 || done["state"] != "Ran" || done["receipt"] != "tickets:comment/9" || done["approved_by"] != "usr_erin" || tgt.acted != 1 {
		t.Fatalf("approve: %d %v", code, done)
	}
	code, b := call(t, mux, "usr_dana", nil, "POST", "/v1/actions", `{"workflow":"ticket-update","target":"tickets:OPS-1183","payload":{"comment":"x"}}`)
	if code != 200 {
		t.Fatal(code)
	}
	if code, r := call(t, mux, "usr_erin", []string{"grp_it_leads"}, "POST", "/v1/actions/"+b["id"].(string)+"/reject", `{"note":""}`); code != 200 || r["state"] != "Rejected" {
		t.Errorf("reject: %d %v", code, r)
	}
	if code, _ := call(t, mux, "usr_dana", nil, "POST", "/v1/actions", `{"workflow":"ticket-update","target":"t","payload":{"n":1.5}}`); code != 400 {
		t.Errorf("float payload: %d", code)
	}
}

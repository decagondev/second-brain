package workflow

import (
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/manifest"
)

func TestRegistry(t *testing.T) {
	tools, err := Registry(manifest.Tenant{Workflows: []string{"ticket-update"}, Approval: map[string]string{"ticket-update": "group:it-leads"}})
	tool := tools["ticket-update.comment"]
	if err != nil || !tool.Consequential || tool.Connector != "tickets" || tool.Approvers != "it-leads" || tool.Workflow != "ticket-update" {
		t.Fatalf("%+v %v", tools, err)
	}
	if _, err := Registry(manifest.Tenant{Workflows: []string{"payroll"}}); !errors.Is(err, ErrUnknown) {
		t.Errorf("unknown workflow: %v", err)
	}
	if _, err := Registry(manifest.Tenant{Workflows: []string{"ticket-update"}}); err == nil {
		t.Error("a workflow without an approval group was accepted")
	}
	if tools, _ := Registry(manifest.Tenant{}); len(tools) != 0 {
		t.Error("no workflows should mean no tools")
	}
}

func TestValidateComment(t *testing.T) {
	v := Templates["ticket-update"].Validate
	if err := v([]byte(`{"comment":"ok"}`)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{}`, `{"comment":""}`, `{"comment":" "}`, `{"comment":"x","assignee":"me"}`, `{"comment":1}`, `[]`, `{"comment":"` + strings.Repeat("x", MaxComment+1) + `"}`} {
		if err := v([]byte(bad)); !errors.Is(err, approvals.ErrPayload) {
			t.Errorf("%.40s accepted", bad)
		}
	}
}

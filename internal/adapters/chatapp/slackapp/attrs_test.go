package slackapp

import (
	"context"
	"testing"
)

// SB-1401: a chat ask carries the directory's attributes, as a web ask does.
func TestChatScopeCarriesAttributes(t *testing.T) {
	dir := users{{ID: "alice", Tenant: tenant, Emails: []string{"alice@acme.example"}, Active: true, Department: "Finance", CostCenter: "CC-100"}}
	a, _ := newApp(dir, &agent{})
	s, err := a.Scope(context.Background(), "UALICE")
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Attrs(); got["department"] != "Finance" || got["cost_center"] != "CC-100" {
		t.Fatalf("attributes %v", got)
	}
}

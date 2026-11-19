package cedar

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
)

func base(t *testing.T) Engine {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "policy", "platform", "base.cedar"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := Load(map[string][]byte{"platform/base.cedar": b})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func scope(t *testing.T, tenant, user string, groups ...string) tenancy.Scope {
	t.Helper()
	s, err := tenancy.NewIssuer(tenant).Issue(tenant, user, groups)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The platform base policies through the real kernel.
func TestBasePolicies(t *testing.T) {
	e := base(t)
	ctx := context.Background()
	ann := scope(t, "acme-logistics", "ann", "it-leads")
	cases := []struct {
		name   string
		s      tenancy.Scope
		action string
		res    policy.Entity
		want   policy.Decision
	}{
		{"read own chunk", ann, "read", policy.Entity{Type: "Chunk", ID: "c1", Attrs: map[string]any{"cite": "docs:a@1"}}, policy.Allow},
		{"read other tenant's chunk", ann, "read", policy.Entity{Type: "Chunk", ID: "c2", Tenant: "beta-freight", Attrs: map[string]any{"cite": "docs:b@1"}}, policy.Deny},
		{"use ask", ann, "use_agent", policy.Entity{Type: "Agent", ID: "ask", Attrs: map[string]any{"name": "ask", "consequential": false}}, policy.Allow},
		{"use another agent", ann, "use_agent", policy.Entity{Type: "Agent", ID: "draft-reply", Attrs: map[string]any{"name": "draft-reply", "consequential": false}}, policy.Deny},
		{"use draft-reply where enabled", ann, "use_agent", policy.Entity{Type: "Agent", ID: "draft-reply", Attrs: map[string]any{"name": "draft-reply", "consequential": false, "enabled": true}}, policy.Allow},
		{"use draft-reply where disabled", ann, "use_agent", policy.Entity{Type: "Agent", ID: "draft-reply", Attrs: map[string]any{"name": "draft-reply", "consequential": false, "enabled": false}}, policy.Deny},
		{"approve own action", ann, "approve", policy.Entity{Type: "ProposedAction", ID: "a1",
			Attrs: map[string]any{"workflow": "ticket-update", "requester": policy.Ref{Type: "User", ID: "ann"}, "approvers": policy.Ref{Type: "Group", ID: "it-leads"}}}, policy.Deny},
		{"approve as the approver group", ann, "approve", policy.Entity{Type: "ProposedAction", ID: "a2",
			Attrs: map[string]any{"workflow": "ticket-update", "requester": policy.Ref{Type: "User", ID: "bob"}, "approvers": policy.Ref{Type: "Group", ID: "it-leads"}}}, policy.Allow},
		{"approve outside the approver group", ann, "approve", policy.Entity{Type: "ProposedAction", ID: "a3",
			Attrs: map[string]any{"workflow": "ticket-update", "requester": policy.Ref{Type: "User", ID: "bob"}, "approvers": policy.Ref{Type: "Group", ID: "finance"}}}, policy.Deny},
		{"approve without approvers (Unknown refuses)", ann, "approve", policy.Entity{Type: "ProposedAction", ID: "a4",
			Attrs: map[string]any{"workflow": "ticket-update", "requester": policy.Ref{Type: "User", ID: "bob"}}}, policy.Unknown},
		{"call a plain tool", ann, "call_tool", policy.Entity{Type: "Tool", ID: "lookup", Attrs: map[string]any{"name": "lookup", "connector": "tickets", "consequential": false}}, policy.Allow},
		{"call a consequential tool directly", ann, "call_tool", policy.Entity{Type: "Tool", ID: "update", Attrs: map[string]any{"name": "update", "connector": "tickets", "consequential": true}}, policy.Deny},
		{"call another tenant's tool", ann, "call_tool", policy.Entity{Type: "Tool", ID: "lookup", Tenant: "beta-freight", Attrs: map[string]any{"name": "lookup", "connector": "tickets", "consequential": false}}, policy.Deny},
		{"propose for the workflow", ann, "propose", policy.Entity{Type: "Workflow", ID: "ticket-update", Attrs: map[string]any{"name": "ticket-update", "approvers": policy.Ref{Type: "Group", ID: "it-leads"}}}, policy.Allow},
		{"admin (no permit)", ann, "admin", policy.Entity{Type: "Cell", ID: "acme-logistics"}, policy.Deny},
		{"no scope", tenancy.Scope{}, "read", policy.Entity{Type: "Chunk", ID: "c1", Attrs: map[string]any{"cite": "x"}}, policy.Unknown},
	}
	for _, c := range cases {
		if got := kernel(e).Authorise(ctx, c.s, c.action, c.res, nil); got.Decision != c.want || got.Reason == "" {
			t.Errorf("%s: %v (%s), want %v", c.name, got.Decision, got.Reason, c.want)
		}
	}
}

// A policy that errors while evaluating (here: a missing attribute) makes the answer
// Unknown, not a quiet Deny or Allow.
func TestEvaluationErrorIsUnknown(t *testing.T) {
	e, err := Load(map[string][]byte{"t.cedar": []byte(`permit (principal, action == Action::"read", resource) when { resource.missing == "x" };`)})
	if err != nil {
		t.Fatal(err)
	}
	got := kernel(e).Authorise(context.Background(), scope(t, "acme-logistics", "ann"), "read", policy.Entity{Type: "Chunk", ID: "c"}, nil)
	if got.Decision != policy.Unknown {
		t.Fatalf("got %v (%s), want Unknown", got.Decision, got.Reason)
	}
}

func TestBadPolicyFailsToLoad(t *testing.T) {
	if _, err := Load(map[string][]byte{"bad.cedar": []byte(`permit (principal, action, resource`)}); err == nil {
		t.Fatal("want a load error")
	}
}

type discard struct{}

func (discard) RecordDecision(context.Context, policy.Record) error { return nil }

func kernel(e Engine) policy.Kernel { return policy.Kernel{Engine: e, Recorder: discard{}} }

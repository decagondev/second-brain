package reorg

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

const treeManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  customer:
    template: support
    members: [group:support, group:sales]
    stewards: [group:cx-leads]
    glossary: { SLA: "service level agreement" }
    scope: { docs: ["/Customer/**"] }
  ops:
    template: engineering
    members: [group:support, group:eng, group:finance]
    stewards: [group:ops-leads]
    glossary: { runbook: "steps for an incident" }
    scope: { docs: ["/Ops/**"] }
  support:
    template: support
    parent: ops
    members: [group:support]
    scope: { docs: ["/Support/**"] }
  helpdesk:
    template: support
    parent: ops
    members: [group:support]
    scope: { docs: ["/Helpdesk/**"] }
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/**"] }
  payroll:
    parent: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/Payroll/**"] }
`

func treeTenant(t *testing.T) manifest.Tenant {
	t.Helper()
	m, ds := manifest.Check([]byte(treeManifest))
	if len(ds) > 0 {
		t.Fatalf("fixture: %v", ds)
	}
	return m
}

func effect(p TreePlan, space string) SpaceEffect {
	for _, e := range p.Spaces {
		if e.Space == space {
			return e
		}
	}
	return SpaceEffect{}
}

func TestMovePlansWhatChanges(t *testing.T) {
	m := treeTenant(t)
	p, err := Move(m, "support", "customer")
	if err != nil {
		t.Fatal(err)
	}
	e := effect(p, "support")
	if e.ParentBefore != "ops" || e.ParentAfter != "customer" || e.Gone {
		t.Errorf("effect %+v", e)
	}
	// The manifest the plan leaves is the one the change makes, and it passes the check.
	again, ds := manifest.Check(manifest.Format(p.After))
	if len(ds) > 0 || again.Spaces["support"].Parent != "customer" {
		t.Errorf("after: %v %+v", ds, again.Spaces["support"])
	}
	if effect(p, "ops").Space == "" {
		t.Error("the old parent isn't in the plan")
	}
	if m.Spaces["support"].Parent != "ops" {
		t.Error("the dry run changed the manifest it read")
	}
}

func TestMergeFoldsOneSpaceIntoAnother(t *testing.T) {
	m := treeTenant(t)
	p, err := Merge(m, "helpdesk", "support")
	if err != nil {
		t.Fatal(err)
	}
	if !effect(p, "helpdesk").Gone {
		t.Errorf("helpdesk isn't gone: %+v", p.Spaces)
	}
	if _, ok := p.After.Spaces["helpdesk"]; ok || !slices.Contains(p.After.Spaces["support"].Scope["docs"], "/Helpdesk/**") {
		t.Errorf("after: %+v", p.After.Spaces["support"])
	}
	if _, ok := m.Spaces["helpdesk"]; !ok {
		t.Error("the dry run changed the manifest it read")
	}
}

func TestTreeRefusesWhatWouldWiden(t *testing.T) {
	m := treeTenant(t)
	if _, err := Merge(m, "support", "customer"); !errors.Is(err, ErrWiden) || !strings.Contains(err.Error(), "group:sales") {
		t.Errorf("a merge whose members differ: %v", err)
	}
	after := copyTenant(m)
	f := after.Spaces["finance"]
	f.Label = "internal"
	after.Spaces["finance"] = f
	if _, err := plan(m, after, OpMove, "finance", "", nil); !errors.Is(err, ErrWiden) {
		t.Errorf("a label that loosens: %v", err)
	} else if !strings.Contains(err.Error(), "spaces.finance.label") {
		t.Errorf("the refusal doesn't name the space: %v", err)
	}
	for name, err := range map[string]error{
		"unknown space":  func() error { _, err := Move(m, "nope", "ops"); return err }(),
		"unknown parent": func() error { _, err := Move(m, "support", "nope"); return err }(),
		"a cycle":        func() error { _, err := Move(m, "ops", "support"); return err }(),
		"into itself":    func() error { _, err := Merge(m, "ops", "ops"); return err }(),
	} {
		if !errors.Is(err, ErrTree) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

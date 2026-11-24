package ask

import (
	"context"
	"fmt"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
)

// Department is a read-only department agent (sb-agents(7), Department agents; SB-1102):
// a bundled prompt, the one template whose spaces it runs in, and no tools.
type Department struct {
	Name, Template, Prompt, Lead string
	Tools                        []string // always empty: a department agent can't act (LAW-09)
	// Restricted runs it only in a restricted space (SB-4003: exec-brief).
	Restricted bool
}

// Departments are the department agents the release ships.
var Departments = map[string]Department{
	"incident-summary": {Name: "incident-summary", Template: "engineering", Prompt: "incident-summary", Lead: "Incident: "},
	"spec-digest":      {Name: "spec-digest", Template: "product", Prompt: "spec-digest", Lead: "Spec: "},
	"close-checklist":  {Name: "close-checklist", Template: "finance", Prompt: "close-checklist", Lead: "Close: "},
	"runbook-answer":   {Name: "runbook-answer", Template: "it-security", Prompt: "runbook-answer", Lead: "Question: "},
	// An agent for every department (SB-4001 to SB-4003).
	"policy-answer":    {Name: "policy-answer", Template: "people", Prompt: "policy-answer", Lead: "Policy: "},
	"clause-finder":    {Name: "clause-finder", Template: "legal", Prompt: "clause-finder", Lead: "Topic: "},
	"account-brief":    {Name: "account-brief", Template: "sales", Prompt: "account-brief", Lead: "Account: "},
	"case-brief":       {Name: "case-brief", Template: "support", Prompt: "case-brief", Lead: "Case: "},
	"brand-check":      {Name: "brand-check", Template: "marketing", Prompt: "brand-check", Lead: "Draft: "},
	"sop-steps":        {Name: "sop-steps", Template: "operations", Prompt: "sop-steps", Lead: "Procedure: "},
	"design-rationale": {Name: "design-rationale", Template: "design", Prompt: "design-rationale", Lead: "Design: "},
	"exec-brief":       {Name: "exec-brief", Template: "executive", Prompt: "exec-brief", Lead: "Question: ", Restricted: true},
}

// department checks that a department agent may run for this scope: enabled by the
// manifest, asked in a space of its own template that lists it, not killed, and allowed
// by the policy kernel. It returns the agent, or an error before any retrieval.
func (a Agent) department(ctx context.Context, s tenancy.Scope, name string) (Department, error) {
	d, ok := Departments[name]
	if !ok {
		return Department{}, fmt.Errorf("%w: no agent %q", ErrInvalid, name)
	}
	if a.Enabled == nil || !a.Enabled(name) {
		return Department{}, fmt.Errorf("%w: agent %s isn't enabled in the manifest", ErrInvalid, name)
	}
	if s.Space() == "company" || a.SpaceOf == nil {
		return Department{}, fmt.Errorf("%w: agent %s runs only in a %s space, never company-wide", ErrInvalid, name, d.Template)
	}
	sp, found, err := a.SpaceOf(ctx, s.Space())
	if err != nil {
		return Department{}, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	if !found || sp.Template != d.Template || !contains(sp.Agents, name) {
		return Department{}, fmt.Errorf("%w: agent %s runs only in a %s space that lists it", ErrInvalid, name, d.Template)
	}
	if d.Restricted && sp.Label != "restricted" {
		return Department{}, fmt.Errorf("%w: agent %s runs only in a restricted %s space", ErrInvalid, name, d.Template)
	}
	if a.AgentKilled == nil || a.AgentKilled(name) {
		return Department{}, ErrKilled // no kill-switch reader is treated as killed
	}
	use := a.Auth.Policy.Authorise(ctx, s, "use_agent", policy.Entity{Type: "Agent", ID: name, Tenant: a.Tenant,
		Attrs: map[string]any{"name": name, "consequential": false, "enabled": true}}, nil)
	if use.Decision != policy.Allow {
		return Department{}, fmt.Errorf("%w: agent %s: %s", ErrInvalid, name, use.Reason)
	}
	return d, nil
}

// citedOnly keeps the cited claims and counts the rest (sb-agents(7): every claim cited).
// With none left, the answer is the one uncited "nothing to cite" claim (LAW-14).
func citedOnly(claims []answer.Claim) ([]answer.Claim, int) {
	var kept []answer.Claim
	for _, c := range claims {
		if !c.Uncited && len(c.Cites) > 0 {
			kept = append(kept, c)
		}
	}
	dropped := len(claims) - len(kept)
	if len(kept) == 0 {
		kept = []answer.Claim{{Text: NothingToCite, Cites: []answer.Cite{}, Uncited: true}}
	}
	return kept, dropped
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

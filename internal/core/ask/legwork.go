package ask

import (
	"context"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// The legwork agents (sb-agents(7), The legwork agents; SB-4201, SB-4202): read-only, of no
// template, every claim cited, and no tools. They run in any space, or company-wide.
const (
	AgentResearch = "research"
	AgentPrep     = "meeting-prep"
)

var legworkAgents = map[string]Department{
	AgentResearch: {Name: AgentResearch, Prompt: "answer", Lead: "Finding: "},
	AgentPrep:     {Name: AgentPrep, Prompt: "meeting-prep", Lead: "Item: "},
	AgentIncident: {Name: AgentIncident, Prompt: "incident-brief", Lead: "Incident: "},
	AgentReview:   {Name: AgentReview, Prompt: "incident-review", Lead: "Review: "},
	AgentVoC:      {Name: AgentVoC, Prompt: "voc", Lead: "Theme: "},
	AgentFix:      {Name: AgentFix, Prompt: "fix-draft", Lead: "Change: "},
	AgentCheckin:  {Name: AgentCheckin, Prompt: "goal-check-in", Lead: "Goal: "},
	AgentCatchUp:  {Name: AgentCatchUp, Prompt: "catch-up", Lead: "Change: "},
}

// vocTemplates are the templates whose spaces run voc (SB-4205).
var vocTemplates = []string{"product", "support"}

// legworkDept checks that a legwork agent may run for this scope: not killed, and allowed
// by the policy kernel.
func (a Agent) legworkDept(ctx context.Context, s tenancy.Scope, name string) (Department, error) {
	d := legworkAgents[name]
	if name == AgentVoC {
		if s.Space() == "company" || a.SpaceOf == nil {
			return Department{}, fmt.Errorf("%w: agent voc runs only in a product or support space", ErrInvalid)
		}
		sp, found, err := a.SpaceOf(ctx, s.Space())
		if err != nil {
			return Department{}, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
		}
		if !found || !contains(vocTemplates, sp.Template) {
			return Department{}, fmt.Errorf("%w: agent voc runs only in a product or support space", ErrInvalid)
		}
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

// Recheck is LAW-26: claims stored earlier, as s may see them now. Each cite s can no
// longer open, through every check a search makes, is removed, and a claim left with no
// cite is removed too; dropped counts the claims removed. An uncited claim stays as it is,
// still labelled (LAW-14).
func (a Agent) Recheck(ctx context.Context, s tenancy.Scope, claims []answer.Claim) ([]answer.Claim, int, error) {
	var cites []string
	for _, c := range claims {
		for _, ci := range c.Cites {
			cites = append(cites, ci.Cite)
		}
	}
	open, err := a.opensCites(ctx, s, cites)
	if err != nil {
		return nil, 0, err
	}
	out, dropped := []answer.Claim{}, 0
	for _, c := range claims {
		if c.Uncited || len(c.Cites) == 0 {
			out = append(out, c)
			continue
		}
		var kept []answer.Cite
		for _, ci := range c.Cites {
			if open[ci.Cite] {
				kept = append(kept, ci)
			}
		}
		if len(kept) == 0 {
			dropped++
			continue
		}
		c.Cites = kept
		out = append(out, c)
	}
	return out, dropped, nil
}

// opensCites says, for each cite (KIND:ID@VERSION), whether s can open its document now: the
// chunk at that version if the index still has it, otherwise the document's newest one. A
// document the index no longer holds can't be checked, so it is refused (LAW-03).
func (a Agent) opensCites(ctx context.Context, s tenancy.Scope, cites []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(cites) == 0 {
		return out, nil
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var refs []ports.SourceRef
	seen := map[string]bool{}
	for _, c := range cites {
		doc, _, _ := strings.Cut(c, "@")
		kind, id, ok := strings.Cut(doc, ":")
		if !ok || seen[doc] {
			continue
		}
		seen[doc] = true
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	byCite, newest := map[string]ports.Chunk{}, firsts(latest(chunks))
	for _, c := range chunks {
		if have, ok := byCite[c.Cite]; !ok || c.Start < have.Start {
			byCite[c.Cite] = c
		}
	}
	pick := map[string]ports.Chunk{} // cite → the chunk that decides it
	var check []ports.Chunk
	for _, c := range cites {
		if _, done := pick[c]; done {
			continue
		}
		ch, ok := byCite[c]
		if !ok {
			doc, _, _ := strings.Cut(c, "@")
			if ch, ok = newest[doc]; !ok {
				continue
			}
		}
		pick[c] = ch
		check = append(check, ch)
	}
	if len(check) == 0 {
		return out, nil
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, check, len(check))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]bool{}
	for _, c := range res.Context {
		allowed[c.Kind+":"+c.SourceID+"@"+fmt.Sprint(c.Version)] = true
	}
	for c, ch := range pick {
		out[c] = allowed[ch.Kind+":"+ch.SourceID+"@"+fmt.Sprint(ch.Version)]
	}
	return out, nil
}

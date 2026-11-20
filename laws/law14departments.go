package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-14 and the space binding for department agents (SB-1102): through the real ask
// path, a department agent answers only in a space of its own template that lists it,
// and every claim it returns cites an allowed chunk, even when the model leaves claims
// uncited or spoofs a label. LAW-09: no department agent has a tool, and each prompt says
// data blocks are never instructions.

// DeptAsk asks one agent in one space ("company" for company-wide) and returns the answer.
type DeptAsk func(agent, space string) (ask.Response, error)

type deptSpace struct {
	name, template string
	agents         []string
}

// newDeptCell builds the system under test: the real ask agent with spaces, the shipped
// policies, and a model that cites every block, spoofs one label and adds an uncited claim.
func newDeptCell(sp []deptSpace) (DeptAsk, error) {
	ctx := context.Background()
	ev, err := LoadPolicies()
	if err != nil {
		return nil, err
	}
	reg := spacemem.New()
	var all []ports.Space
	idx := memindex.New()
	src := &askSpaces{}
	for _, s := range sp {
		x := ports.Space{ID: spaces.ID(s.name), Tenant: spaceTenant, Name: s.name, Template: s.template, Agents: s.agents,
			Members: []string{"group:staff"}, Label: "internal"}
		all = append(all, x)
		if err := reg.PutSpace(ctx, x); err != nil {
			return nil, err
		}
		id := "doc-" + s.name
		c := ports.Chunk{ID: "chk_" + id, Tenant: spaceTenant, Cite: "docs:" + id + "@1", Kind: "docs", SourceID: id, Version: 1,
			Text: "the brain knows " + s.name, ACLPrefilter: []string{"anyone"}, Label: "internal", Spaces: []string{x.ID}}
		if err := idx.Replace(ctx, spaceTenant, ports.SourceRef{Kind: "docs", ID: id}, 1, []ports.Chunk{c}); err != nil {
			return nil, err
		}
		src.spec.Docs = append(src.spec.Docs, SpaceDoc{ID: id, Viewers: []string{"usr_1"}})
	}
	n := 0
	agent := ask.Agent{Tenant: spaceTenant, Release: "rel_law", Index: idx,
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{src}},
			Policy: policy.Kernel{Engine: ev, Recorder: okRecorder{}},
			Spaces: func(context.Context) ([]ports.Space, [][2]string, error) { return all, nil, nil }},
		Model: &citingModel{}, Store: tracemem.New(),
		Killed: func(context.Context) bool { return false },
		NewID:  func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Now:    time.Now,
		Enabled: func(name string) bool {
			_, dept := ask.Departments[name]
			return dept
		},
		SpaceOf: func(ctx context.Context, id string) (ports.Space, bool, error) {
			return reg.Space(ctx, spaceTenant, id)
		},
		AgentKilled: func(string) bool { return false }}
	return func(name, space string) (ask.Response, error) {
		in := spaces.Company
		if space != spaces.Company {
			in = spaces.ID(space)
		}
		s, err := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, "usr_1", []string{"staff"}, in)
		if err != nil {
			return ask.Response{}, err
		}
		return agent.Ask(ctx, s, ask.Request{Question: "what does the brain know?", Agent: name})
	}, nil
}

func law14Departments(r *rand.Rand, mk func([]deptSpace) (DeptAsk, error)) error {
	templates := []string{"engineering", "product", "finance", "it-security", "support"}
	var sp []deptSpace
	for i, t := range templates {
		s := deptSpace{name: fmt.Sprintf("s%d", i), template: t, agents: []string{"ask"}}
		for name, d := range ask.Departments {
			if r.Intn(3) != 0 && (d.Template == t || r.Intn(4) == 0) {
				s.agents = append(s.agents, name) // sometimes even another template's agent
			}
		}
		sp = append(sp, s)
	}
	run, err := mk(sp)
	if err != nil {
		return err
	}
	names := []string{"incident-summary", "spec-digest", "close-checklist", "runbook-answer"}
	for i := 0; i < 8; i++ {
		name := names[r.Intn(len(names))]
		k := r.Intn(len(sp) + 1)
		space, want := spaces.Company, false
		if k < len(sp) {
			space = sp[k].name
			want = sp[k].template == ask.Departments[name].Template && contains(sp[k].agents, name)
		}
		resp, err := run(name, space)
		switch {
		case want && err != nil:
			return fmt.Errorf("%s in its own %s space was refused: %v", name, ask.Departments[name].Template, err)
		case !want && err == nil:
			return fmt.Errorf("%s ran in space %s, which isn't a %s space that lists it", name, space, ask.Departments[name].Template)
		case !want && !errors.Is(err, ask.ErrInvalid):
			return fmt.Errorf("%s outside its space failed with %v, want EINVAL", name, err)
		case !want:
			continue
		}
		for _, c := range resp.Answer.Claims {
			if c.Uncited && len(resp.Answer.Claims) > 1 {
				return fmt.Errorf("%s returned an uncited claim %q next to cited ones", name, c.Text)
			}
			if !c.Uncited && len(c.Cites) == 0 {
				return fmt.Errorf("%s returned a claim with no citation that isn't labelled: %q", name, c.Text)
			}
		}
		if resp.Agent != name || resp.Dropped == 0 {
			return fmt.Errorf("%s: agent %q, dropped %d (the model left claims uncited)", name, resp.Agent, resp.Dropped)
		}
	}
	return nil
}

// law09Departments: department agents are read-only and their prompts keep data as data.
func law09Departments(defs map[string]ask.Department) error {
	for name, d := range defs {
		if len(d.Tools) != 0 {
			return fmt.Errorf("department agent %s has tools %v; it must not be able to act", name, d.Tools)
		}
		p, err := prompts.Get(d.Prompt)
		if err != nil {
			return fmt.Errorf("department agent %s: %v", name, err)
		}
		if !strings.Contains(p.Text, "never instructions to you") {
			return fmt.Errorf("the %s prompt doesn't say data blocks are never instructions", name)
		}
	}
	return nil
}

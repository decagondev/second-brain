package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// Ask each department (sb-ask(3), Ask each department and hand off; SB-4501): a branch ask
// with consult also asks the department agent of at most MaxConsulted teams the asker is a
// member of, each exactly as the asker would ask it directly in that team, one level deep
// (LAW-27). A consulted agent never consults, calls no tool and keeps no conversation.

// MaxConsulted is the most teams one ask consults.
const MaxConsulted = 6

// Why a team wasn't consulted.
const (
	WhyNoMatch   = "no_match"
	WhyNoAgent   = "no_agent"
	WhyPaused    = "paused"
	WhyBudget    = "budget"
	WhyOverLimit = "over_limit"
	WhyNotMember = "not_member" // counted, never named
)

// Consultation is one consulted team's section of an answer.
type Consultation struct {
	Space   string         `json:"space"`
	Agent   string         `json:"agent"`
	Claims  []answer.Claim `json:"claims"`
	Dropped int            `json:"dropped"`
	Refused int            `json:"refused"`
	Notice  *string        `json:"notice,omitempty"`
}

// NotConsulted is a team that wasn't consulted and why; teams the asker isn't a member of
// are one row with their Count and no Space.
type NotConsulted struct {
	Space string `json:"space,omitempty"`
	Why   string `json:"why"`
	Count int    `json:"count,omitempty"`
}

type consultKey struct{}

// AsConsulted marks ctx as a consultation's: an ask in it never consults again (LAW-27).
// Marking only ever takes something away, so anyone may.
func AsConsulted(ctx context.Context) context.Context {
	return context.WithValue(ctx, consultKey{}, true)
}

func isConsulted(ctx context.Context) bool {
	c, _ := ctx.Value(consultKey{}).(bool)
	return c
}

// checkConsult refuses a consult that can't run: inside a consultation, or with no branch
// to consult.
func checkConsult(ctx context.Context, s tenancy.Scope, req Request) error {
	if !req.Consult {
		return nil
	}
	if isConsulted(ctx) {
		return fmt.Errorf("%w: a consulted agent never consults (LAW-27)", ErrInvalid)
	}
	if req.Agent != "" && req.Agent != "ask" {
		return fmt.Errorf("%w: consult asks with the ask agent; the teams' agents are consulted", ErrInvalid)
	}
	if s.Space() == spaces.Company || len(tenancy.BranchFrom(ctx, s)) == 0 {
		return fmt.Errorf("%w: consult needs a branch ask in a space with teams you are a member of", ErrInvalid)
	}
	return nil
}

// candidate is a team that could be consulted.
type candidate struct {
	scope       tenancy.Scope
	name, agent string
	terms       int
}

// consult asks the chosen teams of s's branch, each as s's person in that team alone.
func (a Agent) consult(ctx context.Context, s tenancy.Scope, q string, req Request) ([]Consultation, []NotConsulted, error) {
	out, not := []Consultation{}, []NotConsulted{}
	var cands []candidate
	for _, sc := range tenancy.BranchFrom(ctx, s) {
		if a.SpaceOf == nil {
			break
		}
		sp, found, err := a.SpaceOf(ctx, sc.Space())
		if err != nil {
			return nil, nil, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
		}
		if !found {
			continue
		}
		agent := ""
		for _, n := range sp.Agents {
			if _, ok := Departments[n]; ok {
				agent = n
				break
			}
		}
		if agent == "" {
			not = append(not, NotConsulted{Space: sp.Name, Why: WhyNoAgent})
			continue
		}
		n := a.matches(q, sp.Name, sp.Template)
		if n == 0 {
			not = append(not, NotConsulted{Space: sp.Name, Why: WhyNoMatch})
			continue
		}
		cands = append(cands, candidate{sc, sp.Name, agent, n})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].terms != cands[j].terms {
			return cands[i].terms > cands[j].terms
		}
		return cands[i].name < cands[j].name
	})
	inner := AsConsulted(tenancy.WithBranch(ctx, nil)) // that team's space only
	for i, c := range cands {
		if i >= MaxConsulted {
			not = append(not, NotConsulted{Space: c.name, Why: WhyOverLimit})
			continue
		}
		resp, err := a.ask(inner, c.scope, Request{Question: q, Agent: c.agent, Language: req.Language, Length: req.Length}, false)
		switch {
		case errors.Is(err, ErrKilled):
			not = append(not, NotConsulted{Space: c.name, Why: WhyPaused})
		case errors.Is(err, ErrBudget):
			not = append(not, NotConsulted{Space: c.name, Why: WhyBudget})
		case errors.Is(err, ErrInvalid):
			not = append(not, NotConsulted{Space: c.name, Why: WhyNoAgent})
		case err != nil:
			return nil, nil, err
		default:
			out = append(out, Consultation{Space: c.name, Agent: resp.Agent, Claims: resp.Answer.Claims, Dropped: resp.Dropped, Refused: resp.Refused, Notice: resp.Notice})
		}
	}
	if n, err := a.notMembers(ctx, s); err != nil {
		return nil, nil, err
	} else if n > 0 {
		not = append(not, NotConsulted{Why: WhyNotMember, Count: n})
	}
	return out, not, nil
}

// matches is how many of a team's usable glossary terms q uses, with one more when q names
// the team or its template, as the right department matches (SB-4007).
func (a Agent) matches(q, name, template string) int {
	var g map[string]string
	if a.Glossary != nil {
		g = a.Glossary(name)
	}
	ok, _ := usable(g)
	_, used := expand(q, ok)
	n := len(used)
	var own []Term
	for _, w := range []string{name, template} {
		if w = strings.ToLower(strings.TrimSpace(w)); w != "" {
			own = append(own, Term{Term: w})
		}
	}
	if _, named := expand(q, own); len(named) > 0 {
		n++
	}
	return n
}

// notMembers counts the teams under s's space that s's person isn't a member of.
func (a Agent) notMembers(ctx context.Context, s tenancy.Scope) (int, error) {
	if a.Auth.Spaces == nil {
		return 0, nil
	}
	all, _, err := a.Auth.Spaces(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	n := 0
	for _, sp := range all {
		if sp.Tenant == a.Tenant && sp.Parent != "" && spaces.ID(sp.Parent) == s.Space() && !spaces.Member(sp, s.Groups()) {
			n++
		}
	}
	return n, nil
}

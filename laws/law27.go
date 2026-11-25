package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-27 A consultation is the asker's own (SB-4501; sb-laws(7)): a branch ask that
// consults department agents consults only teams the asker is a member of, at most
// ask.MaxConsulted, each section exactly what asking that team's agent directly would give
// them, never names a team they aren't in, and a consulted agent never consults again. A
// random department with teams of random templates, agents, members, glossaries and
// documents with random viewers.

// ConsultTeam is one team under the department: its template, agents, member groups and
// glossary.
type ConsultTeam struct {
	Name, Template string
	Agents         []string
	Members        []string
	Glossary       map[string]string
}

// ConsultSpec is a department, "hq", with its teams, its users' groups, and documents by
// team with their viewers.
type ConsultSpec struct {
	Teams []ConsultTeam
	Users map[string][]string
	Docs  map[string]map[string][]string // team -> doc id -> viewers
}

// ConsultSystem asks a department and its teams.
type ConsultSystem interface {
	Load(spec ConsultSpec) error
	// Consult asks q in hq across its branch, consulting, as user.
	Consult(user, q string) (ask.Response, error)
	// Direct asks q with agent in team, as user.
	Direct(user, team, agent, q string) (ask.Response, error)
	// Nested asks q in hq across its branch, consulting, from inside a consultation.
	Nested(user, q string) error
}

// NewConsultSystem builds the system under test: the real ask.Agent, branch scopes as the
// auth middleware issues them (spaces.Teams, the Issuer), the shipped policies.
var NewConsultSystem = func() ConsultSystem { return &consultCell{} }

type consultCell struct {
	spec  ConsultSpec
	agent ask.Agent
	reg   *spacemem.Store
	src   *askSpaces
	// wide is the mutation: the branch is every team, as though the asker were in each.
	wide bool
}

func (c *consultCell) Load(spec ConsultSpec) error {
	ctx := context.Background()
	ev, err := LoadPolicies()
	if err != nil {
		return err
	}
	c.spec, c.reg, c.src = spec, spacemem.New(), &askSpaces{}
	var groups []string
	for _, t := range spec.Teams {
		groups = append(groups, t.Members...)
	}
	all := []ports.Space{{ID: spaces.ID("hq"), Tenant: spaceTenant, Name: "hq", Template: "executive", Agents: []string{"ask"},
		Members: groups, Label: "internal"}}
	for _, t := range spec.Teams {
		all = append(all, ports.Space{ID: spaces.ID(t.Name), Tenant: spaceTenant, Name: t.Name, Template: t.Template, Agents: t.Agents,
			Members: t.Members, Label: "internal", Parent: "hq"})
	}
	idx := memindex.New()
	for _, sp := range all {
		if err := c.reg.PutSpace(ctx, sp); err != nil {
			return err
		}
		for id, vs := range spec.Docs[sp.Name] {
			ch := ports.Chunk{ID: "chk_" + id, Tenant: spaceTenant, Cite: "docs:" + id + "@1", Kind: "docs", SourceID: id, Version: 1,
				Text: "the brain knows " + id, ACLPrefilter: []string{"anyone"}, Label: "internal", Spaces: []string{sp.ID}}
			if err := idx.Replace(ctx, spaceTenant, ports.SourceRef{Kind: "docs", ID: id}, 1, []ports.Chunk{ch}); err != nil {
				return err
			}
			c.src.spec.Docs = append(c.src.spec.Docs, SpaceDoc{ID: id, Viewers: vs})
		}
	}
	gloss := map[string]map[string]string{}
	for _, t := range spec.Teams {
		gloss[t.Name] = t.Glossary
	}
	n := 0
	c.agent = ask.Agent{Tenant: spaceTenant, Release: "rel_law", Index: idx,
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{c.src}},
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
			return c.reg.Space(ctx, spaceTenant, id)
		},
		Glossary:    func(name string) map[string]string { return gloss[name] },
		AgentKilled: func(string) bool { return false }}
	return nil
}

func (c *consultCell) scopeIn(user, space string, extra []string) (tenancy.Scope, error) {
	return tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, user, append(dirGroups(c.spec.Users[user]), extra...), spaces.ID(space))
}

// branch is ctx with user's branch scopes under hq, as the auth middleware issues them.
func (c *consultCell) branch(ctx context.Context, user string) (context.Context, tenancy.Scope, error) {
	s, err := c.scopeIn(user, "hq", nil)
	if err != nil {
		return nil, tenancy.Scope{}, err
	}
	teams, err := spaces.Teams(ctx, c.reg, spaceTenant, dirGroups(c.spec.Users[user]), s.Space())
	if err != nil {
		return nil, tenancy.Scope{}, err
	}
	var scopes []tenancy.Scope
	if c.wide {
		for _, t := range c.spec.Teams {
			in, err := c.scopeIn(user, t.Name, dirGroups(t.Members))
			if err != nil {
				return nil, tenancy.Scope{}, err
			}
			scopes = append(scopes, in)
		}
	} else {
		for _, id := range teams {
			in, err := c.scopeIn(user, strings.TrimPrefix(id, "spc_"), nil)
			if err != nil {
				return nil, tenancy.Scope{}, err
			}
			scopes = append(scopes, in)
		}
	}
	return tenancy.WithBranch(ctx, scopes), s, nil
}

func (c *consultCell) Consult(user, q string) (ask.Response, error) {
	ctx, s, err := c.branch(context.Background(), user)
	if err != nil {
		return ask.Response{}, err
	}
	return c.agent.Ask(ctx, s, ask.Request{Question: q, Consult: true})
}

func (c *consultCell) Direct(user, team, agent, q string) (ask.Response, error) {
	s, err := c.scopeIn(user, team, nil)
	if err != nil {
		return ask.Response{}, err
	}
	return c.agent.Ask(context.Background(), s, ask.Request{Question: q, Agent: agent})
}

func (c *consultCell) Nested(user, q string) error {
	ctx, s, err := c.branch(context.Background(), user)
	if err != nil {
		return err
	}
	_, err = c.agent.Ask(ask.AsConsulted(ctx), s, ask.Request{Question: q, Consult: true})
	return err
}

// genConsult makes a department of 2 to 9 teams, three users in random groups, and one or
// two documents per space, each open to random users.
func genConsult(r *rand.Rand) (ConsultSpec, []string) {
	names := make([]string, 0, len(ask.Departments))
	for n := range ask.Departments {
		if !ask.Departments[n].Restricted {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	users := []string{"usr_1", "usr_2", "usr_3"}
	sp := ConsultSpec{Users: map[string][]string{}, Docs: map[string]map[string][]string{}}
	var terms []string
	for i := 0; i < 2+r.Intn(8); i++ {
		d := ask.Departments[names[r.Intn(len(names))]]
		t := ConsultTeam{Name: fmt.Sprintf("team%d", i), Template: d.Template, Agents: []string{"ask"},
			Members: []string{fmt.Sprintf("group:g%d", i)}, Glossary: map[string]string{}}
		if r.Intn(5) != 0 {
			t.Agents = append(t.Agents, d.Name)
		}
		for j := 0; j < r.Intn(3); j++ {
			term := fmt.Sprintf("term%d%c", i, 'a'+j)
			t.Glossary[term] = "a word of " + t.Name
			terms = append(terms, term)
		}
		sp.Teams = append(sp.Teams, t)
	}
	for _, u := range users {
		for _, t := range sp.Teams {
			if r.Intn(2) == 0 {
				sp.Users[u] = append(sp.Users[u], t.Members...)
			}
		}
		if len(sp.Users[u]) == 0 {
			sp.Users[u] = append(sp.Users[u], sp.Teams[0].Members...) // a member of hq
		}
	}
	for _, s := range append([]string{"hq"}, teamNames(sp)...) {
		sp.Docs[s] = map[string][]string{}
		for j := 0; j < 1+r.Intn(2); j++ {
			var vs []string
			for _, u := range users {
				if r.Intn(3) != 0 {
					vs = append(vs, u)
				}
			}
			sp.Docs[s][fmt.Sprintf("%s-doc%d", s, j)] = vs
		}
	}
	var qs []string
	for i := 0; i < 3; i++ {
		q := "what does the brain know"
		for _, t := range terms {
			if r.Intn(3) == 0 {
				q += " about " + t
			}
		}
		if r.Intn(3) == 0 {
			q += " in " + sp.Teams[r.Intn(len(sp.Teams))].Template
		}
		qs = append(qs, q+"?")
	}
	return sp, qs
}

func teamNames(sp ConsultSpec) []string {
	var out []string
	for _, t := range sp.Teams {
		out = append(out, t.Name)
	}
	return out
}

func citesOf(claims []answer.Claim) []string {
	var out []string
	for _, c := range claims {
		for _, ci := range c.Cites {
			out = append(out, ci.Cite)
		}
	}
	sort.Strings(out)
	return out
}

func law27(r *rand.Rand, sys ConsultSystem) error {
	sp, qs := genConsult(r)
	if err := sys.Load(sp); err != nil {
		return err
	}
	member := func(u string, t ConsultTeam) bool {
		for _, g := range sp.Users[u] {
			if contains(t.Members, g) {
				return true
			}
		}
		return false
	}
	byName := map[string]ConsultTeam{}
	for _, t := range sp.Teams {
		byName[t.Name] = t
	}
	for u := range sp.Users {
		for _, q := range qs {
			resp, err := sys.Consult(u, q)
			if err != nil {
				return fmt.Errorf("%s consulting: %v", u, err)
			}
			if len(resp.Departments) > ask.MaxConsulted {
				return fmt.Errorf("%s consulted %d teams, more than %d", u, len(resp.Departments), ask.MaxConsulted)
			}
			outside := 0
			for _, t := range sp.Teams {
				if !member(u, t) {
					outside++
				}
			}
			counted := 0
			for _, nc := range resp.NotConsulted {
				if nc.Why == ask.WhyNotMember {
					counted += nc.Count
					if nc.Space != "" {
						return fmt.Errorf("%s was told the name of a team they aren't in: %s", u, nc.Space)
					}
					continue
				}
				if t, ok := byName[nc.Space]; !ok || !member(u, t) {
					return fmt.Errorf("%s was told about team %q, which they aren't a member of (%s)", u, nc.Space, nc.Why)
				}
			}
			if counted != outside {
				return fmt.Errorf("%s: %d teams counted as not theirs, want %d", u, counted, outside)
			}
			for _, d := range resp.Departments {
				t, ok := byName[d.Space]
				if !ok || !member(u, t) {
					return fmt.Errorf("%s consulted team %q, which they aren't a member of", u, d.Space)
				}
				direct, err := sys.Direct(u, d.Space, d.Agent, q)
				if err != nil {
					return fmt.Errorf("%s asking %s directly: %v", u, d.Space, err)
				}
				got, want := citesOf(d.Claims), citesOf(direct.Answer.Claims)
				if strings.Join(got, ",") != strings.Join(want, ",") {
					return fmt.Errorf("%s consulting %s got cites %v; asking it directly gives %v", u, d.Space, got, want)
				}
			}
			if err := sys.Nested(u, q); !errors.Is(err, ask.ErrInvalid) {
				return fmt.Errorf("a consultation asked to consult gave %v, want EINVAL", err)
			}
		}
	}
	return nil
}

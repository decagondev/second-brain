package ask

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/tenancy"
)

// Scenarios before you approve (sb-agents(7), Scenarios; SB-4904): a steward writes up to
// MaxScenarios questions, each with what its answer should cite or say and what it must
// never say, and runs them against the space as applied and as a proposal would make it,
// side by side, before an operator approves the proposal. A scenario runs as the steward
// who runs it, never as anyone else, and keeps no conversation. One that passes on the
// proposal and names a document to cite can be kept as an eval case.
const (
	MaxScenarios    = 10
	MaxScenarioText = 300
)

// Scenario is one question and what its answer should do: cite Cites (KIND:ID), say Says,
// and never say Never; at least one of the three. Agent is a department agent to ask, ""
// for ask.
type Scenario struct {
	Question string `json:"question"`
	Agent    string `json:"agent,omitempty"`
	Cites    string `json:"cites,omitempty"`
	Says     string `json:"says,omitempty"`
	Never    string `json:"never,omitempty"`
}

// ScenarioVersion is one version of a space as a scenario runs against it: the department
// agents its `agents` lists and its skills.
type ScenarioVersion struct {
	Agents []string
	Skills []Skill
}

// ScenarioSide is one version's run: whether it passed, why not, and its answer's lines and
// cites, for the steward who ran it alone.
type ScenarioSide struct {
	Pass  bool     `json:"pass"`
	Why   string   `json:"why,omitempty"`
	Text  string   `json:"text,omitempty"`
	Cites []string `json:"cites,omitempty"`
}

// ScenarioResult is a scenario run against both versions; Kept is the eval case it became.
type ScenarioResult struct {
	Scenario
	Current  ScenarioSide `json:"current"`
	Proposed ScenarioSide `json:"proposed"`
	Kept     string       `json:"kept,omitempty"`
}

type scenarioKey struct{}

// withScenario runs an ask against version v of its space: its agents and skills.
func withScenario(ctx context.Context, v ScenarioVersion) context.Context {
	return context.WithValue(ctx, scenarioKey{}, v)
}

func scenarioOf(ctx context.Context) (ScenarioVersion, bool) {
	v, ok := ctx.Value(scenarioKey{}).(ScenarioVersion)
	return v, ok
}

// CleanScenarios checks and trims scenarios: 1 to MaxScenarios, each a question and at least
// one expectation, every text one line of at most MaxScenarioText characters.
func CleanScenarios(scs []Scenario) ([]Scenario, error) {
	if len(scs) == 0 || len(scs) > MaxScenarios {
		return nil, fmt.Errorf("%w: 1 to %d scenarios", ErrInvalid, MaxScenarios)
	}
	out := make([]Scenario, 0, len(scs))
	for i, sc := range scs {
		for _, f := range []*string{&sc.Question, &sc.Agent, &sc.Cites, &sc.Says, &sc.Never} {
			*f = strings.TrimSpace(*f)
			if utf8.RuneCountInString(*f) > MaxScenarioText || !printableText(*f) {
				return nil, fmt.Errorf("%w: scenario %d: each text is one line of at most %d characters", ErrInvalid, i+1, MaxScenarioText)
			}
		}
		if sc.Cites != "" {
			sc.Cites = docKey(sc.Cites)
		}
		switch {
		case sc.Question == "":
			return nil, fmt.Errorf("%w: scenario %d has no question", ErrInvalid, i+1)
		case sc.Cites == "" && sc.Says == "" && sc.Never == "":
			return nil, fmt.Errorf("%w: scenario %d needs cites, says or never", ErrInvalid, i+1)
		case sc.Cites != "" && !strings.Contains(sc.Cites, ":"):
			return nil, fmt.Errorf("%w: scenario %d: cites is KIND:ID", ErrInvalid, i+1)
		}
		out = append(out, sc)
	}
	return out, nil
}

// judge is one run of sc: an answer, or the error that stopped it.
func judge(sc Scenario, resp Response, err error) ScenarioSide {
	if err != nil {
		why := err.Error()
		if errors.Is(err, ErrKilled) {
			why = "the agent is switched off or paused"
		}
		return ScenarioSide{Why: why}
	}
	var lines, cites []string
	for _, c := range resp.Answer.Claims {
		lines = append(lines, strings.Join(strings.Fields(c.Text), " "))
		for _, ci := range c.Cites {
			if k := docKey(ci.Cite); k != "" && !contains(cites, k) {
				cites = append(cites, k)
			}
		}
	}
	side := ScenarioSide{Text: strings.Join(lines, "\n"), Cites: cites}
	low := strings.ToLower(side.Text)
	var why []string
	if sc.Cites != "" && !contains(cites, sc.Cites) {
		why = append(why, "doesn't cite "+sc.Cites)
	}
	if sc.Says != "" && !strings.Contains(low, strings.ToLower(sc.Says)) {
		why = append(why, "doesn't say \""+sc.Says+"\"")
	}
	if sc.Never != "" && strings.Contains(low, strings.ToLower(sc.Never)) {
		why = append(why, "says \""+sc.Never+"\"")
	}
	side.Pass, side.Why = len(why) == 0, strings.Join(why, "; ")
	return side
}

// RunScenarios runs scs as s, a steward of space name, against cur and next, the space as
// applied and as proposed. Each run is a probe: it keeps a trace but no conversation, and
// every check a search makes applies, as s.
func (x Stewards) RunScenarios(ctx context.Context, s tenancy.Scope, name string, scs []Scenario, cur, next ScenarioVersion) ([]ScenarioResult, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	if s.Space() != SpaceKey(name) {
		return nil, fmt.Errorf("%w: a scenario runs in its own space", ErrInvalid)
	}
	scs, err := CleanScenarios(scs)
	if err != nil {
		return nil, err
	}
	out := make([]ScenarioResult, 0, len(scs))
	for _, sc := range scs {
		r := ScenarioResult{Scenario: sc}
		for _, side := range []struct {
			v  ScenarioVersion
			to *ScenarioSide
		}{{cur, &r.Current}, {next, &r.Proposed}} {
			resp, err := x.Agent.ProbeAs(withScenario(ctx, side.v), s, sc.Question, sc.Agent)
			if errors.Is(err, ErrUnavailable) {
				return nil, err
			}
			*side.to = judge(sc, resp, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// KeepScenario makes sc, which passed on the proposal, an eval case of space name run as s:
// its question, its agent and the document it cites, counted toward MaxEvalCases.
func (x Stewards) KeepScenario(ctx context.Context, s tenancy.Scope, name string, r ScenarioResult) (EvalCase, error) {
	if x.Evals == nil {
		return EvalCase{}, fmt.Errorf("%w: this cell keeps no eval cases", ErrUnavailable)
	}
	if !r.Proposed.Pass || r.Cites == "" {
		return EvalCase{}, fmt.Errorf("%w: only a scenario that passed on the proposal and names a document to cite can be kept", ErrInvalid)
	}
	all, err := x.Evals.EvalCases(ctx, x.Agent.Tenant)
	if err != nil {
		return EvalCase{}, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	if len(handCases(all)) >= MaxEvalCases {
		return EvalCase{}, fmt.Errorf("%w: a tenant has at most %d eval cases", ErrInvalid, MaxEvalCases)
	}
	version := 0
	if c, found, err := x.current(ctx, r.Cites); err == nil && found {
		version = c.Version
	}
	agent := r.Agent
	if agent == "ask" {
		agent = ""
	}
	ec := EvalCase{ID: x.NewID("ev_"), Tenant: x.Agent.Tenant, Space: SpaceKey(name), Question: r.Question, Expect: r.Cites,
		ExpectVersion: version, Agent: agent, By: s.User(), At: x.Now().UTC()}
	if err := x.Evals.PutEvalCase(ctx, ec); err != nil {
		return EvalCase{}, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	if err := x.auditCase(ctx, s, ec.ID, "added", name); err != nil {
		return EvalCase{}, err
	}
	return ec, nil
}

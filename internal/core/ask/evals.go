package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Cases from wrong citations (sb-evals(7); SB-2204).
const (
	MaxEvalCases = 50 // a tenant's cases
	MaxReports   = 10 // reports listed on a review item
)

// EvalCase is one of the tenant's own eval cases: a question, in a space, and the document
// its answer should cite. No answer is ever kept.
type EvalCase struct {
	ID       string    `json:"id"`
	Tenant   string    `json:"tenant"`
	Space    string    `json:"space"` // spc_NAME
	Question string    `json:"question"`
	Expect   string    `json:"expect"` // KIND:ID
	Wrong    string    `json:"wrong"`  // KIND:ID the reported answer cited wrongly
	By       string    `json:"by"`     // the steward it runs as
	At       time.Time `json:"at"`
	// The last run's release and result, and the last release it passed on.
	LastRelease string `json:"last_release,omitempty"`
	LastPassed  bool   `json:"last_passed"`
	PassedOn    string `json:"passed_on,omitempty"`
}

// EvalStore keeps a tenant's cases.
type EvalStore interface {
	PutEvalCase(ctx context.Context, c EvalCase) error
	EvalCases(ctx context.Context, tenant string) ([]EvalCase, error)
	DeleteEvalCase(ctx context.Context, tenant, id string) error
}

// reviewer checks the scope may review space name (the review list's decision).
func (x Stewards) reviewer(ctx context.Context, s tenancy.Scope, name string) error {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant || name == "company" {
		return ErrNotSteward
	}
	stewards, _, ok := x.Facts(SpaceKey(name))
	if !ok {
		return ErrNotSteward
	}
	var groups []string
	for _, g := range s.Groups() {
		groups = append(groups, "group:"+g)
	}
	if x.Policy.Authorise(ctx, s, "review_space", policy.Entity{Type: "SpaceConfig", ID: name, Tenant: x.Agent.Tenant,
		Attrs: map[string]any{"stewards": stewards}}, map[string]any{"groups": groups}).Decision != policy.Allow {
		return ErrNotSteward
	}
	return nil
}

// AddEvalCase turns a wrong-citation report on a document the space holds into a case:
// the reported answer's question, and expect, a document the steward can open now.
func (x Stewards) AddEvalCase(ctx context.Context, s tenancy.Scope, name, report, expect string) (EvalCase, error) {
	if err := x.reviewer(ctx, s, name); err != nil {
		return EvalCase{}, err
	}
	if x.Evals == nil || x.Feedback == nil {
		return EvalCase{}, fmt.Errorf("%w: this cell keeps no eval cases", ErrUnavailable)
	}
	expect = docKey(strings.TrimSpace(expect))
	if !strings.Contains(expect, ":") {
		return EvalCase{}, fmt.Errorf("%w: expect is KIND:ID", ErrInvalid)
	}
	all, err := x.Evals.EvalCases(ctx, x.Agent.Tenant)
	if err != nil {
		return EvalCase{}, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	if len(all) >= MaxEvalCases {
		return EvalCase{}, fmt.Errorf("%w: a tenant has at most %d eval cases", ErrInvalid, MaxEvalCases)
	}
	f, found, err := x.Feedback.GetFeedback(ctx, x.Agent.Tenant, report)
	if err != nil {
		return EvalCase{}, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	if !found || f.WrongCitation == "" {
		return EvalCase{}, fmt.Errorf("%w: no wrong-citation report %s", ErrInvalid, report)
	}
	wrong := docKey(f.WrongCitation)
	t, found, err := x.Agent.Store.GetTrace(ctx, x.Agent.Tenant, report)
	if err != nil {
		return EvalCase{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	if !found || t.Response.Conversation == "" {
		return EvalCase{}, fmt.Errorf("%w: the reported answer is gone", ErrInvalid)
	}
	qs, found, err := x.Agent.Store.Questions(ctx, x.Agent.Tenant, t.User, t.Response.Conversation)
	if err != nil {
		return EvalCase{}, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if !found || t.Turn >= len(qs) {
		return EvalCase{}, fmt.Errorf("%w: the reported question is gone", ErrInvalid)
	}
	if err := x.heldAndOpen(ctx, s, name, wrong, expect); err != nil {
		return EvalCase{}, err
	}
	c := EvalCase{ID: x.NewID("ev_"), Tenant: x.Agent.Tenant, Space: SpaceKey(name), Question: qs[t.Turn], Expect: expect,
		Wrong: wrong, By: s.User(), At: x.Now().UTC()}
	if err := x.Evals.PutEvalCase(ctx, c); err != nil {
		return EvalCase{}, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	if err := x.auditCase(ctx, s, c.ID, "added", name); err != nil {
		return EvalCase{}, err
	}
	return c, nil
}

// heldAndOpen checks that the space holds the wrong document and the expected one, and that
// the steward can open the expected one now, as a search checks it.
func (x Stewards) heldAndOpen(ctx context.Context, s tenancy.Scope, name, wrong, expect string) error {
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var refs []ports.SourceRef
	for _, k := range []string{wrong, expect} {
		kind, id, _ := strings.Cut(k, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	docs := firsts(latest(cs))
	for _, k := range []string{wrong, expect} {
		if c, ok := docs[k]; !ok || !contains(c.Spaces, SpaceKey(name)) {
			return fmt.Errorf("%w: space %s doesn't hold %s", ErrInvalid, name, k)
		}
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, []ports.Chunk{docs[expect]}, 1)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		return fmt.Errorf("%w: you can't open %s now", ErrInvalid, expect)
	}
	return nil
}

// EvalCasesOf lists a space's cases for its stewards.
func (x Stewards) EvalCasesOf(ctx context.Context, s tenancy.Scope, name string) ([]EvalCase, error) {
	if err := x.reviewer(ctx, s, name); err != nil {
		return nil, err
	}
	if x.Evals == nil {
		return []EvalCase{}, nil
	}
	all, err := x.Evals.EvalCases(ctx, x.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	out := []EvalCase{}
	for _, c := range all {
		if c.Space == SpaceKey(name) {
			out = append(out, c)
		}
	}
	return out, nil
}

// DeleteEvalCase removes one of the space's cases.
func (x Stewards) DeleteEvalCase(ctx context.Context, s tenancy.Scope, name, id string) error {
	cs, err := x.EvalCasesOf(ctx, s, name)
	if err != nil {
		return err
	}
	for _, c := range cs {
		if c.ID == id {
			if err := x.Evals.DeleteEvalCase(ctx, x.Agent.Tenant, id); err != nil {
				return fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
			}
			return x.auditCase(ctx, s, id, "deleted", name)
		}
	}
	return fmt.Errorf("%w: no such case in %s", ErrInvalid, name)
}

// auditCase writes a config row for a case: its id and space, never its question.
func (x Stewards) auditCase(ctx context.Context, s tenancy.Scope, id, state, name string) error {
	if x.Audit == nil {
		return nil
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: id, State: state,
		Reason: "eval case from a wrong citation", Space: auditSpace(SpaceKey(name))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// Probe answers a case's question as its steward, in its space, as Ask does, but keeps no
// conversation: the run is the cell's, not the steward's. Its trace is kept, with agent
// "eval".
func (a Agent) Probe(ctx context.Context, s tenancy.Scope, q string) (Response, error) {
	return a.ask(ctx, s, Request{Question: q}, true)
}

// Passes is a case's verdict on an answer: it cites the expected document and not the
// wrong one.
func (c EvalCase) Passes(r Response) bool {
	cited := map[string]bool{}
	for _, cl := range r.Answer.Claims {
		for _, ci := range cl.Cites {
			cited[docKey(ci.Cite)] = true
		}
	}
	return cited[c.Expect] && !cited[c.Wrong]
}

// ErrSkipCase is a case that can't run now, such as one whose steward has left: it keeps
// its last result and holds nothing.
var ErrSkipCase = errors.New("ask: the eval case can't run now")

// Hold is a release held for this tenant: it regressed cases that passed on Back.
type Hold struct {
	Release string   `json:"release"`
	Back    string   `json:"back"`
	Cases   []string `json:"cases"`
}

// Instead is the release to run when the channel names target: Back while target is the held
// release, else "" (run target).
func (h Hold) Instead(target string) string {
	if h.Release != "" && h.Release == target {
		return h.Back
	}
	return ""
}

// RunCases runs, on release, every case whose last run was on another release, and returns
// the cases with their new results and the hold their regressions make, if any. A case
// that never passed holds nothing.
func RunCases(cases []EvalCase, release string, run func(EvalCase) (bool, error)) ([]EvalCase, *Hold, error) {
	out := make([]EvalCase, 0, len(cases))
	var hold *Hold
	for _, c := range cases {
		if c.LastRelease == release {
			out = append(out, c)
			continue
		}
		ok, err := run(c)
		if errors.Is(err, ErrSkipCase) {
			out = append(out, c) // not run: its steward has gone, or the agent is off
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if !ok && c.PassedOn != "" && c.PassedOn != release {
			if hold == nil {
				hold = &Hold{Release: release, Back: c.PassedOn}
			}
			hold.Cases = append(hold.Cases, c.ID)
		}
		c.LastRelease, c.LastPassed = release, ok
		if ok {
			c.PassedOn = release
		}
		out = append(out, c)
	}
	if hold != nil {
		sort.Strings(hold.Cases)
	}
	return out, hold, nil
}

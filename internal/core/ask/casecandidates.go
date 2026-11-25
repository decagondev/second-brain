package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Cases from feedback (sb-evals(7), Cases from feedback; SB-4401): a down-rated or reported
// answer of an agent becomes a candidate case for the people who answer for the agent.

// The candidates' window and length, and the inbox kind.
const (
	CandidateWindow = 30 * 24 * time.Hour
	MaxCandidates   = 20
	InboxCase       = "case"
)

// CaseCandidate is one answer that could become a case. Its id is the answer's trace. The
// asker is never named; only claims whose every cite the owner can open now are shown.
type CaseCandidate struct {
	ID       string           `json:"id"`
	Agent    string           `json:"agent"`
	Space    string           `json:"space"`
	Question string           `json:"question"`
	Claims   []CandidateClaim `json:"claims"`
	Hidden   int              `json:"hidden"`
	At       time.Time        `json:"at"`
	wrong    string
}

// CandidateClaim is a shown claim and its cites, as the owner can open them now.
type CandidateClaim struct {
	Text  string   `json:"text"`
	Cites []Result `json:"cites"`
}

// FeedbackLister lists a window's feedback.
type FeedbackLister interface {
	FeedbackBetween(ctx context.Context, tenant string, from, to time.Time) ([]Feedback, error)
}

// answering is whether an agent's runs are answers to a question: ask and the department
// agents. The other agents' runs are drafts and briefs, not answers a case can rerun.
func answering(agent string) bool {
	_, dept := Departments[agent]
	return agent == "ask" || dept
}

// ownedAnswering are the answering agents s answers for.
func (x Stewards) ownedAnswering(ctx context.Context, s tenancy.Scope) ([]string, error) {
	var emails []string
	if x.Agent.Emails != nil {
		es, err := x.Agent.Emails(ctx, s.User())
		if err != nil {
			return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		emails = es
	}
	var out []string
	for _, a := range AgentsOwnedBy(emails, s.Groups()) {
		if answering(a) {
			out = append(out, a)
		}
	}
	return out, nil
}

// CaseCandidates are s's candidates, newest first, at most MaxCandidates.
func (x Stewards) CaseCandidates(ctx context.Context, s tenancy.Scope) ([]CaseCandidate, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return nil, ErrNotSteward
	}
	out := []CaseCandidate{}
	fl, ok := x.Feedback.(FeedbackLister)
	if x.Evals == nil || !ok {
		return out, nil
	}
	owned, err := x.ownedAnswering(ctx, s)
	if err != nil || len(owned) == 0 {
		return out, err
	}
	all, err := x.Evals.EvalCases(ctx, x.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	decided := map[string]bool{}
	for _, c := range all {
		decided[c.From] = c.From != ""
	}
	now := x.Now()
	fbs, err := fl.FeedbackBetween(ctx, x.Agent.Tenant, now.Add(-CandidateWindow), now)
	if err != nil {
		return nil, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	sort.SliceStable(fbs, func(i, j int) bool { return fbs[i].At.After(fbs[j].At) })
	for _, f := range fbs {
		if len(out) == MaxCandidates {
			break
		}
		bad := f.Rating == "down" || f.WrongCitation != "" || f.OutdatedCitation != ""
		if f.Tenant != x.Agent.Tenant || !bad || decided[f.Trace] {
			continue
		}
		decided[f.Trace] = true // one candidate a trace
		c, ok, err := x.candidate(ctx, s, f, owned)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// candidate is f's answer as a candidate for s, if it is one of an agent s owns.
func (x Stewards) candidate(ctx context.Context, s tenancy.Scope, f Feedback, owned []string) (CaseCandidate, bool, error) {
	t, found, err := x.Agent.Store.GetTrace(ctx, x.Agent.Tenant, f.Trace)
	if err != nil {
		return CaseCandidate{}, false, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	agent := t.Response.Agent
	if agent == "" {
		agent = "ask"
	}
	if !found || t.Tenant != x.Agent.Tenant || t.Response.Conversation == "" || !contains(owned, agent) {
		return CaseCandidate{}, false, nil
	}
	qs, found, err := x.Agent.Store.Questions(ctx, x.Agent.Tenant, t.User, t.Response.Conversation)
	if err != nil {
		return CaseCandidate{}, false, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if !found || t.Turn >= len(qs) {
		return CaseCandidate{}, false, nil // the question is gone: nothing to rerun
	}
	var cites []string
	for _, cl := range t.Response.Answer.Claims {
		for _, ci := range cl.Cites {
			cites = append(cites, ci.Cite)
		}
	}
	rs, _, err := x.Agent.CheckCites(ctx, s, cites)
	if err != nil {
		return CaseCandidate{}, false, err
	}
	open := map[string]Result{}
	for _, r := range rs {
		open[r.Cite] = r
	}
	c := CaseCandidate{ID: t.ID, Agent: agent, Space: t.Space, Question: qs[t.Turn], Claims: []CandidateClaim{}, At: f.At,
		wrong: docKey(f.WrongCitation)}
	for _, cl := range t.Response.Answer.Claims {
		shown := CandidateClaim{Text: cl.Text, Cites: []Result{}}
		ok := true
		for _, ci := range cl.Cites {
			r, can := open[ci.Cite]
			ok = ok && can
			shown.Cites = append(shown.Cites, r)
		}
		if !ok {
			c.Hidden++ // never named (LAW-01)
			continue
		}
		c.Claims = append(c.Claims, shown)
	}
	return c, true, nil
}

// CaseDecision is the body of POST /v1/agents/cases/{id}: an expect, or dismiss.
type CaseDecision struct {
	Expect  string `json:"expect,omitempty"`
	Dismiss bool   `json:"dismiss,omitempty"`
}

// DecideCase accepts or dismisses one of s's candidates, and audits it.
func (x Stewards) DecideCase(ctx context.Context, s tenancy.Scope, id string, d CaseDecision) (EvalCase, error) {
	cs, err := x.CaseCandidates(ctx, s)
	if err != nil {
		return EvalCase{}, err
	}
	var c *CaseCandidate
	for i := range cs {
		if cs[i].ID == id {
			c = &cs[i]
		}
	}
	if c == nil {
		return EvalCase{}, fmt.Errorf("%w: no candidate %s for you", ErrNotFound, id)
	}
	ec := EvalCase{ID: x.NewID("ev_"), Tenant: x.Agent.Tenant, Space: c.Space, Agent: c.Agent, From: c.ID, By: s.User(), At: x.Now().UTC()}
	state := "dismissed"
	if !d.Dismiss {
		if ec, err = x.acceptCase(ctx, s, *c, ec, d.Expect); err != nil {
			return EvalCase{}, err
		}
		state = "accepted"
	} else {
		ec.Dismissed = true
	}
	if err := x.Evals.PutEvalCase(ctx, ec); err != nil {
		return EvalCase{}, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	if x.Audit != nil {
		if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: ec.ID, State: state,
			Reason: "eval_case", Space: auditSpace(c.Space)}); err != nil {
			return EvalCase{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
		}
	}
	return ec, nil
}

// acceptCase fills an accepted case: a candidate with nothing hidden, under the limit, and
// an expect the owner can open now.
func (x Stewards) acceptCase(ctx context.Context, s tenancy.Scope, c CaseCandidate, ec EvalCase, expect string) (EvalCase, error) {
	if c.Hidden > 0 {
		return EvalCase{}, fmt.Errorf("%w: the answer has claims you can't open", ErrInvalid)
	}
	expect = docKey(strings.TrimSpace(expect))
	if !strings.Contains(expect, ":") {
		return EvalCase{}, fmt.Errorf("%w: expect is KIND:ID", ErrInvalid)
	}
	all, err := x.Evals.EvalCases(ctx, x.Agent.Tenant)
	if err != nil {
		return EvalCase{}, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	if len(ActiveCases(all)) >= MaxEvalCases {
		return EvalCase{}, fmt.Errorf("%w: a tenant has at most %d eval cases", ErrInvalid, MaxEvalCases)
	}
	version, err := x.openNow(ctx, s, expect)
	if err != nil {
		return EvalCase{}, err
	}
	ec.Question, ec.Expect, ec.ExpectVersion, ec.Wrong = c.Question, expect, version, c.wrong
	return ec, nil
}

// openNow is the version of a document s can open now, as a search checks it.
func (x Stewards) openNow(ctx context.Context, s tenancy.Scope, key string) (int, error) {
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return 0, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	kind, id, _ := strings.Cut(key, ":")
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return 0, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	doc, ok := firsts(latest(cs))[key]
	if !ok {
		return 0, fmt.Errorf("%w: you can't open %s now", ErrInvalid, key)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return 0, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, []ports.Chunk{doc}, 1)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		return 0, fmt.Errorf("%w: you can't open %s now", ErrInvalid, key) // the same words: never says it exists
	}
	return doc.Version, nil
}

// inboxCases adds s's candidates to Needs you.
func (l Legwork) inboxCases(ctx context.Context, s tenancy.Scope, in *Inbox) error {
	if l.Stewards.Evals == nil {
		return nil
	}
	cs, err := l.Stewards.CaseCandidates(ctx, s)
	if err != nil {
		return err
	}
	for _, c := range cs {
		in.NeedsYou = append(in.NeedsYou, InboxItem{Kind: InboxCase, ID: c.ID, Space: spaceName(c.Space), Title: c.Question, At: c.At,
			Why: "a " + c.Agent + " answer was rated down", Link: "/?tab=questions"})
	}
	return nil
}

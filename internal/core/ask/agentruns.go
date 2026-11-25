package ask

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// What agents did as me (SB-4305): every run an agent made as a person in the last
// AgentRunDays, drawn from what is already kept (the steward queues' drafts, their reports
// and the traces each run wrote). No record of its own is kept, and what each run read is
// checked again as the person now (LAW-01).

// AgentRunDays is how far back the list reaches.
const AgentRunDays = 90

// RunDoc is one document a run read that the person can open now.
type RunDoc struct {
	Cite  string `json:"cite"`
	Title string `json:"title"`
}

// AgentRun is one run made as the person.
type AgentRun struct {
	Kind    string    `json:"kind"`
	Agent   string    `json:"agent"`
	Space   string    `json:"space"`
	At      time.Time `json:"at"`
	Title   string    `json:"title"`
	Read    []RunDoc  `json:"read"`
	Hidden  int       `json:"hidden"`
	Drafted string    `json:"drafted"`
	Link    string    `json:"link"`
}

// researchAgents is the agent behind each kind of report.
var researchAgents = map[string]string{KindResearch: AgentResearch, KindPrep: AgentPrep, KindRoutine: "ask", KindFix: AgentFix}

// AgentRuns lists the runs made as s's person, newest first.
func (l Legwork) AgentRuns(ctx context.Context, s tenancy.Scope) ([]AgentRun, error) {
	if err := l.check(s); err != nil {
		return nil, err
	}
	since := l.now().Add(-AgentRunDays * 24 * time.Hour)
	type run struct {
		AgentRun
		traces []string
	}
	var runs []run
	if l.Stewards.Store != nil {
		qs, err := l.Stewards.Store.StewardQuestionsOf(ctx, l.Agent.Tenant, s.User())
		if err != nil {
			return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		for _, q := range qs {
			if q.Thread == nil || q.Thread.Agent == "" || q.Thread.Discarded || q.At.Before(since) {
				continue
			}
			runs = append(runs, run{AgentRun{Kind: "draft", Agent: q.Thread.Agent, Space: spaceName(q.Space), At: q.At, Title: q.Question,
				Drafted: "a " + q.Thread.Kind + " draft for the stewards", Link: queueLink(spaceName(q.Space))}, []string{q.Trace}})
		}
	}
	if l.Research != nil {
		rs, err := l.Research.ResearchOf(ctx, l.Agent.Tenant, s.User(), since)
		if err != nil {
			return nil, fmt.Errorf("%w: research: %v", ErrUnavailable, err)
		}
		for _, r := range rs {
			if r.At.Before(since) || r.State == StatePlanned {
				continue // a plan no one ran is no run
			}
			agent := researchAgents[r.Kind]
			if agent == "" {
				agent = r.Kind
			}
			claims := 0
			var traces []string
			for _, sec := range r.Sections {
				claims += len(sec.Claims)
				if sec.Trace != "" {
					traces = append(traces, sec.Trace)
				}
			}
			runs = append(runs, run{AgentRun{Kind: r.Kind, Agent: agent, Space: spaceName(r.Space), At: r.At, Title: r.Question,
				Drafted: "a " + r.Kind + " of " + strconv.Itoa(claims) + " claims, " + r.State, Link: "/?tab=legwork"}, traces})
		}
	}
	out := []AgentRun{}
	for _, r := range runs {
		cites, err := l.runCites(ctx, s, r.traces)
		if err != nil {
			return nil, err
		}
		rs, hidden, err := l.Agent.CheckCites(ctx, s, cites)
		if err != nil {
			return nil, err
		}
		r.Read, r.Hidden = []RunDoc{}, hidden
		for _, x := range rs {
			r.Read = append(r.Read, RunDoc{Cite: x.Cite, Title: x.Title})
		}
		out = append(out, r.AgentRun)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, nil
}

// runCites are the chunks a run's traces allowed, each once; an expired trace adds none.
func (l Legwork) runCites(ctx context.Context, s tenancy.Scope, traces []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, id := range traces {
		t, found, err := l.Agent.Store.GetTrace(ctx, l.Agent.Tenant, id)
		if err != nil {
			return nil, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
		}
		if !found || t.Tenant != l.Agent.Tenant || t.User != s.User() {
			continue
		}
		for _, d := range t.Decisions {
			if d.Verdict == "allow" && !seen[d.Cite] {
				seen[d.Cite] = true
				out = append(out, d.Cite)
			}
		}
	}
	return out, nil
}

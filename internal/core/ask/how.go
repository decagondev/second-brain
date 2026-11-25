package ask

import (
	"context"
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/tenancy"
)

// How this was made (sb-ask(3), How this was made; SB-4606): an answer's, draft's or report's
// making, read from its traces and checked as the viewer each time it is opened (LAW-26). A
// source the viewer can't open now is counted, never named (LAW-01).

// HowMade is how one answer, draft or report was made.
type HowMade struct {
	Trace         string         `json:"trace,omitempty"`
	Report        string         `json:"report,omitempty"`
	Agent         string         `json:"agent"`
	Release       string         `json:"release"`
	Skill         string         `json:"skill,omitempty"`
	Tier          string         `json:"tier,omitempty"`
	Steps         []string       `json:"steps,omitempty"`
	Departments   []HowDept      `json:"departments"`
	NotConsulted  []NotConsulted `json:"not_consulted"`
	Sources       []HowSource    `json:"sources"`
	SourcesHidden int            `json:"sources_hidden"`
	Refused       int            `json:"refused"`
	Dropped       int            `json:"dropped"`
	CostUSD       float64        `json:"cost_usd"`
}

// HowDept is one team an answer consulted (SB-4501).
type HowDept struct {
	Space string `json:"space"`
	Agent string `json:"agent"`
}

// HowSource is one source an answer read that the viewer can open now.
type HowSource struct {
	Cite  string `json:"cite"`
	Title string `json:"title,omitempty"`
}

// How is how the answer or draft of trace id was made, for whoever may read the trace.
func (a Agent) How(ctx context.Context, s tenancy.Scope, id string) (HowMade, error) {
	t, err := a.Trace(ctx, s, id)
	if err != nil {
		return HowMade{}, err
	}
	out := HowMade{Trace: t.ID, Agent: t.Response.Agent, Release: t.Response.Release, Skill: t.Response.Skill, Tier: t.Tier}
	if err := a.howOf(ctx, s, []Trace{t}, &out); err != nil {
		return HowMade{}, err
	}
	return out, nil
}

// howOf adds traces' consultations, sources, counts and cost to out, the sources checked as
// s now.
func (a Agent) howOf(ctx context.Context, s tenancy.Scope, ts []Trace, out *HowMade) error {
	out.Departments, out.NotConsulted, out.Sources = []HowDept{}, []NotConsulted{}, []HowSource{}
	var docs []string
	for _, t := range ts {
		for _, d := range t.Response.Departments {
			out.Departments = append(out.Departments, HowDept{Space: d.Space, Agent: d.Agent})
			out.Dropped += d.Dropped
		}
		out.NotConsulted = append(out.NotConsulted, t.Response.NotConsulted...)
		out.Refused += t.Response.Refused
		out.Dropped += t.Response.Dropped
		out.CostUSD += t.CostUSD
		for _, d := range t.Decisions {
			if k := docKey(d.Cite); d.Verdict == "allow" && !contains(docs, k) {
				docs = append(docs, k)
			}
		}
	}
	titles, hidden, err := a.docTitles(ctx, s, docs)
	if err != nil {
		return err
	}
	out.SourcesHidden = hidden // counted, never named
	for _, d := range docs {
		if t, ok := titles[d]; ok {
			out.Sources = append(out.Sources, HowSource{Cite: d, Title: t})
		}
	}
	sort.SliceStable(out.Sources, func(i, j int) bool { return out.Sources[i].Title < out.Sources[j].Title })
	return nil
}

// How is how one of s's reports, briefs or routine answers was made: its sub-questions,
// and its sections' sources, counts and cost together.
func (l Legwork) How(ctx context.Context, s tenancy.Scope, id string) (HowMade, error) {
	r, err := l.own(ctx, s, id)
	if err != nil {
		return HowMade{}, err
	}
	agent := map[string]string{KindResearch: AgentResearch, KindPrep: AgentPrep, KindRoutine: "ask", KindAccount: ""}[r.Kind]
	out := HowMade{Report: r.ID, Agent: agent, Release: l.Agent.Release, Steps: append([]string{}, r.Steps...)}
	out.Skill, _ = SkillName(r.Question)
	var ts []Trace
	for _, sec := range r.Sections {
		t, found, err := l.Agent.Store.GetTrace(ctx, l.Agent.Tenant, sec.Trace)
		if err != nil {
			return HowMade{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
		}
		if found && t.User == s.User() {
			ts = append(ts, t)
			if out.Agent == "" {
				out.Agent = t.Response.Agent
			}
			out.Release = t.Response.Release
		}
	}
	if err := l.Agent.howOf(ctx, s, ts, &out); err != nil {
		return HowMade{}, err
	}
	if r.CostUSD > out.CostUSD {
		out.CostUSD = r.CostUSD
	}
	return out, nil
}

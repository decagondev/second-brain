package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Shadow runs (sb-agents(7), Shadow runs; SB-4704): in the first ring, an agent whose
// definition carries a shadow prompt makes each of its drafts a second time with it. The
// second is never delivered, kept in a trace or shown to the person it ran as; its owner sees
// the two side by side for the window, a week at most, and then it is deleted.

// ShadowSide is one side of a shadow run: its text and cites, and the cell's own checks.
type ShadowSide struct {
	Text    string   `json:"text"`
	Cites   []string `json:"cites"`
	Chars   int      `json:"chars"`
	Claims  int      `json:"claims"`
	Cited   int      `json:"cited"`
	Dropped int      `json:"dropped"`
	CostUSD float64  `json:"cost_usd"`
}

// ShadowRun is one draft made both ways, kept until its window ends.
type ShadowRun struct {
	ID      string     `json:"id"`
	Tenant  string     `json:"tenant"`
	Agent   string     `json:"agent"`
	Release string     `json:"release"`
	Space   string     `json:"space"`
	User    string     `json:"user"`
	At      time.Time  `json:"at"`
	Until   time.Time  `json:"until"`
	Current ShadowSide `json:"current"`
	Shadow  ShadowSide `json:"shadow"`
}

// ShadowWindow is an agent's shadow under one release: it opens with its first run and
// lasts the definition's days. It holds no text and outlives its runs.
type ShadowWindow struct {
	Tenant  string    `json:"tenant"`
	Agent   string    `json:"agent"`
	Release string    `json:"release"`
	Prompt  string    `json:"prompt"`
	From    time.Time `json:"from"`
	Until   time.Time `json:"until"`
}

// ShadowStore keeps shadow windows and runs.
type ShadowStore interface {
	// OpenShadow is the agent's window under release, opened from w when it has none.
	OpenShadow(ctx context.Context, w ShadowWindow) (ShadowWindow, error)
	PutShadow(ctx context.Context, r ShadowRun) error
	ShadowRuns(ctx context.Context, tenant string) ([]ShadowRun, error)
	ShadowWindows(ctx context.Context, tenant string) ([]ShadowWindow, error)
	// DeleteShadows deletes every run whose window ended at or before now.
	DeleteShadows(ctx context.Context, tenant string, now time.Time) (int, error)
}

type shadowKey struct{}

// withShadow marks ctx as a shadow run's, with the prompt it answers with.
func withShadow(ctx context.Context, prompt string) context.Context {
	return context.WithValue(ctx, shadowKey{}, prompt)
}

// shadowPrompt is the shadow run's prompt, or "" outside one.
func shadowPrompt(ctx context.Context) string {
	p, _ := ctx.Value(shadowKey{}).(string)
	return p
}

// shadow makes a draft of kind a second time with the agent's shadow prompt, when the cell
// is in the first ring and the window is open. Nothing it does reaches anyone; a failure is
// counted and never stops the draft.
func (x Stewards) shadow(ctx context.Context, s tenancy.Scope, kind, agent string, req Request, cur Response, text string) {
	def, ok := AgentDefOf(agent)
	if !ok || def.Shadow == nil || x.Shadows == nil || x.FirstRing == nil || !x.FirstRing() {
		return
	}
	outcome := "kept"
	defer func() {
		if x.Agent.Metrics != nil {
			x.Agent.Metrics.Add("sb_shadow_runs_total", "Shadow runs by outcome (SB-4704).", map[string]string{"tenant": x.Agent.Tenant, "outcome": outcome}, 1)
		}
	}()
	now := x.Now().UTC()
	w, err := x.Shadows.OpenShadow(ctx, ShadowWindow{Tenant: x.Agent.Tenant, Agent: agent, Release: x.Agent.Release, Prompt: def.Shadow.Prompt,
		From: now, Until: now.Add(time.Duration(def.Shadow.Days) * 24 * time.Hour)})
	if err != nil {
		outcome = "failed"
		return
	}
	if !now.Before(w.Until) {
		outcome = "closed"
		return
	}
	sctx, tally := budget.WithTally(withShadow(ctx, def.Shadow.Prompt))
	sresp, stext, err := x.draftAnswer(sctx, s, kind, req)
	if err != nil {
		outcome = "failed"
		return
	}
	curCost := 0.0
	if t, found, err := x.Agent.Store.GetTrace(ctx, x.Agent.Tenant, cur.Trace); err == nil && found {
		curCost = t.CostUSD
	}
	run := ShadowRun{ID: x.NewID("sh_"), Tenant: x.Agent.Tenant, Agent: agent, Release: x.Agent.Release, Space: s.Space(), User: s.User(),
		At: now, Until: w.Until, Current: sideOf(cur, text, curCost), Shadow: sideOf(sresp, stext, tally.USD())}
	if err := x.Shadows.PutShadow(ctx, run); err != nil {
		outcome = "failed"
	}
}

// sideOf is a draft's side: its text, its distinct cites, and the checks' counts.
func sideOf(r Response, text string, cost float64) ShadowSide {
	sd := ShadowSide{Text: text, Cites: []string{}, Chars: utf8.RuneCountInString(text), Claims: len(r.Answer.Claims), Dropped: r.Dropped, CostUSD: cost}
	seen := map[string]bool{}
	for _, c := range r.Answer.Claims {
		if !c.Uncited && len(c.Cites) > 0 {
			sd.Cited++
		}
		for _, ci := range c.Cites {
			if !seen[ci.Cite] {
				seen[ci.Cite] = true
				sd.Cites = append(sd.Cites, ci.Cite)
			}
		}
	}
	sort.Strings(sd.Cites)
	return sd
}

// ShadowSideView is one side as the owner sees it: the counts, never the cites.
type ShadowSideView struct {
	Chars   int     `json:"chars"`
	Claims  int     `json:"claims"`
	Cited   int     `json:"cited"`
	Cites   int     `json:"cites"`
	Dropped int     `json:"dropped"`
	CostUSD float64 `json:"cost_usd"`
}

// ShadowRunView is one run for the owner; Text only when they can open every cite of both.
type ShadowRunView struct {
	ID      string            `json:"id"`
	At      time.Time         `json:"at"`
	Space   string            `json:"space"`
	Current ShadowSideView    `json:"current"`
	Shadow  ShadowSideView    `json:"shadow"`
	Text    *ShadowTextsShown `json:"text,omitempty"`
}

// ShadowTextsShown are the two texts side by side.
type ShadowTextsShown struct {
	Current string `json:"current"`
	Shadow  string `json:"shadow"`
}

// ShadowItem is one open window and its runs.
type ShadowItem struct {
	Agent   string          `json:"agent"`
	Release string          `json:"release"`
	Prompt  string          `json:"prompt"`
	From    time.Time       `json:"from"`
	Until   time.Time       `json:"until"`
	Runs    []ShadowRunView `json:"runs"`
}

// ShadowReport is GET /v1/agents/shadow.
type ShadowReport struct {
	Items []ShadowItem `json:"items"`
}

func sideView(sd ShadowSide) ShadowSideView {
	return ShadowSideView{Chars: sd.Chars, Claims: sd.Claims, Cited: sd.Cited, Cites: len(sd.Cites), Dropped: sd.Dropped, CostUSD: sd.CostUSD}
}

// ShadowsFor is the open windows of the agents s owns, each run's texts shown only when s
// can open every cite of both sides now.
func (x Stewards) ShadowsFor(ctx context.Context, s tenancy.Scope) (ShadowReport, error) {
	out := ShadowReport{Items: []ShadowItem{}}
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return out, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if x.Shadows == nil {
		return out, nil
	}
	owned, err := x.Agent.OwnedAgents(ctx, s)
	if err != nil || len(owned) == 0 {
		return out, err
	}
	now := x.Now().UTC()
	ws, err := x.Shadows.ShadowWindows(ctx, x.Agent.Tenant)
	if err != nil {
		return out, fmt.Errorf("%w: shadows: %v", ErrUnavailable, err)
	}
	runs, err := x.Shadows.ShadowRuns(ctx, x.Agent.Tenant)
	if err != nil {
		return out, fmt.Errorf("%w: shadows: %v", ErrUnavailable, err)
	}
	for _, w := range ws {
		if !contains(owned, w.Agent) || !now.Before(w.Until) {
			continue
		}
		it := ShadowItem{Agent: w.Agent, Release: w.Release, Prompt: w.Prompt, From: w.From, Until: w.Until, Runs: []ShadowRunView{}}
		for _, r := range runs {
			if r.Agent != w.Agent || r.Release != w.Release {
				continue
			}
			v := ShadowRunView{ID: r.ID, At: r.At, Space: strings.TrimPrefix(r.Space, "spc_"), Current: sideView(r.Current), Shadow: sideView(r.Shadow)}
			open, err := x.opensAll(ctx, s, append(append([]string{}, r.Current.Cites...), r.Shadow.Cites...))
			if err != nil {
				return out, err
			}
			if open {
				v.Text = &ShadowTextsShown{Current: r.Current.Text, Shadow: r.Shadow.Text}
			}
			it.Runs = append(it.Runs, v)
		}
		sort.Slice(it.Runs, func(i, j int) bool { return it.Runs[i].At.After(it.Runs[j].At) })
		out.Items = append(out.Items, it)
	}
	sort.Slice(out.Items, func(i, j int) bool {
		return out.Items[i].Agent+out.Items[i].Release < out.Items[j].Agent+out.Items[j].Release
	})
	return out, nil
}

// opensAll is whether s can open every one of cites at the source now; a cite gone from the
// index can't be opened.
func (x Stewards) opensAll(ctx context.Context, s tenancy.Scope, cites []string) (bool, error) {
	if len(cites) == 0 {
		return true, nil
	}
	lk, ok := x.Agent.Index.(ports.CiteLookup)
	if !ok {
		return false, nil
	}
	chunks, err := lk.ChunksOf(ctx, x.Agent.Tenant, cites)
	if err != nil {
		return false, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	have := map[string]bool{}
	for _, c := range chunks {
		have[c.Cite] = true
	}
	for _, c := range cites {
		if !have[c] {
			return false, nil
		}
	}
	who := ports.UserRef{ID: s.User()}
	if x.Agent.Emails != nil {
		es, err := x.Agent.Emails(ctx, s.User())
		if err != nil {
			return false, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		if len(es) > 0 {
			who.Email = es[0]
		}
	}
	vs := x.Agent.Auth.Check(ctx, who, chunks)
	for _, c := range chunks {
		if vs[c.Kind+":"+c.SourceID] != ports.VerdictAllow {
			return false, nil
		}
	}
	return true, nil
}

// SweepShadows deletes the runs of every window that has ended (the notify job).
func (x Stewards) SweepShadows(ctx context.Context) (int, error) {
	if x.Shadows == nil {
		return 0, nil
	}
	return x.Shadows.DeleteShadows(ctx, x.Agent.Tenant, x.Now().UTC())
}

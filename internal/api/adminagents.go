package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/report"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Every agent in one place (sb-admin(3), Agents; SB-4004): each agent the manifest enables,
// its department, the spaces that list it, its runs and the share rated unhelpful in 30
// days, its eval result and whether it is killed. Counts only: never a question, an
// answer or who asked.

// AgentWindow is how far back the registry counts runs.
const AgentWindow = 30 * 24 * time.Hour

// AdminAgent is one agent as an admin or a steward sees it.
type AdminAgent struct {
	Name       string   `json:"name"`
	Department string   `json:"department,omitempty"` // the template it belongs to; "" runs everywhere
	Spaces     []string `json:"spaces"`
	Runs       int      `json:"runs"`
	Rated      int      `json:"rated"`
	Unhelpful  int      `json:"unhelpful"`
	// Eval is "passed" (the release's gate, which every running release passed) or "held"
	// (a tenant eval case regressed and holds the release, sb-evals(7)).
	Eval   string `json:"eval"`
	Suite  string `json:"suite"`
	Killed bool   `json:"killed"`
}

// agentOf is the agent a trace counts for: "" is ask; searches, evals and refused rate
// limits are no agent's run.
func agentOf(t ask.Trace) string {
	switch a := t.Response.Agent; a {
	case "":
		return "ask"
	case ask.AgentSearch, ask.AgentEval, ask.AgentLimited, ask.AgentWelcome:
		return ""
	default:
		return a
	}
}

// AgentRegistry is the agents of m, in name order, counted from traces and feedback. A
// non-empty space counts only that space's runs and lists only the agents it can run.
func AgentRegistry(m manifest.Tenant, space string, traces []ask.Trace, fb []ask.Feedback, killed func(string) bool, held bool) []AdminAgent {
	names := append([]string{}, m.Agents...)
	if !contains(names, ask.AgentHandover) {
		names = append(names, ask.AgentHandover)
	}
	sort.Strings(names)
	rows := map[string]*AdminAgent{}
	out := []AdminAgent{}
	for _, n := range names {
		a := AdminAgent{Name: n, Spaces: []string{}, Eval: "passed", Suite: "gate", Killed: killed != nil && killed(n)}
		if held {
			a.Eval = "held"
		}
		if d, ok := ask.Departments[n]; ok {
			a.Department = d.Template
		}
		if a.Department != "" || n == ask.AgentHandover {
			a.Suite = "agent-" + n
		}
		for _, sn := range sortedSpaces(m) {
			if sp := m.Spaces[sn]; a.Department == "" || contains(sp.Agents, n) {
				a.Spaces = append(a.Spaces, sn)
			}
		}
		if space != "" && !contains(a.Spaces, space) {
			continue
		}
		if space != "" {
			a.Spaces = []string{space}
		}
		out = append(out, a)
	}
	for i := range out {
		rows[out[i].Name] = &out[i]
	}
	by := map[string]string{}
	for _, t := range traces {
		if space != "" && t.Space != "spc_"+space {
			continue
		}
		if a, ok := rows[agentOf(t)]; ok {
			a.Runs++
			by[t.ID] = a.Name
		}
	}
	for _, f := range fb {
		if a, ok := rows[by[f.Trace]]; ok {
			a.Rated++
			if f.Rating == "down" {
				a.Unhelpful++
			}
		}
	}
	return out
}

func sortedSpaces(m manifest.Tenant) []string {
	out := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// AgentCounts reads the 30 days of traces and feedback the registry counts.
func AgentCounts(ctx context.Context, src report.Source, tenant string, now time.Time) ([]ask.Trace, []ask.Feedback, error) {
	if src == nil {
		return nil, nil, errs.New(errs.EUNAVAIL, "Runs can't be read in this cell.")
	}
	from := now.Add(-AgentWindow)
	traces, err := src.Traces(ctx, tenant, from, now)
	if err != nil {
		return nil, nil, err
	}
	var fb []ask.Feedback
	if fs, ok := src.(report.FeedbackSource); ok {
		if fb, err = fs.FeedbackBetween(ctx, tenant, from, now); err != nil {
			return nil, nil, err
		}
	}
	return traces, fb, nil
}

// agents is GET /v1/admin/agents.
func (h Admin) agents(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	m, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	traces, fb, err := AgentCounts(r.Context(), h.Usage, h.Tenant, now())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Runs can't be read right now."))
		return
	}
	held := false
	if h.EvalHeld != nil {
		if held, err = h.EvalHeld(); err != nil {
			h.fail(w, errs.New(errs.EUNAVAIL, "The eval hold can't be read."))
			return
		}
	}
	h.ok(w, map[string]any{"agents": AgentRegistry(m, "", traces, fb, h.AgentKilled, held), "days": 30})
}

// spaceAgents is GET /v1/spaces/NAME/agents: the registry's rows for the agents one space
// can run, counted in that space only, for its stewards.
func (h Stewards) spaceAgents(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	traces, fb, err := AgentCounts(r.Context(), h.Runs, h.Tenant, now())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Runs can't be read right now."))
		return
	}
	held := false
	if h.EvalHeld != nil {
		if held, err = h.EvalHeld(); err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "The eval hold can't be read."))
			return
		}
	}
	can := []string{}
	for n, d := range ask.Departments {
		if d.Template == cur.Spaces[name].Template && contains(cur.Agents, n) {
			can = append(can, n)
		}
	}
	sort.Strings(can)
	writeJSON(w, map[string]any{"space": name, "agents": AgentRegistry(cur, name, traces, fb, h.AgentKilled, held),
		"listed": nonNil(cur.Spaces[name].Agents), "can_list": can, "days": 30})
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// proposeAgents is POST /v1/spaces/NAME/proposals/agents {"agents": […]}: the space as
// applied with its agents list replaced, proposed and checked as any steward proposal, so
// an operator approves the exact manifest diff.
func (h Stewards) proposeAgents(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	var req struct {
		Agents []string `json:"agents"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil || req.Agents == nil {
		fail(w, errs.New(errs.EINVAL, "The body must be {\"agents\": […]}."))
		return
	}
	sp := cur.Spaces[name]
	sp.Agents = req.Agents
	raw, err := json.Marshal(sp)
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The space's definition can't be written."))
		return
	}
	h.submit(w, r, s, cur, name, raw)
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/report"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Stewards serves delegated space changes (sb-admin(3), Proposals; SB-1201): stewards
// propose, operators approve the exact manifest, nobody approves their own.
type Stewards struct {
	Tenant    string
	Policy    policy.Kernel
	Manifests Manifests
	Audit     *audit.Log
	Proposals proposals.Store
	NewID     func(prefix string) string
	Now       func() time.Time
	// Change is the steward rule; nil is manifest.StewardChange, the only one a cell ships.
	Change func(t manifest.Tenant, space string, proposed json.RawMessage) (manifest.Tenant, []manifest.Diag, error)
	// Runs, AgentKilled and EvalHeld show a steward their space's agents (SB-4004).
	Runs        report.Source
	AgentKilled func(agent string) bool
	// OwnerActive is whether an agent's owner has an active person now (SB-4402).
	OwnerActive func(ctx context.Context, owner string) bool
	EvalHeld    func() (bool, error)
	// PausedIn are the spaces an agent is paused in (SB-4702); nil shows none.
	PausedIn func(agent string) []string
	// Scenarios runs and keeps a proposal's scenarios, as the steward in the space ScopeIn
	// makes (SB-4904); nil serves none.
	Scenarios interface {
		RunScenarios(ctx context.Context, s tenancy.Scope, name string, scs []ask.Scenario, cur, next ask.ScenarioVersion) ([]ask.ScenarioResult, error)
		KeepScenario(ctx context.Context, s tenancy.Scope, name string, r ask.ScenarioResult) (ask.EvalCase, error)
	}
	ScopeIn func(ctx context.Context, user, space string) (tenancy.Scope, error)
	// Lift lifts the personal limit of the member of space whose address is email for this
	// month (SB-5003), after audit has written its row; ask.ErrNotMember for no such member.
	// nil serves none.
	Lift func(ctx context.Context, space, email string, audit func(user, month string) error) (user, month string, err error)
	// Checks are one space's kept "did it help?" checks (SB-5005); nil serves none.
	Checks func(space string) ([]ask.Check, error)
	// SharedCount counts a space's shared routines for its branch moderators (SB-5105).
	SharedCount SharedCounter
	// Feature says a feature is on for a user (SB-5106); nil is on for everyone.
	Feature func(ctx context.Context, user, name string) bool
}

// Register mounts the endpoints.
func (h Stewards) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/spaces/{name}/proposals", h.propose)
	mux.HandleFunc("POST /v1/spaces/{name}/proposals/term", h.proposeTerm)
	mux.HandleFunc("GET /v1/spaces/{name}/agents", h.spaceAgents)
	mux.HandleFunc("POST /v1/spaces/{name}/proposals/agents", h.proposeAgents)
	mux.HandleFunc("POST /v1/spaces/{name}/proposals/{id}/scenarios", h.scenarios)              // SB-4904
	mux.HandleFunc("POST /v1/spaces/{name}/limits/lift", h.lift)                                // SB-5003
	mux.HandleFunc("GET /v1/spaces/{name}/checks", h.spaceChecks)                               // SB-5005
	mux.HandleFunc("GET /v1/spaces/{name}/branch", h.branch)                                    // SB-5105
	mux.HandleFunc("GET /v1/spaces/{name}/skills/{skill}/history", h.skillHistory)              // SB-5104
	mux.HandleFunc("POST /v1/spaces/{name}/skills/{skill}/history/{n}/restore", h.restoreSkill) // SB-5104
	mux.HandleFunc("GET /v1/admin/proposals", h.list)
	mux.HandleFunc("POST /v1/admin/proposals/{id}/approve", h.decide(true))
	mux.HandleFunc("POST /v1/admin/proposals/{id}/reject", h.decide(false))
}

func groupsOf(s tenancy.Scope) []string {
	var out []string
	for _, g := range s.Groups() {
		out = append(out, "group:"+g)
	}
	return out
}

// stewardOf reads the applied manifest and checks the caller is one of space name's
// stewards, answering the request itself when not.
func (h Stewards) stewardOf(w http.ResponseWriter, r *http.Request, name string) (tenancy.Scope, manifest.Tenant, bool) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return s, manifest.Tenant{}, false
	}
	cur, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return s, cur, false
	}
	sp, ok := cur.Spaces[name]
	if !ok {
		fail(w, errs.New(errs.ESPACE, "No space %q for you.", name))
		return s, cur, false
	}
	dec := h.Policy.Authorise(r.Context(), s, "propose_space", policy.Entity{Type: "SpaceConfig", ID: name, Tenant: h.Tenant,
		Attrs: map[string]any{"stewards": sp.Stewards}}, map[string]any{"groups": groupsOf(s)})
	if dec.Decision != policy.Allow {
		fail(w, errs.New(errs.EPERM, "Only the space's stewards can propose changes to it."))
		return s, cur, false
	}
	return s, cur, true
}

func (h Stewards) propose(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	var req struct {
		Space json.RawMessage `json:"space"`
	}
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil || len(req.Space) == 0 {
		fail(w, errs.New(errs.EINVAL, "The body must be {\"space\": {…}}."))
		return
	}
	h.submit(w, r, s, cur, name, req.Space)
}

// MaxSpellings is the most other spellings a suggested glossary term takes (SB-3107).
const MaxSpellings = 10

// proposeTerm is Suggest a term (SB-3107): a proposal of the space as applied plus one
// glossary term and its other spellings, made and checked as any steward proposal.
func (h Stewards) proposeTerm(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	var req struct {
		Term      string   `json:"term"`
		Meaning   string   `json:"meaning"`
		Spellings []string `json:"spellings"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil || strings.TrimSpace(req.Term) == "" || strings.TrimSpace(req.Meaning) == "" {
		fail(w, errs.New(errs.EINVAL, "The body must be JSON with a term and its meaning."))
		return
	}
	if len(req.Spellings) > MaxSpellings {
		fail(w, errs.New(errs.EINVAL, "At most %d other spellings.", MaxSpellings))
		return
	}
	sp := cur.Spaces[name]
	g := map[string]string{}
	for k, v := range sp.Glossary {
		g[k] = v
	}
	meaning := strings.TrimSpace(req.Meaning)
	for _, t := range append([]string{req.Term}, req.Spellings...) {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		for k := range g {
			if strings.EqualFold(k, t) {
				fail(w, errs.New(errs.EINVAL, "The glossary already has %q.", k))
				return
			}
		}
		g[t] = meaning
	}
	sp.Glossary = g
	raw, err := json.Marshal(sp)
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The space's definition can't be written."))
		return
	}
	h.submit(w, r, s, cur, name, raw)
}

// submit checks a proposed definition of space name against the applied manifest, and
// audits and stores it as an open proposal.
func (h Stewards) submit(w http.ResponseWriter, r *http.Request, s tenancy.Scope, cur manifest.Tenant, name string, proposed json.RawMessage) {
	change := h.Change
	if change == nil {
		change = manifest.StewardChange
	}
	next, ds, err := change(cur, name, proposed)
	switch {
	case err != nil:
		fail(w, errs.New(errs.EMANIFEST, "%v", err))
		return
	case len(ds) > 0:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errs.HTTP(errs.EMANIFEST))
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": "EMANIFEST", "message": manifest.Count(len(ds)), "errors": ds, "remedy": errs.Remedy(errs.EMANIFEST)}})
		return
	}
	p := proposals.Proposal{ID: h.NewID("prop_"), Space: name, Requester: s.User(), Base: proposals.Hash(cur), Hash: proposals.Hash(next),
		Manifest: string(manifest.Format(next)), Diff: manifest.Diff(cur, next), State: proposals.Open, At: h.Now().UTC()}
	if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: p.ID, State: "proposed",
		PayloadSHA256: release.SHA256([]byte(p.Manifest)), Reason: "steward proposal for space " + name, Space: "spc_" + name}); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The proposal couldn't be audited, so it wasn't made."))
		return
	}
	if err := h.Proposals.Put(r.Context(), p); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The proposal couldn't be stored."))
		return
	}
	pl, _ := plan.Make(next, plan.Desired(cur), true)
	writeJSON(w, map[string]any{"id": p.ID, "space": name, "base": p.Base, "hash": p.Hash, "diff": p.Diff, "plan": pl.Text()})
}

// scenarioVersion is space name as manifest t makes it, for a scenario (SB-4904).
func scenarioVersion(t manifest.Tenant, name string) ask.ScenarioVersion {
	v := ask.ScenarioVersion{Agents: append([]string{}, t.Spaces[name].Agents...), Skills: []ask.Skill{}}
	for n, sk := range t.Effective(name).Skills {
		v.Skills = append(v.Skills, ask.Skill{Name: n, Instruction: sk.Instruction})
	}
	sort.Slice(v.Skills, func(i, j int) bool { return v.Skills[i].Name < v.Skills[j].Name })
	return v
}

// scenarios runs a steward's scenarios against space name as applied and as open proposal
// id would make it, as the steward, and keeps the ones asked for as eval cases (SB-4904).
// The proposal keeps each scenario's pass or fail, never an answer.
func (h Stewards) scenarios(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	if h.Scenarios == nil || h.ScopeIn == nil {
		fail(w, errs.New(errs.EUNAVAIL, "This cell can't run scenarios."))
		return
	}
	var req struct {
		Scenarios []ask.Scenario `json:"scenarios"`
		Keep      []int          `json:"keep"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		fail(w, errs.New(errs.EINVAL, `The body must be JSON such as {"scenarios": [{"question": "…", "cites": "docs:…", "never": "…"}], "keep": [1]}.`))
		return
	}
	p, found, err := h.Proposals.Get(r.Context(), r.PathValue("id"))
	switch {
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "Proposals can't be read."))
		return
	case !found || p.State != proposals.Open || p.Space != name:
		fail(w, errs.New(errs.EINVAL, "No open proposal %q for space %s.", r.PathValue("id"), name))
		return
	}
	next, ds := manifest.Check([]byte(p.Manifest))
	if len(ds) > 0 {
		fail(w, errs.New(errs.EMANIFEST, "The proposal's manifest no longer checks."))
		return
	}
	in, err := h.ScopeIn(r.Context(), s.User(), "spc_"+name)
	if err != nil {
		fail(w, errs.New(errs.EPERM, "You can't ask in space %s now.", name))
		return
	}
	res, err := h.Scenarios.RunScenarios(r.Context(), in, name, req.Scenarios, scenarioVersion(cur, name), scenarioVersion(next, name))
	if err != nil {
		fail(w, questionErr(err))
		return
	}
	for _, k := range req.Keep {
		if k < 1 || k > len(res) {
			fail(w, errs.New(errs.EINVAL, "keep lists scenarios by number, from 1 to %d.", len(res)))
			return
		}
		ec, err := h.Scenarios.KeepScenario(r.Context(), in, name, res[k-1])
		if err != nil {
			fail(w, questionErr(err))
			return
		}
		res[k-1].Kept = ec.ID
	}
	marks, passed := make([]proposals.ScenarioMark, 0, len(res)), 0
	for _, x := range res {
		marks = append(marks, proposals.ScenarioMark{Question: x.Question, Current: x.Current.Pass, Proposed: x.Proposed.Pass})
		if x.Proposed.Pass {
			passed++
		}
	}
	if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: p.ID, State: "scenarios",
		Reason: fmt.Sprintf("%d of %d scenarios pass on the proposal", passed, len(res)), Space: "spc_" + name}); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The run couldn't be audited."))
		return
	}
	if _, err := h.Proposals.Update(r.Context(), p.ID, func(x *proposals.Proposal) error {
		x.Scenarios, x.ScenariosBy = marks, s.User()
		return nil
	}); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The proposal couldn't be stored."))
		return
	}
	writeJSON(w, map[string]any{"id": p.ID, "results": res, "passed": passed, "of": len(res)})
}

// lift lifts one member's personal limit for the month, for a steward of the space, audited
// first (SB-5003).
func (h Stewards) lift(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, _, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	if h.Lift == nil {
		fail(w, errs.New(errs.EUNAVAIL, "This cell has no personal limits to lift."))
		return
	}
	var req struct {
		Person string `json:"person"`
		Reason string `json:"reason"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || json.Unmarshal(b, &req) != nil || strings.TrimSpace(req.Person) == "" || strings.TrimSpace(req.Reason) == "" {
		fail(w, errs.New(errs.EINVAL, `The body must be {"person": "EMAIL", "reason": "…"}.`))
		return
	}
	user, month, err := h.Lift(r.Context(), name, strings.TrimSpace(req.Person), func(user, month string) error {
		_, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "access", Subject: "limit:" + user, State: "lifted",
			Reason: "personal limit lifted for " + month + ": " + strings.TrimSpace(req.Reason), Space: "spc_" + name})
		return err
	})
	switch {
	case errors.Is(err, ask.ErrNotMember):
		fail(w, errs.New(errs.EINVAL, "No active member of %s has that address.", name))
		return
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "The limit can't be lifted right now; nothing was lifted."))
		return
	}
	writeJSON(w, map[string]any{"user": user, "month": month, "lifted": true})
}

func (h Stewards) operator(w http.ResponseWriter, r *http.Request) (tenancy.Scope, bool) {
	return requireOperator(w, r, h.Policy, h.Tenant)
}

func (h Stewards) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.operator(w, r); !ok {
		return
	}
	all, err := h.Proposals.List(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Proposals can't be read."))
		return
	}
	open := []proposals.Proposal{}
	for _, p := range all {
		if p.State == proposals.Open {
			open = append(open, p)
		}
	}
	writeJSON(w, map[string]any{"items": open})
}

func (h Stewards) decide(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := h.operator(w, r)
		if !ok {
			return
		}
		p, found, err := h.Proposals.Get(r.Context(), r.PathValue("id"))
		switch {
		case err != nil:
			fail(w, errs.New(errs.EUNAVAIL, "Proposals can't be read."))
			return
		case !found || p.State != proposals.Open:
			fail(w, errs.New(errs.EINVAL, "No open proposal %q.", r.PathValue("id")))
			return
		}
		dec := h.Policy.Authorise(r.Context(), s, "approve_space", policy.Entity{Type: "SpaceProposal", ID: p.ID, Tenant: h.Tenant,
			Attrs: map[string]any{"requester": policy.Ref{Type: "User", ID: p.Requester}}}, nil)
		if dec.Decision != policy.Allow {
			fail(w, errs.New(errs.EPERM, "You can't decide on this proposal: %s.", strings.TrimSpace(dec.Reason)))
			return
		}
		var req struct{ Hash, Reason string }
		b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
		if err != nil || json.Unmarshal(b, &req) != nil {
			fail(w, errs.New(errs.EINVAL, "The body must be JSON."))
			return
		}
		if !approve && req.Reason == "" {
			fail(w, errs.New(errs.EINVAL, "A rejection needs a reason."))
			return
		}
		if approve && req.Hash != p.Hash {
			fail(w, errs.New(errs.EINVAL, "That hash isn't the proposal's; review the proposal again."))
			return
		}
		cur, _, err := h.Manifests.Applied(r.Context())
		if err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
			return
		}
		// One decision per proposal, audited before it is stored, under the store's lock
		// (SB-1401): a second decision finds it no longer open.
		var fault *errs.Error
		p, err = h.Proposals.Update(r.Context(), p.ID, func(x *proposals.Proposal) error {
			if x.State != proposals.Open {
				fault = errs.New(errs.EINVAL, "No open proposal %q.", x.ID)
				return fault
			}
			state, reason := proposals.Rejected, "reject steward proposal "+x.ID+": "+req.Reason
			if approve {
				state, reason = proposals.Approved, "approve steward proposal "+x.ID
				if proposals.Hash(cur) != x.Base {
					state, reason = proposals.Stale, "steward proposal "+x.ID+" is stale: the manifest changed since it was made"
				}
			}
			if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: x.ID, State: state,
				PayloadSHA256: release.SHA256([]byte(x.Manifest)), Reason: reason, Space: "spc_" + x.Space}); err != nil {
				fault = errs.New(errs.EUNAVAIL, "The decision couldn't be audited, so it wasn't made.")
				return fault
			}
			x.State, x.DecidedBy = state, s.User()
			return nil
		})
		switch {
		case fault != nil:
			fail(w, fault)
			return
		case err != nil:
			fail(w, errs.New(errs.EUNAVAIL, "The decision couldn't be stored."))
			return
		case p.State == proposals.Stale:
			fail(w, errs.New(errs.EDRIFT, "The manifest changed since this proposal was made; the steward must propose again."))
			return
		}
		out := map[string]any{"id": p.ID, "approved": approve}
		if approve {
			out["manifest"] = p.Manifest
		}
		writeJSON(w, out)
	}
}

// spaceChecks is GET /v1/spaces/NAME/checks, for its stewards: the space's checks (SB-5005).
func (h Stewards) spaceChecks(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, _, ok := h.stewardOf(w, r, name); !ok {
		return
	}
	cs := []ask.Check{}
	if h.Checks != nil {
		var err error
		if cs, err = h.Checks(name); err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "Checks can't be read right now."))
			return
		}
	}
	writeJSON(w, map[string]any{"checks": cs})
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Agent and skill history (sb-agents(7), Agent history; SB-5104): every version, a diff
// between any two, and Restore as a proposal.

// ErrNoVersion is a version number the history doesn't have.
var ErrNoVersion = errors.New("api: no such version")

// historyBody is a history, with the changes between from and to when both are given.
func historyBody(w http.ResponseWriter, r *http.Request, vs []proposals.Version) (map[string]any, bool) {
	out := map[string]any{"versions": vs}
	fs, ts := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if fs == "" && ts == "" {
		return out, true
	}
	f, err1 := strconv.Atoi(fs)
	t, err2 := strconv.Atoi(ts)
	if err1 != nil || err2 != nil || f < 1 || t < 1 || f > len(vs) || t > len(vs) {
		fail(w, errs.New(errs.EINVAL, "from and to must be version numbers from 1 to %d.", len(vs)))
		return nil, false
	}
	out["changes"] = proposals.Changes(vs[f-1], vs[t-1])
	return out, true
}

// version reads the path's version number from vs.
func version(w http.ResponseWriter, r *http.Request, vs []proposals.Version) (proposals.Version, bool) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 || n > len(vs) {
		fail(w, errs.New(errs.EINVAL, "No version %q.", r.PathValue("n")))
		return proposals.Version{}, false
	}
	if vs[n-1].Current {
		fail(w, errs.New(errs.EINVAL, "Version %d is the one applied now.", n))
		return proposals.Version{}, false
	}
	return vs[n-1], true
}

// agentHistory is GET /v1/admin/agents/NAME/history[?from=&to=].
func (h Admin) agentHistory(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.AgentHistory == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Agent history isn't available in this cell."))
		return
	}
	vs, err := h.AgentHistory(r.Context(), r.PathValue("name"))
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The agent's history can't be read right now."))
		return
	}
	if out, ok := historyBody(w, r, vs); ok {
		h.ok(w, out)
	}
}

// restoreAgent is POST /v1/admin/agents/NAME/history/N/restore: a proposal of version N's
// settings, for another operator to approve.
func (h Admin) restoreAgent(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	if h.AgentHistory == nil || h.Restore == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Agent history isn't available in this cell."))
		return
	}
	name := r.PathValue("name")
	vs, err := h.AgentHistory(r.Context(), name)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The agent's history can't be read right now."))
		return
	}
	v, ok := version(w, r, vs)
	if !ok {
		return
	}
	p, err := h.Restore(r.Context(), s, "company", func(applied manifest.Tenant) manifest.Tenant {
		return proposals.RestoreAgent(applied, v.Manifest, name)
	}, "restore "+name+" to version "+strconv.Itoa(v.N))
	if err != nil {
		h.fail(w, mapErr(err))
		return
	}
	h.ok(w, map[string]any{"id": p.ID, "diff": p.Diff, "hash": p.Hash})
}

// skillHistory is GET /v1/spaces/NAME/skills/SKILL/history[?from=&to=], for the space's
// stewards.
func (h Stewards) skillHistory(w http.ResponseWriter, r *http.Request) {
	name, skill := r.PathValue("name"), r.PathValue("skill")
	_, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	vs, ok := h.skillVersions(w, r, cur, name, skill)
	if !ok {
		return
	}
	if out, ok := historyBody(w, r, vs); ok {
		writeJSON(w, out)
	}
}

// restoreSkill is POST /v1/spaces/NAME/skills/SKILL/history/N/restore: a steward proposal
// of version N, reviewed like any other.
func (h Stewards) restoreSkill(w http.ResponseWriter, r *http.Request) {
	name, skill := r.PathValue("name"), r.PathValue("skill")
	s, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	vs, ok := h.skillVersions(w, r, cur, name, skill)
	if !ok {
		return
	}
	v, ok := version(w, r, vs)
	if !ok {
		return
	}
	sp := cur.Spaces[name]
	skills := map[string]manifest.Skill{}
	for k, x := range sp.Skills {
		skills[k] = x
	}
	if old, had := v.Manifest.Spaces[name].Skills[skill]; had {
		skills[skill] = old
	} else {
		delete(skills, skill)
	}
	sp.Skills = skills
	raw, err := json.Marshal(sp)
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The space's definition can't be written."))
		return
	}
	h.submit(w, r, s, cur, name, raw)
}

func (h Stewards) skillVersions(w http.ResponseWriter, r *http.Request, cur manifest.Tenant, name, skill string) ([]proposals.Version, bool) {
	ps, err := h.Proposals.List(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Proposals can't be read."))
		return nil, false
	}
	vs := proposals.History(ps, cur, func(t manifest.Tenant) []string { return proposals.SkillLines(t, name, skill) })
	if len(vs) == 1 && len(vs[0].Lines) == 0 {
		fail(w, errs.New(errs.EINVAL, "No skill %q in %s, now or before.", skill, name))
		return nil, false
	}
	return vs, true
}

// RestoreFunc makes a proposal of what change does to the applied manifest, as s, for
// space ("company" for the tenant's own settings), audited with reason.
type RestoreFunc func(ctx context.Context, s tenancy.Scope, space string, change func(manifest.Tenant) manifest.Tenant, reason string) (proposals.Proposal, error)

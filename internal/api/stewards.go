package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/release"
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
}

// Register mounts the endpoints.
func (h Stewards) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/spaces/{name}/proposals", h.propose)
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

func (h Stewards) propose(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	name := r.PathValue("name")
	cur, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return
	}
	sp, ok := cur.Spaces[name]
	if !ok {
		fail(w, errs.New(errs.ESPACE, "No space %q for you.", name))
		return
	}
	dec := h.Policy.Authorise(r.Context(), s, "propose_space", policy.Entity{Type: "SpaceConfig", ID: name, Tenant: h.Tenant,
		Attrs: map[string]any{"stewards": sp.Stewards}}, map[string]any{"groups": groupsOf(s)})
	if dec.Decision != policy.Allow {
		fail(w, errs.New(errs.EPERM, "Only the space's stewards can propose changes to it."))
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
	change := h.Change
	if change == nil {
		change = manifest.StewardChange
	}
	next, ds, err := change(cur, name, req.Space)
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

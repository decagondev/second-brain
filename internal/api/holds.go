package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"

	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Holds serves legal holds (sb-admin(3), Legal holds; SB-1207): Legal stewards place and
// release them; operators can read them.
type Holds struct {
	Tenant    string
	Policy    policy.Kernel
	Manifests Manifests
	Service   holds.Service
}

// Register mounts the endpoints.
func (h Holds) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/holds", h.place)
	mux.HandleFunc("POST /v1/holds/{id}/release", h.release)
	mux.HandleFunc("GET /v1/holds", h.list)
}

// LegalStewards are the stewards of every space whose template is legal.
func LegalStewards(t manifest.Tenant) []string {
	set := map[string]bool{}
	for _, sp := range t.Spaces {
		if sp.Template == "legal" {
			for _, g := range sp.Stewards {
				set[g] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for g := range set {
		out = append(out, g)
	}
	sort.Strings(out)
	return out
}

// legal authorises a Legal steward for action (place_hold or release_hold).
func (h Holds) legal(w http.ResponseWriter, r *http.Request, action string) (tenancy.Scope, manifest.Tenant, bool) {
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
	dec := h.Policy.Authorise(r.Context(), s, action, policy.Entity{Type: "LegalHold", ID: h.Tenant, Tenant: h.Tenant,
		Attrs: map[string]any{"stewards": LegalStewards(cur)}}, map[string]any{"groups": groupsOf(s)})
	if dec.Decision != policy.Allow {
		fail(w, errs.New(errs.EPERM, "Only a Legal steward can do this."))
		return s, cur, false
	}
	return s, cur, true
}

func (h Holds) place(w http.ResponseWriter, r *http.Request) {
	s, cur, ok := h.legal(w, r, "place_hold")
	if !ok {
		return
	}
	var req struct {
		Matter, Space, Reason string
		Users                 []string
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		fail(w, errs.New(errs.EINVAL, "The body must be {\"matter\", \"space\", \"users\", \"reason\"}."))
		return
	}
	if _, exists := cur.Spaces[req.Space]; req.Space != "" && !exists {
		fail(w, errs.New(errs.EINVAL, "No space %q.", req.Space))
		return
	}
	hold, err := h.Service.Place(r.Context(), s.User(), req.Matter, req.Space, req.Users, req.Reason)
	switch {
	case errors.Is(err, holds.ErrInvalid):
		fail(w, errs.New(errs.EINVAL, "%v", err))
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "The hold couldn't be placed: %v", err))
	default:
		writeJSON(w, hold)
	}
}

func (h Holds) release(w http.ResponseWriter, r *http.Request) {
	s, _, ok := h.legal(w, r, "release_hold")
	if !ok {
		return
	}
	var req struct{ Reason string }
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		fail(w, errs.New(errs.EINVAL, "The body must be {\"reason\": \"…\"}."))
		return
	}
	hold, err := h.Service.Release(r.Context(), s.User(), r.PathValue("id"), req.Reason)
	switch {
	case errors.Is(err, holds.ErrInvalid), errors.Is(err, holds.ErrState):
		fail(w, errs.New(errs.EINVAL, "%v", err))
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "The hold couldn't be released: %v", err))
	default:
		writeJSON(w, hold)
	}
}

func (h Holds) list(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	cur, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return
	}
	legal := h.Policy.Authorise(r.Context(), s, "place_hold", policy.Entity{Type: "LegalHold", ID: h.Tenant, Tenant: h.Tenant,
		Attrs: map[string]any{"stewards": LegalStewards(cur)}}, map[string]any{"groups": groupsOf(s)}).Decision == policy.Allow
	op := isOperator(r, h.Policy, s, h.Tenant)
	if !legal && !op {
		fail(w, errs.New(errs.EPERM, "Only Legal stewards and operators can read holds."))
		return
	}
	all, err := h.Service.Store.List(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Holds can't be read."))
		return
	}
	writeJSON(w, map[string]any{"items": all})
}

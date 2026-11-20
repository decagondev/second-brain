package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Spaces serves GET /v1/spaces (sb-ask(3), Spaces; SB-1007): the spaces the asker is a
// member of, now. Nothing about any other space is ever returned.
type Spaces struct {
	Store ports.SpaceStore // nil is a cell without spaces
}

// SpaceItem is one listed space.
type SpaceItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

// Register mounts the endpoint on mux.
func (h Spaces) Register(mux *http.ServeMux) { mux.HandleFunc("GET /v1/spaces", h.list) }

func (h Spaces) list(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	out := []SpaceItem{}
	if h.Store != nil {
		all, err := h.Store.Spaces(r.Context(), s.Tenant())
		if err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "Spaces are unavailable right now."))
			return
		}
		for _, sp := range all {
			if sp.Tenant == s.Tenant() && spaces.Member(sp, s.Groups()) {
				out = append(out, SpaceItem{Name: sp.Name, Label: sp.Label})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	_ = json.NewEncoder(w).Encode(map[string]any{"spaces": out})
}

func fail(w http.ResponseWriter, e *errs.Error) {
	w.WriteHeader(errs.HTTP(e.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
		"code": string(e.Code), "message": e.Msg, "remedy": errs.Remedy(e.Code)}})
}

// writeJSON writes v as the response body.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// isOperator reports whether s holds the operator role in tenant's cell (sb-admin(3)).
func isOperator(r *http.Request, pol policy.Kernel, s tenancy.Scope, tenant string) bool {
	return pol.Authorise(r.Context(), s, "admin", policy.Entity{Type: "Cell", ID: tenant, Tenant: tenant}, nil).Decision == policy.Allow
}

// requireOperator answers EAUTH or EPERM unless the caller is an operator.
func requireOperator(w http.ResponseWriter, r *http.Request, pol policy.Kernel, tenant string) (tenancy.Scope, bool) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return s, false
	}
	if !isOperator(r, pol, s, tenant) {
		fail(w, errs.New(errs.EPERM, "You need the operator role for this."))
		return s, false
	}
	return s, true
}

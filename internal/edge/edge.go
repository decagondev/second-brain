// Package edge is the edge router (sb-tenancy(7), Registry and routing; SB-105). It maps
// a request's host to exactly one cell through the registry, checks that the token
// belongs to that cell's tenant, and forwards to that cell only. Unknown is refusal.
package edge

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/errs"
)

// TokenVerifier checks a request's credentials and returns the tenant they belong to.
// SB-106 provides the OIDC one.
type TokenVerifier func(r *http.Request) (tenant string, err error)

// Forwarder returns the handler that serves one cell.
type Forwarder func(c ports.CellRecord) http.Handler

// Router serves every request in the order of sb-tenancy(7).
type Router struct {
	Cells   ports.Cells
	Verify  TokenVerifier
	Forward Forwarder
}

// Host normalises a Host header: lower case, no port, no trailing dot.
func Host(h string) string {
	if hp, _, err := net.SplitHostPort(h); err == nil {
		h = hp
	}
	return strings.TrimSuffix(strings.ToLower(h), ".")
}

// Decide returns the cell for r, or the coded error that refuses it.
func (rt Router) Decide(r *http.Request) (ports.CellRecord, *errs.Error) {
	host := Host(r.Host)
	c, found, err := rt.Cells.CellByHost(r.Context(), host)
	switch {
	case err != nil:
		return ports.CellRecord{}, errs.New(errs.EUNAVAIL, "tenant registry unavailable")
	case !found || host == "":
		return ports.CellRecord{}, errs.New(errs.EPERM, "no cell serves this host")
	}
	tenant, err := rt.Verify(r)
	if err != nil || tenant == "" {
		return ports.CellRecord{}, errs.New(errs.EAUTH, "sign in again")
	}
	if tenant != c.Tenant {
		return ports.CellRecord{}, errs.New(errs.ETENANT, "this token belongs to another tenant")
	}
	return c, nil
}

// ServeHTTP refuses with a JSON error (sb-api(3) error shape) or forwards to the cell.
func (rt Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, e := rt.Decide(r)
	if e != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errs.HTTP(e.Code))
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
			"code": string(e.Code), "message": e.Msg, "remedy": errs.Remedy(e.Code)}})
		return
	}
	rt.Forward(c).ServeHTTP(w, r)
}

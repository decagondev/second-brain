// Package tenancy is the tenant scope every cell operation carries (sb-tenancy(7),
// SB-108). A Scope can only be made by an Issuer, and a cell has one Issuer, for its
// own tenant, held by the auth middleware.
package tenancy

import (
	"context"
	"errors"
)

// Scope is who is asking, in which tenant. Its fields are unexported, so no code can
// make one up; the zero Scope is invalid.
type Scope struct {
	tenant, user string
	groups       []string
}

// Tenant is the scope's tenant; the only value repositories may filter rows by.
func (s Scope) Tenant() string { return s.tenant }

// User is the signed-in user's id.
func (s Scope) User() string { return s.user }

// Groups are the user's directory groups at the time the scope was issued.
func (s Scope) Groups() []string { return append([]string(nil), s.groups...) }

// Valid is false for the zero Scope.
func (s Scope) Valid() bool { return s.tenant != "" && s.user != "" }

// ErrInvalid is returned by every scoped method given an invalid Scope.
var ErrInvalid = errors.New("tenancy: no valid scope")

// Check returns ErrInvalid for an invalid Scope. Scoped methods call it first.
func (s Scope) Check() error {
	if !s.Valid() {
		return ErrInvalid
	}
	return nil
}

// Issuer makes scopes for one tenant. Only the auth middleware may hold one
// (enforced in CI: tools/check/deps.py).
type Issuer struct{ tenant string }

// NewIssuer makes the cell's one Issuer, at start-up.
func NewIssuer(cellTenant string) Issuer { return Issuer{tenant: cellTenant} }

// ErrOtherTenant means a session of another tenant reached this cell.
var ErrOtherTenant = errors.New("tenancy: session belongs to another tenant")

// Issue makes the scope for a signed-in user of this cell's tenant.
func (i Issuer) Issue(sessionTenant, user string, groups []string) (Scope, error) {
	switch {
	case i.tenant == "":
		return Scope{}, ErrInvalid
	case sessionTenant != i.tenant:
		return Scope{}, ErrOtherTenant
	case user == "":
		return Scope{}, ErrInvalid
	}
	return Scope{tenant: i.tenant, user: user, groups: append([]string(nil), groups...)}, nil
}

type ctxKey struct{}

// WithScope returns ctx carrying s.
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

// FromContext returns the request's scope, or ErrInvalid if there is none.
func FromContext(ctx context.Context) (Scope, error) {
	s, ok := ctx.Value(ctxKey{}).(Scope)
	if !ok || !s.Valid() {
		return Scope{}, ErrInvalid
	}
	return s, nil
}

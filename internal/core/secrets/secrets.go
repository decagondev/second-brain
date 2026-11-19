// Package secrets resolves vault references (sb-tenant(5), Secrets; SB-205). Only the
// documented paths are valid, only for the cell's own tenant, and every value resolved
// is registered with the log redactor before the caller sees it (LAW-07).
package secrets

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"example.com/second-brain/internal/core/ports"
)

var refRe = regexp.MustCompile(`^vault://tenants/([a-z0-9][a-z0-9-]{1,38}[a-z0-9])/(connectors/(docs|chat|tickets|crm)|oidc|scim|model|canary)$`)

// Ref is a parsed, valid vault reference.
type Ref struct{ Raw, Tenant, Path string }

// ErrBadRef is a reference outside the documented paths.
var ErrBadRef = errors.New("secrets: not a valid vault reference")

// ParseRef validates a reference against sb-tenant(5).
func ParseRef(raw string) (Ref, error) {
	m := refRe.FindStringSubmatch(raw)
	if m == nil {
		return Ref{}, fmt.Errorf("%w: %q", ErrBadRef, raw)
	}
	return Ref{Raw: raw, Tenant: m[1], Path: m[2]}, nil
}

// ConnectorRef is the reference for a connector kind's credential.
func ConnectorRef(tenant, kind string) string {
	return "vault://tenants/" + tenant + "/connectors/" + kind
}

// Registrar learns secret values so the logger can redact them.
type Registrar interface{ Register(secret string) }

// Resolver resolves references for one cell's tenant.
type Resolver struct {
	Tenant string
	Vault  ports.Secrets
	Redact Registrar
}

// ErrOtherTenant is a reference to another tenant's vault path.
var ErrOtherTenant = errors.New("secrets: reference belongs to another tenant")

// Get resolves raw at call time. The value is registered for redaction before it is
// returned; callers must not keep it beyond the call.
func (r Resolver) Get(ctx context.Context, raw string) (string, error) {
	ref, err := ParseRef(raw)
	if err != nil {
		return "", err
	}
	if ref.Tenant != r.Tenant {
		return "", ErrOtherTenant
	}
	if r.Redact == nil {
		return "", errors.New("secrets: no redactor; refusing to hand out a secret that logs could leak")
	}
	v, err := r.Vault.Get(ctx, raw)
	if err != nil {
		return "", fmt.Errorf("secrets: %s: %w", raw, err) // the reference, never the value
	}
	if v == "" {
		return "", fmt.Errorf("secrets: %s is empty", raw)
	}
	r.Redact.Register(v)
	return v, nil
}

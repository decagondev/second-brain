package cell

import (
	"regexp"
	"strings"
)

// The custody record's values (sb-factory(7), ADR-0015). This is the only file that may
// spell them; tools/check/custody.py fails on these literals anywhere else.
const (
	CustodyFactory  = "factory"
	CustodyTenant   = "tenant"
	CustodyDetached = "detached"

	// PlacementFactoryAccount is v1's only placement: the factory's own account.
	PlacementFactoryAccount = "factory-account"

	KeyCustodianFactory = "factory"
	KeyCustodianTenant  = "tenant"
)

// TenantAccountPrefix starts a tenant-account placement, tenant-account://PROVIDER/ID.
const TenantAccountPrefix = "tenant-account://"

// ParseTenantAccount checks a tenant-account placement and returns its provider and id.
func ParseTenantAccount(s string) (provider, id string, ok bool) {
	rest, found := strings.CutPrefix(s, TenantAccountPrefix)
	if !found {
		return "", "", false
	}
	provider, id, found = strings.Cut(rest, "/")
	if !found || !safeName.MatchString(provider) || !safeName.MatchString(id) {
		return "", "", false
	}
	return provider, id, true
}

var safeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

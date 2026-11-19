package cell

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

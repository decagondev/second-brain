package ports

import "context"

// Space is one department space in a cell's registry (sb-spaces(7), SB-1002): every field
// of sb-space(5) after the template's defaults are filled in. ID is "spc_" + Name.
type Space struct {
	ID, Tenant, Name, Template, Label string
	Members, Stewards, Agents         []string
	Scope                             map[string][]string
	RetentionDays                     int
	Channels                          []string            // chat channels that ask in this space (SB-1007)
	Fields                            map[string][]string // kind -> sensitive fields allowed (SB-1106)
}

// SpaceStore is a cell's space registry. Only `sb tenant apply` writes it (spaces.Sync);
// requests only read it. An error is never "no such space".
type SpaceStore interface {
	PutSpace(ctx context.Context, s Space) error
	DeleteSpace(ctx context.Context, tenant, id string) error
	Space(ctx context.Context, tenant, id string) (s Space, found bool, err error)
	Spaces(ctx context.Context, tenant string) ([]Space, error)
}

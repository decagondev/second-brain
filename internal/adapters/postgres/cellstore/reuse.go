package cellstore

import (
	"context"

	"example.com/second-brain/internal/core/ask"
)

var _ ask.ReuseStore = Traces{}

// PutReuse implements ask.ReuseStore (SB-5603): the newest per key.
func (t Traces) PutReuse(ctx context.Context, e ask.ReuseEntry) error {
	return put(ctx, t.Pool, e.Tenant, "reuse", e.Key, meta{at: e.At}, e, false)
}

// Reuse implements ask.ReuseStore.
func (t Traces) Reuse(ctx context.Context, tenant, key string) (ask.ReuseEntry, bool, error) {
	return get[ask.ReuseEntry](ctx, t.Pool, tenant, "reuse", key, false)
}

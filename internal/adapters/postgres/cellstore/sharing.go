package cellstore

import (
	"context"
	"time"

	"example.com/second-brain/internal/core/ask"
)

var (
	_ ask.WideConfirmStore = Traces{}
	_ ask.SharingStore     = Traces{}
)

// PutWideConfirm implements ask.WideConfirmStore (SB-5402): one per document.
func (t Traces) PutWideConfirm(ctx context.Context, c ask.WideConfirm) error {
	return put(ctx, t.Pool, c.Tenant, "wideconfirm", c.Doc, meta{at: c.At, user: c.By}, c, false)
}

// WideConfirm implements ask.WideConfirmStore.
func (t Traces) WideConfirm(ctx context.Context, tenant, doc string) (ask.WideConfirm, bool, error) {
	return get[ask.WideConfirm](ctx, t.Pool, tenant, "wideconfirm", doc, false)
}

// PutSharingEvent implements ask.SharingStore (SB-5403).
func (t Traces) PutSharingEvent(ctx context.Context, e ask.SharingEvent) error {
	return put(ctx, t.Pool, e.Tenant, "sharing", e.ID, meta{at: e.At}, e, true)
}

// SharingEvents implements ask.SharingStore: those at or after since, oldest first.
func (t Traces) SharingEvents(ctx context.Context, tenant string, since time.Time) ([]ask.SharingEvent, error) {
	return list[ask.SharingEvent](ctx, t.Pool, tenant, "sharing", "at >= $3", "at, id", since)
}

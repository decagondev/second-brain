package wire

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditfs "example.com/second-brain/internal/adapters/audit/localfs"
	"example.com/second-brain/internal/adapters/connectors/slack"
	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	pgaudit "example.com/second-brain/internal/adapters/postgres/audit"
	"example.com/second-brain/internal/adapters/postgres/cellstore"
	pgindex "example.com/second-brain/internal/adapters/postgres/index"
	pgspaces "example.com/second-brain/internal/adapters/postgres/spaces"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/toolbus"
)

// DirectoryStore is the directory as the cell uses it: SCIM writes it, sign-in, policy,
// the chat app, barriers, reviews and the report read it.
type DirectoryStore interface {
	ports.DirectoryStore
	ports.People
	ports.UserFinder
	Members(ctx context.Context, tenant, name string) ([]string, error)
	ActiveUsers(ctx context.Context, tenant string) (int, error)
	ActiveUserIDs(ctx context.Context, tenant string) ([]string, error)
}

// TraceStore is traces, conversations and feedback.
type TraceStore interface {
	ask.Store
	ask.FeedbackStore
	ask.SpaceTagger
	Retention() []retention.Store
	Between(ctx context.Context, tenant string, from, to time.Time) ([]ask.Trace, error)
}

// ActionStore is the tool bus's actions, with the backup's full list.
type ActionStore interface {
	approvals.Store
	All(ctx context.Context, tenant string) ([]approvals.Action, error)
}

// IndexStore is the index both ingest writes and retrieval searches.
type IndexStore interface {
	ports.Index
	ports.Retriever
}

// stores is every store the cell keeps records in.
type stores struct {
	audit     audit.Store
	directory DirectoryStore
	sessions  ports.Sessions
	spaces    ports.SpaceStore
	index     IndexStore
	traces    TraceStore
	holds     holds.Store
	actions   ActionStore
	receipts  toolbus.Receipts
	proposals proposals.Store
	reviews   reviews.Store
	inbox     slack.Inbox
	close     func()
}

// memoryStores are the in-memory stores (development and tests; sb-cell(5), Stores).
func memoryStores(cfg Config) stores {
	ix := memindex.New().Bind(cfg.Tenant)
	return stores{audit: auditfs.Dir{Root: filepath.Join(cfg.State, "audit"), Tenant: cfg.Tenant}, directory: dirmem.New(), sessions: sessmem.New(),
		spaces: spacemem.New(), index: ix, traces: tracemem.New(), holds: &holds.MemoryStore{}, actions: actmem.New(),
		receipts: actmem.NewReceipts(), proposals: &proposals.MemoryStore{}, reviews: &reviews.MemoryStore{}, inbox: &slack.MemoryInbox{},
		close: func() {}}
}

// postgresStores are the durable stores in the cell's own database (SB-1508), each table
// created if it isn't there.
func postgresStores(ctx context.Context, cfg Config) (stores, error) {
	pool, err := cellstore.Open(ctx, cfg.Database)
	if err != nil {
		return stores{}, fmt.Errorf("%w: database: %v", ErrConfig, err)
	}
	db := cellstore.DB{Pool: pool, Tenant: cfg.Tenant}
	au := pgaudit.Store{Pool: pool, Tenant: cfg.Tenant}
	ix := pgindex.Index{Pool: pool, Tenant: cfg.Tenant, Dim: hashing.Dim}
	sp := pgspaces.DB{Pool: pool}
	// The services of one cell start together: one migrates at a time.
	unlock, err := cellstore.Lock(ctx, pool, "sb-cell-migrate")
	if err != nil {
		pool.Close()
		return stores{}, fmt.Errorf("%w: database: %v", ErrConfig, err)
	}
	defer unlock()
	for name, mig := range map[string]func(context.Context) error{"records": db.Migrate, "audit": au.Migrate, "index": ix.Migrate, "spaces": sp.Migrate} {
		if err := mig(ctx); err != nil {
			pool.Close()
			return stores{}, fmt.Errorf("%w: database: %s tables: %v", ErrConfig, name, err)
		}
	}
	return stores{audit: au, directory: cellstore.Directory{DB: db}, sessions: cellstore.Sessions{DB: db}, spaces: sp, index: ix,
		traces: cellstore.Traces{DB: db}, holds: cellstore.Holds{DB: db}, actions: cellstore.Actions{DB: db},
		receipts: cellstore.Receipts{DB: db}, proposals: cellstore.Proposals{DB: db}, reviews: cellstore.Reviews{DB: db},
		inbox: queueInbox{cellstore.Queue{DB: db, Kind: "slack-events"}}, close: pool.Close}, nil
}

// queueInbox is the Slack events inbox over a durable queue.
type queueInbox struct{ q cellstore.Queue }

func (i queueInbox) Append(ctx context.Context, e slack.Event) error { return i.q.Append(ctx, e) }
func (i queueInbox) Len(ctx context.Context) (int, error)            { return i.q.Len(ctx) }
func (i queueInbox) Read(ctx context.Context, from int) ([]slack.Event, int, error) {
	raw, next, err := i.q.Read(ctx, from)
	if err != nil {
		return nil, from, err
	}
	out := make([]slack.Event, 0, len(raw))
	for _, r := range raw {
		var e slack.Event
		if err := json.Unmarshal(r, &e); err != nil {
			return nil, from, err
		}
		out = append(out, e)
	}
	return out, next, nil
}

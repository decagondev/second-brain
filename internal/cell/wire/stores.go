package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditfs "example.com/second-brain/internal/adapters/audit/localfs"
	backupfs "example.com/second-brain/internal/adapters/backup/localfs"
	"example.com/second-brain/internal/adapters/backup/s3"
	"example.com/second-brain/internal/adapters/connectors/slack"
	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	"example.com/second-brain/internal/adapters/embed/hashing"
	embedoai "example.com/second-brain/internal/adapters/embed/openaicompat"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	pgaudit "example.com/second-brain/internal/adapters/postgres/audit"
	"example.com/second-brain/internal/adapters/postgres/cellstore"
	pgindex "example.com/second-brain/internal/adapters/postgres/index"
	pgspaces "example.com/second-brain/internal/adapters/postgres/spaces"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/health"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/toolbus"
	"sort"
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
	ask.FeedbackReader
	ask.SpaceTagger
	ask.QuestionStore
	ask.StewardStore
	notify.PrefStore
	ask.WatchStore
	ask.EvalStore
	ask.AccessStore
	ask.MarkStore
	ask.TemplateStore
	ask.RetireStore
	ask.NoticeStore
	ask.SuggestedStore
	ask.PinStore
	ask.CaretakerStore
	ask.PolicyStore
	ask.ResearchStore
	ask.ActionStore
	ask.RoutineStore
	ask.MemoryStore
	ask.CollectionStore
	ask.ShadowStore
	ask.RhythmStore
	ask.ProjectStore
	ask.WideConfirmStore
	ask.SharingStore
	ask.ReuseStore
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
func postgresStores(ctx context.Context, cfg Config) (st stores, err error) {
	dsn := cfg.Database
	if strings.HasPrefix(dsn, "vault://") {
		if dsn != "vault://tenants/"+cfg.Tenant+"/database" {
			return stores{}, fmt.Errorf("%w: database: only vault://tenants/%s/database names the DSN", ErrConfig, cfg.Tenant)
		}
		if dsn, err = (localvault.Dir{Root: cfg.Vault}).Get(ctx, dsn); err != nil {
			return stores{}, fmt.Errorf("%w: database: %v", ErrConfig, err)
		}
		dsn = strings.TrimSpace(dsn)
	}
	pool, err := cellstore.Open(ctx, dsn)
	if err != nil {
		return stores{}, fmt.Errorf("%w: database: %v", ErrConfig, err)
	}
	db := cellstore.DB{Pool: pool, Tenant: cfg.Tenant}
	au := pgaudit.Store{Pool: pool, Tenant: cfg.Tenant}
	ix := pgindex.Index{Pool: pool, Tenant: cfg.Tenant, Dim: embedDim(cfg)}
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

// WormTarget is the cell's write-once store: the configured bucket, or files under state
// for development (sb-cell(5), worm). A bucket's lock is checked before each use, so a
// store whose lock was loosened is refused, never written to.
func (c *Cell) WormTarget(ctx context.Context) (backup.Target, error) {
	w := c.Config.Worm
	if w == (WormConfig{}) {
		return backupfs.Dir{Root: filepath.Join(c.Config.State, "worm"), Label: "worm", Media: "disk"}, nil
	}
	b := s3.Bucket{Endpoint: w.Endpoint, BucketName: w.Bucket, Region: w.Region, Creds: c.Secret("worm"), Label: "worm", Away: w.Offsite,
		HTTP: &http.Client{Timeout: time.Minute}}
	if err := b.Check(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

// OffsiteBucket is a backups[] s3 copy (SB-2303): the company's bucket, with no object
// lock, its store and key pair at vault://tenants/TENANT/offsite. It is checked before each
// use; the caller seals every object to it.
func (c *Cell) OffsiteBucket(ctx context.Context, b Backup) (s3.Bucket, error) {
	name, prefix, err := s3.URL(b.S3)
	if err != nil {
		return s3.Bucket{}, err
	}
	bk := s3.Bucket{BucketName: name, Prefix: prefix, Plain: true, Creds: c.Secret("offsite"), Label: b.S3, Away: b.Offsite,
		HTTP: &http.Client{Timeout: time.Minute}}
	if err := bk.Check(ctx); err != nil {
		return s3.Bucket{}, err
	}
	return bk, nil
}

// Health is the cell's health from its own records (sb-admin(3); SB-1607): the job
// finishes every service writes and the sync records the ingest service writes.
func (c *Cell) Health(ctx context.Context, now time.Time) (health.Report, error) {
	kinds := make([]string, 0, len(c.Parts().Connectors))
	for k := range c.Parts().Connectors {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	r, err := health.Build(ctx, c.Runs, c.Syncs, kinds, now)
	if err != nil {
		return r, err
	}
	r.Box = c.box(now)
	r.Model = c.ModelState() // SB-5601, SB-5602
	if via := c.channel(); via != "" {
		rec, found, err := lastNotify(c.Config.State)
		if err != nil {
			found = false // unreadable: as if it hadn't run, a warning
		}
		r.Notify = health.NotifyFrom(rec, found, via, now)
	}
	l, found := lastLeftovers(c.Config.State)
	r.Index = health.IndexFrom(l.At, l.Gone, l.Unheld, found, now)
	if c.Usage != nil {
		week, err := c.Usage.InputTokens(ctx, c.Config.Tenant, now.Add(-7*24*time.Hour), now)
		if err != nil {
			return r, err
		}
		before, err := c.Usage.InputTokens(ctx, c.Config.Tenant, now.Add(-14*24*time.Hour), now.Add(-7*24*time.Hour))
		if err != nil {
			return r, err
		}
		r.Prompt = health.PromptFrom(week, before)
		wa, err := c.Usage.AnswerMillis(ctx, c.Config.Tenant, now.Add(-7*24*time.Hour), now)
		if err != nil {
			return r, err
		}
		ba, err := c.Usage.AnswerMillis(ctx, c.Config.Tenant, now.Add(-14*24*time.Hour), now.Add(-7*24*time.Hour))
		if err != nil {
			return r, err
		}
		r.AnswerTime = health.AnswerFrom(wa, ba)
	}
	r.Unstewarded = c.Unstewarded(ctx)
	return r, nil
}

// box judges the host step's facts on a self-hosted cell, whose host step keeps them under
// the state's host/ directory (SB-1802); a cell with no such directory has no box.
func (c *Cell) box(now time.Time) *health.Box {
	dir := filepath.Join(c.Config.State, "host")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(dir, "facts.json"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return health.BoxFrom(nil, true, now) // there, but unreadable: Unknown
	}
	if err == nil {
		b = c.withDiskHistory(b, now)
	}
	return health.BoxFrom(b, err == nil, now)
}

// embedderFor is the configured embedder (sb-cell(5), embed; SB-1710).
func embedderFor(cfg Config) ports.Embedder {
	if cfg.Embed.Provider == "openaicompat" {
		return embedoai.Embedder{Base: cfg.Embed.Base, Name: cfg.Embed.Model, Dim: cfg.Embed.Dim, HTTP: &http.Client{Timeout: 2 * time.Minute}}
	}
	return hashing.Embedder{}
}

// embedDim is the configured embedder's vector size, which the Postgres index is made with.
func embedDim(cfg Config) int {
	if cfg.Embed.Provider == "openaicompat" {
		return cfg.Embed.Dim
	}
	return hashing.Dim
}

// syncLag is the answer's connector kinds past the freshness target, or unknown (SB-2704):
// the same rule as the health report's stale connectors.
func (c *Cell) syncLag(ctx context.Context, kinds []string) ([]ask.SyncNotice, error) {
	now := time.Now()
	var out []ask.SyncNotice
	conns := c.Parts().Connectors
	for _, k := range kinds {
		if _, ok := conns[k]; !ok {
			continue // not a connector this cell syncs
		}
		st, found, err := c.Syncs.Sync(ctx, k)
		switch {
		case err != nil || !found || st.LastSuccess.IsZero():
			out = append(out, ask.SyncNotice{Kind: k})
		case now.Sub(st.LastSuccess) > health.FreshnessSLA:
			at := st.LastSuccess.UTC()
			out = append(out, ask.SyncNotice{Kind: k, LastSuccess: &at})
		}
	}
	return out, nil
}

// stewardDir is the directory reads a steward check needs.
type stewardDir interface {
	Members(ctx context.Context, tenant, name string) ([]string, error)
	User(ctx context.Context, tenant, id string) (u ports.User, found bool, err error)
}

// unstewarded says whether a space (spc_NAME) has no active member in any of its steward
// groups now (SB-3102).
func (c *Cell) unstewarded(ctx context.Context, space string) (none, known bool, err error) {
	none, known = stewardless(ctx, c.Directory, c.Config.Tenant, c.Manifest(), space)
	return none, known, nil
}

// stewardless is unstewarded on a directory and a manifest. A group or user the directory
// can't answer for, or a steward that isn't a group, makes it unknown: never "has a
// steward" and never "has none".
func stewardless(ctx context.Context, dir stewardDir, tenant string, m manifest.Tenant, space string) (none, known bool) {
	sp, ok := m.Spaces[strings.TrimPrefix(space, "spc_")]
	if !ok || len(sp.Stewards) == 0 || dir == nil {
		return false, false
	}
	known = true
	for _, g := range sp.Stewards {
		name, ok := strings.CutPrefix(g, "group:")
		if !ok {
			return false, false
		}
		ids, err := dir.Members(ctx, tenant, name)
		if err != nil {
			known = false
			continue
		}
		for _, id := range ids {
			u, found, err := dir.User(ctx, tenant, id)
			if err != nil {
				known = false
				continue
			}
			if found && u.Active {
				return false, true
			}
		}
	}
	return known, known
}

// Unstewarded is health's spaces with no active steward, and those it couldn't tell (SB-3102).
func (c *Cell) Unstewarded(ctx context.Context) *health.Unstewarded {
	return unstewardedOf(ctx, c.Directory, c.Config.Tenant, c.Manifest())
}

func unstewardedOf(ctx context.Context, dir stewardDir, tenant string, m manifest.Tenant) *health.Unstewarded {
	if len(m.Spaces) == 0 {
		return nil
	}
	out := &health.Unstewarded{Spaces: []string{}, Unknown: []string{}}
	names := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if len(m.Spaces[n].Stewards) == 0 {
			continue
		}
		none, known := stewardless(ctx, dir, tenant, m, "spc_"+n)
		switch {
		case !known:
			out.Unknown = append(out.Unknown, n)
		case none:
			out.Spaces = append(out.Spaces, n)
		}
	}
	out.Warning = len(out.Spaces)+len(out.Unknown) > 0
	return out
}

// activeByEmail are the active user ids with email, for the audit API's actor= (SB-3204).
func (c *Cell) activeByEmail(ctx context.Context, email string) ([]string, error) {
	if c.Directory == nil {
		return nil, errors.New("wire: no directory")
	}
	us, err := c.Directory.UsersByEmail(ctx, c.Config.Tenant, email)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, u := range us {
		if u.Active {
			out = append(out, u.ID)
		}
	}
	return out, nil
}

// metricsHandler serves /metrics with health's lines as gauges, read on each scrape
// (SB-3203).
func (c *Cell) metricsHandler() http.Handler {
	inner := c.Metrics.Handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.healthGauges(r.Context(), time.Now())
		inner.ServeHTTP(w, r)
	})
}

// healthGauges sets sb_health_warning and sb_health_unknown for every line of health; a
// report that can't be built is unknown on check "health", never fine.
func (c *Cell) healthGauges(ctx context.Context, now time.Time) {
	set := func(check string, warn, unknown bool) {
		l := map[string]string{"check": check}
		c.Metrics.Set("sb_health_warning", "A health line that warns, 1, or not, 0 (SB-3203).", l, one(warn))
		c.Metrics.Set("sb_health_unknown", "A health line the cell can't read, 1 (SB-3203).", l, one(unknown))
	}
	r, err := c.Health(ctx, now)
	if err != nil {
		set("health", false, true)
		return
	}
	set("health", false, false)
	for _, ch := range r.Checks() {
		set(ch.Name, ch.Warning, ch.Unknown)
	}
}

func one(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// diskHistoryPath is the box's disk samples, a day each for 14 days (SB-3206).
func diskHistoryPath(state string) string { return filepath.Join(state, "box", "disk.json") }

// withDiskHistory adds today's disk use to the kept samples and puts them in the facts, so
// the Box section judges the runway; facts it can't read are judged as they are.
func (c *Cell) withDiskHistory(raw []byte, now time.Time) []byte {
	f, err := box.Parse(raw)
	if err != nil || now.Sub(f.At) > box.FactsMaxAge {
		return raw
	}
	p := diskHistoryPath(c.Config.State)
	var hist []box.Sample
	if old, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(old, &hist) // an unreadable file starts again
	}
	next := box.AddSamples(hist, f.Disks, f.At)
	if !reflect.DeepEqual(next, hist) {
		if b, err := json.Marshal(next); err == nil && os.MkdirAll(filepath.Dir(p), 0o700) == nil {
			if os.WriteFile(p+".tmp", b, 0o600) == nil {
				_ = os.Rename(p+".tmp", p)
			}
		}
	}
	f.DiskHistory = next
	out, err := json.Marshal(f)
	if err != nil {
		return raw
	}
	return out
}

// noticeLine is the admin's notice as the chat apps' first line, or "" (SB-3207).
func (c *Cell) noticeLine(ctx context.Context) string {
	if c.Traces == nil {
		return ""
	}
	return ask.NoticeLine(ctx, c.Traces, c.Config.Tenant, time.Now())
}

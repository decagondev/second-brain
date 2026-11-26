package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"example.com/second-brain/internal/adapters/alert/webhook"
	"example.com/second-brain/internal/adapters/chatapp/mmapp"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/adapters/connectors/bamboohr"
	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/connectors/gcal"
	"example.com/second-brain/internal/adapters/connectors/gdrive"
	"example.com/second-brain/internal/adapters/connectors/github"
	"example.com/second-brain/internal/adapters/connectors/hubspot"
	"example.com/second-brain/internal/adapters/connectors/linear"
	"example.com/second-brain/internal/adapters/connectors/quickbooks"
	"example.com/second-brain/internal/adapters/connectors/slack"
	cursorfs "example.com/second-brain/internal/adapters/cursor/filestore"
	killfs "example.com/second-brain/internal/adapters/kill/localfs"
	logfs "example.com/second-brain/internal/adapters/logs/localfs"
	"example.com/second-brain/internal/adapters/model/anthropic"
	"example.com/second-brain/internal/adapters/model/extractive"
	"example.com/second-brain/internal/adapters/model/groq"
	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/adapters/model/openrouter"
	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/adapters/policy/cedar"
	runsfs "example.com/second-brain/internal/adapters/runs/localfs"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	usagefs "example.com/second-brain/internal/adapters/usage/localfs"
	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/barrier"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/costs"
	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/reorg"
	"example.com/second-brain/internal/core/report"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/session"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
	"example.com/second-brain/internal/core/workflow"
	"example.com/second-brain/internal/obs"
	"example.com/second-brain/internal/scim"
	"example.com/second-brain/internal/version"
)

// MemoryStores are the stores kept in memory when the configuration names no database
// (sb-cell(5), Stores); a durable cell needs a database for them.
var MemoryStores = []string{"directory and sessions", "traces and conversations", "actions and receipts", "spaces registry",
	"index", "chat events inbox", "proposals", "access reviews", "legal holds", "audit log (in files)"}

// Vendors are the connector adapters a cell can run, by the manifest's vendor name.
var Vendors = map[string]string{"gdrive": "docs", "folder": "docs", "slack": "chat", "linear": "tickets", "hubspot": "crm", "github": "code",
	"bamboohr": "hris", "quickbooks": "accounting"}

// Cell is a built cell: every part, and the API handler that serves them.
type Cell struct {
	Config  Config
	Release string
	// asks counts each person's asks for asks_per_minute (SB-2305), kept across a
	// manifest apply.
	asks ask.Rate
	// Send is where the API's own notifications go, such as access requests (SB-2205);
	// nil is Notifier().
	Send notify.Sender

	Audit     *audit.Log
	Kill      *kill.Checker
	Policy    policy.Kernel
	Directory DirectoryStore
	Sessions  ports.Sessions
	Spaces    ports.SpaceStore
	Index     IndexStore
	Traces    TraceStore
	Holds     holds.Store
	Metrics   *obs.Metrics
	Auth      *auth.Cell

	// The parts the ingest and jobs services run (SB-1503, SB-1504).
	Cursors   ports.CursorStore
	Runs      runsfs.Dir
	Syncs     runsfs.Syncs // each connector's last sync (SB-1607)
	Usage     *usagefs.Ledger
	Report    report.Source // the weekly report's numbers
	Reviews   reviews.Store
	Proposals proposals.Store
	Actions   ActionStore
	Logs      *logfs.Files
	// Embedder is the configured embedder (sb-cell(5), embed); ingest and asks share it.
	Embedder ports.Embedder
	Redact   *obs.Redactor
	// Secret reads vault://tenants/TENANT/PATH at call time.
	Secret func(path string) func(context.Context) (string, error)
	// HeldNow is the active legal holds, read when asked; holds that can't be read hold
	// everything (PT-0011).
	HeldNow func(ctx context.Context) holds.Set
	NewID   func(prefix string) string
	// Log is the cell's redacting logger; Alerter sends alerts to it and to alerts.webhook.
	Log     *slog.Logger
	Alerter canary.Alerter

	opt      Options
	model    func(m manifest.Tenant) (ports.Model, error)
	small    func(m manifest.Tenant) (ports.Model, error)
	eventKey func(ctx context.Context) (string, error)
	inbox    slack.Inbox
	closeDB  func()
	receipts toolbus.Receipts
	engine   policy.Evaluator
	parts    atomic.Pointer[Parts]
	applyMu  sync.Mutex
	ready    atomic.Bool
}

// Parts are everything built from the applied manifest. An apply builds a new set and
// swaps it in whole (SB-1506), so a request sees one manifest from start to end.
type Parts struct {
	Manifest   manifest.Tenant
	SHA256     string // of the applied manifest's text
	Connectors map[string]ports.Connector
	Agent      ask.Agent
	Bus        *toolbus.Bus
	Retention  retention.Job
	// Stewards is the questions-to-stewards service, where agent drafts wait (SB-4005).
	Stewards ask.Stewards
	// Legwork is research, meeting prep and action items (SB-4201 to SB-4203).
	Legwork ask.Legwork
	// Rhythms are the spaces' playbooks and projects (SB-4801 to SB-4804).
	Rhythms ask.Rhythms
	handler http.Handler
}

// Parts returns the parts built from the manifest the cell runs now.
func (c *Cell) Parts() *Parts { return c.parts.Load() }

// Manifest is the manifest the cell runs now.
func (c *Cell) Manifest() manifest.Tenant { return c.Parts().Manifest }

// Handler serves the whole API (sb-api(3)) with the parts of the manifest applied now.
func (c *Cell) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { c.Parts().handler.ServeHTTP(w, r) })
}

type applied struct {
	t      manifest.Tenant
	commit string
}

func (a applied) Applied(context.Context) (manifest.Tenant, string, error) { return a.t, a.commit, nil }

// usage is the weekly report's source over the cell's own stores (SB-606, SB-1504).
type usage struct {
	traces  TraceStore
	ledger  *usagefs.Ledger
	actions ActionStore
	dir     DirectoryStore
	index   IndexStore
	state   string
	tree    func() manifest.Tenant // the manifest now, for usage by space (SB-3905); nil is flat
}

// ParentOf is a space's department by name, "" for none (SB-3905).
func (u usage) ParentOf(space string) string {
	if u.tree == nil {
		return ""
	}
	return u.tree().Spaces[space].Parent
}

func (u usage) Traces(ctx context.Context, tenant string, from, to time.Time) ([]ask.Trace, error) {
	return u.traces.Between(ctx, tenant, from, to)
}

// LastNotify is the notify job's last run (report.NotifySource; SB-2607).
func (u usage) LastNotify(context.Context) (notify.RunRecord, bool, error) {
	return lastNotify(u.state)
}

// FeedbackBetween is the week's feedback, for usage by space (SB-3506).
func (u usage) FeedbackBetween(ctx context.Context, tenant string, from, to time.Time) ([]ask.Feedback, error) {
	return u.traces.FeedbackBetween(ctx, tenant, from, to)
}

func (u usage) StewardQuestions(ctx context.Context, tenant string, from, to time.Time) ([]ask.StewardQuestion, error) {
	return u.traces.StewardQuestionsBetween(ctx, tenant, from, to)
}

// ChangedDocs counts documents whose version the index first saw in [from, to), by the
// space holding them ("company" for none), never a rebuilt version (SB-1708).
func (u usage) ChangedDocs(ctx context.Context, tenant string, from, to time.Time) (map[string]int, error) {
	lk, ok := u.index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	chunks, err := lk.Docs(ctx, tenant, ports.DocQuery{SeenSince: from})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := map[string]int{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if seen[k] || !c.Seen.Before(to) {
			continue
		}
		seen[k] = true
		if len(c.Spaces) == 0 {
			out["company"]++
		}
		for _, sp := range c.Spaces {
			out[strings.TrimPrefix(sp, "spc_")]++
		}
	}
	return out, nil
}

func (u usage) Drafts(ctx context.Context, tenant string, from, to time.Time) (int, int, error) {
	return u.ledger.Drafts(ctx, tenant, from, to)
}
func (u usage) Tokens(ctx context.Context, tenant string, from, to time.Time) (int, int, error) {
	return u.ledger.Tokens(ctx, tenant, from, to)
}
func (u usage) InputTokens(ctx context.Context, tenant string, from, to time.Time) ([]int, error) {
	return u.ledger.InputTokens(ctx, tenant, from, to)
}
func (u usage) AnswerMillis(ctx context.Context, tenant string, from, to time.Time) ([]int, error) {
	return u.ledger.AnswerMillis(ctx, tenant, from, to)
}

// askMeter is the ask agent's counters: the usage ledger, which keeps each answer's
// duration and forwards everything to the metrics registry (SB-3205), or the registry
// alone when there is no ledger.
func (c *Cell) askMeter() interface {
	Add(name, help string, labels map[string]string, v float64)
	Observe(name, help string, labels map[string]string, v float64)
} {
	if c.Usage != nil {
		return c.Usage
	}
	return c.Metrics
}
func (u usage) Seats(ctx context.Context, tenant string) (int, error) {
	return u.dir.ActiveUsers(ctx, tenant)
}

// Actions are those finished in [from, to), and those still open whose expiry is after from.
func (u usage) Actions(ctx context.Context, tenant string, from, to time.Time) ([]approvals.Action, error) {
	all, err := u.actions.All(ctx, tenant)
	if err != nil {
		return nil, err
	}
	var out []approvals.Action
	for _, a := range all {
		if f := a.FinishedAt; (!f.IsZero() && !f.Before(from) && f.Before(to)) || (f.IsZero() && !a.ExpiresAt.Before(from)) {
			out = append(out, a)
		}
	}
	return out, nil
}

var _ report.Source = usage{}
var _ report.NotifySource = usage{}

// Options are what a test may change about how a cell reaches the outside: the zero value
// is what the services run with.
type Options struct {
	HTTP *http.Client // for the identity provider; nil is http.DefaultClient
	// ConnectorHTTP is the connectors' client; nil is one with a 30-second timeout.
	ConnectorHTTP *http.Client
	// Log is where the cell logs; nil is standard error.
	Log io.Writer
	// EventsHTTP posts the events out (SB-4404); nil is one with a 10-second timeout.
	// Sleep is the wait between retries; nil is time.Sleep.
	EventsHTTP *http.Client
	Sleep      func(time.Duration)
}

// Build builds every part of a cell from its configuration. Anything it can't build stops
// it: a cell never runs with a part switched off (sb-cell(5)).
func Build(ctx context.Context, cfg Config, opt Options) (*Cell, error) {
	if opt.HTTP == nil {
		opt.HTTP = http.DefaultClient
	}
	if cfg.Durable && cfg.Database == "" {
		return nil, fmt.Errorf("%w: durable is true, but with no database these stores are in memory only: %s", ErrConfig, strings.Join(MemoryStores, ", "))
	}
	src, err := os.ReadFile(cfg.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%w: manifest: %v", ErrConfig, err)
	}
	m, err := checkManifest(cfg, src, cfg.Manifest)
	if err != nil {
		return nil, err
	}
	engine, err := loadPolicies(cfg.Policies)
	if err != nil {
		return nil, fmt.Errorf("%w: policies: %v", ErrConfig, err)
	}
	for _, d := range []string{cfg.State, cfg.Vault} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			return nil, fmt.Errorf("%w: %s is not a directory", ErrConfig, d)
		}
	}
	c := &Cell{Config: cfg, Release: "rel_" + version.Version, Metrics: obs.NewMetrics(), opt: opt, engine: engine}
	tenant := cfg.Tenant
	vault := localvault.Dir{Root: cfg.Vault}
	secret := func(path string) func(context.Context) (string, error) {
		return func(ctx context.Context) (string, error) { return vault.Get(ctx, "vault://tenants/"+tenant+"/"+path) }
	}
	redact := obs.NewRedactor()
	n := atomic.Int64{}
	newID := func(p string) string { return fmt.Sprintf("%s%d%06d", p, time.Now().UnixMilli(), n.Add(1)%1000000) }

	st := memoryStores(cfg)
	if cfg.Database != "" {
		if st, err = postgresStores(ctx, cfg); err != nil {
			return nil, err
		}
	}
	c.closeDB = st.close
	c.Audit = &audit.Log{Tenant: tenant, Release: func() string { return c.Release }, Store: st.audit, Now: time.Now, Redact: redact}
	c.Policy = policy.Kernel{Engine: engine, Recorder: c.Audit}
	c.Secret, c.Redact, c.NewID = secret, redact, newID
	if opt.Log == nil {
		opt.Log = os.Stderr
	}
	c.Log = obs.NewLogger(opt.Log, redact)
	al := alerter{log: c.Log, metrics: c.Metrics}
	if cfg.Alerts.Webhook != "" {
		al.webhook = webhook.Sender{URL: cfg.Alerts.Webhook, HTTP: &http.Client{Timeout: 10 * time.Second}}
	}
	c.Alerter = al
	c.Cursors = cursorfs.Store{Root: cfg.State}
	c.Runs = runsfs.Dir{Root: filepath.Join(cfg.State, "runs")}
	c.Syncs = runsfs.Syncs{Root: filepath.Join(cfg.State, "runs")}
	c.Usage = &usagefs.Ledger{Root: filepath.Join(cfg.State, "usage"), Tenant: tenant, Next: c.Metrics, Now: time.Now}
	c.Kill = &kill.Checker{Tenant: tenant, Store: killfs.Dir{Root: filepath.Join(cfg.State, "kill")}, Now: time.Now}
	if err := c.Kill.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("%w: kill switches: %v", ErrConfig, err)
	}
	c.Directory, c.Sessions, c.Spaces, c.Index, c.Traces, c.Holds = st.directory, st.sessions, st.spaces, st.index, st.traces, st.holds
	c.Embedder = embedderFor(cfg)
	c.HeldNow = func(ctx context.Context) holds.Set {
		set, err := holds.Active(ctx, c.Holds)
		if err != nil {
			return holds.Set{{Space: ""}} // holds can't be read: everything is held
		}
		return set
	}
	if err := spaces.SyncHeld(ctx, c.Spaces, nil, m, func(id string) bool { return c.HeldNow(ctx).HeldSpace(id) }); err != nil {
		return nil, fmt.Errorf("%w: spaces: %v", ErrConfig, err)
	}
	if len(m.Groups) > 0 {
		if _, err := groupsync.Apply(ctx, c.Directory, c.Sessions, tenant, nil, m.Groups); err != nil {
			return nil, fmt.Errorf("%w: groups: %v", ErrConfig, err)
		}
	}
	c.inbox, c.receipts, c.Actions, c.Proposals, c.Reviews = st.inbox, st.receipts, st.actions, st.proposals, st.reviews
	c.Logs = &logfs.Files{Dir: filepath.Join(cfg.State, "logs"), Now: time.Now}
	c.Report = usage{traces: c.Traces, ledger: c.Usage, actions: c.Actions, dir: c.Directory, index: c.Index, state: c.Config.State, tree: c.Manifest}
	c.eventKey = secret("events") // SB-4404
	tierOf := func(m manifest.Tenant, small bool) (ports.Model, error) {
		mc := cfg
		if small { // SB-4405: the same provider, base and key, another id and price
			mc.Model.ID, mc.Model.Price.InPerM, mc.Model.Price.OutPerM = cfg.Model.Small.ID, cfg.Model.Small.Price.InPerM, cfg.Model.Small.Price.OutPerM
		}
		guard := &budget.Guard{Tenant: tenant, MonthlyUSD: float64(m.BudgetUSDMonth), Meter: c.Usage.Spend(), Metrics: c.Metrics, Now: time.Now,
			Price: budget.Price{InputPerM: mc.Model.Price.InPerM, OutputPerM: mc.Model.Price.OutPerM}, Once: c.Usage,
			Forecast: func(ctx context.Context) (float64, bool) {
				prices := modelPrices(cfg)
				f, ok, err := costs.Forecast(ctx, c.Usage, tenant, prices, time.Now())
				return f, ok && err == nil
			},
			// The warning goes through whichever alerter the cell has when it fires (SB-1608).
			Warn: func(ctx context.Context, month string, _ float64) {
				if c.Alerter != nil {
					_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev3", Kind: "budget_warning", Tenant: tenant, Where: "budget", Ref: month})
				}
			}}
		return buildModel(mc, tenant, secret, redact, c.Usage, guard)
	}
	c.model = func(m manifest.Tenant) (ports.Model, error) { return tierOf(m, false) }
	c.small = func(m manifest.Tenant) (ports.Model, error) {
		if cfg.Model.Small.ID == "" {
			return nil, nil
		}
		return tierOf(m, true)
	}

	// Sign-in (sb-identity(7)).
	issuer := cfg.OIDC.Issuer
	if issuer == "" {
		if issuer, err = oidc.IssuerFromIdP(m.IdP); err != nil {
			return nil, fmt.Errorf("%w: idp: %v", ErrConfig, err)
		}
	}
	provider, err := oidc.Discover(ctx, issuer, cfg.OIDC.ClientID, tenant, opt.HTTP, time.Now)
	if err != nil {
		return nil, fmt.Errorf("%w: the identity provider can't be reached: %v", ErrConfig, err)
	}
	if _, _, ip := manifest.IdentityPlatform(m.IdP); ip {
		if err := provider.IdentityPlatform(m.IdP); err != nil {
			return nil, fmt.Errorf("%w: identity platform: %v", ErrConfig, err)
		}
	}
	c.Auth = auth.WithIssuer(&auth.Cell{Tenant: tenant, Provider: provider, Sessions: c.Sessions, Directory: c.Directory, Secret: secret("oidc"),
		Redirect: strings.TrimRight(cfg.BaseURL, "/") + "/callback", Now: time.Now,
		GroupsOf: c.Directory.GroupsOf, Spaces: c.Spaces, Attributes: auth.DirectoryAttributes(c.Directory), Audit: c.Audit})

	p, err := c.assemble(m, sha(src))
	if err != nil {
		return nil, err
	}
	c.parts.Store(p)
	c.ready.Store(true)
	return c, nil
}

// assemble builds every part that comes from the manifest, without changing anything the
// cell runs: Build and Apply swap the result in.
func (c *Cell) assemble(m manifest.Tenant, digest string) (*Parts, error) {
	cfg, tenant, secret, redact, newID := c.Config, c.Config.Tenant, c.Secret, c.Redact, c.NewID
	if m.Connectors["chat"] == "slack" && cfg.Chat.TeamID == "" {
		return nil, fmt.Errorf("%w: the manifest's chat is slack, so chat.team_id is required", ErrConfig)
	}
	conns, err := connectors(cfg, m, secret, redact, c.inbox, c.opt.ConnectorHTTP)
	if err != nil {
		return nil, err
	}
	model, err := c.model(m)
	if err != nil {
		return nil, err
	}
	small, err := c.small(m)
	if err != nil {
		return nil, err
	}
	spaceFacts := func(ctx context.Context) ([]ports.Space, [][2]string, error) {
		all, err := c.Spaces.Spaces(ctx, tenant)
		return all, m.Barriers, err
	}

	// Asking (sb-ask(3)).
	authz := retrieve.Authoriser{Connectors: conns, Policy: c.Policy, Killed: c.Kill.ConnectorKilled, Spaces: spaceFacts,
		SpaceKilled: c.Kill.SpaceKilled, Owners: barrier.Directory(c.Directory, tenant), Metrics: c.Metrics}
	emails := func(ctx context.Context, user string) ([]string, error) {
		u, found, err := c.Directory.User(ctx, tenant, user)
		if err != nil || !found {
			return nil, err
		}
		return u.Emails, nil
	}
	agent := ask.Agent{Tenant: tenant, Release: c.Release, Index: c.Index, Embedder: c.Embedder, Auth: authz,
		Model: model, Store: c.Traces, Emails: emails, Killed: c.Kill.AgentKilled("ask"), SpaceKilled: c.Kill.SpaceKilled,
		NewID: newID, Now: time.Now, Metrics: c.askMeter(), Enabled: func(a string) bool { return contains(m.Agents, a) }, Audit: c.Audit,
		Link: linker(authz.Connectors), Caretaker: ask.CaretakerOf(c.Traces, tenant),
		SpaceOf: func(ctx context.Context, id string) (ports.Space, bool, error) {
			return c.Spaces.Space(ctx, tenant, id)
		},
		AgentKilled: func(a string) bool { return c.Kill.AgentKilled(a)(context.Background()) },
		AgentPaused: c.Kill.Paused, AgentCapped: c.agentCapped, // SB-4702, SB-4703
		Small: small, SmallHeld: c.smallHeld, PinMain: func(key string) bool { return m.Spaces[strings.TrimPrefix(key, "spc_")].Model == "main" },
		Departmental: departmental(m),
		Updates:      func(name string) bool { return m.Spaces[name].Updates },
		LeaderWeek:   m.LeaderWeek, // SB-4503
		ReviewAfter: func(ids []string) int {
			least := 0
			for _, id := range ids {
				d := m.Effective(strings.TrimPrefix(id, "spc_")).ReviewAfterDays
				if d == 0 {
					d = manifest.DefaultReviewAfterDays
				}
				if least == 0 || d < least {
					least = d
				}
			}
			return least
		},
		Owner: ask.ActiveOwner(func(ctx context.Context, email string) ([]ports.User, error) {
			return c.Directory.UsersByEmail(ctx, tenant, email)
		}),
		Glossary: func(name string) map[string]string { return m.Effective(name).Glossary },
		Skills:   func(name string) []ask.Skill { return spaceSkills(m, name) },                      // SB-4601
		Routing:  routing(m), Suspect: suspectMode(m), Guard: c.guardEvent, PersonOver: c.personOver, // SB-5002, SB-5003
		Rate: &c.asks, AsksPerMinute: func() int { return m.AsksPerMinute },
		Marks: c.Traces, SyncLag: c.syncLag, Retired: c.Traces, Pins: c.Traces, FileText: folder.Text,
		Climb: func(ctx context.Context, space string) []ask.PersonRef { // SB-3906: up to the department's stewards
			up := parentOf(m)(space)
			if up == "" {
				return nil
			}
			ps, err := c.stewardPeople(m)(ctx, up)
			if err != nil {
				return nil // unknown names no one; never a guess
			}
			var out []ask.PersonRef
			for _, p := range ps {
				out = append(out, ask.PersonRef{Email: p.Email, Title: p.Title, Steward: true})
			}
			if len(out) > ask.MaxPeople {
				out = out[:ask.MaxPeople]
			}
			return out
		}}
	if m.MemoryOn() { // SB-4303: memory: off keeps and sends none
		agent.Memory = c.Traces
	}
	agent.Collections = c.Traces          // SB-4304
	agent.Requests = requestCategories(m) // SB-4502

	// Actions (sb-actions(3)).
	tools, err := workflow.Registry(m)
	if err != nil {
		return nil, fmt.Errorf("%w: workflows: %v", ErrConfig, err)
	}
	if m.Notes {
		tools[ask.NoteTool] = toolbus.Tool{Name: ask.NoteTool, Connector: "docs"} // a steward's publish (SB-1705)
	}
	locate := func(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
		conn, ok := conns[ref.Kind]
		if !ok {
			return ports.Document{}, fmt.Errorf("no connector for %s", ref.Kind)
		}
		return conn.Fetch(ctx, ref)
	}
	bus := &toolbus.Bus{Tenant: tenant, Tools: tools, Kill: c.Kill, Policy: c.Policy, Audit: c.Audit, Actions: c.Actions,
		Connectors: conns, Now: time.Now, NewID: newID, Receipts: c.receipts,
		Barriers: barrier.Cell{Spaces: spaceFacts, GroupsOf: func(ctx context.Context, u string) ([]string, error) { return c.Directory.GroupsOf(ctx, tenant, u) },
			Owners: barrier.Directory(c.Directory, tenant), Locate: locate},
		Managers: func(ctx context.Context, u string) (string, error) {
			usr, found, err := c.Directory.User(ctx, tenant, u)
			if err != nil || !found {
				return "", err
			}
			return usr.Manager, nil
		}}

	// Retention and erasure, holds always read at the moment they're needed (PT-0011).
	spaceDays := map[string]int{}
	for name, sp := range m.Spaces {
		if sp.RetentionDays > 0 && sp.RetentionDays < m.RetentionDays {
			spaceDays["spc_"+name] = sp.RetentionDays
		}
	}
	ret := retention.Job{Tenant: tenant, Days: m.RetentionDays, Stores: append(c.Traces.Retention(), c.Logs), Audit: c.Audit, Now: time.Now,
		Held: func(space, user string) bool { return c.HeldNow(context.Background()).Held(space, user) }, SpaceDays: spaceDays}

	revoker := session.Revoker{Tenant: tenant, Store: c.Sessions, Audit: c.Audit}
	man := applied{t: m, commit: "sha256:" + digest}
	mux := http.NewServeMux()
	public := http.NewServeMux()
	public.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	public.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !c.ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	public.HandleFunc("GET /login", c.Auth.Login)
	public.HandleFunc("GET /callback", c.Auth.Callback)
	public.Handle("/scim/v2/", (&scim.Endpoint{Tenant: tenant, Directory: c.Directory, Sessions: c.Sessions, Token: secret("scim"), Joined: c.joined}).Handler())
	api.Verifier{Release: c.Release, Metrics: c.metricsHandler(), ControlPlane: c.controlPlane, Attest: c.attestor(), TargetFor: c.VerifyTarget}.Register(public)
	var savers []*slackapp.App // their Saver is the stewards' service, below (SB-3703)
	if sc, ok := conns["chat"].(*slack.Connector); ok {
		public.Handle("POST /connectors/slack/events", sc.EventsHandler(time.Now))
		creds := secret("connectors/chat")
		app := &slackapp.App{Tenant: tenant, TeamID: cfg.Chat.TeamID, Secret: slackField(creds, "signing_secret"),
			Email: slackapp.WebEmail(cfg.Connectors["chat"].Base, slackField(creds, "bot_token"), &http.Client{Timeout: 10 * time.Second}),
			Users: c.Directory, Groups: c.Directory.GroupsOf, Issuer: c.Auth.Issuer, Agent: agent, Spaces: c.Spaces, Now: time.Now,
			Language: func() string { return c.Manifest().Language }, Notice: c.noticeLine,
			Masked: func(ctx context.Context, n int) { c.masked(ctx, "slack", n) }, // SB-5102
			Thread: slackapp.WebThread(cfg.Connectors["chat"].Base, slackField(creds, "bot_token"), &http.Client{Timeout: 10 * time.Second})}
		app.Register(public)
		savers = append(savers, app)
	}
	if mm := cfg.Mattermost; mm.Base != "" {
		creds := secret("mattermost")
		client := &http.Client{Timeout: 10 * time.Second}
		rules := &slackapp.App{Tenant: tenant, Users: c.Directory, Groups: c.Directory.GroupsOf, Issuer: c.Auth.Issuer, Spaces: c.Spaces,
			Email: mmapp.APIEmail(mm.Base, slackField(creds, "bot_token"), client), Language: func() string { return c.Manifest().Language },
			Notice: c.noticeLine}
		(&mmapp.App{Tenant: tenant, Base: mm.Base, TeamID: mm.TeamID, Token: slackField(creds, "command_token"), Rules: rules,
			Agent: agent, ActionURL: strings.TrimSuffix(cfg.BaseURL, "/") + "/chat/mattermost/actions",
			Masked: func(ctx context.Context, n int) { c.masked(ctx, "mattermost", n) }, // SB-5102
			Thread: mmapp.APIThread(mm.Base, slackField(creds, "bot_token"), client)}).Register(public)
		savers = append(savers, rules)
	}
	if cfg.Web != "" {
		api.Web{Files: os.DirFS(cfg.Web), Base: cfg.BaseURL, Language: func() string { return c.Manifest().Language }}.Register(public)
	}
	api.Ask{Agent: agent, Feedback: ask.Feedbacks{Agent: agent, Store: c.Traces}, Release: c.Release, NewID: newID}.Register(mux)
	api.Drafts{Agent: ask.Drafts{Agent: agent, Killed: c.Kill.AgentKilled("draft-reply"), Policy: c.Policy, Metrics: c.Usage},
		Ask: api.Ask{Agent: agent, Release: c.Release, NewID: newID}}.Register(mux)
	api.Actions{Bus: bus, Release: c.Release, NewID: newID}.Register(mux)
	stewards := ask.Stewards{Shadows: c.Traces, FirstRing: func() bool { return c.Manifest().Channel == "testing" }, // SB-4704
		Agent: agent, Store: c.Traces, Audit: c.Audit, Policy: c.Policy, NewID: newID, Now: time.Now,
		Unstewarded: c.unstewarded, Suggested: c.Traces, Zone: workZone(m), Caretakers: c.Traces,
		Facts: stewardFacts(m), Gone: func(ctx context.Context, email string) (bool, error) {
			us, err := c.Directory.UsersByEmail(ctx, tenant, email)
			if err != nil {
				return false, err
			}
			for _, u := range us {
				if u.Active {
					return false, nil
				}
			}
			return true, nil // no active user has it: they left, or were deprovisioned (SB-2201)
		}, Evals: c.Traces, Feedback: c.Traces, Marks: c.Traces, Replies: c.Traces, Claimant: c.claimant,
		StartItems: func(name string) []string { return m.Effective(name).StartHere }, Members: c.noteReaders(m),
		Children: m.Children, Policies: c.Traces, ScopeOf: c.ownerScope, ReviewAfter: func(name string) int { return m.Effective(name).ReviewAfterDays },
		Parent: func(name string) string { return m.Spaces[name].Parent }, StewardPeople: c.stewardPeople(m),
		BudgetShare: c.budgetShare, // SB-4503
		Actions:     c.Traces,      // SB-4504
		Rescope:     c.rescope,     // SB-4602
		Saved:       c.Traces,      // SB-4603
		Requests: func(name string) ([]string, int, bool) { // SB-4104
			if r := m.Spaces[name].Requests; r != nil {
				return r.Categories, r.DueDays, true
			}
			return nil, 0, false
		},
		Owner: func(ctx context.Context, email string) (string, bool, error) { // SB-3706: one active user, or no one
			us, err := c.Directory.UsersByEmail(ctx, tenant, email)
			if err != nil {
				return "", false, err
			}
			id := ""
			for _, u := range us {
				if u.Active {
					if id != "" {
						return "", false, nil
					}
					id = u.ID
				}
			}
			return id, id != "", nil
		}}
	if m.Notes {
		stewards.Bus, stewards.Readers = bus, c.noteReaders(m)
	}
	legwork := c.legwork(m, agent, stewards)
	legwork.Approvals = inboxApprovals(bus)   // SB-4301
	stewards.ActionsOf = legwork.DraftActions // a decisions draft carries its action items (SB-4203)
	for _, app := range savers {
		app.Saver = stewards
	}
	api.Legwork{Service: legwork, Release: c.Release, NewID: newID}.Register(mux)
	rhythms := c.rhythms(m, stewards, legwork, newID) // SB-4801 to SB-4804
	api.Rhythms{Service: rhythms, Release: c.Release, NewID: newID}.Register(mux)
	api.Collections{Service: ask.CollectionService{Agent: agent, Store: c.Traces, Saved: c.Traces, Audit: c.Audit, Known: legwork.Known,
		NewID: newID, Now: time.Now}, Release: c.Release, NewID: newID}.Register(mux) // SB-4304
	api.Questions{Service: stewards, Release: c.Release, NewID: newID, Reviews: c.Traces, TryTrigger: c.tryTrigger, LendAs: c.ScopeAs,
		ReviewAfter: func(name string) int { return m.Effective(name).ReviewAfterDays }}.Register(mux)
	api.Knowledge{Search: agent, Release: c.Release, NewID: newID,
		Start: func(ctx context.Context, s tenancy.Scope) (ask.StartResponse, error) {
			return agent.StartHere(ctx, s, m.Effective(strings.TrimPrefix(s.Space(), "spc_")).StartHere)
		},
		Changes: agent.Changes,
		Access: func(ctx context.Context, s tenancy.Scope, trace string) (ask.AccessResponse, error) {
			return agent.RequestAccess(ctx, s, trace, c.Traces, c.accessNote())
		},
		Handover: func(ctx context.Context, s tenancy.Scope, person string) (ask.HandoverResponse, error) {
			return agent.Handover(ctx, s, person, emails, c.relations(m), c.Policy)
		},
		People: func(ctx context.Context, s tenancy.Scope, q string) ([]ask.Briefable, error) {
			return agent.HandoverPeople(ctx, s, q, c.briefable(m), c.relations(m), c.Policy)
		},
		Questions: ask.Questions{Tenant: tenant, Store: c.Traces, Audit: c.Audit, NewID: newID, Now: time.Now,
			Watches: c.Traces, Sources: c.Sources}}.Register(mux)
	admin := api.Admin{Tenant: tenant, Policy: c.Policy, Manifests: man, Kill: kill.Control{Store: c.Kill.Store, Audit: c.Audit, Now: time.Now},
		Switches: c.Kill.Store, Audit: c.Audit, AuditRows: c.Audit.Store, Release: c.Release, Approvers: approvers(m),
		Usage: c.Report,
		Price: report.Price{InputPerM: cfg.Model.Price.InPerM, OutputPerM: cfg.Model.Price.OutPerM},
		Meter: c.Usage, Prices: modelPrices(cfg), Tiers: modelTiers(cfg), Guardrails: c.Guardrails, GuardrailEvents: c.GuardrailEvents,
		BudgetUSD: m.BudgetUSDMonth, Now: time.Now, Erasure: ret, Sessions: revoker, Tokens: c.tokenLister(), NotifyTest: c.testEmail,
		AgentKilled: c.agentKilled, EvalHeld: c.evalHeld, Index: indexReports{c},
		ActiveByEmail: c.activeByEmail, Notices: c.Traces, SpaceSpend: c.spaceSpend, OwnerActive: c.ownerActive,
		AgentValue: func(ctx context.Context) (ask.ValueReport, error) { return c.agentValue(ctx, nil) }, PausedIn: c.Kill.PausedIn, // SB-4701, SB-4702
		AIUsage: c.aiUsage,                        // SB-5001
		Checks:  c.Checks, Inventory: c.Inventory, // SB-5005, SB-5007
		AgentRecord:  c.AgentRecord,                      // SB-4707
		AgentHistory: c.agentHistory, Restore: c.restore, // SB-5104
		Rollouts: c.Rollouts, HoldFeature: c.holdAdmin} // SB-5106
	admin.InFlight = c.inFlight(m) // SB-4806
	admin.Register(mux)
	api.Sessions{Revoker: revoker}.Register(mux)
	api.Tokens{Sessions: c.Sessions, Audit: c.Audit, Now: time.Now}.Register(mux)
	if cfg.MCP {
		api.MCP{Agent: agent, Release: c.Release, ScopeIn: c.ScopeAs, Legwork: legwork, // SB-4306
			Agents: func(ctx context.Context, s tenancy.Scope) (map[string][]string, error) {
				return agent.UsableAgents(ctx, s, c.ScopeAs)
			}, Skills: agent.UsableSkills, Starters: stewards.UsableStarters}.Register(mux) // SB-4601, SB-4906
	}
	api.Me{Data: ret, Agent: agent, Sessions: c.Sessions, Now: time.Now}.Register(mux)
	api.Notifications{Prefs: c.Traces, Release: c.Release, NewID: newID, Test: c.testSelf}.Register(mux)
	api.Operations{Admin: admin, Feedback: c.Traces, Traces: c.Traces, Health: c.Health}.Register(mux)
	api.Spaces{Store: c.Spaces}.Register(mux)
	api.Stewards{Tenant: tenant, Policy: c.Policy, Manifests: man, Audit: c.Audit, Proposals: c.Proposals, NewID: newID, Now: time.Now,
		Runs: c.Report, AgentKilled: c.agentKilled, EvalHeld: c.evalHeld, OwnerActive: c.ownerActive, PausedIn: c.Kill.PausedIn,
		Scenarios: stewards, ScopeIn: c.ScopeAs, Lift: c.lift, Checks: c.Checks, // SB-4904, SB-5003, SB-5005
		SharedCount: stewards.SharedRoutineCount, Feature: c.featureOn}.Register(mux) // SB-5105, SB-5106
	api.AgentControl{Value: c.ownerValue, Pause: c.pauseAgent, Shadow: stewards.ShadowsFor}.Register(mux) // SB-4701, SB-4702, SB-4704
	api.Reviews{Tenant: tenant, Policy: c.Policy, Manifests: man, Audit: c.Audit, Reviews: c.Reviews, Proposals: c.Proposals,
		NewID: newID, Now: time.Now}.Register(mux)
	api.Holds{Tenant: tenant, Policy: c.Policy, Manifests: man, Service: holds.Service{Store: c.Holds, Audit: c.Audit, NewID: newID, Now: time.Now}}.Register(mux)
	public.HandleFunc("POST /v1/cell/manifest", c.applyHandler)
	public.HandleFunc("GET /v1/cell/live", c.liveHandler)
	public.HandleFunc("GET /v1/cell/audit", c.auditRows)
	public.HandleFunc("GET /v1/cell/audit/export", c.auditExport)
	public.HandleFunc("GET /v1/cell/agents/{name}/record", c.agentRecordCP)  // SB-4707
	public.HandleFunc("GET /v1/cell/rollouts", c.rolloutsCP)                 // SB-5106
	public.HandleFunc("POST /v1/cell/rollouts/{feature}/{op}", c.rolloutsCP) // SB-5106
	public.HandleFunc("GET /v1/cell/inventory", c.inventoryCP)               // SB-5007
	public.HandleFunc("GET /v1/cell/package", c.packageHandler)
	public.Handle("/v1/", c.Auth.Middleware(mux))
	return &Parts{Manifest: m, SHA256: digest, Connectors: conns, Agent: agent, Bus: bus, Retention: ret, Stewards: stewards,
		Legwork: legwork, Rhythms: rhythms, handler: api.MaskOut(public, c.maskedOut)}, nil
}

// legwork is the research, meeting-prep and action-item service (sb-ask(3); SB-4201 to
// SB-4203). Every run is as the person, with a scope read from the directory now.
func (c *Cell) legwork(m manifest.Tenant, agent ask.Agent, stewards ask.Stewards) ask.Legwork {
	tenant := c.Config.Tenant
	zone, _ := manifest.DigestZone(m.Digest)
	l := ask.Legwork{Agent: agent, Stewards: stewards, Research: c.Traces, Actions: c.Traces, Audit: c.Audit, Routines: c.Traces, Zone: zone,
		NewID: c.NewID, Now: time.Now, ResearchUSD: func() float64 { return m.ResearchUSD }, ScopeIn: c.ScopeAs, Tell: c.tell,
		CatchUpOffer: c.catchUpOffer, AccountSpaces: c.accountSpaces, // SB-4506, SB-4605
		Known: func(ctx context.Context, email string) bool {
			ids, err := c.activeByEmail(ctx, email)
			return err == nil && len(ids) == 1 // unknown, or two people: no one (LAW-12)
		},
		Users: func(ctx context.Context) ([]ports.User, error) {
			ids, err := c.Directory.ActiveUserIDs(ctx, tenant)
			if err != nil {
				return nil, err
			}
			var out []ports.User
			for _, id := range ids {
				u, found, err := c.Directory.User(ctx, tenant, id)
				if err != nil {
					return nil, err
				}
				if found && u.Active {
					out = append(out, u)
				}
			}
			return out, nil
		}}
	if cal := c.Config.Calendar; cal.Kind == "google" {
		base := cal.Base
		if base == "" {
			base = "https://www.googleapis.com"
		}
		hc := c.opt.ConnectorHTTP
		if hc == nil {
			hc = &http.Client{Timeout: 30 * time.Second}
		}
		l.Calendar = &gcal.Calendar{Base: base, HTTP: hc, Metrics: c.Metrics,
			Tokens: &gdrive.ServiceAccount{Key: c.Secret("calendar"), HTTP: hc, Redact: c.Redact, Now: time.Now, Scope: gcal.Scope}}
	}
	return l
}

// Ready reports whether every part was built (GET /readyz).
func (c *Cell) Ready() bool { return c.ready.Load() }

// Stop marks the cell not ready, so the platform stops sending it requests.
func (c *Cell) Stop() { c.ready.Store(false) }

// Close releases the cell's database connections, after it has stopped serving.
func (c *Cell) Close() {
	if c.closeDB != nil {
		c.closeDB()
	}
}

func loadPolicies(dir string) (policy.Evaluator, error) {
	files := map[string][]byte{}
	for _, glob := range []string{"platform/*.cedar", "*.cedar"} {
		ms, _ := filepath.Glob(filepath.Join(dir, glob))
		for _, p := range ms {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(dir, p)
			files[filepath.ToSlash(rel)] = b
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no policies in %s", dir)
	}
	return cedar.Load(files)
}

func connectors(cfg Config, m manifest.Tenant, secret func(string) func(context.Context) (string, error), redact *obs.Redactor, inbox slack.Inbox,
	hc *http.Client) (map[string]ports.Connector, error) {
	out := map[string]ports.Connector{}
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	for kind, vendor := range m.Connectors {
		base := cfg.Connectors[kind].Base
		creds := secret("connectors/" + kind)
		switch vendor {
		case "gdrive":
			cls, err := classification(secret("connectors/" + kind + "-classification"))
			if err != nil {
				return nil, fmt.Errorf("%w: connector %s: %v", ErrConfig, kind, err)
			}
			out[kind] = &gdrive.Connector{Base: base, HTTP: hc, Classification: cls,
				Tokens: &gdrive.ServiceAccount{Key: creds, HTTP: hc, Redact: redact, Now: time.Now}}
		case "folder":
			u, err := url.Parse(base)
			if err != nil || u.Scheme != "file" || u.Path == "" {
				return nil, fmt.Errorf("%w: connector %s: folder needs connectors.%s.base as file:///DIR", ErrConfig, kind, kind)
			}
			c, err := folder.Open(u.Path, filepath.Join(cfg.State, "connectors", kind))
			if err != nil {
				return nil, fmt.Errorf("%w: connector %s: %v", ErrConfig, kind, err)
			}
			if m.Notes && kind == "docs" {
				if err := c.EnableNotes(u.Path); err != nil {
					return nil, fmt.Errorf("%w: connector %s: %v", ErrConfig, kind, err)
				}
			}
			out[kind] = c
		case "slack":
			out[kind] = &slack.Connector{Base: base, HTTP: hc, Creds: creds, Redact: redact, Exclude: m.Exclude[kind], Inbox: inbox}
		case "linear":
			out[kind] = &linear.Connector{Endpoint: base, HTTP: hc, Key: creds, Redact: redact}
		case "hubspot":
			out[kind] = &hubspot.Connector{Base: base, HTTP: hc, Creds: creds, Redact: redact}
		case "github":
			out[kind] = &github.Connector{Base: base, HTTP: hc, Redact: redact, Creds: creds}
		case "bamboohr":
			out[kind] = &bamboohr.Connector{Base: base, HTTP: hc, Redact: redact, Creds: creds}
		case "quickbooks":
			out[kind] = &quickbooks.Connector{Base: base, HTTP: hc, Redact: redact, Creds: creds, Now: time.Now}
		default:
			return nil, fmt.Errorf("%w: connector %s: no adapter for vendor %q", ErrConfig, kind, vendor)
		}
	}
	return out, nil
}

func buildModel(cfg Config, tenant string, secret func(string) func(context.Context) (string, error), redact *obs.Redactor,
	meter gateway.Counter, guard *budget.Guard) (ports.Model, error) {
	var adapter ports.Model
	hc := &http.Client{Timeout: 60 * time.Second}
	key := secret("model")
	if cfg.Model.Local {
		key = func(context.Context) (string, error) { return "", nil } // a model on the box holds no secret
	}
	switch cfg.Model.Provider {
	case "extractive":
		adapter = extractive.Model{}
	case "openaicompat":
		adapter = &openaicompat.Client{BaseURL: cfg.Model.Base, Key: key, HTTP: hc}
	case "anthropic":
		adapter = &anthropic.Client{BaseURL: cfg.Model.Base, Key: key, HTTP: hc}
	case "groq":
		adapter = &groq.Client{BaseURL: cfg.Model.Base, Key: key, HTTP: hc}
	case "openrouter":
		adapter = &openrouter.Client{BaseURL: cfg.Model.Base, Key: key, HTTP: hc}
	default:
		return nil, fmt.Errorf("%w: model.provider %q", ErrConfig, cfg.Model.Provider)
	}
	return gateway.Gateway{Tenant: tenant, Provider: cfg.Model.Provider, ModelID: cfg.Model.ID, Model: adapter, Secrets: redact,
		Metrics: meter, Budget: guard}, nil
}

// classification reads the Drive label that carries sensitivity (sb-connector(3)): none
// when the vault has no such secret, and an error, never none, when it can't be read.
func classification(read func(context.Context) (string, error)) (*gdrive.Classification, error) {
	raw, err := read(context.Background())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c gdrive.Classification
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Label == "" || c.Field == "" {
		return nil, fmt.Errorf("the docs classification isn't {label, field, choices}")
	}
	return &c, nil
}

// slackField reads one field of the chat credentials (sb-connector(3), Slack) at call time.
func slackField(creds func(context.Context) (string, error), field string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		raw, err := creds(ctx)
		if err != nil {
			return "", err
		}
		var v map[string]string
		if err := json.Unmarshal([]byte(raw), &v); err != nil || v[field] == "" {
			return "", fmt.Errorf("the chat credentials have no %s", field)
		}
		return v[field], nil
	}
}

func approvers(m manifest.Tenant) []string {
	var out []string
	for _, g := range m.Approval {
		out = append(out, strings.TrimPrefix(g, "group:"))
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// alerter sends every alert to the log and, when one is configured, to the webhook
// (sb-observability(7)); a webhook that fails is logged and counted, never dropped quietly.
type alerter struct {
	webhook canary.Alerter
	log     *slog.Logger
	metrics *obs.Metrics
}

func (a alerter) Alert(ctx context.Context, al canary.Alert) error {
	a.log.Warn("alert", "kind", al.Kind, "severity", al.Severity, "where", al.Where, "ref", al.Ref)
	a.metrics.Add("sb_alerts_total", "Alerts raised, by kind and severity.", map[string]string{"kind": al.Kind, "severity": al.Severity}, 1)
	if a.webhook == nil {
		return nil
	}
	if err := a.webhook.Alert(ctx, al); err != nil {
		a.log.Error("alert not delivered", "kind", al.Kind, "error", err.Error())
		a.metrics.Add("sb_alert_failures_total", "Alerts the webhook didn't take.", map[string]string{"kind": al.Kind}, 1)
		return err
	}
	return nil
}

// noteReaders are a note's default readers (SB-1705): the active members of the space now,
// by their directory email, or every active user for a company-wide note.
func (c *Cell) noteReaders(m manifest.Tenant) func(ctx context.Context, space string) ([]string, error) {
	tenant := c.Config.Tenant
	return func(ctx context.Context, space string) ([]string, error) {
		var ids []string
		if space == "company" {
			all, err := c.Directory.ActiveUserIDs(ctx, tenant)
			if err != nil {
				return nil, err
			}
			ids = all
		} else {
			sp, ok := m.Spaces[strings.TrimPrefix(space, "spc_")]
			if !ok {
				return nil, fmt.Errorf("no space %s", space)
			}
			for _, g := range sp.Members {
				ms, err := c.Directory.Members(ctx, tenant, strings.TrimPrefix(g, "group:"))
				if err != nil {
					return nil, err
				}
				ids = append(ids, ms...)
			}
		}
		var out []string
		seen := map[string]bool{}
		for _, id := range ids {
			u, found, err := c.Directory.User(ctx, tenant, id)
			if err != nil {
				return nil, err
			}
			if found && u.Active && len(u.Emails) > 0 && !seen[u.Emails[0]] {
				seen[u.Emails[0]] = true
				out = append(out, strings.ToLower(u.Emails[0]))
			}
		}
		return out, nil
	}
}

// relations are how the asker stands to a person, for handover briefs (sb-policy(5),
// Handover; SB-1707): self, their SCIM manager, or a steward of a space they're a member of.
// briefable are the people a handover search looks among (SB-2106): the caller, the people
// whose manager they are, and the members of the spaces they steward, read now.
func (c *Cell) briefable(m manifest.Tenant) func(ctx context.Context, s tenancy.Scope) ([]ports.User, error) {
	tenant := c.Config.Tenant
	return func(ctx context.Context, s tenancy.Scope) ([]ports.User, error) {
		ids := []string{s.User()}
		all, err := c.Directory.ActiveUserIDs(ctx, tenant)
		if err != nil {
			return nil, err
		}
		for _, id := range all {
			u, found, err := c.Directory.User(ctx, tenant, id)
			if err != nil {
				return nil, err
			}
			if found && u.Manager == s.User() {
				ids = append(ids, id)
			}
		}
		mine := map[string]bool{}
		for _, g := range s.Groups() {
			mine["group:"+g] = true
		}
		for _, sp := range m.Spaces {
			steward := false
			for _, g := range sp.Stewards {
				steward = steward || mine[g]
			}
			if !steward {
				continue
			}
			for _, g := range sp.Members {
				ms, err := c.Directory.Members(ctx, tenant, strings.TrimPrefix(g, "group:"))
				if err != nil {
					continue // a group the directory doesn't know has no members
				}
				ids = append(ids, ms...)
			}
		}
		var out []ports.User
		for _, id := range ids {
			u, found, err := c.Directory.User(ctx, tenant, id)
			if err != nil {
				return nil, err
			}
			if found {
				out = append(out, u)
			}
		}
		return out, nil
	}
}

func (c *Cell) relations(m manifest.Tenant) ask.Relations {
	tenant := c.Config.Tenant
	return func(ctx context.Context, s tenancy.Scope, person string) ([]string, error) {
		u, found, err := c.Directory.User(ctx, tenant, person)
		if err != nil || !found {
			return nil, err
		}
		var out []string
		if person == s.User() {
			out = append(out, "self")
		}
		if u.Manager != "" && u.Manager == s.User() {
			out = append(out, "manager")
		}
		theirs, err := c.Directory.GroupsOf(ctx, tenant, person)
		if err != nil {
			return nil, err
		}
		mine := map[string]bool{}
		for _, g := range s.Groups() {
			mine["group:"+g] = true
		}
		in := map[string]bool{}
		for _, g := range theirs {
			in["group:"+g] = true
		}
		for _, sp := range m.Spaces {
			member, steward := false, false
			for _, g := range sp.Members {
				member = member || in[g]
			}
			for _, g := range sp.Stewards {
				steward = steward || mine[g]
			}
			if member && steward {
				out = append(out, "steward")
				break
			}
		}
		return out, nil
	}
}

// linker opens a cite at its source for a connector that can link (SB-3006), or "".
func linker(conns map[string]ports.Connector) func(ports.Chunk) string {
	return func(ch ports.Chunk) string {
		l, ok := conns[ch.Kind].(interface {
			Link(ctx context.Context, ch ports.Chunk) string
		})
		if !ok {
			return ""
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return l.Link(ctx, ch)
	}
}

// workZone is the working week's time zone: the manifest's digest zone, or UTC (SB-3404).
func workZone(m manifest.Tenant) *time.Location {
	if z, _ := manifest.DigestZone(m.Digest); z != nil {
		return z
	}
	return time.UTC
}

// modelPrices are the price of each model the cell calls, by provider/id (SB-4405).
func modelPrices(cfg Config) map[string]costs.Price {
	out := map[string]costs.Price{cfg.Model.Provider + "/" + cfg.Model.ID: {InPerM: cfg.Model.Price.InPerM, OutPerM: cfg.Model.Price.OutPerM}}
	if s := cfg.Model.Small; s.ID != "" {
		out[cfg.Model.Provider+"/"+s.ID] = costs.Price{InPerM: s.Price.InPerM, OutPerM: s.Price.OutPerM}
	}
	return out
}

// modelTiers name each model's tier, only when the cell has a small one.
func modelTiers(cfg Config) map[string]string {
	if cfg.Model.Small.ID == "" {
		return nil
	}
	return map[string]string{cfg.Model.Provider + "/" + cfg.Model.ID: "main", cfg.Model.Provider + "/" + cfg.Model.Small.ID: "small"}
}

// rhythms is the rhythms and projects service (SB-4801 to SB-4804): every step is drafted
// as its approver, with a scope read from the directory now, and a member is someone in
// one of the space's member groups now.
func (c *Cell) rhythms(m manifest.Tenant, stewards ask.Stewards, legwork ask.Legwork, newID func(string) string) ask.Rhythms {
	tenant := c.Config.Tenant
	readers := func(ctx context.Context, space string) ([]string, error) { // active members' user ids
		sp, ok := m.Spaces[strings.TrimPrefix(space, "spc_")]
		if !ok {
			return nil, fmt.Errorf("no space %s", space)
		}
		var out []string
		for _, g := range sp.Members {
			ms, err := c.Directory.Members(ctx, tenant, strings.TrimPrefix(g, "group:"))
			if err != nil {
				return nil, err
			}
			for _, id := range ms {
				u, found, err := c.Directory.User(ctx, tenant, id)
				if err != nil {
					return nil, err
				}
				if found && u.Active && !slices.Contains(out, id) {
					out = append(out, id)
				}
			}
		}
		return out, nil
	}
	return ask.Rhythms{Stewards: stewards, Legwork: legwork, Store: c.Traces, Projects: c.Traces,
		Collections: ask.CollectionService{Agent: stewards.Agent, Store: c.Traces, Saved: c.Traces, Audit: c.Audit, Known: legwork.Known, NewID: newID, Now: time.Now},
		Defs: func(space string) map[string]ask.RhythmDef {
			if _, ok := m.Spaces[space]; !ok {
				return nil
			}
			out := map[string]ask.RhythmDef{}
			for name, r := range m.Rhythms(space) {
				d := ask.RhythmDef{Subject: r.Subject, Plan: r.Plan}
				for _, st := range r.Steps {
					d.Steps = append(d.Steps, ask.RhythmStepDef{Day: st.Day, Kind: st.Kind, Ask: st.Ask, Approver: st.Approver, Agent: st.Agent})
				}
				out[name] = d
			}
			return out
		},
		Glossary: func(space string) map[string]string { return m.Effective(space).Glossary }, // SB-4805
		Spaces: func() []string {
			out := make([]string, 0, len(m.Spaces))
			for name := range m.Spaces {
				out = append(out, name)
			}
			sort.Strings(out)
			return out
		},
		ScopeIn: func(ctx context.Context, user, space string) (tenancy.Scope, error) {
			ids, err := readers(ctx, "spc_"+space)
			if err != nil {
				return tenancy.Scope{}, err
			}
			if !slices.Contains(ids, user) {
				return tenancy.Scope{}, errors.New("wire: not a member of the space")
			}
			return c.ScopeAs(ctx, user, "spc_"+space)
		},
		Manager: func(ctx context.Context, user string) (string, error) {
			u, found, err := c.Directory.User(ctx, tenant, user)
			if err != nil || !found {
				return "", err
			}
			return u.Manager, nil
		},
		Members: func(ctx context.Context, space string) ([]string, error) { return readers(ctx, "spc_"+space) },
		TracesOf: func(ctx context.Context, user string, from time.Time) ([]ask.Trace, error) {
			ts, err := c.Traces.Between(ctx, tenant, from, time.Now().Add(time.Minute))
			if err != nil {
				return nil, err
			}
			var out []ask.Trace
			for _, t := range ts {
				if t.User == user {
					out = append(out, t)
				}
			}
			return out, nil
		},
	}
}

// inFlight counts what is under way in each space a reorg touches (SB-4806): counts only.
func (c *Cell) inFlight(m manifest.Tenant) func(ctx context.Context, spaces []string) (map[string]reorg.InFlight, error) {
	tenant := c.Config.Tenant
	return func(ctx context.Context, spaces []string) (map[string]reorg.InFlight, error) {
		runs, err := c.Traces.RhythmRuns(ctx, tenant)
		if err != nil {
			return nil, err
		}
		projects, err := c.Traces.Projects(ctx, tenant)
		if err != nil {
			return nil, err
		}
		routines, err := c.Traces.Routines(ctx, tenant)
		if err != nil {
			return nil, err
		}
		out := map[string]reorg.InFlight{}
		for _, sp := range spaces {
			in := reorg.InFlight{Triggered: []string{}}
			for _, r := range runs {
				if r.Space == sp && (r.State == ask.RhythmRunning || r.State == ask.RhythmProposed) {
					in.Rhythms++
				}
			}
			for _, p := range projects {
				if p.Space == sp && p.State == ask.ProjectOpen {
					in.Projects++
				}
			}
			for _, q := range routines {
				if q.Space == sp || q.Space == ask.SpaceKey(sp) {
					in.Routines++
				}
			}
			qs, err := c.Traces.StewardQueue(ctx, tenant, ask.SpaceKey(sp))
			if err != nil {
				return nil, err
			}
			for _, q := range qs {
				if q.Reply == nil && (q.Thread == nil || !q.Thread.Discarded) {
					in.Open++
					if q.Waiting != nil {
						in.Waiting++
					}
				}
			}
			e := m.Effective(sp)
			if e.Brief != nil {
				in.Triggered = append(in.Triggered, "brief")
			}
			if e.Updates {
				in.Triggered = append(in.Triggered, "update")
			}
			out[sp] = in
		}
		return out, nil
	}
}

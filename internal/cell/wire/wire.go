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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"example.com/second-brain/internal/adapters/alert/webhook"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/adapters/connectors/bamboohr"
	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/connectors/gdrive"
	"example.com/second-brain/internal/adapters/connectors/github"
	"example.com/second-brain/internal/adapters/connectors/hubspot"
	"example.com/second-brain/internal/adapters/connectors/linear"
	"example.com/second-brain/internal/adapters/connectors/quickbooks"
	"example.com/second-brain/internal/adapters/connectors/slack"
	cursorfs "example.com/second-brain/internal/adapters/cursor/filestore"
	"example.com/second-brain/internal/adapters/embed/hashing"
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
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/report"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/reviews"
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
	Redact    *obs.Redactor
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
	handler    http.Handler
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
}

func (u usage) Traces(ctx context.Context, tenant string, from, to time.Time) ([]ask.Trace, error) {
	return u.traces.Between(ctx, tenant, from, to)
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

// Options are what a test may change about how a cell reaches the outside: the zero value
// is what the services run with.
type Options struct {
	HTTP *http.Client // for the identity provider; nil is http.DefaultClient
	// ConnectorHTTP is the connectors' client; nil is one with a 30-second timeout.
	ConnectorHTTP *http.Client
	// Log is where the cell logs; nil is standard error.
	Log io.Writer
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
	c.Report = usage{traces: c.Traces, ledger: c.Usage, actions: c.Actions, dir: c.Directory, index: c.Index}
	c.model = func(m manifest.Tenant) (ports.Model, error) {
		guard := &budget.Guard{Tenant: tenant, MonthlyUSD: float64(m.BudgetUSDMonth), Meter: c.Usage.Spend(), Metrics: c.Metrics, Now: time.Now,
			Price: budget.Price{InputPerM: cfg.Model.Price.InPerM, OutputPerM: cfg.Model.Price.OutPerM}, Once: c.Usage,
			// The warning goes through whichever alerter the cell has when it fires (SB-1608).
			Warn: func(ctx context.Context, month string, _ float64) {
				if c.Alerter != nil {
					_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev3", Kind: "budget_warning", Tenant: tenant, Where: "budget", Ref: month})
				}
			}}
		return buildModel(cfg, tenant, secret, redact, c.Usage, guard)
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
		GroupsOf: c.Directory.GroupsOf, Spaces: c.Spaces, Attributes: auth.DirectoryAttributes(c.Directory)})

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
	agent := ask.Agent{Tenant: tenant, Release: c.Release, Index: c.Index, Embedder: hashing.Embedder{}, Auth: authz,
		Model: model, Store: c.Traces, Emails: emails, Killed: c.Kill.AgentKilled("ask"), SpaceKilled: c.Kill.SpaceKilled,
		NewID: newID, Now: time.Now, Metrics: c.Metrics, Enabled: func(a string) bool { return contains(m.Agents, a) },
		SpaceOf: func(ctx context.Context, id string) (ports.Space, bool, error) {
			return c.Spaces.Space(ctx, tenant, id)
		},
		AgentKilled: func(a string) bool { return c.Kill.AgentKilled(a)(context.Background()) },
		ReviewAfter: func(ids []string) int {
			least := 0
			for _, id := range ids {
				d := m.Spaces[strings.TrimPrefix(id, "spc_")].ReviewAfterDays
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
		})}

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
	public.Handle("/scim/v2/", (&scim.Endpoint{Tenant: tenant, Directory: c.Directory, Sessions: c.Sessions, Token: secret("scim")}).Handler())
	api.Verifier{Release: c.Release, Metrics: c.Metrics.Handler(), ControlPlane: c.controlPlane, Attest: c.attestor(), TargetFor: c.VerifyTarget}.Register(public)
	if sc, ok := conns["chat"].(*slack.Connector); ok {
		public.Handle("POST /connectors/slack/events", sc.EventsHandler(time.Now))
		creds := secret("connectors/chat")
		app := &slackapp.App{Tenant: tenant, TeamID: cfg.Chat.TeamID, Secret: slackField(creds, "signing_secret"),
			Email: slackapp.WebEmail(cfg.Connectors["chat"].Base, slackField(creds, "bot_token"), &http.Client{Timeout: 10 * time.Second}),
			Users: c.Directory, Groups: c.Directory.GroupsOf, Issuer: c.Auth.Issuer, Agent: agent, Spaces: c.Spaces, Now: time.Now}
		app.Register(public)
	}
	if cfg.Web != "" {
		api.Web{Files: os.DirFS(cfg.Web)}.Register(public)
	}
	api.Ask{Agent: agent, Feedback: ask.Feedbacks{Agent: agent, Store: c.Traces}, Release: c.Release, NewID: newID}.Register(mux)
	api.Drafts{Agent: ask.Drafts{Agent: agent, Killed: c.Kill.AgentKilled("draft-reply"), Policy: c.Policy, Metrics: c.Usage},
		Ask: api.Ask{Agent: agent, Release: c.Release, NewID: newID}}.Register(mux)
	api.Actions{Bus: bus, Release: c.Release, NewID: newID}.Register(mux)
	stewards := ask.Stewards{Agent: agent, Store: c.Traces, Audit: c.Audit, Policy: c.Policy, NewID: newID, Now: time.Now,
		Facts: func(space string) ([]string, string, bool) {
			if space == "company" {
				return []string{m.QuestionsTo}, "", m.QuestionsTo != ""
			}
			sp, ok := m.Spaces[strings.TrimPrefix(space, "spc_")]
			return sp.Stewards, sp.Label, ok && len(sp.Stewards) > 0
		}}
	if m.Notes {
		stewards.Bus, stewards.Readers = bus, c.noteReaders(m)
	}
	api.Questions{Service: stewards, Release: c.Release, NewID: newID, Reviews: c.Traces,
		ReviewAfter: func(name string) int { return m.Spaces[name].ReviewAfterDays }}.Register(mux)
	api.Knowledge{Search: agent, Release: c.Release, NewID: newID,
		Start: func(ctx context.Context, s tenancy.Scope) (ask.StartResponse, error) {
			return agent.StartHere(ctx, s, m.Spaces[strings.TrimPrefix(s.Space(), "spc_")].StartHere)
		},
		Changes: agent.Changes,
		Handover: func(ctx context.Context, s tenancy.Scope, person string) (ask.HandoverResponse, error) {
			return agent.Handover(ctx, s, person, emails, c.relations(m), c.Policy)
		},
		Questions: ask.Questions{Tenant: tenant, Store: c.Traces, Audit: c.Audit, NewID: newID, Now: time.Now}}.Register(mux)
	admin := api.Admin{Tenant: tenant, Policy: c.Policy, Manifests: man, Kill: kill.Control{Store: c.Kill.Store, Audit: c.Audit, Now: time.Now},
		Switches: c.Kill.Store, Audit: c.Audit, AuditRows: c.Audit.Store, Release: c.Release, Approvers: approvers(m),
		Usage: c.Report,
		Price: report.Price{InputPerM: cfg.Model.Price.InPerM, OutputPerM: cfg.Model.Price.OutPerM},
		Meter: c.Usage, Prices: map[string]costs.Price{cfg.Model.Provider + "/" + cfg.Model.ID: {InPerM: cfg.Model.Price.InPerM, OutPerM: cfg.Model.Price.OutPerM}},
		BudgetUSD: m.BudgetUSDMonth, Now: time.Now, Erasure: ret}
	admin.Register(mux)
	api.Operations{Admin: admin, Feedback: c.Traces, Traces: c.Traces, Health: c.Health}.Register(mux)
	api.Spaces{Store: c.Spaces}.Register(mux)
	api.Stewards{Tenant: tenant, Policy: c.Policy, Manifests: man, Audit: c.Audit, Proposals: c.Proposals, NewID: newID, Now: time.Now}.Register(mux)
	api.Reviews{Tenant: tenant, Policy: c.Policy, Manifests: man, Audit: c.Audit, Reviews: c.Reviews, Proposals: c.Proposals,
		NewID: newID, Now: time.Now}.Register(mux)
	api.Holds{Tenant: tenant, Policy: c.Policy, Manifests: man, Service: holds.Service{Store: c.Holds, Audit: c.Audit, NewID: newID, Now: time.Now}}.Register(mux)
	public.HandleFunc("POST /v1/cell/manifest", c.applyHandler)
	public.HandleFunc("GET /v1/cell/live", c.liveHandler)
	public.HandleFunc("GET /v1/cell/audit", c.auditRows)
	public.HandleFunc("GET /v1/cell/audit/export", c.auditExport)
	public.HandleFunc("GET /v1/cell/package", c.packageHandler)
	public.Handle("/v1/", c.Auth.Middleware(mux))
	return &Parts{Manifest: m, SHA256: digest, Connectors: conns, Agent: agent, Bus: bus, Retention: ret, handler: public}, nil
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

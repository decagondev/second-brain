package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditfs "example.com/second-brain/internal/adapters/audit/localfs"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/adapters/connectors/bamboohr"
	"example.com/second-brain/internal/adapters/connectors/gdrive"
	"example.com/second-brain/internal/adapters/connectors/github"
	"example.com/second-brain/internal/adapters/connectors/hubspot"
	"example.com/second-brain/internal/adapters/connectors/linear"
	"example.com/second-brain/internal/adapters/connectors/quickbooks"
	"example.com/second-brain/internal/adapters/connectors/slack"
	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	killfs "example.com/second-brain/internal/adapters/kill/localfs"
	logfs "example.com/second-brain/internal/adapters/logs/localfs"
	"example.com/second-brain/internal/adapters/model/anthropic"
	"example.com/second-brain/internal/adapters/model/extractive"
	"example.com/second-brain/internal/adapters/model/groq"
	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/adapters/model/openrouter"
	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/adapters/policy/cedar"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/barrier"
	"example.com/second-brain/internal/core/gateway"
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
	"example.com/second-brain/internal/core/toolbus"
	"example.com/second-brain/internal/core/workflow"
	"example.com/second-brain/internal/obs"
	"example.com/second-brain/internal/scim"
	"example.com/second-brain/internal/version"
)

// MemoryStores are the stores that have only an in-memory adapter today (sb-cell(5),
// Stores); a durable cell refuses to start while this list isn't empty.
var MemoryStores = []string{"directory and sessions", "traces and conversations", "actions and receipts", "spaces registry",
	"index", "chat events inbox", "proposals", "access reviews", "legal holds"}

// Vendors are the connector adapters a cell can run, by the manifest's vendor name.
var Vendors = map[string]string{"gdrive": "docs", "slack": "chat", "linear": "tickets", "hubspot": "crm", "github": "code",
	"bamboohr": "hris", "quickbooks": "accounting"}

// Cell is a built cell: every part, and the API handler that serves them.
type Cell struct {
	Config   Config
	Manifest manifest.Tenant
	Release  string

	Audit     *audit.Log
	Kill      *kill.Checker
	Policy    policy.Kernel
	Directory *dirmem.Dir
	Sessions  *sessmem.Store
	Spaces    *spacemem.Store
	Index     *memindex.Index
	Traces    *tracemem.Store
	Holds     *holds.MemoryStore
	Bus       *toolbus.Bus
	Agent     ask.Agent
	Metrics   *obs.Metrics
	Auth      *auth.Cell

	handler http.Handler
	ready   atomic.Bool
}

// Handler serves the whole API (sb-api(3)).
func (c *Cell) Handler() http.Handler { return c.handler }

type applied struct {
	t      manifest.Tenant
	commit string
}

func (a applied) Applied(context.Context) (manifest.Tenant, string, error) { return a.t, a.commit, nil }

// noUsage is the usage source until the jobs service keeps one (SB-1504): it answers
// EUNAVAIL, never numbers it doesn't have.
type noUsage struct{}

var errNoUsage = errors.New("usage is kept by the jobs service (SB-1504)")

func (noUsage) Traces(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	return nil, errNoUsage
}
func (noUsage) Drafts(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 0, 0, errNoUsage
}
func (noUsage) Actions(context.Context, string, time.Time, time.Time) ([]approvals.Action, error) {
	return nil, errNoUsage
}
func (noUsage) Seats(context.Context, string) (int, error) { return 0, errNoUsage }
func (noUsage) Tokens(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 0, 0, errNoUsage
}

var _ report.Source = noUsage{}

// Options are what a test may change about how a cell reaches the outside: the zero value
// is what the services run with.
type Options struct {
	HTTP *http.Client // for the identity provider; nil is http.DefaultClient
}

// Build builds every part of a cell from its configuration. Anything it can't build stops
// it: a cell never runs with a part switched off (sb-cell(5)).
func Build(ctx context.Context, cfg Config, opt Options) (*Cell, error) {
	if opt.HTTP == nil {
		opt.HTTP = http.DefaultClient
	}
	if cfg.Durable {
		return nil, fmt.Errorf("%w: durable is true, but these stores are in memory only: %s", ErrConfig, strings.Join(MemoryStores, ", "))
	}
	src, err := os.ReadFile(cfg.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%w: manifest: %v", ErrConfig, err)
	}
	m, ds := manifest.Check(src)
	if len(ds) > 0 {
		return nil, fmt.Errorf("%w: the manifest doesn't check: %s", ErrConfig, ds[0].Format(cfg.Manifest))
	}
	if m.Tenant != cfg.Tenant {
		return nil, fmt.Errorf("%w: the manifest is for %s, not %s", ErrConfig, m.Tenant, cfg.Tenant)
	}
	for kind, vendor := range m.Connectors {
		if Vendors[vendor] != kind {
			return nil, fmt.Errorf("%w: connector %s: no adapter for vendor %q", ErrConfig, kind, vendor)
		}
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
	c := &Cell{Config: cfg, Manifest: m, Release: "rel_" + version.Version, Metrics: obs.NewMetrics()}
	tenant := cfg.Tenant
	vault := localvault.Dir{Root: cfg.Vault}
	secret := func(path string) func(context.Context) (string, error) {
		return func(ctx context.Context) (string, error) { return vault.Get(ctx, "vault://tenants/"+tenant+"/"+path) }
	}
	redact := obs.NewRedactor()
	n := atomic.Int64{}
	newID := func(p string) string { return fmt.Sprintf("%s%d%06d", p, time.Now().UnixMilli(), n.Add(1)%1000000) }

	c.Audit = &audit.Log{Tenant: tenant, Release: func() string { return c.Release }, Store: auditfs.Dir{Root: filepath.Join(cfg.State, "audit"), Tenant: tenant},
		Now: time.Now, Redact: redact}
	c.Policy = policy.Kernel{Engine: engine, Recorder: c.Audit}
	c.Kill = &kill.Checker{Tenant: tenant, Store: killfs.Dir{Root: filepath.Join(cfg.State, "kill")}, Now: time.Now}
	if err := c.Kill.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("%w: kill switches: %v", ErrConfig, err)
	}
	c.Directory, c.Sessions = dirmem.New(), sessmem.New()
	c.Spaces, c.Index, c.Traces, c.Holds = spacemem.New(), memindex.New(), tracemem.New(), &holds.MemoryStore{}
	heldNow := func(ctx context.Context) holds.Set {
		set, err := holds.Active(ctx, c.Holds)
		if err != nil {
			return holds.Set{{Space: ""}} // holds can't be read: everything is held
		}
		return set
	}
	if err := spaces.SyncHeld(ctx, c.Spaces, nil, m, func(id string) bool { return heldNow(ctx).HeldSpace(id) }); err != nil {
		return nil, fmt.Errorf("%w: spaces: %v", ErrConfig, err)
	}
	spaceFacts := func(ctx context.Context) ([]ports.Space, [][2]string, error) {
		all, err := c.Spaces.Spaces(ctx, tenant)
		return all, m.Barriers, err
	}

	if m.Connectors["chat"] == "slack" && cfg.Chat.TeamID == "" {
		return nil, fmt.Errorf("%w: the manifest's chat is slack, so chat.team_id is required", ErrConfig)
	}
	inbox := &slack.MemoryInbox{}
	conns, err := connectors(cfg, m, secret, redact, inbox)
	if err != nil {
		return nil, err
	}
	model, err := buildModel(cfg, tenant, secret, redact)
	if err != nil {
		return nil, err
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
	c.Agent = ask.Agent{Tenant: tenant, Release: c.Release, Index: c.Index.Bind(tenant), Embedder: hashing.Embedder{}, Auth: authz,
		Model: model, Store: c.Traces, Emails: emails, Killed: c.Kill.AgentKilled("ask"), SpaceKilled: c.Kill.SpaceKilled,
		NewID: newID, Now: time.Now, Metrics: c.Metrics, Enabled: func(a string) bool { return contains(m.Agents, a) },
		SpaceOf: func(ctx context.Context, id string) (ports.Space, bool, error) {
			return c.Spaces.Space(ctx, tenant, id)
		},
		AgentKilled: func(a string) bool { return c.Kill.AgentKilled(a)(context.Background()) }}

	// Actions (sb-actions(3)).
	tools, err := workflow.Registry(m)
	if err != nil {
		return nil, fmt.Errorf("%w: workflows: %v", ErrConfig, err)
	}
	locate := func(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
		conn, ok := conns[ref.Kind]
		if !ok {
			return ports.Document{}, fmt.Errorf("no connector for %s", ref.Kind)
		}
		return conn.Fetch(ctx, ref)
	}
	c.Bus = &toolbus.Bus{Tenant: tenant, Tools: tools, Kill: c.Kill, Policy: c.Policy, Audit: c.Audit, Actions: actmem.New(),
		Connectors: conns, Now: time.Now, NewID: newID, Receipts: actmem.NewReceipts(),
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
	logs := &logfs.Files{Dir: filepath.Join(cfg.State, "logs"), Now: time.Now}
	erasure := retention.Job{Tenant: tenant, Days: m.RetentionDays, Stores: append(c.Traces.Retention(), logs), Audit: c.Audit, Now: time.Now,
		Held: func(space, user string) bool { return heldNow(context.Background()).Held(space, user) }}

	man := applied{t: m, commit: "local"}
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
	api.Verifier{Release: c.Release, Metrics: c.Metrics.Handler(), ControlPlane: func(*http.Request) bool { return false }}.Register(public)
	if sc, ok := conns["chat"].(*slack.Connector); ok {
		public.Handle("POST /connectors/slack/events", sc.EventsHandler(time.Now))
		creds := secret("connectors/chat")
		app := &slackapp.App{Tenant: tenant, TeamID: cfg.Chat.TeamID, Secret: slackField(creds, "signing_secret"),
			Email: slackapp.WebEmail(cfg.Connectors["chat"].Base, slackField(creds, "bot_token"), &http.Client{Timeout: 10 * time.Second}),
			Users: c.Directory, Groups: c.Directory.GroupsOf, Issuer: c.Auth.Issuer, Agent: c.Agent, Spaces: c.Spaces, Now: time.Now}
		app.Register(public)
	}
	if cfg.Web != "" {
		api.Web{Files: os.DirFS(cfg.Web)}.Register(public)
	}
	api.Ask{Agent: c.Agent, Feedback: ask.Feedbacks{Agent: c.Agent, Store: c.Traces}, Release: c.Release, NewID: newID}.Register(mux)
	api.Drafts{Agent: ask.Drafts{Agent: c.Agent, Killed: c.Kill.AgentKilled("draft-reply"), Policy: c.Policy, Metrics: c.Metrics},
		Ask: api.Ask{Agent: c.Agent, Release: c.Release, NewID: newID}}.Register(mux)
	api.Actions{Bus: c.Bus, Release: c.Release, NewID: newID}.Register(mux)
	api.Admin{Tenant: tenant, Policy: c.Policy, Manifests: man, Kill: kill.Control{Store: c.Kill.Store, Audit: c.Audit, Now: time.Now},
		Switches: c.Kill.Store, Audit: c.Audit, AuditRows: c.Audit.Store, Release: c.Release, Usage: noUsage{}, Approvers: approvers(m),
		BudgetUSD: m.BudgetUSDMonth, Now: time.Now, Erasure: erasure}.Register(mux)
	api.Spaces{Store: c.Spaces}.Register(mux)
	proposalStore := &proposals.MemoryStore{}
	api.Stewards{Tenant: tenant, Policy: c.Policy, Manifests: man, Audit: c.Audit, Proposals: proposalStore, NewID: newID, Now: time.Now}.Register(mux)
	api.Reviews{Tenant: tenant, Policy: c.Policy, Manifests: man, Audit: c.Audit, Reviews: &reviews.MemoryStore{}, Proposals: proposalStore,
		NewID: newID, Now: time.Now}.Register(mux)
	api.Holds{Tenant: tenant, Policy: c.Policy, Manifests: man, Service: holds.Service{Store: c.Holds, Audit: c.Audit, NewID: newID, Now: time.Now}}.Register(mux)
	public.Handle("/v1/", c.Auth.Middleware(mux))
	c.handler = public
	c.ready.Store(true)
	return c, nil
}

// Ready reports whether every part was built (GET /readyz).
func (c *Cell) Ready() bool { return c.ready.Load() }

// Stop marks the cell not ready, so the platform stops sending it requests.
func (c *Cell) Stop() { c.ready.Store(false) }

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

func connectors(cfg Config, m manifest.Tenant, secret func(string) func(context.Context) (string, error), redact *obs.Redactor, inbox slack.Inbox) (map[string]ports.Connector, error) {
	out := map[string]ports.Connector{}
	hc := &http.Client{Timeout: 30 * time.Second}
	for kind, vendor := range m.Connectors {
		base := cfg.Connectors[kind].Base
		creds := secret("connectors/" + kind)
		switch vendor {
		case "gdrive":
			out[kind] = &gdrive.Connector{Base: base, HTTP: hc, Tokens: &gdrive.ServiceAccount{Key: creds, HTTP: hc, Redact: redact, Now: time.Now}}
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

func buildModel(cfg Config, tenant string, secret func(string) func(context.Context) (string, error), redact *obs.Redactor) (ports.Model, error) {
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
	return gateway.Gateway{Tenant: tenant, Provider: cfg.Model.Provider, ModelID: cfg.Model.ID, Model: adapter, Secrets: redact}, nil
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

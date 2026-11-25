package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/costs"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/reorg"
	"example.com/second-brain/internal/core/report"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/route"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Manifests is the applied manifest of this cell's tenant.
type Manifests interface {
	Applied(ctx context.Context) (manifest.Tenant, string, error) // manifest and its git commit
}

// Admin serves sb-admin(3). The console is a client of this API and has no other powers.
type Admin struct {
	// InFlight counts what is under way in each space a reorg touches (SB-4806); nil
	// counts nothing.
	InFlight  func(ctx context.Context, spaces []string) (map[string]reorg.InFlight, error)
	Tenant    string
	Policy    policy.Kernel
	Manifests Manifests
	Kill      kill.Control
	Switches  kill.Store
	Audit     *audit.Log
	AuditRows audit.Store
	Release   string
	// Usage: the weekly numbers (SB-606), readable by operators and approvers.
	Usage     report.Source
	Price     report.Price
	Approvers []string // the manifest's approval groups
	// Meter, Prices and BudgetUSD make the monthly cost statement (SB-709); Now dates it.
	Meter     costs.Meter
	Prices    map[string]costs.Price // provider/model -> price, from the release
	BudgetUSD int
	Now       func() time.Time
	// Erasure erases one user's derived data (SB-611), exports it (SB-1903) and plans the
	// retention job (SB-1905); nil means EUNAVAIL.
	Erasure interface {
		Erase(ctx context.Context, actor, user, reason string) (retention.Report, error)
		Export(ctx context.Context, actor, user, reason string) (retention.Export, error)
		Plan(ctx context.Context, in time.Duration) (retention.Report, error)
	}
	// Sessions ends a person's sessions (SB-1906); nil means EUNAVAIL.
	Sessions Revoker
	// Tokens lists and ends API tokens (SB-3803); nil means EUNAVAIL.
	Tokens TokenLister
	// NotifyTest sends the test notification to a person by email (SB-2602); nil is EUNAVAIL.
	NotifyTest NotifyTest
	// Index reports on the cell's index (SB-2903); nil is EUNAVAIL.
	Index IndexReports
	// ActiveByEmail are the active user ids with an email, for actor= (SB-3204); nil
	// matches actor= as a user id only.
	ActiveByEmail func(ctx context.Context, email string) ([]string, error)
	// Notices keeps the cell's one notice to everyone (SB-3207); nil is EUNAVAIL to set it
	// and no notice to read.
	Notices ask.NoticeStore
	// AgentKilled and EvalHeld make the agent registry (SB-4004); a hold that can't be
	// read is an error, never no hold.
	AgentKilled func(agent string) bool
	EvalHeld    func() (bool, error)
	// OwnerActive says whether an agent's owner, an email or group:NAME, has an active
	// person (SB-4402); nil flags no owner.
	OwnerActive func(ctx context.Context, owner string) bool
	// Guardrails are the 12 weeks of guardrail counts (SB-4406); nil is EUNAVAIL.
	Guardrails func(ctx context.Context, now time.Time) (report.Guardrails, error)
	// Tiers names each "provider/model" of the cost statement main or small (SB-4405).
	Tiers map[string]string
	// SpaceSpend is each space's branch spend in a month (SB-4105); nil is EUNAVAIL.
	SpaceSpend func(ctx context.Context, month string) ([]SpaceSpend, error)
	// AgentValue is every agent's value (SB-4701), and PausedIn the spaces an agent is
	// paused in (SB-4702); nil has neither.
	AgentValue func(ctx context.Context) (ask.ValueReport, error)
	PausedIn   func(agent string) []string
	// AgentRecord is an agent's signed record (SB-4707); nil is EUNAVAIL.
	AgentRecord func(ctx context.Context, name string) (ask.SignedRecord, error)
	// AIUsage is the AI command centre (SB-5001); nil is EUNAVAIL.
	AIUsage func(ctx context.Context) (ask.AIUsage, error)
}

// SpaceSpend is one space's model spend in a month: its own and its teams' (the branch),
// its budget (0 for none) and, for the month still running, the month-end forecast.
type SpaceSpend struct {
	Space       string   `json:"space"`
	Parent      string   `json:"parent,omitempty"`
	OwnUSD      float64  `json:"own_usd"`
	SpentUSD    float64  `json:"spent_usd"`
	BudgetUSD   int      `json:"budget_usd_month"`
	ForecastUSD *float64 `json:"forecast_usd,omitempty"`
}

// IndexReports are the operator's reports on the index: counts and locations only.
type IndexReports interface {
	ScopeReport(ctx context.Context) ([]route.PatternCount, error)
	// Leftovers are the documents their source no longer lists and the chunks no space
	// holds; Prune removes the first, audited (SB-2905).
	Leftovers(ctx context.Context) (gone, unheld []ports.Chunk, err error)
	Prune(ctx context.Context, actor, reason string) (int, error)
	// Refresh re-reads one document from its source now, audited (SB-3202).
	Refresh(ctx context.Context, actor, reason string, ref ports.SourceRef) (ingest.Refreshed, error)
}

// Register mounts the endpoints.
func (h Admin) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/admin/manifest", h.guard(h.manifest))
	mux.HandleFunc("GET /v1/admin/tree/plan", h.guard(h.treePlan)) // SB-4806
	mux.HandleFunc("POST /v1/admin/manifest/diff", h.guard(h.diff))
	mux.HandleFunc("POST /v1/admin/kill", h.guard(h.kill(true)))
	mux.HandleFunc("POST /v1/admin/unkill", h.guard(h.kill(false)))
	mux.HandleFunc("GET /v1/admin/kill", h.guard(h.switches))
	mux.HandleFunc("GET /v1/admin/audit", h.guard(h.audit))
	mux.HandleFunc("GET /v1/admin/usage", h.readers(h.usage))
	mux.HandleFunc("GET /v1/admin/costs", h.readers(h.costs))
	mux.HandleFunc("GET /v1/admin/costs/spaces", h.readers(h.spaceCosts))
	mux.HandleFunc("GET /v1/admin/guardrails", h.guard(h.guardrails))
	mux.HandleFunc("POST /v1/admin/subjects/erase", h.guard(h.erase))
	mux.HandleFunc("POST /v1/admin/subjects/export", h.guard(h.export))
	mux.HandleFunc("GET /v1/admin/retention/plan", h.guard(h.plan))
	mux.HandleFunc("GET /v1/admin/index/scopes", h.guard(h.scopes))
	mux.HandleFunc("GET /v1/admin/index/leftovers", h.guard(h.leftovers))
	mux.HandleFunc("POST /v1/admin/index/prune", h.guard(h.prune))
	mux.HandleFunc("POST /v1/admin/index/refresh", h.guard(h.refresh))
	mux.HandleFunc("POST /v1/admin/sessions/revoke", h.guard(h.revokeSessions))
	mux.HandleFunc("GET /v1/admin/agents", h.guard(h.agents))
	mux.HandleFunc("GET /v1/admin/agents/value", h.guard(h.agentValue))          // SB-4701
	mux.HandleFunc("GET /v1/admin/agents/{name}/record", h.guard(h.agentRecord)) // SB-4707
	mux.HandleFunc("GET /v1/admin/ai/usage", h.guard(h.aiUsage))                 // SB-5001
	mux.HandleFunc("GET /v1/admin/tokens", h.guard(h.listTokens))
	mux.HandleFunc("DELETE /v1/admin/tokens/{id}", h.guard(h.revokeToken))
	mux.HandleFunc("POST /v1/admin/notifications/test", h.guard(h.notifyTest))
	mux.HandleFunc("PUT /v1/admin/notice", h.guard(h.setNotice))
	mux.HandleFunc("DELETE /v1/admin/notice", h.guard(h.clearNotice))
	mux.HandleFunc("GET /v1/notice", h.notice)
	mux.HandleFunc("GET /v1/admin/refusals", h.guard(func(w http.ResponseWriter, _ *http.Request, _ tenancy.Scope) {
		h.fail(w, errs.New(errs.ENOTIMPL, "refusal reports arrive with SB-607"))
	}))
}

// readers lets operators and approvers through (sb-admin(3): usage and costs).
func (h Admin) readers(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := tenancy.FromContext(r.Context())
		if err != nil {
			h.fail(w, errs.New(errs.EAUTH, "Sign in again."))
			return
		}
		approver := false
		for _, g := range s.Groups() {
			for _, a := range h.Approvers {
				approver = approver || g == a
			}
		}
		if !approver && h.Policy.Authorise(r.Context(), s, "admin", policy.Entity{Type: "Cell", ID: h.Tenant, Tenant: h.Tenant}, nil).Decision != policy.Allow {
			h.fail(w, errs.New(errs.EPERM, "You need the operator or approver role for this."))
			return
		}
		next(w, r)
	}
}

// costs is the month's cost statement (sb-tenant(5), Model billing and cost statements).
func (h Admin) costs(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if _, _, err := costs.ParseMonth(month); err != nil {
		h.fail(w, errs.New(errs.EINVAL, "month must be like 2027-03"))
		return
	}
	if h.Meter == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Metering isn't available in this cell."))
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	st, err := costs.Build(r.Context(), h.Meter, h.Tenant, month, h.Prices, h.BudgetUSD, now())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The cost statement can't be built right now."))
		return
	}
	for i := range st.Lines { // SB-4405: each line's tier
		st.Lines[i].Tier = h.Tiers[st.Lines[i].Provider+"/"+st.Lines[i].Model]
	}
	if r.URL.Query().Get("format") == "csv" {
		b, err := st.CSV()
		if err != nil {
			h.fail(w, errs.New(errs.EUNAVAIL, "The cost statement can't be written."))
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("X-SB-Release", h.Release)
		_, _ = w.Write(b)
		return
	}
	h.ok(w, st)
}

// spaceCosts is each branch's spend in a month (SB-4105): amounts only.
// guardrails are the guardrail counts (sb-observability(7), Guardrail insights; SB-4406).
func (h Admin) guardrails(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Guardrails == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Guardrail insights aren't available in this cell."))
		return
	}
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}
	g, err := h.Guardrails(r.Context(), now)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The guardrail counts can't be read right now."))
		return
	}
	h.ok(w, g)
}

func (h Admin) spaceCosts(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if _, _, err := costs.ParseMonth(month); err != nil {
		h.fail(w, errs.New(errs.EINVAL, "month must be like 2027-03"))
		return
	}
	if h.SpaceSpend == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Metering by space isn't available in this cell."))
		return
	}
	rows, err := h.SpaceSpend(r.Context(), month)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The spend by space can't be read right now."))
		return
	}
	h.ok(w, map[string]any{"month": month, "spaces": rows})
}

// usage is the week's numbers (SB-606).
func (h Admin) usage(w http.ResponseWriter, r *http.Request) {
	from, to, err := report.ParseWeek(r.URL.Query().Get("week"))
	if err != nil {
		h.fail(w, errs.New(errs.EINVAL, "week must be like 2027-W10"))
		return
	}
	rep, err := report.Build(r.Context(), h.Usage, h.Tenant, from, to, h.Price)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Usage can't be read right now."))
		return
	}
	h.ok(w, rep)
}

// guard requires the operator role: policy must allow admin on this tenant's Cell.
func (h Admin) guard(next func(http.ResponseWriter, *http.Request, tenancy.Scope)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := tenancy.FromContext(r.Context())
		if err != nil {
			h.fail(w, errs.New(errs.EAUTH, "Sign in again."))
			return
		}
		res := h.Policy.Authorise(r.Context(), s, "admin", policy.Entity{Type: "Cell", ID: h.Tenant, Tenant: h.Tenant}, nil)
		if res.Decision != policy.Allow {
			h.fail(w, errs.New(errs.EPERM, "You need the operator role for this."))
			return
		}
		next(w, r, s)
	}
}

func (h Admin) manifest(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	m, commit, err := h.Manifests.Applied(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return
	}
	h.ok(w, map[string]any{"manifest": string(manifest.Format(m)), "commit": commit})
}

// treePlan is a reorg's dry run on the applied manifest (sb-space(1), tree; SB-4806):
// ?op=move&space=S&to=P or ?op=merge&space=A&to=B. It changes nothing; the proposed
// manifest goes through the usual review.
func (h Admin) treePlan(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	m, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return
	}
	q := r.URL.Query()
	var p reorg.TreePlan
	switch q.Get("op") {
	case reorg.OpMove:
		p, err = reorg.Move(m, q.Get("space"), q.Get("to"))
	case reorg.OpMerge:
		p, err = reorg.Merge(m, q.Get("space"), q.Get("to"))
	default:
		err = reorg.ErrTree
	}
	if errors.Is(err, reorg.ErrWiden) {
		h.fail(w, errs.New(errs.EPERM, "%s", strings.TrimPrefix(err.Error(), reorg.ErrWiden.Error()+": ")))
		return
	}
	if err != nil {
		h.fail(w, errs.New(errs.EINVAL, "%v", err))
		return
	}
	out := map[string]any{"plan": p, "manifest": string(manifest.Format(p.After))}
	if h.InFlight != nil {
		var spaces []string
		for _, e := range p.Spaces {
			spaces = append(spaces, e.Space)
		}
		in, err := h.InFlight(r.Context(), spaces)
		if err != nil {
			h.fail(w, errs.New(errs.EUNAVAIL, "What is under way can't be counted right now."))
			return
		}
		out["in_flight"] = in
	}
	h.ok(w, out)
}

func (h Admin) diff(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	var ops []manifest.Op
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &ops) != nil || len(ops) == 0 {
		h.fail(w, errs.New(errs.EINVAL, "The body must be a JSON Patch array."))
		return
	}
	cur, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return
	}
	next, ds, err := manifest.Patch(cur, ops)
	switch {
	case err != nil:
		h.fail(w, errs.New(errs.EMANIFEST, "%v", err))
		return
	case len(ds) > 0:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errs.HTTP(errs.EMANIFEST))
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": "EMANIFEST", "message": manifest.Count(len(ds)), "errors": ds, "remedy": errs.Remedy(errs.EMANIFEST)}})
		return
	}
	if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: "manifest", State: "proposed",
		PayloadSHA256: release.SHA256(b), Reason: "console manifest diff"}); err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The proposal couldn't be audited, so it wasn't made."))
		return
	}
	h.ok(w, map[string]any{"manifest": string(manifest.Format(next)), "diff": manifest.Diff(cur, next)})
}

func (h Admin) kill(set bool) func(http.ResponseWriter, *http.Request, tenancy.Scope) {
	return func(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
		var req struct{ Scope, Name, Reason string }
		b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
		if err != nil || json.Unmarshal(b, &req) != nil || req.Reason == "" {
			h.fail(w, errs.New(errs.EINVAL, "scope, name and reason are required."))
			return
		}
		scope := req.Scope + ":" + req.Name
		if req.Scope == "tenant" {
			scope = "tenant"
		}
		if set {
			err = h.Kill.Kill(r.Context(), h.Tenant, scope, req.Reason, s.User())
		} else {
			err = h.Kill.Unkill(r.Context(), h.Tenant, scope, req.Reason, s.User())
		}
		switch {
		case errors.Is(err, kill.ErrScope):
			h.fail(w, errs.New(errs.EINVAL, "Unknown scope."))
		case err != nil:
			h.fail(w, errs.New(errs.EUNAVAIL, "The switch couldn't be changed."))
		default:
			h.ok(w, map[string]any{"scope": scope, "killed": set})
		}
	}
}

// erase is subject deletion (sb-admin(3), sb-tenant(5) Retention and deletion).
func (h Admin) erase(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	user, reason, ok := h.subject(w, r)
	if !ok {
		return
	}
	rep, err := h.Erasure.Erase(r.Context(), s.User(), user, reason)
	switch {
	case errors.Is(err, retention.ErrInvalid):
		h.fail(w, errs.New(errs.EINVAL, "A usr_ id and a reason are required."))
	case errors.Is(err, retention.ErrHeld):
		h.fail(w, errs.New(errs.EHOLD, "A legal hold covers this person; nothing was erased. Ask a Legal steward."))
	case err != nil:
		h.fail(w, errs.New(errs.EUNAVAIL, "The erasure didn't finish; the audit log says how far it got."))
	default:
		h.ok(w, map[string]any{"erased": rep.Counts, "audit": rep.Audit})
	}
}

// subject reads {user, reason} for erase and export.
func (h Admin) subject(w http.ResponseWriter, r *http.Request) (user, reason string, ok bool) {
	return h.subjectFor(w, r, h.Erasure != nil, "Erasure isn't available in this cell.")
}

// subjectFor reads {user, reason}; a reason is required, and available false is EUNAVAIL.
func (h Admin) subjectFor(w http.ResponseWriter, r *http.Request, available bool, unavailable string) (user, reason string, ok bool) {
	var req struct {
		User   string `json:"user"`
		Reason string `json:"reason"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		h.fail(w, errs.New(errs.EINVAL, "The body must be {\"user\": \"usr_…\", \"reason\": \"…\"}."))
		return "", "", false
	}
	if !available {
		h.fail(w, errs.New(errs.EUNAVAIL, "%s", unavailable))
		return "", "", false
	}
	return req.User, req.Reason, true
}

// export is a copy of one person's derived data (sb-admin(3); SB-1903): what erase would remove.
func (h Admin) export(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	user, reason, ok := h.subject(w, r)
	if !ok {
		return
	}
	ex, err := h.Erasure.Export(r.Context(), s.User(), user, reason)
	switch {
	case errors.Is(err, retention.ErrInvalid):
		h.fail(w, errs.New(errs.EINVAL, "A usr_ id and a reason are required."))
	case err != nil:
		h.fail(w, errs.New(errs.EUNAVAIL, "The export didn't finish, so nothing is handed over; the audit log says which store failed."))
	default:
		h.ok(w, ex)
	}
}

// plan is what the retention job would delete (sb-admin(3); SB-1905), deleting nothing.
func (h Admin) plan(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	days := 0
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > retention.MaxDays {
			h.fail(w, errs.New(errs.EINVAL, "days must be 0 to 3650."))
			return
		}
		days = n
	}
	if h.Erasure == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Retention isn't available in this cell."))
		return
	}
	rep, err := h.Erasure.Plan(r.Context(), time.Duration(days)*24*time.Hour)
	switch {
	case errors.Is(err, retention.ErrDays):
		h.fail(w, errs.New(errs.EINVAL, "The manifest's retention_days is outside 30 to 3650."))
	case err != nil:
		h.fail(w, errs.New(errs.EUNAVAIL, "A store can't be read: %s.", strings.Join(rep.Failed, ", ")))
	default:
		h.ok(w, map[string]any{"days": days, "cutoff": rep.Cutoff, "would_delete": rep.Counts, "held": rep.Held})
	}
}

// scopes counts what each space's scope patterns match now (SB-2903); it changes nothing.
func (h Admin) scopes(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Index == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The index can't be read in this cell."))
		return
	}
	rows, err := h.Index.ScopeReport(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The index can't be read."))
		return
	}
	if rows == nil {
		rows = []route.PatternCount{}
	}
	h.ok(w, map[string]any{"patterns": rows})
}

// leftovers lists the index's leftovers by cite (SB-2905); it changes nothing.
func (h Admin) leftovers(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Index == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The index can't be read in this cell."))
		return
	}
	gone, unheld, err := h.Index.Leftovers(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "A source or the index can't be read, so nothing is called a leftover."))
		return
	}
	cites := func(cs []ports.Chunk) []string {
		out := []string{}
		for _, c := range cs {
			out = append(out, c.Cite)
		}
		return out
	}
	h.ok(w, map[string]any{"gone": cites(gone), "unheld": cites(unheld)})
}

// prune removes the documents their source no longer lists, with a reason (SB-2905).
func (h Admin) prune(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		h.fail(w, errs.New(errs.EINVAL, "Say why: the body is {\"reason\": \"…\"}."))
		return
	}
	if h.Index == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The index can't be read in this cell."))
		return
	}
	n, err := h.Index.Prune(r.Context(), s.User(), strings.TrimSpace(req.Reason))
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The prune didn't finish; the audit log says how far it got."))
		return
	}
	h.ok(w, map[string]any{"pruned": n})
}

// refresh re-reads one document from its source now (SB-3202).
func (h Admin) refresh(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	var req struct {
		Cite   string `json:"cite"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		h.fail(w, errs.New(errs.EINVAL, "Say which document and why: the body is {\"cite\": \"docs:…\", \"reason\": \"…\"}."))
		return
	}
	doc, _, _ := strings.Cut(req.Cite, "@")
	kind, id, _ := strings.Cut(doc, ":")
	if kind == "" || id == "" {
		h.fail(w, errs.New(errs.EINVAL, "A cite is KIND:ID, with or without @VERSION."))
		return
	}
	if h.Index == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The index can't be read in this cell."))
		return
	}
	res, err := h.Index.Refresh(r.Context(), s.User(), strings.TrimSpace(req.Reason), ports.SourceRef{Kind: kind, ID: id})
	switch {
	case errors.Is(err, ingest.ErrNoConnector):
		h.fail(w, errs.New(errs.EINVAL, "This cell has no %s connector.", kind))
		return
	case err != nil:
		h.fail(w, errs.New(errs.EUNAVAIL, "The document couldn't be read from its source now; nothing changed."))
		return
	}
	h.ok(w, map[string]any{"document": doc, "version": res.Version, "changed": res.Changed, "removed": res.Removed})
}

func (h Admin) switches(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	list, err := h.Switches.List(r.Context(), h.Tenant)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The kill store can't be read."))
		return
	}
	h.ok(w, map[string]any{"switches": list})
}

// auditFilter is GET /v1/admin/audit's actor=, from= and to= (SB-3204): an email is the one
// active user it names, and the dates are whole UTC days, both inclusive.
func (h Admin) auditFilter(r *http.Request) (actor, from, to string, err *errs.Error) {
	q := r.URL.Query()
	actor, from, to = strings.TrimSpace(q.Get("actor")), q.Get("from"), q.Get("to")
	for _, d := range []string{from, to} {
		if _, e := time.Parse("2006-01-02", d); d != "" && e != nil {
			return "", "", "", errs.New(errs.EINVAL, "from and to are dates, YYYY-MM-DD.")
		}
	}
	if strings.Contains(actor, "@") && h.ActiveByEmail != nil {
		ids, e := h.ActiveByEmail(r.Context(), actor)
		switch {
		case e != nil:
			return "", "", "", errs.New(errs.EUNAVAIL, "The directory can't be read.")
		case len(ids) != 1:
			return "", "", "", errs.New(errs.EINVAL, "No one active has that email; give their user id.")
		}
		actor = ids[0]
	}
	return actor, from, to, nil
}

func (h Admin) audit(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	kind := r.URL.Query().Get("kind")
	space := r.URL.Query().Get("space")
	actor, from, to, ferr := h.auditFilter(r)
	if ferr != nil {
		h.fail(w, ferr)
		return
	}
	rows, err := h.AuditRows.Range(r.Context(), since, 500)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The audit log can't be read."))
		return
	}
	var out []audit.Row
	for _, row := range rows {
		day := row.At
		if len(day) >= 10 {
			day = day[:10]
		}
		if (kind == "" || row.Kind == kind) && (space == "" || audit.InSpace(row, space)) && (actor == "" || row.Actor == actor) &&
			(from == "" || day >= from) && (to == "" || day <= to) {
			out = append(out, row)
		}
	}
	next := ""
	if len(rows) == 500 {
		next = strconv.FormatInt(rows[len(rows)-1].Seq+1, 10)
	}
	h.ok(w, map[string]any{"items": out, "next": next})
}

func (h Admin) ok(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SB-Release", h.Release)
	_ = json.NewEncoder(w).Encode(v)
}

func (h Admin) fail(w http.ResponseWriter, e *errs.Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errs.HTTP(e.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": string(e.Code), "message": e.Msg, "remedy": errs.Remedy(e.Code)}})
}

package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Reviews serves quarterly access reviews (sb-admin(3), Access reviews; SB-1202): stewards
// decide, removals become change requests, operators read the evidence.
type Reviews struct {
	Tenant    string
	Policy    policy.Kernel
	Manifests Manifests
	Audit     *audit.Log
	Reviews   reviews.Store
	Proposals proposals.Store
	NewID     func(prefix string) string
	Now       func() time.Time
	// Requests turns a complete review into change requests; nil is reviews.Requests.
	Requests func(t manifest.Tenant, r reviews.Review) (manifest.Tenant, bool, []reviews.Request)
}

// Register mounts the endpoints.
func (h Reviews) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/spaces/{name}/reviews", h.listSpace)
	mux.HandleFunc("POST /v1/spaces/{name}/reviews/{id}/decide", h.decide)
	mux.HandleFunc("GET /v1/admin/reviews", h.listAll)
	mux.HandleFunc("GET /v1/admin/reviews/{id}/evidence", h.evidence)
}

func (h Reviews) ok(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// steward authorises the caller as a steward of the named space.
func (h Reviews) steward(w http.ResponseWriter, r *http.Request) (tenancy.Scope, manifest.Tenant, bool) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return s, manifest.Tenant{}, false
	}
	name := r.PathValue("name")
	cur, _, err := h.Manifests.Applied(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The applied manifest can't be read."))
		return s, cur, false
	}
	sp, ok := cur.Spaces[name]
	if !ok {
		fail(w, errs.New(errs.ESPACE, "No space %q for you.", name))
		return s, cur, false
	}
	dec := h.Policy.Authorise(r.Context(), s, "review_space", policy.Entity{Type: "SpaceConfig", ID: name, Tenant: h.Tenant,
		Attrs: map[string]any{"stewards": sp.Stewards}}, map[string]any{"groups": groupsOf(s)})
	if dec.Decision != policy.Allow {
		fail(w, errs.New(errs.EPERM, "Only the space's stewards can review its access."))
		return s, cur, false
	}
	return s, cur, true
}

func (h Reviews) listSpace(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.steward(w, r); !ok {
		return
	}
	all, err := h.Reviews.List(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Reviews can't be read."))
		return
	}
	out := []reviews.Review{}
	for _, rv := range all {
		if rv.Space == r.PathValue("name") {
			out = append(out, rv)
		}
	}
	h.ok(w, map[string]any{"items": out})
}

func (h Reviews) decide(w http.ResponseWriter, r *http.Request) {
	s, cur, ok := h.steward(w, r)
	if !ok {
		return
	}
	rv, found, err := h.Reviews.Get(r.Context(), r.PathValue("id"))
	switch {
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "Reviews can't be read."))
		return
	case !found || rv.Space != r.PathValue("name"):
		fail(w, errs.New(errs.EINVAL, "No review %q in this space.", r.PathValue("id")))
		return
	}
	var req struct{ User, Decision, Reason string }
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		fail(w, errs.New(errs.EINVAL, "The body must be {\"user\", \"decision\", \"reason\"}."))
		return
	}
	now := h.Now()
	next, err := reviews.Decide(rv, req.User, req.Decision, req.Reason, s.User(), now)
	switch {
	case errors.Is(err, reviews.ErrSelf):
		fail(w, errs.New(errs.EPERM, "Nobody decides on their own access; ask another steward or an operator."))
		return
	case err != nil:
		fail(w, errs.New(errs.EINVAL, "%v", err))
		return
	}
	reason := "access review " + rv.ID + ": " + req.Decision + " " + req.User
	if req.Reason != "" {
		reason += ": " + req.Reason
	}
	row, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "access", Subject: rv.ID + ":" + req.User,
		State: req.Decision, Reason: reason, Space: "spc_" + rv.Space})
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The decision couldn't be audited, so it wasn't made."))
		return
	}
	for i := range next.Items {
		if next.Items[i].User == req.User {
			next.Items[i].AuditSeq = row.Seq
		}
	}
	if next.State == reviews.Complete {
		requests := h.Requests
		if requests == nil {
			requests = reviews.Requests
		}
		nt, changed, reqs := requests(cur, next)
		if changed {
			if t, ds := manifest.Check(manifest.Format(nt)); len(ds) == 0 {
				p := proposals.Proposal{ID: h.NewID("prop_"), Space: rv.Space, Requester: s.User(), Base: proposals.Hash(cur),
					Hash: proposals.Hash(t), Manifest: string(manifest.Format(t)), Diff: manifest.Diff(cur, t), State: proposals.Open, At: now.UTC()}
				if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: p.ID, State: "proposed",
					PayloadSHA256: release.SHA256([]byte(p.Manifest)), Reason: "access review " + rv.ID + " removals", Space: "spc_" + rv.Space}); err != nil {
					fail(w, errs.New(errs.EUNAVAIL, "The removals couldn't be audited."))
					return
				}
				if err := h.Proposals.Put(r.Context(), p); err != nil {
					fail(w, errs.New(errs.EUNAVAIL, "The removals couldn't be stored."))
					return
				}
				for i := range reqs {
					if reqs[i].Kind == "manifest" {
						reqs[i].Proposal = p.ID
					}
				}
			} else {
				// A removal that would leave the manifest invalid goes to the operator as an
				// IdP-style request to resolve by hand, never silently dropped.
				for i := range reqs {
					if reqs[i].Kind == "manifest" {
						reqs[i].Kind = "idp"
					}
				}
			}
		}
		next.Requests = reqs
	}
	if err := h.Reviews.Put(r.Context(), next); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The decision couldn't be stored."))
		return
	}
	h.ok(w, next)
}

func (h Reviews) operator(w http.ResponseWriter, r *http.Request) bool {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return false
	}
	if h.Policy.Authorise(r.Context(), s, "admin", policy.Entity{Type: "Cell", ID: h.Tenant, Tenant: h.Tenant}, nil).Decision != policy.Allow {
		fail(w, errs.New(errs.EPERM, "You need the operator role for this."))
		return false
	}
	return true
}

func (h Reviews) listAll(w http.ResponseWriter, r *http.Request) {
	if !h.operator(w, r) {
		return
	}
	all, err := h.Reviews.List(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Reviews can't be read."))
		return
	}
	now := h.Now()
	type row struct {
		reviews.Review
		Overdue bool `json:"overdue"`
	}
	out := []row{}
	for _, rv := range all {
		out = append(out, row{rv, reviews.Overdue(rv, now)})
	}
	h.ok(w, map[string]any{"items": out})
}

func (h Reviews) evidence(w http.ResponseWriter, r *http.Request) {
	if !h.operator(w, r) {
		return
	}
	rv, found, err := h.Reviews.Get(r.Context(), r.PathValue("id"))
	switch {
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "Reviews can't be read."))
		return
	case !found:
		fail(w, errs.New(errs.EINVAL, "No review %q.", r.PathValue("id")))
		return
	}
	doc, err := reviews.Evidence(rv)
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The evidence couldn't be made."))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(doc)
}

package api

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/health"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

var traceID = regexp.MustCompile(`^tr_[A-Za-z0-9_]{1,64}$`)

// Operations serves the console's health and its unhelpful-answer queue (sb-admin(3);
// SB-1607, SB-1609), behind the operator role like the rest of the admin API.
type Operations struct {
	Admin Admin
	// Health builds the cell's health from its own records.
	Health func(ctx context.Context, now time.Time) (health.Report, error)
	// Feedback and Traces are the cell's feedback and its traces, for the queue.
	Feedback interface {
		ask.FeedbackStore
		ask.FeedbackReader
	}
	Traces ask.Store
}

// Register mounts the endpoints.
func (h Operations) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/admin/health", h.Admin.guard(h.health))
	mux.HandleFunc("GET /v1/admin/feedback", h.Admin.guard(h.queue))
	mux.HandleFunc("POST /v1/admin/feedback/{trace}/reviewed", h.Admin.guard(h.reviewed))
}

func (h Operations) now() time.Time {
	if h.Admin.Now != nil {
		return h.Admin.Now()
	}
	return time.Now()
}

func (h Operations) health(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Health == nil {
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "Health isn't available in this cell."))
		return
	}
	rep, err := h.Health(r.Context(), h.now())
	if err != nil {
		// Never a healthy-looking report the cell couldn't read (LAW-03's rule, for health).
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "The cell's health records can't be read right now."))
		return
	}
	h.Admin.ok(w, rep)
}

func (h Operations) queue(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Feedback == nil || h.Traces == nil {
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "Feedback isn't available in this cell."))
		return
	}
	q := r.URL.Query()
	rating := q.Get("rating")
	if rating == "" {
		rating = "down"
	}
	items, err := ask.Queue(r.Context(), h.Feedback, h.Traces, h.Admin.Tenant, rating, q.Get("reviewed") == "true", h.now())
	switch {
	case errors.Is(err, ask.ErrInvalid):
		h.Admin.fail(w, errs.New(errs.EINVAL, "rating must be up or down"))
		return
	case err != nil:
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "Feedback can't be read right now."))
		return
	}
	h.Admin.ok(w, map[string]any{"items": items})
}

func (h Operations) reviewed(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	if h.Feedback == nil {
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "Feedback isn't available in this cell."))
		return
	}
	trace := r.PathValue("trace")
	if !traceID.MatchString(trace) {
		h.Admin.fail(w, errs.New(errs.EINVAL, "That isn't a trace id."))
		return
	}
	cur, found, err := h.Feedback.GetFeedback(r.Context(), h.Admin.Tenant, trace)
	switch {
	case err != nil:
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "Feedback can't be read right now."))
		return
	case !found:
		h.Admin.fail(w, errs.New(errs.EINVAL, "There's no feedback on that trace."))
		return
	}
	if cur.ReviewedBy == "" && h.Admin.Audit != nil {
		// The audit row first: a review the log doesn't show didn't happen.
		if _, err := h.Admin.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: "feedback:" + trace,
			State: "Reviewed", Reason: "feedback_reviewed"}); err != nil {
			h.Admin.fail(w, errs.New(errs.EUNAVAIL, "The review can't be recorded in the audit log."))
			return
		}
	}
	f, _, err := ask.MarkReviewed(r.Context(), h.Feedback, h.Feedback, h.Admin.Tenant, trace, s.User(), h.now())
	switch {
	case errors.Is(err, ask.ErrNotFound):
		h.Admin.fail(w, errs.New(errs.EINVAL, "There's no feedback on that trace."))
		return
	case err != nil:
		h.Admin.fail(w, errs.New(errs.EUNAVAIL, "The review can't be recorded right now."))
		return
	}
	h.Admin.ok(w, map[string]any{"trace": trace, "reviewed": ask.Review{By: f.ReviewedBy, At: f.ReviewedAt.UTC()}})
}

package api

import (
	"errors"
	"net/http"
	"time"

	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/errs"
)

// Notifications serves a person's notification settings (sb-ask(3), Notifications; SB-2101).
type Notifications struct {
	Prefs   notify.PrefStore
	Release string
	NewID   func(prefix string) string
	Now     func() time.Time // nil is time.Now
	// Test sends the caller the test notification (SB-2602); nil is EUNAVAIL.
	Test NotifyTest
}

// Register mounts the endpoints on mux.
func (h Notifications) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me/notifications", h.get)
	mux.HandleFunc("PUT /v1/me/notifications", h.put)
	mux.HandleFunc("POST /v1/me/notifications/test", h.test)
}

func (h Notifications) k() Knowledge { return Knowledge{Release: h.Release, NewID: h.NewID} }

func (h Notifications) get(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	p, err := h.Prefs.NotifyPrefs(r.Context(), s.Tenant(), s.User())
	if err != nil {
		h.k().out().fail(w, "", errs.New(errs.EUNAVAIL, "Your notification settings can't be read now. Try again shortly."))
		return
	}
	if p.Off == nil {
		p.Off = []string{}
	}
	h.k().out().ok(w, "", p)
}

func (h Notifications) put(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Off       []string `json:"off"`
		PauseDays *int     `json:"pause_days"`
		Digest    *string  `json:"digest"` // SB-3805; nil keeps it
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"off": ["digest"]}.`))
		return
	}
	old, err := h.Prefs.NotifyPrefs(r.Context(), s.Tenant(), s.User())
	if err != nil {
		h.k().out().fail(w, "", errs.New(errs.EUNAVAIL, "Your notification settings can't be saved now. Try again shortly."))
		return
	}
	// A pause is set only by pause_days, from now; one sent as paused_until is ignored (SB-2506).
	digest := old.Digest
	if body.Digest != nil {
		digest = *body.Digest
	}
	p, err := notify.Prefs{Off: body.Off, PausedUntil: old.PausedUntil, Digest: digest}.Check()
	if errors.Is(err, notify.ErrDigest) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The digest is daily or weekly."))
		return
	}
	if errors.Is(err, notify.ErrKind) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "Each kind turned off is one of questions, replies, digest, watch, access, tokens or owned."))
		return
	}
	if body.PauseDays != nil {
		now := time.Now
		if h.Now != nil {
			now = h.Now
		}
		if p, err = p.Pause(*body.PauseDays, now()); err != nil {
			h.k().out().fail(w, "", errs.New(errs.EINVAL, "A pause is 0 to 30 days."))
			return
		}
	}
	if err := h.Prefs.PutNotifyPrefs(r.Context(), s.Tenant(), s.User(), p); err != nil {
		h.k().out().fail(w, "", errs.New(errs.EUNAVAIL, "Your notification settings can't be saved now. Try again shortly."))
		return
	}
	h.k().out().ok(w, "", p)
}

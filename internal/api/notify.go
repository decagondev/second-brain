package api

import (
	"errors"
	"net/http"

	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/errs"
)

// Notifications serves a person's notification settings (sb-ask(3), Notifications; SB-2101).
type Notifications struct {
	Prefs   notify.PrefStore
	Release string
	NewID   func(prefix string) string
}

// Register mounts the endpoints on mux.
func (h Notifications) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me/notifications", h.get)
	mux.HandleFunc("PUT /v1/me/notifications", h.put)
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
	var body notify.Prefs
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"off": ["digest"]}.`))
		return
	}
	p, err := body.Check()
	if errors.Is(err, notify.ErrKind) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "Each kind turned off is one of questions, replies, digest, watch or access."))
		return
	}
	if err := h.Prefs.PutNotifyPrefs(r.Context(), s.Tenant(), s.User(), p); err != nil {
		h.k().out().fail(w, "", errs.New(errs.EUNAVAIL, "Your notification settings can't be saved now. Try again shortly."))
		return
	}
	h.k().out().ok(w, "", p)
}

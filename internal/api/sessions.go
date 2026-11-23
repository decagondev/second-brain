package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/session"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Revoker ends every session of a person (session.Revoker).
type Revoker interface {
	Revoke(ctx context.Context, actor, user, reason string) (int, []string, error)
}

// Sessions serves POST /v1/sessions/revoke (sb-api(3); SB-1906): the caller signs out
// everywhere, this browser included.
type Sessions struct {
	Revoker Revoker
}

// Register mounts the endpoint.
func (h Sessions) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/sessions/revoke", func(w http.ResponseWriter, r *http.Request) {
		s, err := tenancy.FromContext(r.Context())
		if err != nil {
			fail(w, errs.New(errs.EAUTH, "Sign in again."))
			return
		}
		n, ids, err := h.Revoker.Revoke(r.Context(), s.User(), s.User(), "signed out everywhere")
		if err != nil {
			fail(w, revokeErr(err))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "sb_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
		writeJSON(w, map[string]any{"revoked": n, "audit": ids})
	})
}

func revokeErr(err error) *errs.Error {
	if errors.Is(err, session.ErrUser) {
		return errs.New(errs.EINVAL, "A usr_ id is required.")
	}
	return errs.New(errs.EUNAVAIL, "The sessions weren't revoked; try again. The audit log says how far it got.")
}

// revokeSessions is the operator's POST /v1/admin/sessions/revoke (sb-admin(3); SB-1906).
func (h Admin) revokeSessions(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	user, reason, ok := h.subjectFor(w, r, h.Sessions != nil, "Session revocation isn't available in this cell.")
	if !ok {
		return
	}
	if strings.TrimSpace(reason) == "" {
		h.fail(w, errs.New(errs.EINVAL, "A reason is required."))
		return
	}
	n, ids, err := h.Sessions.Revoke(r.Context(), s.User(), user, reason)
	if err != nil {
		h.fail(w, revokeErr(err))
		return
	}
	h.ok(w, map[string]any{"user": user, "revoked": n, "audit": ids})
}

package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/session"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Tokens serves /v1/tokens (sb-api(3), Read-only API tokens; SB-2306): a person makes,
// lists and revokes their own read-only API tokens. Only a browser session reaches these:
// auth refuses a token anywhere but ask and search.
type Tokens struct {
	Sessions ports.Sessions
	Audit    session.Auditor
	Now      func() time.Time
}

type tokenView struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
	Expires time.Time `json:"expires"`
	Token   string    `json:"token,omitempty"` // once, when it is made
}

// Register mounts the endpoints.
func (h Tokens) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/tokens", h.make)
	mux.HandleFunc("GET /v1/tokens", h.list)
	mux.HandleFunc("DELETE /v1/tokens/{id}", h.revoke)
}

func (h Tokens) make(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var req struct {
		Name string `json:"name"`
		Days int    `json:"days"`
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(body) > MaxBody || json.Unmarshal(body, &req) != nil {
		fail(w, errs.New(errs.EINVAL, "The request body must be JSON with a name and days."))
		return
	}
	tok, x, err := session.NewToken(rand.Reader, s.Tenant(), s.User(), req.Name, req.Days, h.Now().UTC())
	if errors.Is(err, session.ErrToken) {
		fail(w, errs.New(errs.EINVAL, "A token needs a name of 1 to 60 characters and 1 to 90 days."))
		return
	}
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The token wasn't made; try again."))
		return
	}
	// Audited first: a token nobody can account for is never made.
	if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "access", Subject: "token:" + x.Token, State: "token_created", Reason: x.Expires.Format(time.DateOnly)}); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The token wasn't made: the audit log is unavailable."))
		return
	}
	if err := h.Sessions.Put(r.Context(), x); err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "The token wasn't made; try again."))
		return
	}
	v := view(x)
	v.Token = tok
	writeJSON(w, v)
}

func view(x ports.Session) tokenView {
	return tokenView{ID: x.Token, Name: x.Name, Created: x.SignedIn, Expires: x.Expires}
}

func (h Tokens) mine(r *http.Request) (tenancy.Scope, []ports.Session, *errs.Error) {
	s, serr := tenancy.FromContext(r.Context())
	if serr != nil {
		return s, nil, errs.New(errs.EAUTH, "Sign in again.")
	}
	all, err := h.Sessions.ListUser(r.Context(), s.Tenant(), s.User())
	if err != nil {
		return s, nil, errs.New(errs.EUNAVAIL, "Your tokens can't be read right now.")
	}
	now := h.Now()
	var out []ports.Session
	for _, x := range all {
		if x.Token != "" && now.Before(x.Expires) {
			out = append(out, x)
		}
	}
	return s, out, nil
}

func (h Tokens) list(w http.ResponseWriter, r *http.Request) {
	_, toks, e := h.mine(r)
	if e != nil {
		fail(w, e)
		return
	}
	items := make([]tokenView, 0, len(toks))
	for _, x := range toks {
		items = append(items, view(x))
	}
	writeJSON(w, map[string]any{"items": items})
}

func (h Tokens) revoke(w http.ResponseWriter, r *http.Request) {
	s, toks, e := h.mine(r)
	if e != nil {
		fail(w, e)
		return
	}
	id := r.PathValue("id")
	for _, x := range toks {
		if x.Token != id {
			continue
		}
		if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "access", Subject: "token:" + id, State: "token_revoked", Reason: "revoked by its owner"}); err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "The token wasn't revoked: the audit log is unavailable."))
			return
		}
		if err := h.Sessions.Delete(r.Context(), x.ID); err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "The token wasn't revoked; try again."))
			return
		}
		writeJSON(w, map[string]any{"id": id, "revoked": true})
		return
	}
	fail(w, errs.New(errs.EPERM, "That token isn't yours, has expired or doesn't exist."))
}

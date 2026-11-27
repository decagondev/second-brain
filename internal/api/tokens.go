package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
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
	// Clients are sb-cell(5) mcp_clients: the AI tools a token may be made for; empty is any
	// (SB-6006). Families are the MCP tool families a person's groups grant; nil is every one.
	Clients  []string
	Families func(s tenancy.Scope) (map[string]bool, bool)
}

type tokenView struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Client   string    `json:"client,omitempty"` // the AI tool it was made for (SB-6006)
	Created  time.Time `json:"created"`
	Expires  time.Time `json:"expires"`
	LastUsed string    `json:"last_used,omitempty"`
	Token    string    `json:"token,omitempty"` // once, when it is made
}

// Register mounts the endpoints.
func (h Tokens) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/tokens", h.make)
	mux.HandleFunc("GET /v1/tokens", h.list)
	mux.HandleFunc("DELETE /v1/tokens/{id}", h.revoke)
	mux.HandleFunc("GET /v1/me/ai-tools", h.aiTools) // SB-6006
	mux.HandleFunc("GET /v1/me/sessions", h.sessions)
	mux.HandleFunc("DELETE /v1/me/sessions/{id}", h.signOut)
}

func (h Tokens) make(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var req struct {
		Name   string `json:"name"`
		Days   int    `json:"days"`
		Client string `json:"client"`
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
	if req.Client != "" || len(h.Clients) > 0 { // SB-6006
		if !session.ValidClient(req.Client) {
			fail(w, errs.New(errs.EINVAL, "Name the AI tool the token is for: a short lower-case word, such as claude-code."))
			return
		}
		if len(h.Clients) > 0 && !slices.Contains(h.Clients, req.Client) {
			fail(w, errs.New(errs.EPERM, "This cell allows tokens only for these AI tools: %s.", strings.Join(h.Clients, ", ")))
			return
		}
		x.Client = req.Client
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
	v := tokenView{ID: x.Token, Name: x.Name, Client: x.Client, Created: x.SignedIn, Expires: x.Expires}
	if x.Validated.After(x.SignedIn) {
		v.LastUsed = x.Validated.UTC().Format(time.DateOnly)
	}
	return v
}

// aiTools is GET /v1/me/ai-tools: the person's own tokens made for an AI tool, each with
// its last use and the MCP families the person's groups grant (SB-6006). Revoke one with
// DELETE /v1/tokens/{id}.
func (h Tokens) aiTools(w http.ResponseWriter, r *http.Request) {
	s, toks, e := h.mine(r)
	if e != nil {
		fail(w, e)
		return
	}
	families := []string{}
	if h.Families == nil {
		families = append(families, MCPFamilies...)
	} else if fs, ok := h.Families(s); ok {
		for _, f := range MCPFamilies {
			if fs[f] {
				families = append(families, f)
			}
		}
	}
	items := []tokenView{}
	for _, x := range toks {
		if x.Client != "" {
			items = append(items, view(x))
		}
	}
	writeJSON(w, map[string]any{"items": items, "families": families})
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

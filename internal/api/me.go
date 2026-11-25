package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/session"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Me serves the person's own data (sb-ask(3), Your own data; SB-3501, SB-3502).
type Me struct {
	Data interface {
		Export(ctx context.Context, actor, user, reason string) (retention.Export, error)
		Forget(ctx context.Context, user string) (retention.Report, error)
	}
	Agent    ask.Agent
	Sessions ports.Sessions
	Now      func() time.Time
}

// ForgetPhrase is what the person types to forget their questions.
const ForgetPhrase = "forget my questions"

// Register mounts the endpoints.
func (h Me) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me/export", h.export)
	mux.HandleFunc("POST /v1/me/forget", h.forget)
	mux.HandleFunc("GET /v1/me/memory", h.memory)
	mux.HandleFunc("POST /v1/me/memory", h.remember)
	mux.HandleFunc("DELETE /v1/me/memory", h.forgetMemory)
	mux.HandleFunc("DELETE /v1/me/memory/{id}", h.forgetMemory)
}

// Memory I control (sb-ask(3); SB-4303): the caller's own lines only.

func (h Me) memoryErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ask.ErrMemoryOff):
		fail(w, errs.New(errs.EUNAVAIL, "Memory is off for your organisation."))
	case errors.Is(err, ask.ErrFromDocument):
		fail(w, errs.New(errs.EINVAL, "That reads like a document, not something about you."))
	case errors.Is(err, ask.ErrInvalid):
		fail(w, errs.New(errs.EINVAL, "%s", strings.TrimPrefix(err.Error(), ask.ErrInvalid.Error()+": ")))
	default:
		fail(w, errs.New(errs.EUNAVAIL, "Your memory can't be read or changed right now; nothing was changed."))
	}
}

func (h Me) memory(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	v, err := h.Agent.MemoryFor(r.Context(), s)
	if err != nil {
		h.memoryErr(w, err)
		return
	}
	writeJSON(w, v)
}

func (h Me) remember(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if !readJSON(r, &body) {
		fail(w, errs.New(errs.EINVAL, `The body must be JSON such as {"text": "I work in finance"}.`))
		return
	}
	l, err := h.Agent.Remember(r.Context(), s, body.Text)
	if err != nil {
		h.memoryErr(w, err)
		return
	}
	writeJSON(w, l)
}

func (h Me) forgetMemory(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	n, err := h.Agent.ForgetLine(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.memoryErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"removed": n})
}

type ownToken struct {
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
	Expires time.Time `json:"expires"`
}

func (h Me) export(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	if h.Data == nil {
		fail(w, errs.New(errs.EUNAVAIL, "Your data can't be exported in this cell."))
		return
	}
	x, err := h.Data.Export(r.Context(), s.User(), s.User(), "own export")
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Your data can't be exported right now; nothing was handed over."))
		return
	}
	records, left, err := h.Agent.OwnExport(r.Context(), s, x.Records)
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Your data can't be checked right now; nothing was handed over."))
		return
	}
	toks := []ownToken{}
	if h.Sessions != nil {
		all, err := h.Sessions.ListUser(r.Context(), s.Tenant(), s.User())
		if err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "Your tokens can't be read right now; nothing was handed over."))
			return
		}
		now := h.Now()
		for _, t := range all {
			if t.Token != "" && now.Before(t.Expires) {
				toks = append(toks, ownToken{Name: t.Name, Created: t.SignedIn, Expires: t.Expires})
			}
		}
		sort.Slice(toks, func(i, j int) bool { return toks[i].Created.After(toks[j].Created) })
	}
	w.Header().Set("Content-Disposition", `attachment; filename="my-second-brain-data.json"`)
	writeJSON(w, map[string]any{"user": s.User(), "records": records, "tokens": toks, "left_out": left})
}

func (h Me) forget(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var body struct {
		Confirm string `json:"confirm"`
	}
	if !readJSON(r, &body) || body.Confirm != ForgetPhrase {
		fail(w, errs.New(errs.EINVAL, `Type "`+ForgetPhrase+`" to confirm.`))
		return
	}
	if h.Data == nil {
		fail(w, errs.New(errs.EUNAVAIL, "Your questions can't be forgotten in this cell."))
		return
	}
	rep, err := h.Data.Forget(r.Context(), s.User())
	if err != nil {
		fail(w, errs.New(errs.EUNAVAIL, "Not everything was forgotten; try again. The audit log says how far it got."))
		return
	}
	writeJSON(w, map[string]any{"erased": rep.Counts, "held": rep.Held})
}

// Where I'm signed in (sb-identity(7); SB-3503).

type sessionView struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	Name     string    `json:"name,omitempty"`
	Started  time.Time `json:"started"`
	LastUsed string    `json:"last_used"`
	Expires  time.Time `json:"expires"`
	Current  bool      `json:"current,omitempty"`
}

// publicID is a session's id as its owner sees it: never its key.
func publicID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "ses_" + hex.EncodeToString(sum[:8])
}

// live are the caller's sessions that haven't ended.
func (h Tokens) live(r *http.Request) (tenancy.Scope, []ports.Session, *errs.Error) {
	s, serr := tenancy.FromContext(r.Context())
	if serr != nil {
		return s, nil, errs.New(errs.EAUTH, "Sign in again.")
	}
	all, err := h.Sessions.ListUser(r.Context(), s.Tenant(), s.User())
	if err != nil {
		return s, nil, errs.New(errs.EUNAVAIL, "Your sessions can't be read right now.")
	}
	now := h.Now()
	var out []ports.Session
	for _, x := range all {
		end := x.SignedIn.Add(session.MaxAge)
		if x.Token != "" {
			end = x.Expires
		}
		if x.Tenant == s.Tenant() && x.User == s.User() && now.Before(end) {
			out = append(out, x)
		}
	}
	return s, out, nil
}

func (h Tokens) sessions(w http.ResponseWriter, r *http.Request) {
	_, all, e := h.live(r)
	if e != nil {
		fail(w, e)
		return
	}
	current := ""
	if ck, err := r.Cookie("sb_session"); err == nil {
		current = ck.Value
	}
	items := []sessionView{}
	for _, x := range all {
		v := sessionView{ID: publicID(x.ID), Kind: "web", Started: x.SignedIn, LastUsed: x.Validated.UTC().Format(time.DateOnly),
			Expires: x.SignedIn.Add(session.MaxAge), Current: x.Token == "" && x.ID == current}
		if x.Token != "" {
			v.Kind, v.Name, v.Expires = "token", x.Name, x.Expires
		}
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Started.After(items[j].Started) })
	writeJSON(w, map[string]any{"items": items})
}

func (h Tokens) signOut(w http.ResponseWriter, r *http.Request) {
	s, all, e := h.live(r)
	if e != nil {
		fail(w, e)
		return
	}
	id := r.PathValue("id")
	for _, x := range all {
		if publicID(x.ID) != id {
			continue
		}
		if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "access", Subject: "session:" + id, State: "signed_out", Reason: "signed out by its owner"}); err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "The session wasn't ended: the audit log is unavailable."))
			return
		}
		if err := h.Sessions.Delete(r.Context(), x.ID); err != nil {
			fail(w, errs.New(errs.EUNAVAIL, "The session wasn't ended; try again."))
			return
		}
		writeJSON(w, map[string]any{"id": id, "signed_out": true})
		return
	}
	fail(w, errs.New(errs.EPERM, "That session isn't yours, has ended or doesn't exist."))
}

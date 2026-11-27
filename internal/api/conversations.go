package api

import (
	"errors"
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Your conversations (sb-api(3), sb-ask(3); SB-5802 to SB-5804): the caller's own list,
// one of them to continue, every device on or off, and a whole conversation shared.

// registerConversations mounts the endpoints.
func (h Me) registerConversations(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me/conversations", h.conversations)
	mux.HandleFunc("PUT /v1/me/conversations/sync", h.syncConversations)
	mux.HandleFunc("GET /v1/me/conversations/{id}", h.conversation)
	mux.HandleFunc("POST /v1/me/conversations/{id}/share", h.shareConversation)
	mux.HandleFunc("GET /v1/conversation-shares/{id}", h.openConversationShare)
	mux.HandleFunc("DELETE /v1/conversation-shares/{id}", h.revokeConversationShare)
}

func (h Me) conversationErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ask.ErrNoConversation), errors.Is(err, ask.ErrNoShare):
		fail(w, errs.New(errs.EPERM, "There's no such conversation for you."))
	case errors.Is(err, ask.ErrInvalid):
		fail(w, errs.New(errs.EINVAL, "That isn't a conversation."))
	default:
		fail(w, errs.New(errs.EUNAVAIL, "Your conversations can't be read right now; nothing was changed."))
	}
}

func (h Me) withScope(w http.ResponseWriter, r *http.Request) (tenancy.Scope, bool) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return tenancy.Scope{}, false
	}
	return s, true
}

func (h Me) conversations(w http.ResponseWriter, r *http.Request) {
	s, ok := h.withScope(w, r)
	if !ok {
		return
	}
	v, err := h.Agent.ListConversations(r.Context(), s)
	if err != nil {
		h.conversationErr(w, err)
		return
	}
	writeJSON(w, v)
}

func (h Me) syncConversations(w http.ResponseWriter, r *http.Request) {
	s, ok := h.withScope(w, r)
	if !ok {
		return
	}
	if tenancy.ClientOf(r.Context()) != tenancy.ClientWeb { // the person's own choice, in the web app
		fail(w, errs.New(errs.EPERM, "Turn this on or off in the web app."))
		return
	}
	var body struct {
		On *bool `json:"on"`
	}
	if !readJSON(r, &body) || body.On == nil {
		fail(w, errs.New(errs.EINVAL, `The body must be JSON such as {"on": true}.`))
		return
	}
	v, err := h.Agent.SetSync(r.Context(), s, *body.On)
	if err != nil {
		h.conversationErr(w, err)
		return
	}
	writeJSON(w, v)
}

func (h Me) conversation(w http.ResponseWriter, r *http.Request) {
	s, ok := h.withScope(w, r)
	if !ok {
		return
	}
	v, err := h.Agent.OpenConversation(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.conversationErr(w, err)
		return
	}
	writeJSON(w, v)
}

func (h Me) shareConversation(w http.ResponseWriter, r *http.Request) {
	s, ok := h.withScope(w, r)
	if !ok {
		return
	}
	if h.Shares == nil {
		fail(w, errs.New(errs.EUNAVAIL, "Conversations can't be shared in this cell."))
		return
	}
	sh, err := h.Shares.Share(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.conversationErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"id": sh.ID, "url": "/c/" + sh.ID, "expires": sh.Expires})
}

func (h Me) openConversationShare(w http.ResponseWriter, r *http.Request) {
	s, ok := h.withScope(w, r)
	if !ok {
		return
	}
	if h.Shares == nil {
		fail(w, errs.New(errs.EUNAVAIL, "Conversations can't be shared in this cell."))
		return
	}
	v, err := h.Shares.Open(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.conversationErr(w, err)
		return
	}
	writeJSON(w, v)
}

func (h Me) revokeConversationShare(w http.ResponseWriter, r *http.Request) {
	s, ok := h.withScope(w, r)
	if !ok {
		return
	}
	if h.Shares == nil {
		fail(w, errs.New(errs.EUNAVAIL, "Conversations can't be shared in this cell."))
		return
	}
	if err := h.Shares.Revoke(r.Context(), s, r.PathValue("id")); err != nil {
		h.conversationErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"revoked": true})
}

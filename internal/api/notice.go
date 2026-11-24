package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// The notice to everyone (sb-admin(3), PUT /v1/admin/notice; SB-3207).

func (h Admin) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h Admin) setNotice(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	var req struct {
		Text  string    `json:"text"`
		Until time.Time `json:"until"`
	}
	now := h.now()
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<14)).Decode(&req); err != nil || ask.CheckNotice(req.Text, req.Until, now) != nil {
		h.fail(w, errs.New(errs.EINVAL, "A notice is one line of plain text of at most %d characters, with no link or markup, ending in the future and within 7 days: {\"text\": \"…\", \"until\": \"2027-03-12T18:00:00Z\"}.", ask.MaxNoticeText))
		return
	}
	if h.Notices == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "This cell can't keep a notice."))
		return
	}
	text := strings.TrimSpace(req.Text)
	if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: "notice", State: "set",
		PayloadSHA256: release.SHA256([]byte(text)), Reason: "notice until " + req.Until.UTC().Format(time.RFC3339)}); err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The notice couldn't be audited, so it wasn't set."))
		return
	}
	n := ask.CellNotice{Tenant: h.Tenant, Text: text, Until: req.Until.UTC(), By: s.User(), At: now.UTC()}
	if err := h.Notices.PutNotice(r.Context(), n); err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The notice couldn't be kept; try again."))
		return
	}
	h.ok(w, map[string]any{"text": n.Text, "until": n.Until})
}

func (h Admin) clearNotice(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	if h.Notices == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "This cell can't keep a notice."))
		return
	}
	n, ok, err := h.Notices.Notice(r.Context(), h.Tenant)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The notice can't be read now; nothing changed."))
		return
	}
	if !ok {
		h.ok(w, map[string]any{"cleared": false})
		return
	}
	if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "config", Subject: "notice", State: "cleared",
		PayloadSHA256: release.SHA256([]byte(n.Text)), Reason: "notice cleared"}); err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Clearing the notice couldn't be audited, so it wasn't cleared."))
		return
	}
	if err := h.Notices.DeleteNotice(r.Context(), h.Tenant); err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The notice couldn't be cleared; try again."))
		return
	}
	h.ok(w, map[string]any{"cleared": true})
}

// notice is GET /v1/notice, for anyone signed in: the notice while it stands, {} after.
func (h Admin) notice(w http.ResponseWriter, r *http.Request) {
	if _, err := tenancy.FromContext(r.Context()); err != nil {
		h.fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	n, ok, err := ask.Standing(r.Context(), h.Notices, h.Tenant, h.now())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The notice can't be read now."))
		return
	}
	if !ok {
		h.ok(w, map[string]any{})
		return
	}
	h.ok(w, map[string]any{"text": n.Text, "until": n.Until})
}

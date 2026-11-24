package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Every token in one place (sb-admin(3); SB-3803): an admin lists the tenant's live API
// tokens, oldest unused first, and revokes one with a reason, audited first.

// TokenLister is every token of a tenant, and the one way to end one.
type TokenLister interface {
	ListTokens(ctx context.Context, tenant string) ([]ports.Session, error)
	Delete(ctx context.Context, id string) error
}

// AdminToken is one token as an admin sees it: never the token or its hash.
type AdminToken struct {
	ID       string    `json:"id"`
	Owner    string    `json:"owner"`
	Name     string    `json:"name"`
	Created  time.Time `json:"created"`
	LastUsed string    `json:"last_used,omitempty"`
	Expires  time.Time `json:"expires"`
}

// AdminTokens is the live tokens of tenant at now, oldest unused first: the ones never
// used, by when they were made, then by the day they were last used.
func AdminTokens(all []ports.Session, tenant string, now time.Time) []AdminToken {
	out := []AdminToken{}
	for _, x := range all {
		if x.Token == "" || x.Tenant != tenant || !now.Before(x.Expires) {
			continue
		}
		t := AdminToken{ID: x.Token, Owner: x.User, Name: x.Name, Created: x.SignedIn.UTC(), Expires: x.Expires.UTC()}
		if x.Validated.After(x.SignedIn) {
			t.LastUsed = x.Validated.UTC().Format(time.DateOnly)
		}
		out = append(out, t)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.LastUsed == "") != (b.LastUsed == "") {
			return a.LastUsed == ""
		}
		if a.LastUsed != b.LastUsed {
			return a.LastUsed < b.LastUsed
		}
		return a.Created.Before(b.Created)
	})
	return out
}

func (h Admin) listTokens(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Tokens == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Tokens can't be listed in this cell."))
		return
	}
	all, err := h.Tokens.ListTokens(r.Context(), h.Tenant)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The tokens can't be read right now."))
		return
	}
	h.ok(w, map[string]any{"items": AdminTokens(all, h.Tenant, h.now())})
}

func (h Admin) revokeToken(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	if h.Tokens == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Tokens can't be revoked in this cell."))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil || strings.TrimSpace(req.Reason) == "" {
		h.fail(w, errs.New(errs.EINVAL, "Say why the token is revoked."))
		return
	}
	id := r.PathValue("id")
	all, err := h.Tokens.ListTokens(r.Context(), h.Tenant)
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The tokens can't be read right now."))
		return
	}
	now := h.now()
	for _, x := range all {
		if x.Token != id || x.Tenant != h.Tenant || !now.Before(x.Expires) {
			continue
		}
		// Audited first: a revoke nobody can account for never happens.
		if _, err := h.Audit.Write(r.Context(), audit.Event{Actor: s.User(), Kind: "access", Subject: "token:" + id,
			State: "token_revoked", Reason: strings.TrimSpace(req.Reason)}); err != nil {
			h.fail(w, errs.New(errs.EUNAVAIL, "The token wasn't revoked: the audit log is unavailable."))
			return
		}
		if err := h.Tokens.Delete(r.Context(), x.ID); err != nil {
			h.fail(w, errs.New(errs.EUNAVAIL, "The token wasn't revoked; try again."))
			return
		}
		h.ok(w, map[string]any{"id": id, "revoked": true})
		return
	}
	h.fail(w, errs.New(errs.EINVAL, "No live token has that id."))
}

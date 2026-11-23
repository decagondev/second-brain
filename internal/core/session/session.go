// Package session holds the session rules of sb-identity(7): at most 12 hours from
// sign-in, never extended, and re-validated against the directory every 60 seconds.
package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
)

const (
	MaxAge     = 12 * time.Hour
	Revalidate = 60 * time.Second
)

var (
	ErrExpired  = errors.New("session: older than 12 hours; sign in again")
	ErrInactive = errors.New("session: the directory says this user is not active")
	ErrNoAnswer = errors.New("session: the directory could not answer")
)

// Check applies the rules at now. It returns the session (with Validated moved forward
// if it was re-validated), whether it changed, or an error that ends the session.
func Check(ctx context.Context, now time.Time, s ports.Session, dir ports.Directory) (ports.Session, bool, error) {
	if !now.Before(s.SignedIn.Add(MaxAge)) {
		return s, false, ErrExpired
	}
	if now.Sub(s.Validated) < Revalidate {
		return s, false, nil
	}
	active, known, err := dir.Active(ctx, s.Tenant, s.User)
	switch {
	case err != nil:
		return s, false, ErrNoAnswer
	case !known || !active:
		return s, false, ErrInactive
	}
	s.Validated = now
	return s, true, nil
}

// Auditor writes audit rows (audit.Log).
type Auditor interface {
	Write(ctx context.Context, e audit.Event) (audit.Row, error)
}

// ErrAudit is a revocation whose first audit row can't be written: nothing was revoked.
var ErrAudit = errors.New("session: the audit row can't be written, so nothing was revoked")

// ErrUser is a user id that isn't usr_….
var ErrUser = errors.New("session: a usr_ id is required")

// Revoker ends every session of one person (sb-identity(7), Signing out everywhere;
// SB-1906). It audits the request before revoking anything, and the count after.
type Revoker struct {
	Tenant string
	Store  ports.Sessions
	Audit  Auditor
}

// Revoke ends user's sessions on actor's request. An actor revoking their own is the
// person signing out everywhere.
func (v Revoker) Revoke(ctx context.Context, actor, user, reason string) (int, []string, error) {
	if !strings.HasPrefix(user, "usr_") || len(user) < 5 {
		return 0, nil, ErrUser
	}
	subject := "sessions:" + user
	first, err := v.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: subject, State: "revoke_requested", Reason: reason})
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrAudit, err)
	}
	n, err := v.Store.RevokeUser(ctx, v.Tenant, user)
	state := "revoked"
	if err != nil {
		state = "revoke_failed"
	}
	last, aerr := v.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: subject, State: state, Reason: fmt.Sprintf("sessions=%d", n)})
	ids := []string{first.ID}
	if aerr == nil {
		ids = append(ids, last.ID)
	}
	if err != nil {
		return n, ids, fmt.Errorf("session: the store failed: %w", err)
	}
	return n, ids, aerr
}

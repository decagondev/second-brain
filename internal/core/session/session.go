// Package session holds the session rules of sb-identity(7): at most 12 hours from
// sign-in, never extended, and re-validated against the directory every 60 seconds.
package session

import (
	"context"
	"errors"
	"time"

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

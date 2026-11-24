package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewToken(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	tok, s, err := NewToken(rand.Reader, "acme-logistics", "usr_ann", " nightly report ", 0, now)
	if err != nil || !IsToken(tok) || s.ID != TokenKey(tok) || strings.Contains(s.ID, tok) || !strings.HasPrefix(s.Token, "tok_") ||
		s.Name != "nightly report" || !s.Expires.Equal(now.AddDate(0, 0, DefaultTokenDays)) {
		t.Fatalf("%q %+v %v", tok, s, err)
	}
	for _, c := range []struct {
		name string
		days int
		user string
	}{{"", 30, "usr_ann"}, {"x", 91, "usr_ann"}, {"x", -1, "usr_ann"}, {strings.Repeat("x", 61), 30, "usr_ann"}, {"x", 30, "ann"}} {
		if _, _, err := NewToken(rand.Reader, "acme-logistics", c.user, c.name, c.days, now); !errors.Is(err, ErrToken) {
			t.Errorf("%+v: %v", c, err)
		}
	}
	if _, _, err := NewToken(bytes.NewReader(nil), "acme-logistics", "usr_ann", "x", 1, now); err == nil {
		t.Fatal("no randomness made a token")
	}
	// A token ends at its own expiry, not after 12 hours, and is still checked against
	// the directory.
	active := &dir{active: true, known: true}
	if _, _, err := Check(context.Background(), now.Add(13*time.Hour), s, active); err != nil {
		t.Fatalf("a token after 13 hours: %v", err)
	}
	if _, _, err := Check(context.Background(), s.Expires, s, active); !errors.Is(err, ErrExpired) {
		t.Fatalf("a token at its expiry: %v", err)
	}
	if _, _, err := Check(context.Background(), now.Add(2*time.Minute), s, &dir{known: true}); !errors.Is(err, ErrInactive) {
		t.Fatalf("a deprovisioned person's token: %v", err)
	}
	if !TokenRoute("POST", "/v1/ask") || !TokenRoute("GET", "/v1/search") || !TokenRoute("POST", "/v1/mcp") || TokenRoute("GET", "/v1/mcp") || TokenRoute("POST", "/v1/actions/act_1/approve") ||
		TokenRoute("GET", "/v1/ask") || TokenRoute("POST", "/v1/tokens") || TokenRoute("GET", "/v1/admin/usage") {
		t.Fatal("TokenRoute")
	}
}

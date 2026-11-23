package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"

	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/session"
)

// LAW-12 and LAW-04 for signing out everywhere (SB-1906): after a revocation every session
// of the person is refused at its next request, through the real middleware, and no one
// else's is; the request is audited before anything is revoked, and if that row can't be
// written nothing is.

// SessionRevoker ends a person's sessions (session.Revoker).
type SessionRevoker interface {
	Revoke(ctx context.Context, actor, user, reason string) (int, []string, error)
}

// NewSessionRevoker builds the system under test over a store and an audit log.
type NewSessionRevoker func(store ports.Sessions, a session.Auditor) SessionRevoker

// RealSessionRevoker is the real one.
var RealSessionRevoker NewSessionRevoker = func(store ports.Sessions, a session.Auditor) SessionRevoker {
	return session.Revoker{Tenant: "acme-logistics", Store: store, Audit: a}
}

type revokeAudit struct {
	rows []audit.Event
	down bool
	at   func() // called on each write, to see what the store held then
}

func (a *revokeAudit) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	if a.at != nil {
		a.at()
	}
	if a.down {
		return audit.Row{}, errors.New("audit down")
	}
	a.rows = append(a.rows, e)
	return audit.Row{ID: fmt.Sprintf("aud_%012d", len(a.rows))}, nil
}

func law12Revoke(r *rand.Rand, build NewSessionRevoker) error {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 9, 0, 0, 0, time.UTC)
	store := sessmem.New()
	users := []string{"usr_ann", "usr_bob", "usr_cy"}
	owner := map[string]string{}
	for i := 0; i < 3+r.Intn(12); i++ {
		u := users[r.Intn(len(users))]
		id := fmt.Sprintf("s%d", i)
		signed := now.Add(-time.Duration(r.Intn(11*60)) * time.Minute)
		_ = store.Put(ctx, ports.Session{ID: id, Tenant: "acme-logistics", User: u, SignedIn: signed, Validated: now})
		owner[id] = u
	}
	cell := auth.WithIssuer(&auth.Cell{Tenant: "acme-logistics", Sessions: store, Directory: allActive{}, Now: func() time.Time { return now },
		GroupsOf: func(context.Context, string, string) ([]string, error) { return nil, nil }})
	h := cell.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	live := func(id string) bool {
		req := httptest.NewRequest("GET", "/v1/me", nil)
		req.AddCookie(&http.Cookie{Name: auth.Cookie, Value: id})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code == http.StatusNoContent
	}
	gone := users[r.Intn(len(users))]
	// The audit is down: nothing may be revoked.
	down := &revokeAudit{down: true}
	if _, _, err := build(store, down).Revoke(ctx, "usr_olga", gone, "lost laptop"); err == nil {
		return fmt.Errorf("a revocation with the audit log down succeeded")
	}
	for id := range owner {
		if !live(id) {
			return fmt.Errorf("with the audit log down, session %s of %s was revoked anyway", id, owner[id])
		}
	}
	// The real run: the first row is written while every session is still there.
	a := &revokeAudit{}
	var aliveAtFirst = -1
	a.at = func() {
		if aliveAtFirst >= 0 {
			return
		}
		aliveAtFirst = 0
		for id := range owner {
			if _, found, _ := store.Get(ctx, id); found {
				aliveAtFirst++
			}
		}
	}
	n, _, err := build(store, a).Revoke(ctx, "usr_olga", gone, "lost laptop")
	if err != nil {
		return fmt.Errorf("revoke: %v", err)
	}
	if aliveAtFirst != len(owner) {
		return fmt.Errorf("sessions were revoked before the request was audited (%d of %d left)", aliveAtFirst, len(owner))
	}
	if len(a.rows) < 1 || a.rows[0].State != "revoke_requested" || a.rows[0].Subject != "sessions:"+gone {
		return fmt.Errorf("the request wasn't audited first: %+v", a.rows)
	}
	want := 0
	for id, u := range owner {
		switch {
		case u == gone && live(id):
			return fmt.Errorf("session %s of %s still works after revoking %s", id, u, gone)
		case u != gone && !live(id):
			return fmt.Errorf("revoking %s ended %s's session %s", gone, u, id)
		}
		if u == gone {
			want++
		}
	}
	if n != want {
		return fmt.Errorf("revoke reported %d sessions, %s had %d", n, gone, want)
	}
	return nil
}

type allActive struct{}

func (allActive) Active(context.Context, string, string) (bool, bool, error) { return true, true, nil }

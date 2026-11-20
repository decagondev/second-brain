// Package auth is a cell's sign-in and session middleware (sb-identity(7), SB-106):
// the OIDC code flow with PKCE against the tenant's own IdP, sessions in the cell's own
// store, and per-request session checks. Every failure is EAUTH.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/session"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Cookie is the session cookie's name.
const Cookie = "sb_session"

// PendingTTL bounds how long a started sign-in may take.
const PendingTTL = 10 * time.Minute

// Cell is one cell's sign-in for its one tenant.
type Cell struct {
	Tenant    string
	Provider  *oidc.Provider
	Sessions  ports.Sessions
	Directory ports.Directory
	// Secret returns the OIDC client secret from vault://tenants/TENANT/oidc at call time.
	Secret   func(ctx context.Context) (string, error)
	Redirect string
	Now      func() time.Time
	// Issuer is the cell's one tenancy.Issuer (SB-108); GroupsOf reads the directory.
	Issuer   tenancy.Issuer
	GroupsOf func(ctx context.Context, tenant, user string) ([]string, error)
	// Attributes reads the user's directory attributes at each request (SB-1205); nil is a
	// cell that passes none. An error refuses the request.
	Attributes func(ctx context.Context, tenant, user string) (map[string]string, error)
	// Spaces is the cell's space registry (SB-1002); nil is a cell with no spaces, where
	// any ?space= other than company is ESPACE.
	Spaces ports.SpaceStore

	mu      sync.Mutex
	pending map[string]started
}

type started struct {
	nonce, verifier string
	at              time.Time
}

// ErrAuth wraps every sign-in or session failure.
var ErrAuth = errors.New("auth: sign in again")

// Login starts a sign-in and redirects to the IdP.
func (c *Cell) Login(w http.ResponseWriter, r *http.Request) {
	state, err1 := oidc.Random(32)
	nonce, err2 := oidc.Random(32)
	verifier, err3 := oidc.Random(48)
	if err1 != nil || err2 != nil || err3 != nil {
		refuse(w, errs.New(errs.EUNAVAIL, "could not start sign-in"))
		return
	}
	c.mu.Lock()
	if c.pending == nil {
		c.pending = map[string]started{}
	}
	c.pending[state] = started{nonce, verifier, c.Now()}
	c.mu.Unlock()
	http.Redirect(w, r, c.Provider.AuthURL(c.Redirect, state, nonce, verifier), http.StatusFound)
}

// Callback finishes a sign-in: exchange the code, verify the ID token, check the
// directory, create a session and set the cookie.
func (c *Cell) Callback(w http.ResponseWriter, r *http.Request) {
	s, err := c.callback(r)
	if err != nil {
		refuse(w, errs.New(errs.EAUTH, "sign-in failed; try again"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: Cookie, Value: s.ID, Path: "/", HttpOnly: true, Secure: true,
		SameSite: http.SameSiteLaxMode, Expires: s.SignedIn.Add(session.MaxAge)})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (c *Cell) callback(r *http.Request) (ports.Session, error) {
	ctx := r.Context()
	state := r.URL.Query().Get("state")
	c.mu.Lock()
	p, ok := c.pending[state]
	delete(c.pending, state) // single use
	c.mu.Unlock()
	now := c.Now()
	if !ok || state == "" || now.Sub(p.at) > PendingTTL {
		return ports.Session{}, ErrAuth
	}
	secret, err := c.Secret(ctx)
	if err != nil {
		return ports.Session{}, ErrAuth
	}
	raw, err := c.Provider.Exchange(ctx, r.URL.Query().Get("code"), c.Redirect, p.verifier, secret)
	if err != nil {
		return ports.Session{}, ErrAuth
	}
	id, err := c.Provider.Verify(ctx, raw, p.nonce)
	if err != nil || id.Tenant != c.Tenant {
		return ports.Session{}, ErrAuth
	}
	// Users exist only if SCIM created them (sb-identity(7)).
	if active, known, err := c.Directory.Active(ctx, c.Tenant, id.Subject); err != nil || !known || !active {
		return ports.Session{}, ErrAuth
	}
	sid, err := oidc.Random(32)
	if err != nil {
		return ports.Session{}, ErrAuth
	}
	s := ports.Session{ID: sid, Tenant: c.Tenant, User: id.Subject, SignedIn: now, Validated: now}
	if err := c.Sessions.Put(ctx, s); err != nil {
		return ports.Session{}, ErrAuth
	}
	return s, nil
}

// Authenticate returns the request's live session, applying the session rules.
func (c *Cell) Authenticate(r *http.Request) (ports.Session, error) {
	ck, err := r.Cookie(Cookie)
	if err != nil || ck.Value == "" {
		return ports.Session{}, ErrAuth
	}
	ctx := r.Context()
	s, found, err := c.Sessions.Get(ctx, ck.Value)
	if err != nil || !found || s.Tenant != c.Tenant {
		return ports.Session{}, ErrAuth
	}
	s, changed, err := session.Check(ctx, c.Now(), s, c.Directory)
	if err != nil {
		_ = c.Sessions.Delete(ctx, s.ID)
		return ports.Session{}, ErrAuth
	}
	if changed {
		if err := c.Sessions.Put(ctx, s); err != nil {
			return ports.Session{}, ErrAuth
		}
	}
	return s, nil
}

// Middleware authenticates every request and puts its tenancy.Scope in the context.
// Handlers get data only through that scope (sb-tenancy(7), Scope in code).
func (c *Cell) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, err := c.Authenticate(r)
		if err != nil {
			refuse(w, errs.New(errs.EAUTH, "sign in again"))
			return
		}
		groups, err := c.GroupsOf(r.Context(), s.Tenant, s.User)
		if err != nil {
			refuse(w, errs.New(errs.EUNAVAIL, "directory unavailable"))
			return
		}
		asked := r.URL.Query()["space"]
		if len(asked) > 1 {
			refuse(w, errs.New(errs.ESPACE, "name one space, or none for company-wide"))
			return
		}
		requested := ""
		if len(asked) == 1 {
			requested = asked[0]
		}
		space, err := spaces.Resolve(r.Context(), c.Spaces, s.Tenant, groups, requested)
		switch {
		case errors.Is(err, spaces.ErrUnavailable):
			refuse(w, errs.New(errs.EUNAVAIL, "space registry unavailable"))
			return
		case err != nil:
			refuse(w, errs.New(errs.ESPACE, "no space %q for you; ask its stewards for membership, or ask company-wide", requested))
			return
		}
		scope, err := c.Issuer.IssueIn(s.Tenant, s.User, groups, space)
		if err != nil {
			refuse(w, errs.New(errs.EAUTH, "sign in again"))
			return
		}
		if c.Attributes != nil {
			attrs, err := c.Attributes(r.Context(), s.Tenant, s.User)
			if err != nil {
				refuse(w, errs.New(errs.EUNAVAIL, "directory unavailable"))
				return
			}
			scope = scope.WithAttrs(attrs)
		}
		next.ServeHTTP(w, r.WithContext(tenancy.WithScope(r.Context(), scope)))
	})
}

// DirectoryAttributes reads a user's attributes from the directory (SB-1205).
func DirectoryAttributes(dir ports.DirectoryStore) func(ctx context.Context, tenant, user string) (map[string]string, error) {
	return func(ctx context.Context, tenant, user string) (map[string]string, error) {
		u, found, err := dir.User(ctx, tenant, user)
		if err != nil {
			return nil, err
		}
		if !found {
			return map[string]string{}, nil
		}
		return u.Attributes(), nil
	}
}

// TenantOf is an edge.TokenVerifier for this cell: the tenant of a live session.
func (c *Cell) TenantOf(r *http.Request) (string, error) {
	s, err := c.Authenticate(r)
	if err != nil {
		return "", err
	}
	return s.Tenant, nil
}

func refuse(w http.ResponseWriter, e *errs.Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errs.HTTP(e.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
		"code": string(e.Code), "message": e.Msg, "remedy": errs.Remedy(e.Code)}})
}

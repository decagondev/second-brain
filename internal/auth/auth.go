// Package auth is a cell's sign-in and session middleware (sb-identity(7), SB-106):
// the OIDC code flow with PKCE against the tenant's own IdP, sessions in the cell's own
// store, and per-request session checks. Every failure is EAUTH.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/core/audit"
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
	// Audit records each use of a read-only API token (SB-2306); nil refuses every token,
	// so no use goes unaudited.
	Audit session.Auditor

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

// Authenticate returns the request's live session, applying the session rules: the
// session cookie, or a read-only API token as a bearer (SB-2306), looked up by its hash.
func (c *Cell) Authenticate(r *http.Request) (ports.Session, error) {
	key, token := "", false
	if b, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && session.IsToken(b) {
		key, token = session.TokenKey(b), true
	} else if ck, err := r.Cookie(Cookie); err == nil {
		key = ck.Value
	}
	if key == "" {
		return ports.Session{}, ErrAuth
	}
	ctx := r.Context()
	s, found, err := c.Sessions.Get(ctx, key)
	if err != nil || !found || s.Tenant != c.Tenant || token != (s.Token != "") {
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
		if s.Token != "" && !c.tokenUse(w, r, s) {
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
		ctx := tenancy.WithScope(r.Context(), scope)
		if s.Token != "" { // SB-5305: an API token's asks count as its own client
			ctx = tenancy.WithClient(ctx, tenancy.ClientToken)
		}
		if r.URL.Query().Get("branch") == "true" { // SB-3903: this department and its teams
			teams, err := spaces.Teams(r.Context(), c.Spaces, s.Tenant, groups, space)
			if err != nil {
				refuse(w, errs.New(errs.EUNAVAIL, "space registry unavailable"))
				return
			}
			var branch []tenancy.Scope
			for _, id := range teams {
				in, err := c.Issuer.IssueIn(s.Tenant, s.User, groups, id)
				if err != nil {
					refuse(w, errs.New(errs.EAUTH, "sign in again"))
					return
				}
				branch = append(branch, in.WithAttrs(scope.Attrs()))
			}
			ctx = tenancy.WithBranch(ctx, branch)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tokenUse lets a read-only API token through only to ask or search, and only once its
// use is audited (SB-2306); otherwise it refuses and says false.
func (c *Cell) tokenUse(w http.ResponseWriter, r *http.Request, s ports.Session) bool {
	if !session.TokenRoute(r.Method, r.URL.Path) {
		refuse(w, errs.New(errs.EPERM, "an API token can only ask and search"))
		return false
	}
	if c.Audit == nil {
		refuse(w, errs.New(errs.EUNAVAIL, "API tokens can't be audited here"))
		return false
	}
	if _, err := c.Audit.Write(r.Context(), audit.Event{Actor: s.User, Kind: "access", Subject: "token:" + s.Token, State: "token_used", Reason: r.Method + " " + r.URL.Path}); err != nil {
		refuse(w, errs.New(errs.EUNAVAIL, "the audit log is unavailable"))
		return false
	}
	return true
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

// WithIssuer gives c the cell's one Issuer, for c's own tenant (SB-108, SB-1501): the
// composition root builds the Cell and auth makes its Issuer, so nothing outside auth
// ever holds one.
func WithIssuer(c *Cell) *Cell {
	c.Issuer = tenancy.NewIssuer(c.Tenant)
	return c
}

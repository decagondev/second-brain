package auth

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/core/ports"
)

type dir map[string]bool // user -> active

func (d dir) Active(_ context.Context, _, user string) (bool, bool, error) {
	a, ok := d[user]
	return a, ok, nil
}

type cellUnderTest struct {
	cell   *Cell
	server *httptest.Server
	idp    *oidctest.IdP
}

// newCell stands up one tenant's IdP and cell: the two-IdP fixture is two of these.
func newCell(t *testing.T, tenant string, clock *time.Time) cellUnderTest {
	t.Helper()
	idp := oidctest.New(tenant, "cell-"+tenant, "secret-"+tenant)
	t.Cleanup(idp.Server.Close)
	now := func() time.Time { return *clock }
	p, err := oidc.Discover(context.Background(), idp.Issuer(), idp.ClientID, tenant, idp.Server.Client(), now)
	if err != nil {
		t.Fatal(err)
	}
	c := &Cell{Tenant: tenant, Provider: p, Sessions: memory.New(), Directory: dir{"usr_1": true}, Now: now,
		Secret: func(context.Context) (string, error) { return "secret-" + tenant, nil }}
	mux := http.NewServeMux()
	mux.HandleFunc("/login", c.Login)
	mux.HandleFunc("/callback", c.Callback)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s, err := c.Authenticate(r)
		if err != nil {
			refuse(w, errsEAUTH())
			return
		}
		_, _ = w.Write([]byte("hello " + s.User + "@" + s.Tenant))
	})
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	c.Redirect = srv.URL + "/callback"
	return cellUnderTest{c, srv, idp}
}

// browser follows redirects across both TLS test servers and keeps cookies.
func browser(t *testing.T, a, b *httptest.Server) *http.Client {
	jar, _ := cookiejar.New(nil)
	pool := a.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	tr := a.Client().Transport.(*http.Transport).Clone()
	tr.TLSClientConfig.RootCAs = pool
	pool.AddCert(b.Certificate())
	return &http.Client{Jar: jar, Transport: tr}
}

func TestLoginWithTestIdP(t *testing.T) {
	clock := time.Now()
	acme := newCell(t, "acme-logistics", &clock)
	br := browser(t, acme.server, acme.idp.Server)
	resp, err := br.Get(acme.server.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	body := readAll(t, resp)
	if resp.StatusCode != 200 || body != "hello usr_1@acme-logistics" {
		t.Fatalf("after sign-in: %d %q", resp.StatusCode, body)
	}
	// 61 seconds later the session is re-validated; 12 hours later it is over.
	clock = clock.Add(61 * time.Second)
	if resp, _ := br.Get(acme.server.URL + "/"); resp.StatusCode != 200 {
		t.Fatalf("after re-validation: %d", resp.StatusCode)
	}
	clock = clock.Add(12 * time.Hour)
	if resp, _ := br.Get(acme.server.URL + "/"); resp.StatusCode != 401 {
		t.Fatalf("after 12h: %d, want 401", resp.StatusCode)
	}
}

// A user the directory doesn't know never gets a session (users come only from SCIM).
func TestUnknownUserRefused(t *testing.T) {
	clock := time.Now()
	acme := newCell(t, "acme-logistics", &clock)
	acme.cell.Directory = dir{}
	br := browser(t, acme.server, acme.idp.Server)
	resp, err := br.Get(acme.server.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 || !strings.Contains(readAll(t, resp), `"EAUTH"`) {
		t.Fatalf("got %d", resp.StatusCode)
	}
}

// Two IdPs, two cells: a session from acme's cell means nothing in beta's cell, and an
// acme ID token never verifies at beta.
func TestTwoIdPs(t *testing.T) {
	clock := time.Now()
	acme := newCell(t, "acme-logistics", &clock)
	beta := newCell(t, "beta-freight", &clock)
	br := browser(t, acme.server, acme.idp.Server)
	if resp, err := br.Get(acme.server.URL + "/login"); err != nil || resp.StatusCode != 200 {
		t.Fatalf("acme sign-in: %v", err)
	}
	var sid string
	for _, c := range br.Jar.Cookies(mustURL(acme.server.URL)) {
		if c.Name == Cookie {
			sid = c.Value
		}
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: Cookie, Value: sid})
	if _, err := beta.cell.Authenticate(r); err == nil {
		t.Fatal("acme's session was accepted by beta's cell")
	}
	tok := acme.idp.Mint(map[string]any{"iss": acme.idp.Issuer(), "aud": beta.idp.ClientID, "sub": "usr_1",
		"tenant": "beta-freight", "nonce": "n"}, acme.idp.Key, acme.idp.KID)
	if _, err := beta.cell.Provider.Verify(context.Background(), tok, "n"); err == nil {
		t.Fatal("a token from acme's IdP verified at beta's cell")
	}
	_ = ports.Session{}
}

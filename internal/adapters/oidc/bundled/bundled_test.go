package bundled

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc"
)

// world is the IdP over TLS and the cell's own OIDC client (internal/adapters/oidc).
type world struct {
	srv  *httptest.Server
	idp  *Server
	prov *oidc.Provider
	now  time.Time
}

const redirect = "https://cell.acme.example/callback"

func newWorld(t *testing.T) *world {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	st := &Store{Dir: t.TempDir()}
	w := &world{now: time.Now()}
	mux := http.NewServeMux()
	w.srv = httptest.NewTLSServer(mux)
	t.Cleanup(w.srv.Close)
	w.idp = New(w.srv.URL+"/idp", "acme-logistics", "sb-acme-logistics", "client-secret-1", redirect, st, k)
	w.idp.Now = func() time.Time { return w.now }
	mux.Handle("/idp/", w.idp.Handler())
	h, _ := Hash("correct horse battery")
	if err := st.Add(User{ID: "usr_ann", Email: "ann@acme.example", Hash: h}); err != nil {
		t.Fatal(err)
	}
	p, err := oidc.Discover(context.Background(), w.srv.URL+"/idp", "sb-acme-logistics", "acme-logistics", w.srv.Client(), func() time.Time { return w.now })
	if err != nil {
		t.Fatal(err)
	}
	w.prov = p
	return w
}

var hidden = regexp.MustCompile(`name="(state|nonce|code_challenge|at|csrf)" value="([^"]*)"`)

// answer is what the sign-in post answered: its status and its headers.
type answer struct {
	StatusCode int
	Header     http.Header
}

// signIn runs the browser's part: the form, then the post. It returns the answer.
func (w *world) signIn(t *testing.T, email, password string, tamper func(url.Values)) (answer, string, string) {
	t.Helper()
	verifier := strings.Repeat("v", 43)
	state, nonce := "st-"+email, "no-"+email
	auth := w.prov.AuthURL(redirect, state, nonce, verifier)
	c := w.srv.Client()
	jar, _ := cookiejar.New(nil)
	c.Jar = jar
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Get(auth)
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("form: %d %s", resp.StatusCode, page)
	}
	f := url.Values{"email": {email}, "password": {password}}
	for _, m := range hidden.FindAllStringSubmatch(string(page), -1) {
		f.Set(m[1], strings.ReplaceAll(m[2], "&#43;", "+"))
	}
	if tamper != nil {
		tamper(f)
	}
	resp, err = c.PostForm(auth, f)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return answer{resp.StatusCode, resp.Header}, verifier, nonce
}

func TestTheCellSignsInThroughIt(t *testing.T) {
	w := newWorld(t)
	resp, verifier, nonce := w.signIn(t, "Ann@acme.example", "correct horse battery", nil)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("sign-in: %d", resp.StatusCode)
	}
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if !strings.HasPrefix(loc.String(), redirect+"?") || loc.Query().Get("state") != "st-Ann@acme.example" {
		t.Fatalf("redirect %s", loc)
	}
	code := loc.Query().Get("code")
	tok, err := w.prov.Exchange(context.Background(), code, redirect, verifier, "client-secret-1")
	if err != nil {
		t.Fatal(err)
	}
	id, err := w.prov.Verify(context.Background(), tok, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "usr_ann" || id.Tenant != "acme-logistics" || id.Email != "ann@acme.example" {
		t.Fatalf("identity %+v", id)
	}
	if _, err := w.prov.Exchange(context.Background(), code, redirect, verifier, "client-secret-1"); err == nil {
		t.Fatal("a code was used twice")
	}
}

func TestRefusals(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	// The form refuses another client or another redirect before showing anything.
	for _, q := range []url.Values{
		{"client_id": {"other"}, "redirect_uri": {redirect}},
		{"client_id": {"sb-acme-logistics"}, "redirect_uri": {"https://evil.example/callback"}},
	} {
		q.Set("response_type", "code")
		q.Set("code_challenge_method", "S256")
		q.Set("code_challenge", strings.Repeat("c", 43))
		q.Set("state", "s")
		q.Set("nonce", "n")
		resp, err := w.srv.Client().Get(w.srv.URL + "/idp/authorize?" + q.Encode())
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest || strings.Contains(string(b), "password") {
			t.Errorf("%v: %d", q, resp.StatusCode)
		}
	}
	// A wrong password, an unknown address, a disabled user and a forged form never sign in.
	if r, _, _ := w.signIn(t, "ann@acme.example", "wrong password here", nil); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong password: %d", r.StatusCode)
	}
	if r, _, _ := w.signIn(t, "nobody@acme.example", "correct horse battery", nil); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("unknown address: %d", r.StatusCode)
	}
	if r, _, _ := w.signIn(t, "ann@acme.example", "correct horse battery", func(f url.Values) { f.Set("csrf", "forged") }); r.StatusCode != http.StatusBadRequest {
		t.Errorf("forged form: %d", r.StatusCode)
	}
	if r, _, _ := w.signIn(t, "ann@acme.example", "correct horse battery", func(f url.Values) { f.Set("nonce", "swapped") }); r.StatusCode != http.StatusBadRequest {
		t.Errorf("swapped nonce: %d", r.StatusCode)
	}
	// A code is bound to its verifier and redirect, and the client must prove its secret.
	r, verifier, _ := w.signIn(t, "ann@acme.example", "correct horse battery", nil)
	loc, _ := url.Parse(r.Header.Get("Location"))
	code := loc.Query().Get("code")
	if _, err := w.prov.Exchange(ctx, code, redirect, verifier, "wrong-secret"); err == nil {
		t.Error("a wrong client secret got a token")
	}
	r, _, _ = w.signIn(t, "ann@acme.example", "correct horse battery", nil)
	loc, _ = url.Parse(r.Header.Get("Location"))
	if _, err := w.prov.Exchange(ctx, loc.Query().Get("code"), redirect, strings.Repeat("x", 43), "client-secret-1"); err == nil {
		t.Error("a wrong verifier got a token")
	}
	// An expired code is refused.
	r, verifier, _ = w.signIn(t, "ann@acme.example", "correct horse battery", nil)
	loc, _ = url.Parse(r.Header.Get("Location"))
	w.now = w.now.Add(CodeTTL + time.Second)
	if _, err := w.prov.Exchange(ctx, loc.Query().Get("code"), redirect, verifier, "client-secret-1"); err == nil {
		t.Error("an expired code got a token")
	}
	// A user disabled between the form and the exchange gets no token.
	r, verifier, _ = w.signIn(t, "ann@acme.example", "correct horse battery", nil)
	loc, _ = url.Parse(r.Header.Get("Location"))
	if _, err := w.idp.Store.Disable("ann@acme.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.prov.Exchange(ctx, loc.Query().Get("code"), redirect, verifier, "client-secret-1"); err == nil {
		t.Error("a disabled user got a token")
	}
	if r, _, _ := w.signIn(t, "ann@acme.example", "correct horse battery", nil); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("a disabled user signed in: %d", r.StatusCode)
	}
}

func TestFailuresAreLimited(t *testing.T) {
	w := newWorld(t)
	for i := 0; i < PerAddress; i++ {
		w.signIn(t, "ann@acme.example", "wrong password here", nil)
	}
	if r, _, _ := w.signIn(t, "ann@acme.example", "correct horse battery", nil); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("the right password after %d failures: %d, want refused", PerAddress, r.StatusCode)
	}
	w.now = w.now.Add(Window + time.Second)
	if r, _, _ := w.signIn(t, "ann@acme.example", "correct horse battery", nil); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("after the window: %d", r.StatusCode)
	}
}

func TestHashes(t *testing.T) {
	if _, err := Hash("short"); err == nil {
		t.Fatal("a short password was kept")
	}
	h, err := Hash("a long enough password")
	if err != nil || !Check(h, "a long enough password") || Check(h, "a long enough passwore") {
		t.Fatalf("hash %q %v", h, err)
	}
	weak := strings.Replace(h, "$600000$", "$1000$", 1)
	if Check(weak, "a long enough password") {
		t.Fatal("a hash with fewer iterations was accepted")
	}
	st := &Store{Dir: t.TempDir()}
	if err := st.Add(User{ID: "usr_a", Email: "a@acme.example", Hash: h}); err != nil {
		t.Fatal(err)
	}
	if err := st.Add(User{ID: "usr_b", Email: "A@acme.example", Hash: h}); !errors.Is(err, ErrExists) {
		t.Fatalf("a second user with the same address: %v", err)
	}
	k1, err := st.Key()
	if err != nil {
		t.Fatal(err)
	}
	k2, _ := st.Key()
	if k1.N.Cmp(k2.N) != 0 {
		t.Fatal("the signing key changed between starts")
	}
}

// Behind the proxy, failures count per client: the last X-Forwarded-For entry, which the
// proxy adds, and never one a client wrote itself in front of it.
func TestClientBehindTheProxy(t *testing.T) {
	s := &Server{BehindProxy: true}
	r := httptest.NewRequest(http.MethodPost, "/idp/authorize", nil)
	r.RemoteAddr = "172.18.0.5:41234"
	r.Header.Add("X-Forwarded-For", "6.6.6.6, 203.0.113.7")
	if got := s.client(r); got != "203.0.113.7" {
		t.Fatalf("client %q, want the proxy's entry", got)
	}
	r.Header.Set("X-Forwarded-For", "not-an-ip")
	if got := s.client(r); got != "172.18.0.5" {
		t.Fatalf("a bad header: %q, want the connection's address", got)
	}
	s.BehindProxy = false
	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	if got := s.client(r); got != "172.18.0.5" {
		t.Fatalf("not behind a proxy: %q", got)
	}
}

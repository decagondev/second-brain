// Package bundled is sb-idp(1), the bundled OIDC identity provider for a self-hosted cell
// with no IdP of its own (SB-1605): one tenant, one client, passwords kept as PBKDF2
// hashes, the authorization-code flow with PKCE that sb-identity(7) describes, and a
// sign-in form that refuses a redirect it wasn't told about before it shows anything.
package bundled

import (
	"crypto"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Parameters of the password hash and the flow (sb-idp(1)).
const (
	Iterations  = 600_000
	MinPassword = 12
	CodeTTL     = 60 * time.Second
	TokenTTL    = 5 * time.Minute
	FormTTL     = 10 * time.Minute
	Window      = 15 * time.Minute
	PerAddress  = 5
	PerClient   = 20
)

// User is one person who may sign in.
type User struct {
	ID       string `json:"id"` // the cell's directory id, usr_…
	Email    string `json:"email"`
	Hash     string `json:"hash"`
	Disabled bool   `json:"disabled,omitempty"`
}

var (
	userID = regexp.MustCompile(`^usr_[A-Za-z0-9_]{1,64}$`)
	// ErrPassword is a password too short to keep.
	ErrPassword = fmt.Errorf("bundled: a password needs at least %d characters", MinPassword)
	// ErrExists is an address already present.
	ErrExists = errors.New("bundled: that address is already a user")
)

// Hash is the stored form of a password: pbkdf2-sha256$ITER$SALT$KEY, base64.
func Hash(password string) (string, error) {
	if len([]rune(password)) < MinPassword {
		return "", ErrPassword
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	k, err := pbkdf2.Key(sha256.New, password, salt, Iterations, 32)
	if err != nil {
		return "", err
	}
	b := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", Iterations, b(salt), b(k)), nil
}

// Check reports whether password matches hash, in constant time for a well-formed hash.
func Check(hash, password string) bool {
	p := strings.Split(hash, "$")
	if len(p) != 4 || p[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(p[1])
	if err != nil || iter < Iterations {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(p[2])
	want, err2 := base64.RawStdEncoding.DecodeString(p[3])
	if err1 != nil || err2 != nil || len(want) != 32 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, 32)
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// dummy is checked against when an address is unknown, so a miss takes as long as a hit.
var dummy, _ = Hash("not-a-password-anyone-has")

// Store is DIR/users.json and DIR/key.pem.
type Store struct {
	Dir string
	mu  sync.Mutex
}

func (s *Store) users() ([]User, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, "users.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var us []User
	if err := json.Unmarshal(b, &us); err != nil {
		return nil, fmt.Errorf("bundled: users.json can't be read: %w", err)
	}
	return us, nil
}

func (s *Store) write(us []User) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(us, "", " ")
	tmp := filepath.Join(s.Dir, ".users.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.Dir, "users.json"))
}

// Users lists the users.
func (s *Store) Users() ([]User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.users()
}

// Find is the user with that address, case-insensitively.
func (s *Store) Find(email string) (User, bool, error) {
	us, err := s.Users()
	if err != nil {
		return User{}, false, err
	}
	for _, u := range us {
		if strings.EqualFold(u.Email, email) {
			return u, true, nil
		}
	}
	return User{}, false, nil
}

// Add stores a new user; the hash must come from Hash.
func (s *Store) Add(u User) error {
	if !userID.MatchString(u.ID) || !strings.Contains(u.Email, "@") || !strings.HasPrefix(u.Hash, "pbkdf2-sha256$") {
		return errors.New("bundled: a user needs a usr_ id, an address and a hash")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	us, err := s.users()
	if err != nil {
		return err
	}
	for _, x := range us {
		if strings.EqualFold(x.Email, u.Email) {
			return ErrExists
		}
	}
	return s.write(append(us, u))
}

// Disable marks the user with that address disabled and returns them.
func (s *Store) Disable(email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	us, err := s.users()
	if err != nil {
		return User{}, err
	}
	for i := range us {
		if strings.EqualFold(us[i].Email, email) {
			us[i].Disabled = true
			return us[i], s.write(us)
		}
	}
	return User{}, fmt.Errorf("bundled: no user %s", email)
}

// Key is the signing key, made on first use and kept in DIR/key.pem.
func (s *Store) Key() (*rsa.PrivateKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := filepath.Join(s.Dir, "key.pem")
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		k, gerr := rsa.GenerateKey(rand.Reader, 2048)
		if gerr != nil {
			return nil, gerr
		}
		if err := os.MkdirAll(s.Dir, 0o700); err != nil {
			return nil, err
		}
		out := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: must(x509.MarshalPKCS8PrivateKey(k))})
		f, ferr := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if ferr != nil {
			return nil, ferr
		}
		if _, werr := f.Write(out); werr != nil {
			_ = f.Close()
			return nil, werr
		}
		return k, f.Close()
	}
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, errors.New("bundled: key.pem isn't PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("bundled: key.pem isn't an RSA key")
	}
	return rk, nil
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

// Server is one running IdP.
type Server struct {
	Issuer, Tenant, ClientID, Redirect string
	Secret                             string // the client secret
	Store                              *Store
	Key                                *rsa.PrivateKey
	Now                                func() time.Time
	// BehindProxy takes the client's address from the proxy's X-Forwarded-For (the last
	// entry, the one the proxy itself added), so failures are counted per person, not
	// for everyone behind the one proxy. Set it only when nothing but the proxy reaches
	// the IdP, as in the self-hosted layout.
	BehindProxy bool

	formKey []byte
	mu      sync.Mutex
	codes   map[string]grant
	fails   map[string][]time.Time
}

type grant struct {
	user                       User
	nonce, challenge, redirect string
	expires                    time.Time
}

// New makes a server; the form key is fresh at every start.
func New(issuer, tenant, clientID, secret, redirect string, st *Store, key *rsa.PrivateKey) *Server {
	fk := make([]byte, 32)
	if _, err := rand.Read(fk); err != nil {
		panic(err)
	}
	return &Server{Issuer: strings.TrimRight(issuer, "/"), Tenant: tenant, ClientID: clientID, Secret: secret, Redirect: redirect,
		Store: st, Key: key, Now: time.Now, formKey: fk, codes: map[string]grant{}, fails: map[string][]time.Time{}}
}

func (s *Server) kid() string {
	h := sha256.Sum256(s.Key.N.Bytes())
	return hex.EncodeToString(h[:8])
}

// Handler serves the IdP under the issuer's path.
func (s *Server) Handler() http.Handler {
	u, _ := url.Parse(s.Issuer)
	base := strings.TrimRight(u.Path, "/")
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+base+"/.well-known/openid-configuration", s.discovery)
	mux.HandleFunc("GET "+base+"/jwks", s.jwks)
	mux.HandleFunc("GET "+base+"/authorize", s.authorize)
	mux.HandleFunc("POST "+base+"/authorize", s.signIn)
	mux.HandleFunc("POST "+base+"/token", s.token)
	return mux
}

func (s *Server) discovery(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"issuer": s.Issuer, "authorization_endpoint": s.Issuer + "/authorize",
		"token_endpoint": s.Issuer + "/token", "jwks_uri": s.Issuer + "/jwks", "response_types_supported": []string{"code"},
		"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"client_secret_basic"}})
}

func (s *Server) jwks(w http.ResponseWriter, _ *http.Request) {
	b64 := base64.RawURLEncoding.EncodeToString
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": s.kid(), "alg": "RS256", "use": "sig",
		"n": b64(s.Key.N.Bytes()), "e": b64(big.NewInt(int64(s.Key.E)).Bytes())}}})
}

// request is an authorization request's parameters, all checked before the form shows.
type request struct{ State, Nonce, Challenge string }

func (s *Server) check(q url.Values) (request, bool) {
	r := request{State: q.Get("state"), Nonce: q.Get("nonce"), Challenge: q.Get("code_challenge")}
	ok := q.Get("client_id") == s.ClientID && q.Get("redirect_uri") == s.Redirect && q.Get("response_type") == "code" &&
		q.Get("code_challenge_method") == "S256" && len(r.Challenge) == 43 && r.State != "" && len(r.State) <= 256 &&
		r.Nonce != "" && len(r.Nonce) <= 256
	return r, ok
}

func headers(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
}

var form = template.Must(template.New("f").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1"><title>Sign in</title>
<style>body{font-family:system-ui,sans-serif;max-width:22rem;margin:4rem auto;padding:0 1rem;color:#1f2328}
label{display:block;margin:.8rem 0 .2rem}input{width:100%;padding:.5rem;box-sizing:border-box;font-size:1rem}
button{margin-top:1.2rem;padding:.55rem 1rem;font-size:1rem}.err{color:#b42318}</style></head><body>
<h1>Sign in</h1><p>{{.Tenant}}</p>{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
<form method="post" action="{{.Action}}">
<input type="hidden" name="state" value="{{.State}}"><input type="hidden" name="nonce" value="{{.Nonce}}">
<input type="hidden" name="code_challenge" value="{{.Challenge}}"><input type="hidden" name="at" value="{{.At}}">
<input type="hidden" name="csrf" value="{{.CSRF}}">
<label for="email">Email</label><input id="email" name="email" type="email" autocomplete="username" required value="{{.Email}}">
<label for="password">Password</label><input id="password" name="password" type="password" autocomplete="current-password" required>
<button type="submit">Sign in</button></form></body></html>`))

func (s *Server) csrf(r request, at string) string {
	m := hmac.New(sha256.New, s.formKey)
	for _, v := range []string{r.State, r.Nonce, r.Challenge, at} {
		m.Write([]byte(v))
		m.Write([]byte{0})
	}
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *Server) show(w http.ResponseWriter, r request, email, msg string, status int) {
	at := strconv.FormatInt(s.Now().Unix(), 10)
	u, _ := url.Parse(s.Issuer)
	headers(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = form.Execute(w, map[string]string{"Tenant": s.Tenant, "Error": msg, "Action": strings.TrimRight(u.Path, "/") + "/authorize?" + url.Values{
		"client_id": {s.ClientID}, "redirect_uri": {s.Redirect}, "response_type": {"code"}, "code_challenge_method": {"S256"},
		"state": {r.State}, "nonce": {r.Nonce}, "code_challenge": {r.Challenge}}.Encode(),
		"State": r.State, "Nonce": r.Nonce, "Challenge": r.Challenge, "At": at, "CSRF": s.csrf(r, at), "Email": email})
}

func (s *Server) authorize(w http.ResponseWriter, req *http.Request) {
	r, ok := s.check(req.URL.Query())
	if !ok {
		headers(w)
		http.Error(w, "this sign-in request isn't for this cell", http.StatusBadRequest)
		return
	}
	s.show(w, r, "", "", http.StatusOK)
}

func (s *Server) client(req *http.Request) string {
	if s.BehindProxy {
		if xff := req.Header.Values("X-Forwarded-For"); len(xff) > 0 {
			parts := strings.Split(xff[len(xff)-1], ",")
			if ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

// limited reports whether key has too many recent failures, dropping old ones.
func (s *Server) limited(key string, max int) bool {
	now := s.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.fails[key][:0]
	for _, t := range s.fails[key] {
		if now.Sub(t) < Window {
			kept = append(kept, t)
		}
	}
	s.fails[key] = kept
	if len(kept) == 0 {
		delete(s.fails, key)
	}
	return len(kept) >= max
}

func (s *Server) fail(keys ...string) {
	now := s.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		s.fails[k] = append(s.fails[k], now)
	}
}

func (s *Server) signIn(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, 16<<10)
	r, ok := s.check(req.URL.Query())
	if !ok || req.ParseForm() != nil {
		headers(w)
		http.Error(w, "this sign-in request isn't for this cell", http.StatusBadRequest)
		return
	}
	posted := request{State: req.PostForm.Get("state"), Nonce: req.PostForm.Get("nonce"), Challenge: req.PostForm.Get("code_challenge")}
	at := req.PostForm.Get("at")
	sec, err := strconv.ParseInt(at, 10, 64)
	good := hmac.Equal([]byte(req.PostForm.Get("csrf")), []byte(s.csrf(posted, at)))
	if posted != r || err != nil || !good || s.Now().Sub(time.Unix(sec, 0)) > FormTTL || time.Unix(sec, 0).After(s.Now().Add(time.Minute)) {
		s.show(w, r, "", "The sign-in page expired. Try again.", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.PostForm.Get("email")))
	addr, from := "a:"+email, "c:"+s.client(req)
	const wrong = "That address and password don't match."
	if s.limited(addr, PerAddress) || s.limited(from, PerClient) {
		Check(dummy, req.PostForm.Get("password"))
		s.show(w, r, email, wrong, http.StatusUnauthorized)
		return
	}
	u, found, ferr := s.Store.Find(email)
	if ferr != nil {
		s.show(w, r, email, "Sign-in is unavailable. Try again later.", http.StatusServiceUnavailable)
		return
	}
	hash := dummy
	if found {
		hash = u.Hash
	}
	if !Check(hash, req.PostForm.Get("password")) || !found || u.Disabled {
		s.fail(addr, from)
		s.show(w, r, email, wrong, http.StatusUnauthorized)
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	code := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	for c, g := range s.codes {
		if s.Now().After(g.expires) {
			delete(s.codes, c)
		}
	}
	s.codes[code] = grant{user: u, nonce: r.Nonce, challenge: r.Challenge, redirect: s.Redirect, expires: s.Now().Add(CodeTTL)}
	s.mu.Unlock()
	dest, _ := url.Parse(s.Redirect)
	dest.RawQuery = url.Values{"code": {code}, "state": {r.State}}.Encode()
	headers(w)
	http.Redirect(w, req, dest.String(), http.StatusSeeOther)
}

func (s *Server) token(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, 16<<10)
	w.Header().Set("Cache-Control", "no-store")
	id, secret, ok := req.BasicAuth()
	if ok {
		id, _ = url.QueryUnescape(id)
		secret, _ = url.QueryUnescape(secret)
	}
	if !ok || subtle.ConstantTimeCompare([]byte(id), []byte(s.ClientID)) != 1 || subtle.ConstantTimeCompare([]byte(secret), []byte(s.Secret)) != 1 {
		http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
		return
	}
	if req.ParseForm() != nil || req.PostForm.Get("grant_type") != "authorization_code" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	code := req.PostForm.Get("code")
	s.mu.Lock()
	g, found := s.codes[code]
	delete(s.codes, code) // single use, whatever happens next
	s.mu.Unlock()
	h := sha256.Sum256([]byte(req.PostForm.Get("code_verifier")))
	if !found || s.Now().After(g.expires) || g.redirect != req.PostForm.Get("redirect_uri") ||
		subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(h[:])), []byte(g.challenge)) != 1 {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	// The user may have been disabled since the form: check again.
	if u, ok, err := s.Store.Find(g.user.Email); err != nil || !ok || u.Disabled || u.ID != g.user.ID {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	now := s.Now()
	tok, err := s.mint(map[string]any{"iss": s.Issuer, "aud": s.ClientID, "sub": g.user.ID, "tenant": s.Tenant, "nonce": g.nonce,
		"email": g.user.Email, "email_verified": true, "iat": now.Unix(), "exp": now.Add(TokenTTL).Unix()})
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id_token": tok, "token_type": "Bearer", "expires_in": int(TokenTTL.Seconds())})
}

func (s *Server) mint(c map[string]any) (string, error) {
	b64 := base64.RawURLEncoding.EncodeToString
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": s.kid(), "typ": "JWT"})
	p, _ := json.Marshal(c)
	in := b64(h) + "." + b64(p)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.Key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return in + "." + b64(sig), nil
}

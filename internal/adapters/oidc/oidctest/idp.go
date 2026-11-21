// Package oidctest is a test OIDC IdP: discovery, key set, an authorize endpoint that
// signs the user in at once, and a token endpoint that checks the client and PKCE.
// It also mints arbitrary tokens, including deliberately wrong ones, for law tests.
package oidctest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"
)

// Now is the clock Mint uses; tests may replace it.
var Now = time.Now

// IdP is one running test IdP.
type IdP struct {
	Server                   *httptest.Server
	Key                      *rsa.PrivateKey
	KID                      string
	ClientID, Secret, Tenant string
	User                     string // who signs in at /authorize
	// URL, when set, is the issuer instead of Server's (an IdP served by Handler).
	URL string

	mu    sync.Mutex
	codes map[string]pending
}

type pending struct{ nonce, challenge, redirect, user string }

// New starts an IdP over TLS for one tenant's client.
func New(tenant, clientID, secret string) *IdP {
	idp := NewAt(tenant, clientID, secret, "")
	idp.Server = httptest.NewTLSServer(idp.Handler())
	return idp
}

// NewAt is an IdP whose issuer is url, served by Handler wherever the caller serves it
// (tools/devidp, for a local cell).
func NewAt(tenant, clientID, secret, url string) *IdP {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return &IdP{Key: k, KID: "k-" + tenant, ClientID: clientID, Secret: secret, Tenant: tenant, User: "usr_1", URL: url, codes: map[string]pending{}}
}

// Handler serves discovery, the key set, /authorize and /token.
func (i *IdP) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", i.discovery)
	mux.HandleFunc("/jwks", i.jwks)
	mux.HandleFunc("/authorize", i.authorize)
	mux.HandleFunc("/token", i.token)
	return mux
}

// Issuer is the IdP's issuer URL.
func (i *IdP) Issuer() string {
	if i.URL != "" {
		return i.URL
	}
	return i.Server.URL
}

func (i *IdP) discovery(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"issuer": i.Issuer(), "authorization_endpoint": i.Issuer() + "/authorize",
		"token_endpoint": i.Issuer() + "/token", "jwks_uri": i.Issuer() + "/jwks"})
}

func (i *IdP) jwks(w http.ResponseWriter, _ *http.Request) {
	b64 := base64.RawURLEncoding.EncodeToString
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": i.KID, "alg": "RS256", "use": "sig",
		"n": b64(i.Key.N.Bytes()), "e": b64(big.NewInt(int64(i.Key.E)).Bytes())}}})
}

func (i *IdP) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != i.ClientID || q.Get("code_challenge_method") != "S256" || q.Get("response_type") != "code" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	code := "code-" + q.Get("state")
	i.mu.Lock()
	i.codes[code] = pending{q.Get("nonce"), q.Get("code_challenge"), q.Get("redirect_uri"), i.User}
	i.mu.Unlock()
	u, _ := url.Parse(q.Get("redirect_uri"))
	u.RawQuery = url.Values{"code": {code}, "state": {q.Get("state")}}.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (i *IdP) token(w http.ResponseWriter, r *http.Request) {
	id, secret, ok := r.BasicAuth()
	if ok {
		id, _ = url.QueryUnescape(id)
		secret, _ = url.QueryUnescape(secret)
	}
	i.mu.Lock()
	p, found := i.codes[r.PostFormValue("code")]
	delete(i.codes, r.PostFormValue("code")) // codes are single use
	i.mu.Unlock()
	h := sha256.Sum256([]byte(r.PostFormValue("code_verifier")))
	if !ok || id != i.ClientID || secret != i.Secret || !found || p.redirect != r.PostFormValue("redirect_uri") ||
		base64.RawURLEncoding.EncodeToString(h[:]) != p.challenge {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	tok := i.Mint(map[string]any{"iss": i.Issuer(), "aud": i.ClientID, "sub": p.user, "tenant": i.Tenant,
		"nonce": p.nonce, "email": p.user + "@" + i.Tenant + ".example"}, i.Key, i.KID)
	_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok, "token_type": "Bearer"})
}

// Mint signs claims with key (RS256). Missing iat and exp are set to now and now+1h by Now.
func (i *IdP) Mint(c map[string]any, key *rsa.PrivateKey, kid string) string {
	if _, ok := c["iat"]; !ok {
		c["iat"] = Now().Unix()
	}
	if _, ok := c["exp"]; !ok {
		c["exp"] = Now().Unix() + 3600
	}
	b64 := base64.RawURLEncoding.EncodeToString
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	p, _ := json.Marshal(c)
	in := b64(h) + "." + b64(p)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	return in + "." + b64(sig)
}

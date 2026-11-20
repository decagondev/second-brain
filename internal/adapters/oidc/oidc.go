// Package oidc verifies a tenant's OIDC ID tokens and runs the authorization-code flow
// with PKCE (sb-identity(7), SB-106). Standard library only: RS256 over the issuer's
// published key set. Every failure is an error; nothing is accepted by default.
package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// Leeway is the clock skew allowed on exp and iat (sb-identity(7)).
const Leeway = 60 * time.Second

// IssuerFromIdP turns a manifest idp (oidc://HOST[/path]) into the issuer URL. An
// identity-platform://PROJECT/IDP_TENANT idp gives Identity Platform's issuer for PROJECT.
func IssuerFromIdP(idp string) (string, error) {
	if project, _, ok := manifest.IdentityPlatform(idp); ok {
		return SecureTokenIssuer + project, nil
	}
	rest, ok := strings.CutPrefix(idp, "oidc://")
	if !ok || rest == "" {
		return "", fmt.Errorf("oidc: %q is not oidc://HOST[/path]", idp)
	}
	return "https://" + rest, nil
}

// SecureTokenIssuer is the prefix of Identity Platform's issuer; the project id follows.
const SecureTokenIssuer = "https://securetoken.google.com/"

// IdentityPlatform configures p for an identity-platform://PROJECT/IDP_TENANT idp
// (sb-identity(7), Identity Platform, exactly; SB-907): the audience is the project, the
// token's firebase.tenant must be the company's own Identity Platform tenant, and the
// directory user is found by the token's verified email.
func (p *Provider) IdentityPlatform(idp string) error {
	project, idpTenant, ok := manifest.IdentityPlatform(idp)
	if !ok {
		return fmt.Errorf("oidc: %q is not identity-platform://PROJECT/IDP_TENANT", idp)
	}
	p.ClientID, p.IdPTenant = project, idpTenant
	return nil
}

// Provider is one tenant's IdP as one cell sees it.
type Provider struct {
	Issuer, ClientID, Tenant string
	// IdPTenant is set only for Identity Platform: the company's own Identity Platform
	// tenant, which every token's firebase.tenant must equal.
	IdPTenant string
	HTTP      *http.Client
	Now       func() time.Time

	AuthEndpoint, TokenEndpoint, JWKSURI string

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// RefreshEvery bounds key-set fetches: a token with an unknown kid triggers at most one
// fetch per interval, so forged tokens can't be used to hammer the IdP.
const RefreshEvery = time.Minute

var _ ports.IdentityProvider = (*Provider)(nil)

// Discover reads the issuer's discovery document. Its issuer must be exactly the expected one.
func Discover(ctx context.Context, issuer, clientID, tenant string, hc *http.Client, now func() time.Time) (*Provider, error) {
	var d struct {
		Issuer        string `json:"issuer"`
		Authorization string `json:"authorization_endpoint"`
		Token         string `json:"token_endpoint"`
		JWKS          string `json:"jwks_uri"`
	}
	if err := getJSON(ctx, hc, strings.TrimSuffix(issuer, "/")+"/.well-known/openid-configuration", &d); err != nil {
		return nil, err
	}
	if d.Issuer != issuer || d.Authorization == "" || d.Token == "" || d.JWKS == "" {
		return nil, fmt.Errorf("oidc: discovery for %s is incomplete or names issuer %q", issuer, d.Issuer)
	}
	return &Provider{Issuer: issuer, ClientID: clientID, Tenant: tenant, HTTP: hc, Now: now,
		AuthEndpoint: d.Authorization, TokenEndpoint: d.Token, JWKSURI: d.JWKS}, nil
}

func getJSON(ctx context.Context, hc *http.Client, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("oidc: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oidc: %s returned %d", u, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v)
}

// key returns the signing key for kid, refreshing the key set once if kid is new.
func (p *Provider) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.mu.Lock()
	k, ok := p.keys[kid]
	recent := !p.fetched.IsZero() && p.Now().Sub(p.fetched) < RefreshEvery
	p.mu.Unlock()
	if ok {
		return k, nil
	}
	if recent {
		return nil, fmt.Errorf("oidc: no key %q in the issuer's key set", kid)
	}
	var set struct {
		Keys []struct{ Kty, Kid, Alg, N, E string } `json:"keys"`
	}
	if err := getJSON(ctx, p.HTTP, p.JWKSURI, &set); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, j := range set.Keys {
		if j.Kty != "RSA" || (j.Alg != "" && j.Alg != "RS256") {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(j.N)
		e, err2 := base64.RawURLEncoding.DecodeString(j.E)
		if err1 != nil || err2 != nil || len(e) > 4 {
			continue
		}
		keys[j.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	p.mu.Lock()
	p.keys, p.fetched = keys, p.Now()
	p.mu.Unlock()
	if k, ok := keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("oidc: no key %q in the issuer's key set", kid)
}

type claims struct {
	Iss    string          `json:"iss"`
	Sub    string          `json:"sub"`
	Aud    json.RawMessage `json:"aud"`
	Exp    int64           `json:"exp"`
	Iat    int64           `json:"iat"`
	Nonce  string          `json:"nonce"`
	Tenant string          `json:"tenant"`
	Email  string          `json:"email"`
	// Identity Platform only.
	EmailVerified bool `json:"email_verified"`
	Firebase      struct {
		Tenant string `json:"tenant"`
	} `json:"firebase"`
}

func (c claims) audience(want string) bool {
	var one string
	if json.Unmarshal(c.Aud, &one) == nil {
		return one == want
	}
	var many []string
	if json.Unmarshal(c.Aud, &many) == nil {
		for _, a := range many {
			if a == want {
				return true
			}
		}
	}
	return false
}

// Verify checks every rule in sb-identity(7) and returns the identity, or an error.
func (p *Provider) Verify(ctx context.Context, raw, nonce string) (ports.Identity, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return ports.Identity{}, errors.New("oidc: malformed token")
	}
	var hdr struct{ Alg, Kid string }
	if err := decode(parts[0], &hdr); err != nil {
		return ports.Identity{}, err
	}
	if hdr.Alg != "RS256" {
		return ports.Identity{}, fmt.Errorf("oidc: algorithm %q refused", hdr.Alg)
	}
	k, err := p.key(ctx, hdr.Kid)
	if err != nil {
		return ports.Identity{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ports.Identity{}, errors.New("oidc: malformed signature")
	}
	h := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(k, crypto.SHA256, h[:], sig); err != nil {
		return ports.Identity{}, errors.New("oidc: bad signature")
	}
	var c claims
	if err := decode(parts[1], &c); err != nil {
		return ports.Identity{}, err
	}
	now := p.Now()
	switch {
	case c.Iss != p.Issuer:
		return ports.Identity{}, fmt.Errorf("oidc: issuer %q is not %q", c.Iss, p.Issuer)
	case !c.audience(p.ClientID):
		return ports.Identity{}, errors.New("oidc: token is for another client")
	case now.After(time.Unix(c.Exp, 0).Add(Leeway)):
		return ports.Identity{}, errors.New("oidc: token expired")
	case time.Unix(c.Iat, 0).After(now.Add(Leeway)):
		return ports.Identity{}, errors.New("oidc: token issued in the future")
	case nonce == "" || c.Nonce != nonce:
		return ports.Identity{}, errors.New("oidc: nonce mismatch")
	case c.Sub == "":
		return ports.Identity{}, errors.New("oidc: no subject")
	}
	if p.IdPTenant != "" {
		// Identity Platform: every company's token has this issuer and audience, so the
		// Identity Platform tenant is what tells them apart (LAW-02).
		if c.Firebase.Tenant == "" || c.Firebase.Tenant != p.IdPTenant {
			return ports.Identity{}, errors.New("oidc: token belongs to another tenant") // LAW-02
		}
		if c.Email == "" || !c.EmailVerified {
			return ports.Identity{}, errors.New("oidc: no verified email")
		}
		return ports.Identity{Tenant: p.Tenant, Subject: groupsync.UserID(c.Email), Email: c.Email, Issuer: c.Iss, IssuedAt: time.Unix(c.Iat, 0)}, nil
	}
	if c.Tenant == "" || c.Tenant != p.Tenant {
		return ports.Identity{}, errors.New("oidc: token belongs to another tenant") // LAW-02
	}
	return ports.Identity{Tenant: c.Tenant, Subject: c.Sub, Email: c.Email, Issuer: c.Iss, IssuedAt: time.Unix(c.Iat, 0)}, nil
}

func decode(seg string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return errors.New("oidc: malformed token")
	}
	if err := json.Unmarshal(b, v); err != nil {
		return errors.New("oidc: malformed token")
	}
	return nil
}

// Random returns n random bytes, base64url-encoded (state, nonce, PKCE verifier).
func Random(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Challenge is the S256 PKCE challenge for verifier.
func Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// AuthURL is where the browser goes to sign in.
func (p *Provider) AuthURL(redirect, state, nonce, verifier string) string {
	q := url.Values{"response_type": {"code"}, "client_id": {p.ClientID}, "redirect_uri": {redirect},
		"scope": {"openid email"}, "state": {state}, "nonce": {nonce},
		"code_challenge": {Challenge(verifier)}, "code_challenge_method": {"S256"}}
	return p.AuthEndpoint + "?" + q.Encode()
}

// Exchange trades an authorization code for a raw ID token. The secret is fetched by the
// caller from the vault at call time and never stored here (LAW-07).
func (p *Provider) Exchange(ctx context.Context, code, redirect, verifier, secret string) (string, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect},
		"client_id": {p.ClientID}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(p.ClientID), url.QueryEscape(secret))
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("oidc: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oidc: token endpoint returned %d", resp.StatusCode)
	}
	var t struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t); err != nil || t.IDToken == "" {
		return "", errors.New("oidc: no id_token in the token response")
	}
	return t.IDToken, nil
}

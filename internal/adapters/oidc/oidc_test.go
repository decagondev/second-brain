package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
)

func setup(t *testing.T) (*oidctest.IdP, *Provider) {
	t.Helper()
	idp := oidctest.New("acme-logistics", "cell-acme", "s")
	t.Cleanup(idp.Server.Close)
	p, err := Discover(context.Background(), idp.Issuer(), "cell-acme", "acme-logistics", idp.Server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return idp, p
}

func good(idp *oidctest.IdP) map[string]any {
	return map[string]any{"iss": idp.Issuer(), "aud": "cell-acme", "sub": "usr_1", "tenant": "acme-logistics", "nonce": "n"}
}

func TestIssuerFromIdP(t *testing.T) {
	if got, err := IssuerFromIdP("oidc://login.acme.example/realms/x"); err != nil || got != "https://login.acme.example/realms/x" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := IssuerFromIdP("https://login.acme.example"); err == nil {
		t.Fatal("want an error")
	}
}

// Every rule in sb-identity(7), one failing token each.
func TestEveryRule(t *testing.T) {
	idp, p := setup(t)
	ctx := context.Background()
	if id, err := p.Verify(ctx, idp.Mint(good(idp), idp.Key, idp.KID), "n"); err != nil || id.Subject != "usr_1" || id.Tenant != "acme-logistics" {
		t.Fatalf("good token: %+v %v", id, err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	mut := func(k string, v any) map[string]any { c := good(idp); c[k] = v; return c }
	now := time.Now().Unix()
	cases := map[string]string{
		"wrong issuer":    idp.Mint(mut("iss", "https://evil.example"), idp.Key, idp.KID),
		"wrong audience":  idp.Mint(mut("aud", "cell-beta"), idp.Key, idp.KID),
		"expired":         idp.Mint(mut("exp", now-120), idp.Key, idp.KID),
		"future iat":      idp.Mint(mut("iat", now+600), idp.Key, idp.KID),
		"wrong nonce":     idp.Mint(mut("nonce", "x"), idp.Key, idp.KID),
		"other tenant":    idp.Mint(mut("tenant", "beta-freight"), idp.Key, idp.KID),
		"no tenant":       idp.Mint(mut("tenant", ""), idp.Key, idp.KID),
		"no subject":      idp.Mint(mut("sub", ""), idp.Key, idp.KID),
		"other key":       idp.Mint(good(idp), other, idp.KID),
		"unknown kid":     idp.Mint(good(idp), other, "k-nope"),
		"malformed":       "a.b",
		"alg none":        noneToken(idp.Mint(good(idp), idp.Key, idp.KID)),
		"audience in set": "", // filled below: a list that doesn't contain us
	}
	cases["audience in set"] = idp.Mint(mut("aud", []string{"cell-beta", "cell-gamma"}), idp.Key, idp.KID)
	for name, tok := range cases {
		if _, err := p.Verify(ctx, tok, "n"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := p.Verify(ctx, idp.Mint(mut("aud", []string{"x", "cell-acme"}), idp.Key, idp.KID), "n"); err != nil {
		t.Errorf("audience list containing us: %v", err)
	}
	if _, err := p.Verify(ctx, idp.Mint(good(idp), idp.Key, idp.KID), ""); err == nil {
		t.Error("an empty expected nonce must refuse")
	}
}

func noneToken(tok string) string {
	parts := strings.Split(tok, ".")
	return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","kid":"k-acme-logistics"}`)) + "." + parts[1] + "."
}

// An IdP that can't be reached is an error, never a pass.
func TestUnreachable(t *testing.T) {
	idp, p := setup(t)
	tok := idp.Mint(good(idp), idp.Key, "k-rotated") // forces a key-set fetch
	idp.Server.Close()
	if _, err := p.Verify(context.Background(), tok, "n"); err == nil {
		t.Fatal("accepted with the IdP down")
	}
	if _, err := Discover(context.Background(), idp.Issuer(), "c", "t", &http.Client{Timeout: time.Second}, time.Now); err == nil {
		t.Fatal("discovery with the IdP down must fail")
	}
}

// Unknown key ids refetch the key set at most once per RefreshEvery.
func TestKeySetRefreshIsBounded(t *testing.T) {
	idp, p := setup(t)
	fetches := 0
	inner := idp.Server.Client().Transport
	p.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/jwks") {
			fetches++
		}
		return inner.RoundTrip(r)
	})}
	for i := 0; i < 5; i++ {
		_, _ = p.Verify(context.Background(), idp.Mint(good(idp), idp.Key, "k-forged"), "n")
	}
	if fetches != 1 {
		t.Fatalf("%d key-set fetches for 5 forged tokens; want 1", fetches)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
)

// LAW-02 at sign-in (SB-106): a cell accepts only ID tokens issued by its own tenant's
// IdP, for its own client, carrying its own tenant claim. A token from tenant A's IdP,
// or claiming tenant A, never signs in to tenant B's cell.

// TokenSpec describes one ID token to mint and present.
type TokenSpec struct {
	IssuerOf  string // tenant whose IdP issues it, or "evil" for an unknown issuer
	SignedBy  string // tenant whose IdP key signs it, or "evil"
	Tenant    string // the tenant claim ("" for none)
	Audience  string // tenant whose cell client id is the audience
	PresentAt string // tenant whose cell verifies it
}

// TokenCheck mints spec with test IdPs for every tenant and verifies it at the cell.
// It returns the tenant of the accepted identity, or "" if the token was refused.
type TokenCheck func(spec TokenSpec) (accepted string, err error)

// NewTokenCheck builds the system under test: real OIDC verification (SB-106) at each
// tenant's cell, with a test IdP per tenant and one unknown ("evil") IdP.
var NewTokenCheck = func() TokenCheck {
	idps := map[string]*oidctest.IdP{}
	cells := map[string]*oidc.Provider{}
	for _, t := range []string{"acme-logistics", "beta-freight", "evil"} {
		idp := oidctest.New(t, "cell-"+t, "secret-"+t)
		idps[t] = idp
		if t == "evil" {
			continue
		}
		p, err := oidc.Discover(context.Background(), idp.Issuer(), idp.ClientID, t, idp.Server.Client(), time.Now)
		if err != nil {
			panic(err)
		}
		cells[t] = p
	}
	return func(s TokenSpec) (string, error) {
		iss, signer := idps[s.IssuerOf], idps[s.SignedBy]
		tok := iss.Mint(map[string]any{"iss": iss.Issuer(), "aud": "cell-" + s.Audience, "sub": "usr_1",
			"tenant": s.Tenant, "nonce": "n"}, signer.Key, iss.KID)
		id, err := cells[s.PresentAt].Verify(context.Background(), tok, "n")
		if err != nil {
			return "", nil // refused
		}
		return id.Tenant, nil
	}
}

func law02Tokens(r *rand.Rand, check TokenCheck) error {
	tenants := []string{"acme-logistics", "beta-freight"}
	pick := func(extra ...string) string { all := append(tenants, extra...); return all[r.Intn(len(all))] }
	for i := 0; i < 12; i++ {
		s := TokenSpec{IssuerOf: pick("evil"), SignedBy: pick("evil"), Tenant: pick(""), Audience: pick(), PresentAt: pick()}
		if r.Intn(2) == 0 { // keep half the tokens well-formed for some cell
			s.SignedBy, s.Audience, s.Tenant = s.IssuerOf, s.PresentAt, s.PresentAt
		}
		got, err := check(s)
		if err != nil {
			return err
		}
		valid := s.IssuerOf == s.PresentAt && s.SignedBy == s.IssuerOf && s.Audience == s.PresentAt && s.Tenant == s.PresentAt
		switch {
		case got != "" && got != s.PresentAt:
			return fmt.Errorf("token %+v signed in as tenant %q at %s's cell", s, got, s.PresentAt)
		case valid && got == "":
			return fmt.Errorf("valid token %+v was refused", s)
		case !valid && got != "":
			return fmt.Errorf("token %+v was accepted", s)
		}
	}
	return nil
}

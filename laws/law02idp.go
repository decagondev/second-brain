package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/adapters/oidc"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
)

// LAW-02 at sign-in through Identity Platform (SB-907): every company's token from one
// Identity Platform project has the same issuer, signing keys and audience, so only the
// firebase.tenant claim tells them apart. A token from another company's Identity
// Platform tenant never signs in to this cell.

// IPWorld is one Identity Platform project (a test IdP standing in for securetoken) and
// two cells that sign in through it, each with its own Identity Platform tenant.
type IPWorld struct {
	IdP   *oidctest.IdP
	Cells map[string]*oidc.Provider // tenant -> verifier
	IdPOf map[string]string         // tenant -> its Identity Platform tenant
}

// NewIPWorld builds the system under test: real verification (SB-907) at each cell.
var NewIPWorld = func() *IPWorld {
	const project = "sb-idp-shared"
	idp := oidctest.New("unused", project, "unused")
	w := &IPWorld{IdP: idp, Cells: map[string]*oidc.Provider{},
		IdPOf: map[string]string{"acme-logistics": "acme-logistics-k3f9", "beta-freight": "beta-freight-p7q2"}}
	for t, it := range w.IdPOf {
		p, err := oidc.Discover(context.Background(), idp.Issuer(), "", t, idp.Server.Client(), time.Now)
		if err != nil {
			panic(err)
		}
		if err := p.IdentityPlatform("identity-platform://" + project + "/" + it); err != nil {
			panic(err)
		}
		w.Cells[t] = p
	}
	return w
}

// Mint issues a token from the shared project for an Identity Platform tenant ("" for none).
func (w *IPWorld) Mint(idpTenant, email string, verified bool) string {
	c := map[string]any{"iss": w.IdP.Issuer(), "aud": w.IdP.ClientID, "sub": "fb-uid-1", "nonce": "n",
		"email": email, "email_verified": verified}
	if idpTenant != "" {
		c["firebase"] = map[string]any{"tenant": idpTenant, "sign_in_provider": "saml.workspace"}
	}
	return w.IdP.Mint(c, w.IdP.Key, w.IdP.KID)
}

// IPVerify presents a token at a cell and returns the tenant it signed in as, or "".
type IPVerify func(w *IPWorld, cell, token string) string

func verifyIP(w *IPWorld, cell, token string) string {
	id, err := w.Cells[cell].Verify(context.Background(), token, "n")
	if err != nil {
		return ""
	}
	return id.Tenant
}

func law02IdentityPlatform(r *rand.Rand, w *IPWorld, verify IPVerify) error {
	tenants := []string{"acme-logistics", "beta-freight"}
	for i := 0; i < 12; i++ {
		at := tenants[r.Intn(2)]
		from := []string{"acme-logistics-k3f9", "beta-freight-p7q2", "", "gamma-parts-z1x8"}[r.Intn(4)]
		if r.Intn(2) == 0 {
			from = w.IdPOf[at] // keep half the tokens well-formed for their cell
		}
		verified := r.Intn(4) != 0
		got := verify(w, at, w.Mint(from, "ana@example.test", verified))
		valid := from == w.IdPOf[at] && verified
		switch {
		case got != "" && got != at:
			return fmt.Errorf("a token from Identity Platform tenant %q signed in as %q at the %s cell", from, got, at)
		case valid && got == "":
			return fmt.Errorf("the %s cell refused its own verified token", at)
		case !valid && got != "":
			return fmt.Errorf("a token from Identity Platform tenant %q (verified email %v) signed in to the %s cell", from, verified, at)
		}
	}
	return nil
}

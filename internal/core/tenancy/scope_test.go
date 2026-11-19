package tenancy

import (
	"context"
	"errors"
	"testing"
)

func TestIssuer(t *testing.T) {
	i := NewIssuer("acme-logistics")
	s, err := i.Issue("acme-logistics", "usr_1", []string{"it-leads"})
	if err != nil || s.Tenant() != "acme-logistics" || s.User() != "usr_1" || len(s.Groups()) != 1 {
		t.Fatalf("%+v %v", s, err)
	}
	s.Groups()[0] = "changed"
	if s.Groups()[0] != "it-leads" {
		t.Fatal("Groups must return a copy")
	}
	if _, err := i.Issue("beta-freight", "usr_1", nil); !errors.Is(err, ErrOtherTenant) {
		t.Fatalf("other tenant: %v", err)
	}
	if _, err := (Issuer{}).Issue("", "usr_1", nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("the zero Issuer must issue nothing")
	}
	if (Scope{}).Check() == nil {
		t.Fatal("the zero Scope must be invalid")
	}
	if _, err := FromContext(context.Background()); err == nil {
		t.Fatal("no scope in context must be an error")
	}
	got, err := FromContext(WithScope(context.Background(), s))
	if err != nil || got.Tenant() != "acme-logistics" {
		t.Fatalf("%+v %v", got, err)
	}
}

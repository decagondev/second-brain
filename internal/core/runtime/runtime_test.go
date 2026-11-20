package runtime

import (
	"context"
	"errors"
	"testing"

	"example.com/second-brain/internal/core/release"
)

type store struct{}

func (store) Get(context.Context, string, string) ([]byte, error) {
	return nil, errors.New("no such release")
}

type deployer struct{ calls int }

func (d *deployer) Attest(context.Context, string) error { d.calls++; return nil }
func (d *deployer) Deploy(context.Context, Service, string, string) error {
	d.calls++
	return nil
}
func (d *deployer) Running(context.Context, Service) (string, string, error) { return "", "", nil }

// A release that can't be admitted changes nothing (the law test covers the signed paths).
func TestUnadmittedChangesNothing(t *testing.T) {
	d := &deployer{}
	u := Updater{Registry: "r", Store: store{}, Verifier: release.Ed25519Verifier{Trusted: "ed25519:x"}, Deploy: d,
		Target: func(context.Context) (string, error) { return "rel_2027-03-02_abcdef", nil }}
	if _, err := u.Run(context.Background()); !errors.Is(err, ErrUnsigned) || d.calls != 0 {
		t.Fatalf("%v, %d calls", err, d.calls)
	}
	if got := u.Image(Services[0], "ab"); got != "r/api@sha256:ab" {
		t.Fatal(got)
	}
}

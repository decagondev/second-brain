package localvault

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/secrets"
	"example.com/second-brain/internal/obs"
)

const ref = "vault://tenants/acme-logistics/connectors/docs"

func TestResolveAndRedact(t *testing.T) {
	v := Dir{Root: t.TempDir()}
	if err := v.Put(ref, "sbcanary-0123456789"); err != nil {
		t.Fatal(err)
	}
	red := obs.NewRedactor()
	r := secrets.Resolver{Tenant: "acme-logistics", Vault: v, Redact: red}
	got, err := r.Get(context.Background(), ref)
	if err != nil || got != "sbcanary-0123456789" {
		t.Fatalf("%q %v", got, err)
	}
	var buf bytes.Buffer
	obs.NewLogger(&buf, red).With("auth", "Bearer "+got).Info("hello")
	if strings.Contains(buf.String(), got) || !strings.Contains(buf.String(), obs.Redacted) {
		t.Fatalf("WithAttrs leaked: %s", buf.String())
	}
}

func TestRefusals(t *testing.T) {
	v := Dir{Root: t.TempDir()}
	_ = v.Put(ref, "sbcanary-0123456789")
	ctx := context.Background()
	r := secrets.Resolver{Tenant: "beta-freight", Vault: v, Redact: obs.NewRedactor()}
	if _, err := r.Get(ctx, ref); !errors.Is(err, secrets.ErrOtherTenant) {
		t.Errorf("other tenant: %v", err)
	}
	for _, bad := range []string{"vault://tenants/acme-logistics/../../etc", "vault://tenants/acme-logistics/connectors/wiki", "https://x"} {
		if _, err := secrets.ParseRef(bad); err == nil {
			t.Errorf("%q: want a bad reference", bad)
		}
	}
	if _, err := (secrets.Resolver{Tenant: "acme-logistics", Vault: v}).Get(ctx, ref); err == nil {
		t.Error("no redactor: want a refusal")
	}
	p, _ := v.path(ref)
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get(ctx, ref); !errors.Is(err, ErrExposed) {
		t.Errorf("world-readable secret: %v", err)
	}
}

// BUG-0032: every secret path the cell reads is a valid reference: the fallback model's
// key (SB-5601), the transcribe provider's (SB-6102) and the Atlassian account's (SB-6201).
func TestEveryPathTheCellReadsIsValid(t *testing.T) {
	for _, p := range []string{"model", "model-fallback", "model-transcribe", "atlassian", "msgraph", "teams", "mattermost", "email", "events"} {
		if _, err := secrets.ParseRef("vault://tenants/acme-logistics/" + p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

package wire

import (
	"errors"
	"strings"
	"testing"
)

const good = `{"tenant": "acme-logistics", "manifest": "/m.yaml", "state": "/s", "vault": "/v", "policies": "/p",
 "listen": {"api": "127.0.0.1:8080"}, "base_url": "https://acme-logistics.sb.localhost",
 "oidc": {"client_id": "sb-acme-logistics"}, "model": {"provider": "extractive"}}`

func TestParseGood(t *testing.T) {
	c, err := Parse([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if c.Tenant != "acme-logistics" || c.Listen.API != "127.0.0.1:8080" || c.Durable {
		t.Fatalf("%+v", c)
	}
}

func TestParseRefuses(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"unknown field":   {strings.Replace(good, `"tenant"`, `"tenantt": "x", "tenant"`, 1), "unknown field"},
		"missing":         {strings.Replace(good, `"vault": "/v",`, "", 1), "missing vault"},
		"wrong type":      {strings.Replace(good, `"client_id": "sb-acme-logistics"`, `"client_id": 7`, 1), "cannot unmarshal"},
		"two values":      {good + ` {}`, "more than one"},
		"bad listen":      {strings.Replace(good, `127.0.0.1:8080`, `8080`, 1), "host:port"},
		"bad url":         {strings.Replace(good, `https://acme-logistics.sb.localhost`, `ftp://x`, 1), "http(s) URL"},
		"bad provider":    {strings.Replace(good, `"extractive"`, `"mystery"`, 1), "not one of"},
		"gateway no base": {strings.Replace(good, `"extractive"`, `"anthropic"`, 1), "needs model.base"},
		"not json":        {"tenant: x", ""},
	} {
		_, err := Parse([]byte(tc.body))
		if !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want ErrConfig with %q", name, err, tc.want)
		}
	}
}

func TestDurableRefuses(t *testing.T) {
	c, _ := Parse([]byte(good))
	c.Durable = true
	if _, err := Build(t.Context(), c, Options{}); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "in memory only") {
		t.Fatalf("Build: %v", err)
	}
}

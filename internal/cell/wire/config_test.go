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

// SB-1710: a model on the box is local (no key, price 0); the embedder is checked whole.
func TestParseModelOnTheBox(t *testing.T) {
	local := strings.Replace(good, `"model": {"provider": "extractive"}`,
		`"model": {"provider": "openaicompat", "base": "http://model:8080/v1", "id": "qwen", "local": true},
		 "embed": {"provider": "openaicompat", "base": "http://model-embed:8080/v1", "model": "nomic", "dim": 768}`, 1)
	c, err := Parse([]byte(local))
	if err != nil || !c.Model.Local || embedDim(c) != 768 || embedderFor(c).Model() != "openaicompat:nomic:768" {
		t.Fatalf("%+v %v", c.Model, err)
	}
	for name, body := range map[string]string{
		"priced elsewhere": strings.Replace(local, `"local": true`, `"local": false`, 1),
		"local anthropic":  strings.Replace(local, `"provider": "openaicompat", "base": "http://model:8080/v1"`, `"provider": "anthropic", "base": "http://model:8080/v1"`, 1),
		"no dim":           strings.Replace(local, `, "dim": 768`, ``, 1),
		"bad embedder":     strings.Replace(local, `"provider": "openaicompat", "base": "http://model-embed`, `"provider": "mystery", "base": "http://model-embed`, 1),
	} {
		if _, err := Parse([]byte(body)); !errors.Is(err, ErrConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if c, _ := Parse([]byte(good)); embedDim(c) != 64 || embedderFor(c).Model() != "local-hashing-64" {
		t.Fatal("the default embedder changed")
	}
}

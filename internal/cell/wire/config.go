// Package wire is a cell's one composition root (sb-cell(5), sb-runtime(7); SB-1501): it
// reads the cell's configuration and builds every part of the running cell from it, once.
package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// Config is a cell's configuration file (sb-cell(5)).
type Config struct {
	Tenant   string `json:"tenant"`
	Manifest string `json:"manifest"`
	State    string `json:"state"`
	Vault    string `json:"vault"`
	Policies string `json:"policies"`
	Listen   struct {
		API string `json:"api"`
	} `json:"listen"`
	BaseURL string `json:"base_url"`
	OIDC    struct {
		ClientID string `json:"client_id"`
		Issuer   string `json:"issuer"`
	} `json:"oidc"`
	Web   string `json:"web"`
	Model struct {
		Provider string `json:"provider"`
		Base     string `json:"base"`
		ID       string `json:"id"`
	} `json:"model"`
	Connectors map[string]struct {
		Base string `json:"base"`
	} `json:"connectors"`
	Chat struct {
		TeamID string `json:"team_id"`
	} `json:"chat"`
	Durable bool `json:"durable"`
}

// ErrConfig is a configuration a cell can't start with.
var ErrConfig = errors.New("wire: bad cell configuration")

// Providers are the model providers a cell can run (ADR-0019), and the local one.
var Providers = map[string]bool{"extractive": true, "openaicompat": true, "anthropic": true, "groq": true, "openrouter": true}

// Load reads and checks a configuration file strictly: unknown fields, missing required
// fields and bad values are errors.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	return Parse(b)
}

// Parse checks a configuration.
func Parse(b []byte) (Config, error) {
	var c Config
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	if d.More() {
		return Config{}, fmt.Errorf("%w: more than one JSON value", ErrConfig)
	}
	var missing []string
	for name, v := range map[string]string{"tenant": c.Tenant, "manifest": c.Manifest, "state": c.State, "vault": c.Vault,
		"policies": c.Policies, "listen.api": c.Listen.API, "base_url": c.BaseURL, "oidc.client_id": c.OIDC.ClientID, "model.provider": c.Model.Provider} {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("%w: missing %s", ErrConfig, strings.Join(sorted(missing), ", "))
	}
	if _, _, err := net.SplitHostPort(c.Listen.API); err != nil {
		return Config{}, fmt.Errorf("%w: listen.api %q must be host:port", ErrConfig, c.Listen.API)
	}
	for name, v := range map[string]string{"base_url": c.BaseURL, "oidc.issuer": c.OIDC.Issuer, "model.base": c.Model.Base} {
		if v == "" {
			continue
		}
		if u, err := url.Parse(v); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return Config{}, fmt.Errorf("%w: %s %q must be an http(s) URL", ErrConfig, name, v)
		}
	}
	if !Providers[c.Model.Provider] {
		return Config{}, fmt.Errorf("%w: model.provider %q is not one of extractive, openaicompat, anthropic, groq, openrouter", ErrConfig, c.Model.Provider)
	}
	if c.Model.Provider != "extractive" && (c.Model.Base == "" || c.Model.ID == "") {
		return Config{}, fmt.Errorf("%w: model.provider %s needs model.base and model.id", ErrConfig, c.Model.Provider)
	}
	return c, nil
}

func sorted(xs []string) []string {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
	return xs
}

package laws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/core/secrets"
	"example.com/second-brain/internal/obs"
)

// LAW-07 (SB-205): no secret in a log. Secrets are resolved through the Secrets port,
// and whatever is resolved can't reach log output by any path: message, attribute,
// error, nested group, or an attribute named like a secret.

// LogEvent is one way a careless caller might log a secret.
type LogEvent struct {
	Path  string // "msg", "attr", "err", "group", "named"
	Value string // the secret, possibly embedded in other text
}

// SecretWorld resolves secrets and logs events, returning everything the logger wrote.
type SecretWorld interface {
	Resolve(tenant, kind, value string) (string, error) // store value at the connector path, resolve it
	Log(events []LogEvent) (output string, err error)
}

// NewSecretWorld builds the system under test: the local vault, the secrets Resolver and
// the redacting logger (SB-205), with the vault under root.
var NewSecretWorld = func(root string) SecretWorld {
	red := obs.NewRedactor()
	v := localvault.Dir{Root: root}
	return &secretWorld{vault: v, red: red, res: secrets.Resolver{Tenant: "acme-logistics", Vault: v, Redact: red}}
}

type secretWorld struct {
	vault localvault.Dir
	red   *obs.Redactor
	res   secrets.Resolver
}

func (w *secretWorld) Resolve(tenant, kind, value string) (string, error) {
	ref := secrets.ConnectorRef(tenant, kind)
	if err := w.vault.Put(ref, value); err != nil {
		return "", err
	}
	return w.res.Get(context.Background(), ref)
}

func (w *secretWorld) Log(evs []LogEvent) (string, error) {
	var buf bytes.Buffer
	l := obs.NewLogger(&buf, w.red)
	for i, e := range evs {
		switch e.Path {
		case "msg":
			l.Info("calling source with " + e.Value)
		case "attr":
			l.Info("call", "header", e.Value)
		case "err":
			l.Error("call failed", "err", errors.New("upstream said: "+e.Value))
		case "group":
			l.Info("call", slog.Group("req", slog.Group("auth", "header", e.Value)))
		case "named":
			l.Info("call", "api_token", fmt.Sprintf("v%d", i), "password", e.Value)
		}
	}
	return buf.String(), nil
}

func law07(r *rand.Rand, w SecretWorld) error {
	var secrets []string
	for i := 0; i < 1+r.Intn(3); i++ {
		v := fmt.Sprintf("sbcanary-%d-%x", i, r.Int63())
		got, err := w.Resolve("acme-logistics", []string{"docs", "chat", "tickets"}[i%3], v)
		if err != nil {
			return err
		}
		if got != v {
			return fmt.Errorf("resolved %q, stored %q", got, v)
		}
		secrets = append(secrets, v)
	}
	var evs []LogEvent
	for i := 0; i < 8; i++ {
		s := secrets[r.Intn(len(secrets))]
		val := []string{s, "Bearer " + s, "url?token=" + s + "&x=1", "prefix " + s + " suffix"}[r.Intn(4)]
		evs = append(evs, LogEvent{Path: []string{"msg", "attr", "err", "group", "named"}[r.Intn(5)], Value: val})
	}
	out, err := w.Log(evs)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		return fmt.Errorf("the logger wrote nothing; the check would prove nothing")
	}
	for _, s := range secrets {
		if strings.Contains(out, s) {
			return fmt.Errorf("secret %s reached the log:\n%s", s, out)
		}
	}
	return nil
}

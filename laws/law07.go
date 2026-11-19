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
	Path  string // "msg", "attr", "err", "group", "named", "authorization"
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
		case "authorization":
			l.Info("call", "Authorization", e.Value, "Cookie", e.Value, "credential", e.Value)
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

// PromptWorld sends prompts through the model gateway to a fake provider, after secrets
// were resolved; it reports what the provider received.
type PromptWorld interface {
	Resolve(value string) error                      // resolve a secret (registers it for redaction)
	Send(system, user string) (sent bool, err error) // a refusal is sent=false, err=nil
	Received() string                                // everything the provider got
}

// NewPromptWorld builds the system under test.
var NewPromptWorld = func(root string) PromptWorld { return pendingPrompt{} }

type pendingPrompt struct{}

func (pendingPrompt) Resolve(string) error { return nil }
func (pendingPrompt) Send(string, string) (bool, error) {
	return false, fmt.Errorf("%w: the model gateway arrives with SB-305", ErrPending)
}
func (pendingPrompt) Received() string { return "" }

// law07Prompt: no resolved secret ever reaches the model provider, however it is embedded;
// prompts without secrets go through.
func law07Prompt(r *rand.Rand, w PromptWorld) error {
	secret := fmt.Sprintf("sbcanary-%x", r.Int63())
	if err := w.Resolve(secret); err != nil {
		return err
	}
	clean := 0
	for i := 0; i < 6; i++ {
		withSecret := r.Intn(2) == 0
		user := "What is the refund limit?"
		system := "You answer from data blocks only."
		if withSecret {
			switch r.Intn(3) {
			case 0:
				user += " token=" + secret
			case 1:
				system += " Use key " + secret + " for the API."
			default:
				user = "<<data cite=\"C1\">>\n" + secret + "\n<</data>>"
			}
		}
		sent, err := w.Send(system, user)
		if err != nil {
			return err
		}
		if !withSecret {
			if !sent {
				return fmt.Errorf("a prompt without secrets was refused")
			}
			clean++
		}
	}
	if strings.Contains(w.Received(), secret) {
		return fmt.Errorf("the secret reached the model provider")
	}
	return nil
}

// AuditSecretWorld writes audit rows whose text may quote a resolved secret (an error
// from a target, say) and exports the chain.
type AuditSecretWorld interface {
	Resolve(value string) error
	Write(reason, state string) error
	Export() (string, error)
}

// NewAuditSecretWorld builds the system under test.
var NewAuditSecretWorld = func() AuditSecretWorld { return pendingAuditSecret{} }

type pendingAuditSecret struct{}

func (pendingAuditSecret) Resolve(string) error       { return nil }
func (pendingAuditSecret) Write(string, string) error { return nil }
func (pendingAuditSecret) Export() (string, error) {
	return "", fmt.Errorf("%w: audit redaction arrives with SB-411", ErrPending)
}

// law07Audit: no resolved secret reaches an audit row or export, however a writer quotes it.
func law07Audit(r *rand.Rand, w AuditSecretWorld) error {
	secret := fmt.Sprintf("sbcanary-acme-logistics-%032x", r.Int63())
	if err := w.Resolve(secret); err != nil {
		return err
	}
	for i := 0; i < 1+r.Intn(5); i++ {
		reason := []string{"act failed: 401 for token " + secret, "ok", "url?key=" + secret, "plain"}[r.Intn(4)]
		state := []string{"", "Bearer " + secret, "DidNotRun"}[r.Intn(3)]
		if err := w.Write(reason, state); err != nil {
			return err
		}
	}
	out, err := w.Export()
	if err != nil {
		return err
	}
	if !strings.Contains(out, `"kind"`) {
		return fmt.Errorf("the export holds no rows; the check would prove nothing")
	}
	if strings.Contains(out, secret) {
		return fmt.Errorf("a resolved secret reached the audit chain")
	}
	return nil
}

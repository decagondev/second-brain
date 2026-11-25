package wire

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/smtpmail"
	"example.com/second-brain/internal/core/notify"
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

// SB-2005: mattermost is none, or an https base and a team.
func TestMattermostConfig(t *testing.T) {
	for _, mm := range []struct{ base, team string }{{"http://chat.example", "t"}, {"https://chat.example", ""}, {"", "t"}, {"https://u:p@chat.example", "t"}} {
		c := Config{}
		c.Mattermost.Base, c.Mattermost.TeamID = mm.base, mm.team
		if err := c.checkMattermost(); !errors.Is(err, ErrConfig) {
			t.Errorf("%+v: %v", mm, err)
		}
	}
	c := Config{}
	c.Mattermost.Base, c.Mattermost.TeamID = "https://chat.example", "t"
	if err := c.checkMattermost(); err != nil {
		t.Fatal(err)
	}
}

// SB-2303: an s3 copy is its own kind, always sealed, at a well-formed s3:// address.
func TestParseS3Copy(t *testing.T) {
	const seal = "sb-recovery-pub:"
	with := func(b string) string {
		return strings.Replace(good, `"model"`, `"backups": [`+b+`], "model"`, 1)
	}
	if _, err := Parse([]byte(with(`{"s3": "s3://acme-backups/sb", "offsite": true}`))); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "needs seal") {
		t.Fatalf("an unsealed s3 copy: %v", err)
	}
	for name, b := range map[string]string{
		"bad address": `{"s3": "https://acme-backups/sb", "offsite": true, "seal": "` + seal + `x"}`,
		"with a dir":  `{"s3": "s3://acme-backups/sb", "dir": "/b", "media": "disk", "seal": "` + seal + `x"}`,
	} {
		if _, err := Parse([]byte(with(b))); !errors.Is(err, ErrConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// SB-2307: email is none, or a server and one sender address; it is the notifier only
// when the cell has no chat bot, and its sign-in is optional.
func TestEmailConfigAndNotifier(t *testing.T) {
	for _, e := range []struct{ server, from string }{{"mail.acme-logistics.example", "brain@acme-logistics.example"},
		{"mail.acme-logistics.example:587", ""}, {"mail.acme-logistics.example:587", "Brain <brain@acme-logistics.example>"}, {"", "brain@acme-logistics.example"}} {
		c := Config{}
		c.Email.Server, c.Email.From = e.server, e.from
		if err := c.checkMattermost(); !errors.Is(err, ErrConfig) {
			t.Errorf("%+v: %v", e, err)
		}
	}
	vault := map[string]string{}
	cell := &Cell{Secret: func(path string) func(context.Context) (string, error) {
		return func(context.Context) (string, error) {
			v, ok := vault[path]
			if !ok {
				return "", fmt.Errorf("vault %s: %w", path, os.ErrNotExist)
			}
			return v, nil
		}
	}}
	if cell.Notifier() != nil {
		t.Fatal("a cell with nothing to send with has a notifier")
	}
	cell.Config.Email.Server, cell.Config.Email.From = "mail.acme-logistics.example:587", "brain@acme-logistics.example"
	if err := cell.Config.checkMattermost(); err != nil {
		t.Fatal(err)
	}
	m, ok := inner(cell.Notifier()).(smtpmail.Mail)
	if !ok || m.Server != "mail.acme-logistics.example:587" {
		t.Fatalf("notifier = %#v", cell.Notifier())
	}
	ctx := context.Background()
	if u, _, err := m.Login(ctx); err != nil || u != "" {
		t.Fatalf("no secret: %q %v", u, err)
	}
	vault["email"] = `{"username": "brain"}`
	if _, _, err := m.Login(ctx); err == nil {
		t.Fatal("a sign-in without a password was used")
	}
	vault["email"] = `{"username": "brain", "password": "pw"}`
	if u, p, err := m.Login(ctx); err != nil || u != "brain" || p != "pw" {
		t.Fatalf("%q %q %v", u, p, err)
	}
	cell.Config.Mattermost.Base = "https://chat.example"
	if _, ok := inner(cell.Notifier()).(smtpmail.Mail); ok {
		t.Fatal("email was chosen over the cell's chat bot")
	}
}

// inner is the sender a notifier masks for (SB-5102).
func inner(s notify.Sender) notify.Sender {
	if m, ok := s.(notify.Masking); ok {
		return m.Next
	}
	return s
}

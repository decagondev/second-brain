package wire

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/adapters/chatapp/teamsapp"
	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/msgraph"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/obs"
)

// Microsoft 365 (ADR-0023; SB-5901 to SB-5907): one Graph client for the cell, built from
// sb-cell(5) microsoft and vault://tenants/TENANT/msgraph, behind the sharepoint connector,
// the Outlook calendar, the Teams bot's people and the spaces' shared mailboxes.

func graphClient(cfg Config, secret func(string) func(context.Context) (string, error), redact *obs.Redactor, hc *http.Client) *msgraph.Client {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &msgraph.Client{Base: cfg.Microsoft.Base, Login: cfg.Microsoft.Login, HTTP: hc, Creds: secret("msgraph"), Redact: redact, Now: time.Now}
}

// graph is the cell's Graph client, made once.
func (c *Cell) graph() *msgraph.Client {
	c.graphOnce.Do(func() { c.graphC = graphClient(c.Config, c.Secret, c.Redact, c.opt.ConnectorHTTP) })
	return c.graphC
}

// sharePoint is the sharepoint connector (SB-5901, SB-5902, SB-5906).
func sharePoint(cfg Config, g *msgraph.Client) *msgraph.SharePoint {
	return &msgraph.SharePoint{Graph: g, Drives: cfg.Microsoft.Drives, Domain: cfg.Microsoft.Domain, Labels: cfg.Labels.Microsoft,
		Organizers: cfg.Microsoft.Organizers, Dir: &msgraph.Directory{Graph: g},
		Readers: msgraph.Readers{Text: folder.Text, Pages: folder.Pages, ReviewBy: folder.OfficeReviewBy, Transcript: folder.TranscriptText}}
}

// unmappedLabels are the Purview label ids the docs connector met that labels.microsoft
// doesn't map, with how many items carried each (SB-5902); nil for none.
func (c *Cell) unmappedLabels() map[string]int {
	sp, ok := c.Parts().Connectors["docs"].(*msgraph.SharePoint)
	if !ok {
		return nil
	}
	if u := sp.Unmapped(); len(u) > 0 {
		return u
	}
	return nil
}

// teams mounts the Teams bot (SB-5904) when sb-cell(5) teams names its app.
func (c *Cell) teams(mux *http.ServeMux, agent ask.Agent) {
	cfg := c.Config
	if cfg.Teams.AppID == "" {
		return
	}
	dir := &msgraph.Directory{Graph: c.graph()}
	rules := &slackapp.App{Tenant: cfg.Tenant, Users: c.Directory, Groups: c.Directory.GroupsOf, Issuer: c.Auth.Issuer, Spaces: c.Spaces,
		Email: dir.Email, Language: func() string { return c.Manifest().Language }, Notice: c.noticeLine}
	hc := c.opt.ConnectorHTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	(&teamsapp.App{Tenant: cfg.Tenant, AppID: cfg.Teams.AppID, OpenID: cfg.Teams.OpenID, Login: cfg.Teams.Login, HTTP: hc, Creds: c.Secret("teams"),
		Rules: rules, Agent: agent, Masked: func(ctx context.Context, n int) { c.masked(ctx, "teams", n) }}).Register(mux)
}

func (c *Cell) mailDir() string { return filepath.Join(c.Config.State, "mail") }

var spaceFile = regexp.MustCompile(`^[a-z0-9-]+$`)

// mailEvents drafts a reply note for each new message in the shared mailbox each space
// names with on: [mail-received] (SB-5907), as the space's first active steward. Only the
// manifest's mailboxes are read; each one's delta link is kept in state/mail/SPACE, so a
// message starts one draft, and a mailbox's first read starts from now.
func (c *Cell) mailEvents(ctx context.Context) error {
	m := c.Manifest()
	if !c.Config.microsoft() {
		return nil
	}
	box := &msgraph.Mailbox{Graph: c.graph()}
	for _, name := range sortedSpaces(m) {
		sp := m.Spaces[name]
		if sp.Mailbox == "" || !sp.Watches(ask.OnMailReceived) || !spaceFile.MatchString(name) {
			continue
		}
		p := filepath.Join(c.mailDir(), name)
		link := ""
		if b, err := os.ReadFile(p); err == nil {
			link = strings.TrimSpace(string(b))
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		mails, next, err := box.New(ctx, sp.Mailbox, link)
		if err != nil {
			c.Metrics.Add("sb_mail_failed_total", "Shared mailbox reads that failed (SB-5907).", map[string]string{"space": name}, 1)
			continue // read again on the next tick
		}
		for _, ml := range mails {
			req := ask.Request{Question: ask.MailQuestion(ml.Subject), File: &ask.FileUpload{Name: "message.txt", Data: mailText(ml)}}
			if _, err := c.runDraft(ctx, ask.SpaceKey(name), ask.DraftMail, "Mail: "+ml.Subject, req, false, ""); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(c.mailDir(), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(next+"\n"), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// mailText is a message as the file its draft reads: who sent it, its subject and body,
// then each attachment's text, cut to the file limit. All of it is data (LAW-09).
func mailText(ml msgraph.Mail) []byte {
	var b strings.Builder
	b.WriteString("From: " + ml.From + "\nSubject: " + ml.Subject + "\n\n" + ml.Body + "\n")
	for _, a := range ml.Attachments {
		b.WriteString("\nAttachment: " + a.Name + "\n")
		if t, ok := folder.Text(a.Name, a.Data); ok && len(a.Data) > 0 {
			b.WriteString(t + "\n")
		}
	}
	out := []byte(b.String())
	if len(out) > ask.MaxFileBytes {
		out = out[:ask.MaxFileBytes]
	}
	return out
}

// GraphFor is cfg's Graph client and docs connector with its secrets read from the vault at
// dir, for sb prove and sb selfhost check-microsoft (SB-6001, SB-6003), which run outside
// a cell. The sharepoint connector is nil when cfg names no drives or organizers.
func GraphFor(cfg Config, dir string, hc *http.Client) (*msgraph.Client, *msgraph.SharePoint) {
	v := localvault.Dir{Root: dir}
	secret := func(p string) func(context.Context) (string, error) {
		return func(ctx context.Context) (string, error) { return v.Get(ctx, "vault://tenants/"+cfg.Tenant+"/"+p) }
	}
	g := graphClient(cfg, secret, obs.NewRedactor(), hc)
	if len(cfg.Microsoft.Drives) == 0 && len(cfg.Microsoft.Organizers) == 0 {
		return g, nil
	}
	return g, sharePoint(cfg, g)
}

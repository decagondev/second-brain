package wire

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/atlassian"
	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

// Confluence and Jira (ADR-0024; SB-6201 to SB-6207): one Atlassian client for the cell,
// built from sb-cell(5) atlassian and vault://tenants/TENANT/atlassian, behind the
// confluence and jira connectors, the service queues and the setup check.

func atlassianClient(cfg Config, secret func(string) func(context.Context) (string, error), redact *obs.Redactor, hc *http.Client) *atlassian.Client {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &atlassian.Client{Base: cfg.Atlassian.Base, Creds: secret("atlassian"), HTTP: hc, Redact: redact}
}

// atlassianConnector is the docs: confluence or tickets: jira connector.
func atlassianConnector(cfg Config, vendor string, secret func(string) func(context.Context) (string, error), redact *obs.Redactor, hc *http.Client) (ports.Connector, error) {
	if cfg.Atlassian.Base == "" {
		return nil, errors.New(vendor + " needs atlassian.base")
	}
	cl := atlassianClient(cfg, secret, redact, hc)
	if vendor == "confluence" {
		if len(cfg.Atlassian.Spaces) == 0 {
			return nil, errors.New("confluence needs atlassian.spaces")
		}
		return &atlassian.Confluence{Client: cl, Spaces: cfg.Atlassian.Spaces, Dir: &atlassian.Directory{Client: cl}, Text: folder.Text}, nil
	}
	if len(cfg.Atlassian.Projects) == 0 {
		return nil, errors.New("jira needs atlassian.projects")
	}
	return &atlassian.Jira{Client: cl, Projects: cfg.Atlassian.Projects}, nil
}

// inboundEvents drafts the shared mailboxes' new messages (SB-5907) and the service queues'
// new requests (SB-6206).
func (c *Cell) inboundEvents(ctx context.Context) error {
	if err := c.mailEvents(ctx); err != nil {
		return err
	}
	return c.serviceEvents(ctx)
}

func (c *Cell) requestsDir() string { return filepath.Join(c.Config.State, "requests") }

// serviceEvents drafts a reply note for each new request in the Jira Service Management
// queue each space names with on: [request-received] (SB-6206), as the space's first active
// steward. Only the manifest's queues are read; the newest request seen is kept in
// state/requests/SPACE, so a request starts one draft, and a queue's first read starts
// from now. Nothing is sent.
func (c *Cell) serviceEvents(ctx context.Context) error {
	m := c.Manifest()
	if c.Config.Atlassian.Base == "" {
		return nil
	}
	j := &atlassian.Jira{Client: atlassianClient(c.Config, c.Secret, c.Redact, c.opt.ConnectorHTTP)}
	for _, name := range sortedSpaces(m) {
		sp := m.Spaces[name]
		if sp.ServiceQueue == "" || !sp.Watches(ask.OnRequestReceived) || !spaceFile.MatchString(name) {
			continue
		}
		p := filepath.Join(c.requestsDir(), name)
		since := time.Now().UTC()
		if b, err := os.ReadFile(p); err == nil {
			if t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b))); err == nil {
				since = t
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		reqs, newest, err := j.Requests(ctx, sp.ServiceQueue, since)
		if err != nil {
			c.Metrics.Add("sb_requests_failed_total", "Service queue reads that failed (SB-6206).", map[string]string{"space": name}, 1)
			continue
		}
		for _, r := range reqs {
			text := "Request " + r.Key + ": " + r.Summary + "\n\n" + r.Text
			req := ask.Request{Question: ask.RequestQuestion(r.Summary), File: &ask.FileUpload{Name: "request.txt", Data: []byte(text)}}
			if _, err := c.runDraft(ctx, ask.SpaceKey(name), ask.DraftServiceRequest, "Request: "+r.Key+" "+r.Summary, req, false, ""); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(c.requestsDir(), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(newest.Format(time.RFC3339Nano)+"\n"), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// jiraHookSync syncs the tickets connector once, for a verified Jira webhook.
var jiraHookSync = func(c *Cell) {
	_ = ingest.Syncer{Pipeline: c.IngestPipeline("tickets", time.Now), Connector: "tickets", Cursors: c.Cursors, Now: time.Now}.Once(context.Background())
}

// jiraHook mounts POST /hooks/jira (SB-6205): a Jira webhook signed with the account's
// webhook_secret (X-Hub-Signature: sha256=HMAC of the body) starts a sync of the tickets
// connector now, so ticket-filed, incident briefs and reviews don't wait for the next
// minute. The body is never read as anything but a signal; an unsigned or wrongly signed
// delivery is refused.
func (c *Cell) jiraHook(mux *http.ServeMux) {
	if c.Config.Atlassian.Base == "" || len(c.Config.Atlassian.Projects) == 0 {
		return
	}
	mux.HandleFunc("POST /hooks/jira", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		cl := atlassianClient(c.Config, c.Secret, c.Redact, nil)
		cr, err := cl.Credentials(r.Context())
		sig, ok := strings.CutPrefix(r.Header.Get("X-Hub-Signature"), "sha256=")
		if err != nil || cr.Webhook == "" || !ok || !hmac.Equal([]byte(sig), []byte(signBody(cr.Webhook, body))) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		c.Metrics.Add("sb_jira_webhooks_total", "Verified Jira webhooks, each a sync of the tickets connector (SB-6205).", nil, 1)
		c.hooks.Add(1)
		go func() { defer c.hooks.Done(); jiraHookSync(c) }()
		w.WriteHeader(http.StatusNoContent)
	})
}

// signBody is HMAC-SHA256 of body with secret, in hex: Jira's X-Hub-Signature.
func signBody(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// jiraCreate is the link to Jira's create form, prefilled for the first project sb-cell(5)
// atlassian.projects names (SB-6207); nil without Jira. The person files the issue under
// their own sign-in; the cell never does.
func jiraCreate(cfg Config) func(summary, description string) string {
	if cfg.Atlassian.Base == "" || len(cfg.Atlassian.Projects) == 0 {
		return nil
	}
	base, project := strings.TrimSuffix(cfg.Atlassian.Base, "/"), cfg.Atlassian.Projects[0]
	return func(summary, description string) string {
		q := url.Values{"pid": {project}, "issuetype": {"Task"}, "summary": {summary}, "description": {description}}
		return base + "/secure/CreateIssueDetails!init.jspa?" + q.Encode()
	}
}

// WaitHooks waits for the syncs webhooks started to finish (SB-6205).
func (c *Cell) WaitHooks() { c.hooks.Wait() }

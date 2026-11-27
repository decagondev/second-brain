// Package msgraph reaches Microsoft 365 through Microsoft Graph, read-only, with the
// standard library only (ADR-0023): one client with OAuth client credentials against
// Entra, and adapters over it for SharePoint and OneDrive with Purview labels and Teams
// transcripts (SB-5901, SB-5902, SB-5906), Entra groups and managers (SB-5903), Outlook
// calendars (SB-5905) and the shared mailboxes spaces name (SB-5907). Every adapter is
// built against the recorded fake in graphtest; a real tenant run is In review.
package msgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// The defaults: Graph's v1.0 API and Entra's sign-in.
const (
	DefaultBase  = "https://graph.microsoft.com/v1.0"
	DefaultLogin = "https://login.microsoftonline.com"
	Scope        = "https://graph.microsoft.com/.default"
)

// Credentials are vault://tenants/TENANT/msgraph: the Entra tenant and the app's id and
// secret, read at call time.
type Credentials struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Registrar is the log redactor, told each token so no log line can hold it.
type Registrar interface{ Register(secret string) }

// Client is one tenant's Graph client.
type Client struct {
	Base, Login string
	HTTP        *http.Client
	Creds       func(ctx context.Context) (string, error)
	Redact      Registrar
	Now         func() time.Time

	mu    sync.Mutex
	token string
	until time.Time
}

// Error is a Graph answer that wasn't 2xx; its status is a ports.StatusError.
type Error struct {
	Status int
	Code   string
}

func (e *Error) Error() string { return fmt.Sprintf("msgraph: HTTP %d %s", e.Status, e.Code) }

// HTTPStatus implements ports.StatusError.
func (e *Error) HTTPStatus() int { return e.Status }

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) base() string {
	if c.Base == "" {
		return DefaultBase
	}
	return strings.TrimRight(c.Base, "/")
}

func (c *Client) http() *http.Client {
	if c.HTTP == nil {
		return http.DefaultClient
	}
	return c.HTTP
}

// Token is the app's access token, from client credentials, kept until a minute before it
// expires.
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && c.now().Before(c.until) {
		t := c.token
		c.mu.Unlock()
		return t, nil
	}
	c.mu.Unlock()
	raw, err := c.Creds(ctx)
	if err != nil {
		return "", fmt.Errorf("msgraph: credentials: %w", err)
	}
	var cr Credentials
	if err := json.Unmarshal([]byte(raw), &cr); err != nil || cr.TenantID == "" || cr.ClientID == "" || cr.ClientSecret == "" {
		return "", errors.New("msgraph: the credentials aren't {tenant_id, client_id, client_secret}")
	}
	if c.Redact != nil {
		c.Redact.Register(cr.ClientSecret)
	}
	login := c.Login
	if login == "" {
		login = DefaultLogin
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {cr.ClientID}, "client_secret": {cr.ClientSecret}, "scope": {Scope}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(login, "/")+"/"+url.PathEscape(cr.TenantID)+"/oauth2/v2.0/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", fmt.Errorf("msgraph: token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tok) != nil || tok.AccessToken == "" {
		return "", &Error{Status: resp.StatusCode, Code: "token"}
	}
	if c.Redact != nil {
		c.Redact.Register(tok.AccessToken)
	}
	c.mu.Lock()
	c.token, c.until = tok.AccessToken, c.now().Add(time.Duration(tok.ExpiresIn)*time.Second-time.Minute)
	c.mu.Unlock()
	return tok.AccessToken, nil
}

// raw GETs path (relative to the base, or a whole URL of the same host from a nextLink or
// deltaLink) and returns its body, retrying a 429 a few times as the context allows.
func (c *Client) raw(ctx context.Context, path string) ([]byte, error) {
	return c.rawWith(ctx, path, "")
}

// rawWith is raw with a Prefer header ("" for none).
func (c *Client) rawWith(ctx context.Context, path, prefer string) ([]byte, error) {
	u := path
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		u = c.base() + path
	} else if !strings.HasPrefix(u, c.base()+"/") {
		return nil, errors.New("msgraph: a link that isn't Graph's")
	}
	wait := 50 * time.Millisecond
	for attempt := 0; ; attempt++ {
		tok, err := c.Token(ctx)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		if prefer != "" {
			req.Header.Set("Prefer", prefer)
		}
		resp, err := c.http().Do(req)
		if err != nil {
			return nil, fmt.Errorf("msgraph: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode/100 == 2 {
			return body, nil
		}
		var e struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		ge := &Error{Status: resp.StatusCode, Code: e.Error.Code}
		if resp.StatusCode != http.StatusTooManyRequests || attempt >= 3 {
			return nil, ge
		}
		select {
		case <-ctx.Done():
			return nil, ge
		case <-time.After(wait):
		}
		wait *= 2
	}
}

// get GETs path and decodes its JSON into v.
func (c *Client) get(ctx context.Context, path string, v any) error {
	return c.getWith(ctx, path, "", v)
}

func (c *Client) getWith(ctx context.Context, path, prefer string, v any) error {
	b, err := c.rawWith(ctx, path, prefer)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("msgraph: a page that isn't JSON: %w", err)
	}
	return nil
}

// page is one page of a Graph collection.
type page[T any] struct {
	Value     []T    `json:"value"`
	NextLink  string `json:"@odata.nextLink"`
	DeltaLink string `json:"@odata.deltaLink"`
}

// all reads every page of a collection from path, at most limit items (0 for no limit);
// more is true when the limit cut it short. The delta link of the last page, if any, is
// returned too.
func all[T any](ctx context.Context, c *Client, path string, limit int) (items []T, delta string, more bool, err error) {
	return allWith[T](ctx, c, path, "", limit)
}

// allWith is all with a Prefer header on every page.
func allWith[T any](ctx context.Context, c *Client, path, prefer string, limit int) (items []T, delta string, more bool, err error) {
	for path != "" {
		var p page[T]
		if err := c.getWith(ctx, path, prefer, &p); err != nil {
			return nil, "", false, err
		}
		for _, x := range p.Value {
			if limit > 0 && len(items) == limit {
				return items, "", true, nil
			}
			items = append(items, x)
		}
		path, delta = p.NextLink, p.DeltaLink
	}
	return items, delta, false, nil
}

func notFound(err error) bool {
	var ge *Error
	return errors.As(err, &ge) && (ge.Status == http.StatusNotFound || ge.Status == http.StatusForbidden)
}

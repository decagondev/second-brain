// Package atlassian reaches Confluence Cloud and Jira Cloud read-only (ADR-0024; SB-6201
// to SB-6207): one REST client with the standard library, a directory of Atlassian users
// and groups, the docs: confluence and tickets: jira connectors, a service queue reader and
// the setup check. Nothing here writes, but an issue a person files with their own token.
package atlassian

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Credentials are vault://tenants/TENANT/atlassian: a read-only account's email and API
// token, and the webhook secret Jira signs its deliveries with.
type Credentials struct {
	Email    string `json:"email"`
	APIToken string `json:"api_token"`
	Webhook  string `json:"webhook_secret,omitempty"`
}

// Registrar is the log redactor, told each secret so no log line can hold it.
type Registrar interface{ Register(secret string) }

// Client is one Atlassian site, https://acme.atlassian.net.
type Client struct {
	Base   string
	Creds  func(ctx context.Context) (string, error)
	HTTP   *http.Client
	Redact Registrar
}

// Error is the site's refusal: its status, and the Retry-After it sent with a 429.
type Error struct {
	Status int
	Wait   time.Duration
}

func (e *Error) Error() string { return fmt.Sprintf("atlassian: HTTP %d", e.Status) }

// HTTPStatus implements ports.StatusError.
func (e *Error) HTTPStatus() int { return e.Status }

// RetryAfter implements ingest.RetryAfter (SB-6004).
func (e *Error) RetryAfter() time.Duration { return e.Wait }

// Credentials reads the vault's credentials.
func (c *Client) Credentials(ctx context.Context) (Credentials, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return Credentials{}, err
	}
	var cr Credentials
	if json.Unmarshal([]byte(raw), &cr) != nil || cr.Email == "" || cr.APIToken == "" {
		return Credentials{}, errors.New("atlassian: the credentials need email and api_token")
	}
	if c.Redact != nil {
		c.Redact.Register(cr.APIToken)
	}
	return cr, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, v any) error {
	cr, err := c.Credentials(ctx)
	if err != nil {
		return err
	}
	u := path
	if !strings.HasPrefix(u, "http") {
		u = strings.TrimSuffix(c.Base, "/") + path
	} else if !strings.HasPrefix(u, strings.TrimSuffix(c.Base, "/")+"/") {
		return errors.New("atlassian: a link that isn't the site's")
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cr.Email+":"+cr.APIToken)))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("atlassian: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		e := &Error{Status: resp.StatusCode}
		if s, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); err == nil && s > 0 {
			e.Wait = time.Duration(s) * time.Second
		}
		return e
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(v)
}

func (c *Client) get(ctx context.Context, path string, v any) error {
	return c.do(ctx, http.MethodGet, path, nil, v)
}

// notFound says an error is the site's 404.
func notFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

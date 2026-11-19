// Package gdrive is the read-only Google Drive connector, `docs: google-drive`
// (sb-connector(3), SB-203). The live ACL check asks Drive as the asker, through
// domain-wide delegation; anything Drive can't answer cleanly is Unknown (LAW-03).
package gdrive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Tokens mints access tokens; subject "" is the service account itself.
type Tokens interface {
	Token(ctx context.Context, subject string) (string, error)
}

const (
	// DefaultBase is Drive's v3 API.
	DefaultBase = "https://www.googleapis.com/drive/v3"
	// ACLDeadline is the caller's deadline for one ACL batch.
	ACLDeadline = 800 * time.Millisecond
	// ACLConcurrency bounds parallel per-document checks.
	ACLConcurrency = 8
	kind           = "docs"
	fileFields     = "id,name,mimeType,version,modifiedTime,trashed,permissions(type,emailAddress,domain)"
)

// Connector talks to one Workspace's Drive.
type Connector struct {
	Base   string
	HTTP   *http.Client
	Tokens Tokens
}

var _ ports.Connector = (*Connector)(nil)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: google-drive connector is read-only")

func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

type apiError struct {
	status int
	reason string
}

func (e *apiError) Error() string { return fmt.Sprintf("gdrive: HTTP %d %s", e.status, e.reason) }

func (e *apiError) rateLimited() bool {
	return e.status == 429 || (e.status == 403 && (e.reason == "rateLimitExceeded" || e.reason == "userRateLimitExceeded"))
}

// get calls path as subject and decodes JSON into v (or returns the raw body if v is nil).
func (c *Connector) get(ctx context.Context, subject, path string, q url.Values, v any) ([]byte, error) {
	tok, err := c.Tokens.Token(ctx, subject)
	if err != nil {
		return nil, err
	}
	u := c.Base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error struct {
				Errors []struct{ Reason string } `json:"errors"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		reason := ""
		if len(e.Error.Errors) > 0 {
			reason = e.Error.Errors[0].Reason
		}
		return nil, &apiError{resp.StatusCode, reason}
	}
	if v != nil {
		return body, json.Unmarshal(body, v)
	}
	return body, nil
}

// retry runs f, backing off on rate limits while the context allows.
func retry(ctx context.Context, f func() error) error {
	wait := 20 * time.Millisecond
	for {
		err := f()
		var ae *apiError
		if err == nil || !errors.As(err, &ae) || !ae.rateLimited() {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(wait):
		}
		wait *= 2
	}
}

type file struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	Version      string `json:"version"`
	ModifiedTime string `json:"modifiedTime"`
	Trashed      bool   `json:"trashed"`
	Permissions  []struct {
		Type, EmailAddress, Domain string
	} `json:"permissions"`
}

func fileQuery() url.Values { return url.Values{"fields": {fileFields}, "supportsAllDrives": {"true"}} }

// Fetch reads one file as the service account.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	var f file
	if err := retry(ctx, func() error { _, err := c.get(ctx, "", "/files/"+url.PathEscape(ref.ID), fileQuery(), &f); return err }); err != nil {
		return ports.Document{}, err
	}
	if f.Trashed {
		return ports.Document{}, fmt.Errorf("gdrive: %s is trashed", ref.ID)
	}
	v, err := strconv.Atoi(f.Version)
	if err != nil {
		return ports.Document{}, fmt.Errorf("gdrive: %s has no usable version", ref.ID)
	}
	text := ""
	switch {
	case f.MimeType == "application/vnd.google-apps.document":
		err = retry(ctx, func() error {
			b, err := c.get(ctx, "", "/files/"+url.PathEscape(ref.ID)+"/export", url.Values{"mimeType": {"text/plain"}}, nil)
			text = string(b)
			return err
		})
	case strings.HasPrefix(f.MimeType, "text/"):
		err = retry(ctx, func() error {
			b, err := c.get(ctx, "", "/files/"+url.PathEscape(ref.ID), url.Values{"alt": {"media"}}, nil)
			text = string(b)
			return err
		})
	}
	if err != nil {
		return ports.Document{}, err
	}
	return ports.Document{Ref: ref, Version: v, Title: f.Name, Text: text, ACLPrefilter: prefilter(f)}, nil
}

func prefilter(f file) []string {
	var out []string
	for _, p := range f.Permissions {
		switch p.Type {
		case "user":
			out = append(out, strings.ToLower(p.EmailAddress))
		case "group":
			out = append(out, "group:"+strings.ToLower(p.EmailAddress))
		case "domain":
			out = append(out, "domain:"+strings.ToLower(p.Domain))
		case "anyone":
			out = append(out, "anyone")
		}
	}
	return out
}

// ACL asks Drive as the user, at most ACLConcurrency at a time, within ACLDeadline.
func (c *Connector) ACL(ctx context.Context, user ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // zero value is Unknown (LAW-03)
	if user.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	sem := make(chan struct{}, ACLConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, r := range refs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			v := c.verdict(ctx, user.Email, id)
			mu.Lock()
			out[i] = v
			mu.Unlock()
		}(i, r.ID)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil // late is Unknown
	}
	return out, nil
}

func (c *Connector) verdict(ctx context.Context, email, id string) ports.Verdict {
	q := url.Values{"fields": {"id,trashed"}, "supportsAllDrives": {"true"}}
	var f file
	err := retry(ctx, func() error { _, err := c.get(ctx, email, "/files/"+url.PathEscape(id), q, &f); return err })
	var ae *apiError
	switch {
	case err == nil && !f.Trashed:
		return ports.VerdictAllow
	case err == nil:
		return ports.VerdictUnknown // trashed
	case errors.As(err, &ae) && (ae.status == 404 || (ae.status == 403 && !ae.rateLimited())):
		// The user can't see it. That's a Deny only if the file still exists for us.
		var g file
		if err := retry(ctx, func() error { _, err := c.get(ctx, "", "/files/"+url.PathEscape(id), q, &g); return err }); err == nil && !g.Trashed {
			return ports.VerdictDeny
		}
	}
	return ports.VerdictUnknown
}

// Changes reads Drive's change feed. An empty cursor lists every file (for first sync
// and rebuild) and returns the page token taken before the listing.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	if cur == "" {
		return c.everything(ctx)
	}
	var set ports.ChangeSet
	token := string(cur)
	for {
		var page struct {
			Changes []struct {
				FileID  string `json:"fileId"`
				Removed bool   `json:"removed"`
				Time    string `json:"time"`
				File    *file  `json:"file"`
			} `json:"changes"`
			NextPageToken     string `json:"nextPageToken"`
			NewStartPageToken string `json:"newStartPageToken"`
		}
		q := url.Values{"pageToken": {token}, "fields": {"changes(fileId,removed,time,file(" + fileFields + ")),nextPageToken,newStartPageToken"},
			"supportsAllDrives": {"true"}, "includeItemsFromAllDrives": {"true"}}
		if err := retry(ctx, func() error { _, err := c.get(ctx, "", "/changes", q, &page); return err }); err != nil {
			return nil, cur, err
		}
		for _, ch := range page.Changes {
			ref := ports.SourceRef{Kind: kind, ID: ch.FileID}
			if ch.Removed || ch.File == nil || ch.File.Trashed {
				set = append(set, ports.Change{Ref: ref, Deleted: true})
				continue
			}
			set = append(set, ports.Change{Ref: ref, ACLOnly: aclOnly(ch.Time, ch.File.ModifiedTime)})
		}
		if page.NewStartPageToken != "" {
			return set, ports.Cursor(page.NewStartPageToken), nil
		}
		if page.NextPageToken == "" {
			return set, cur, errors.New("gdrive: change page has neither next nor new start token")
		}
		token = page.NextPageToken
	}
}

// aclOnly: a change long after the content's last modification is a permission change.
func aclOnly(changeTime, modified string) bool {
	ct, err1 := time.Parse(time.RFC3339Nano, changeTime)
	mt, err2 := time.Parse(time.RFC3339Nano, modified)
	if err1 != nil || err2 != nil {
		return false // unsure: treat as a content change, which re-ingests everything
	}
	d := ct.Sub(mt)
	return d > time.Second || d < -time.Second
}

func (c *Connector) everything(ctx context.Context) (ports.ChangeSet, ports.Cursor, error) {
	var start struct {
		StartPageToken string `json:"startPageToken"`
	}
	if err := retry(ctx, func() error {
		_, err := c.get(ctx, "", "/changes/startPageToken", url.Values{"supportsAllDrives": {"true"}}, &start)
		return err
	}); err != nil {
		return nil, "", err
	}
	var set ports.ChangeSet
	page := ""
	for {
		var list struct {
			Files []struct {
				ID string `json:"id"`
			} `json:"files"`
			NextPageToken string `json:"nextPageToken"`
		}
		q := url.Values{"q": {"trashed = false"}, "fields": {"files(id),nextPageToken"}, "pageSize": {"1000"},
			"supportsAllDrives": {"true"}, "includeItemsFromAllDrives": {"true"}}
		if page != "" {
			q.Set("pageToken", page)
		}
		if err := retry(ctx, func() error { _, err := c.get(ctx, "", "/files", q, &list); return err }); err != nil {
			return nil, "", err
		}
		for _, f := range list.Files {
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: f.ID}})
		}
		if list.NextPageToken == "" {
			return set, ports.Cursor(start.StartPageToken), nil
		}
		page = list.NextPageToken
	}
}

// Act is not supported: read-only.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}

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
	fileFields     = "id,name,mimeType,version,modifiedTime,trashed,parents,permissions(type,emailAddress,domain),owners(emailAddress)"
)

// Connector talks to one Workspace's Drive.
type Connector struct {
	Base   string
	HTTP   *http.Client
	Tokens Tokens
	// Classification is the tenant's Drive label for sensitivity (SB-1005); nil is none.
	Classification *Classification

	mu      sync.Mutex
	folders map[string]folder // id -> name and parent, cached
}

// Classification names one Drive label field whose choices are sensitivity labels
// (sb-connector(3)): from vault://tenants/TENANT/connectors/docs-classification.
type Classification struct {
	Label   string            `json:"label"`
	Field   string            `json:"field"`
	Choices map[string]string `json:"choices"` // choice id -> public|internal|confidential|restricted
}

type folder struct {
	name, parent string
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

// HTTPStatus is the status Drive answered with, a quota 403 as the 429 it means
// (ports.StatusError).
func (e *apiError) HTTPStatus() int {
	if e.rateLimited() {
		return 429
	}
	return e.status
}

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
	Owners []struct {
		EmailAddress string `json:"emailAddress"`
	} `json:"owners"`
	Parents   []string `json:"parents"`
	LabelInfo struct {
		Labels []struct {
			ID     string `json:"id"`
			Fields map[string]struct {
				Selection []string `json:"selection"`
			} `json:"fields"`
		} `json:"labels"`
	} `json:"labelInfo"`
}

// location is the file's folder path from the top of its drive, then its name. A
// folder that can't be read stops the fetch: a document is never routed by a guess.
func (c *Connector) location(ctx context.Context, f file) (string, error) {
	parts := []string{f.Name}
	seen := map[string]bool{}
	parent := ""
	if len(f.Parents) > 0 {
		parent = f.Parents[0]
	}
	for parent != "" {
		if seen[parent] || len(seen) > 64 {
			return "", fmt.Errorf("gdrive: the folders above %s loop or are too deep", f.ID)
		}
		seen[parent] = true
		fo, err := c.folder(ctx, parent)
		if err != nil {
			return "", err
		}
		if fo.parent == "" && parent == "root" {
			break // the root of My Drive has no name in a path
		}
		parts = append([]string{fo.name}, parts...)
		parent = fo.parent
	}
	return "/" + strings.Join(parts, "/"), nil
}

func (c *Connector) folder(ctx context.Context, id string) (folder, error) {
	c.mu.Lock()
	fo, ok := c.folders[id]
	c.mu.Unlock()
	if ok {
		return fo, nil
	}
	var f file
	q := url.Values{"fields": {"id,name,parents"}, "supportsAllDrives": {"true"}}
	if err := retry(ctx, func() error { _, err := c.get(ctx, "", "/files/"+url.PathEscape(id), q, &f); return err }); err != nil {
		return folder{}, err
	}
	fo = folder{name: f.Name}
	if len(f.Parents) > 0 {
		fo.parent = f.Parents[0]
	}
	c.mu.Lock()
	if c.folders == nil {
		c.folders = map[string]folder{}
	}
	c.folders[id] = fo
	c.mu.Unlock()
	return fo, nil
}

// label maps the file's classification choice to a label; a choice the map doesn't
// name is restricted, and no label is "".
func (c *Connector) label(f file) string {
	if c.Classification == nil {
		return ""
	}
	for _, l := range f.LabelInfo.Labels {
		if l.ID != c.Classification.Label {
			continue
		}
		fd, ok := l.Fields[c.Classification.Field]
		if !ok || len(fd.Selection) == 0 {
			return ""
		}
		if v, ok := c.Classification.Choices[fd.Selection[0]]; ok {
			return v
		}
		return "restricted"
	}
	return ""
}

func fileQuery() url.Values { return url.Values{"fields": {fileFields}, "supportsAllDrives": {"true"}} }

func (c *Connector) fetchQuery() url.Values {
	q := fileQuery()
	if c.Classification != nil {
		q.Set("includeLabels", c.Classification.Label)
		q.Set("fields", fileFields+",labelInfo")
	}
	return q
}

// Fetch reads one file as the service account.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	var f file
	if err := retry(ctx, func() error {
		_, err := c.get(ctx, "", "/files/"+url.PathEscape(ref.ID), c.fetchQuery(), &f)
		return err
	}); err != nil {
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
	loc, err := c.location(ctx, f)
	if err != nil {
		return ports.Document{}, err
	}
	return ports.Document{Ref: ref, Version: v, Title: f.Name, Text: text, ACLPrefilter: prefilter(f), Location: loc, Label: c.label(f), Owner: owner(f)}, nil
}

// owner is the file's owner email (SB-1203); a shared drive's file has none.
func owner(f file) string {
	if len(f.Owners) == 0 {
		return ""
	}
	return strings.ToLower(f.Owners[0].EmailAddress)
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
	// Folder names and parents are cached for one sync only: a folder moved or renamed
	// since emits no change for its files, so a cache kept longer routes them by the old
	// path (SB-1401).
	c.mu.Lock()
	c.folders = nil
	c.mu.Unlock()
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

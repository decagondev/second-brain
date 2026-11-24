// Package github is the read-only code connector (sb-connector(3), code: github; SB-1103):
// READMEs, docs and Markdown files, issues and pull-request descriptions of one GitHub
// organisation, and source code only for repositories a space opts in. Access is asked
// live, per repository, and anything GitHub can't answer is Unknown (LAW-03).
package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// PublicToken is the prefilter entry for a public repository: any organisation member.
const PublicToken = "github:members"

// MaxFile is the largest file indexed.
const MaxFile = 1 << 20

// Registrar learns secret values so logs can redact them.
type Registrar interface{ Register(secret string) }

// Connector reads one organisation.
type Connector struct {
	Base   string // the API; "" is https://api.github.com
	HTTP   *http.Client
	Redact Registrar
	// Creds returns vault://tenants/TENANT/connectors/code: {"token": "…", "org": "…"}.
	Creds func(ctx context.Context) (string, error)
	// SourceRepos are the repositories whose source code a space opts in (a code scope of
	// REPO/**); nil opts in none.
	SourceRepos func() []string
}

var _ ports.Connector = (*Connector)(nil)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: github connector is read-only")

// APIVersion is the connector contract version.
func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

// Act is not supported.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}

type apiError struct{ status int }

func (e *apiError) Error() string { return fmt.Sprintf("github: HTTP %d", e.status) }

// HTTPStatus is the status GitHub answered with (ports.StatusError).
func (e *apiError) HTTPStatus() int { return e.status }

func notFound(err error) bool {
	var a *apiError
	return errors.As(err, &a) && a.status == http.StatusNotFound
}

type creds struct{ Token, Org string }

func (c *Connector) creds(ctx context.Context) (creds, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return creds{}, fmt.Errorf("github: credentials: %w", err)
	}
	var k creds
	if err := json.Unmarshal([]byte(raw), &k); err != nil || k.Token == "" || k.Org == "" {
		return creds{}, errors.New("github: credentials must be {\"token\", \"org\"}")
	}
	if c.Redact != nil {
		c.Redact.Register(k.Token)
	}
	return k, nil
}

func (c *Connector) base() string {
	if c.Base == "" {
		return "https://api.github.com"
	}
	return c.Base
}

func (c *Connector) call(ctx context.Context, k creds, method, p string, q url.Values, body any, v any) error {
	u := c.base() + p
	if len(q) > 0 {
		u += "?" + q.Encode()
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
	req.Header.Set("Authorization", "Bearer "+k.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("github: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode/100 != 2 {
		return &apiError{resp.StatusCode}
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(v)
}

func (c *Connector) get(ctx context.Context, k creds, p string, q url.Values, v any) error {
	return c.call(ctx, k, http.MethodGet, p, q, nil, v)
}

// ref is a parsed document id.
type ref struct {
	repo, kind, path string // kind: blob, issues or pulls
	n                int
}

func parse(id string) (ref, error) {
	repo, rest, ok := strings.Cut(id, "/")
	if !ok || repo == "" {
		return ref{}, fmt.Errorf("github: bad document id %q", id)
	}
	kind, tail, ok := strings.Cut(rest, "/")
	if !ok || tail == "" {
		return ref{}, fmt.Errorf("github: bad document id %q", id)
	}
	switch kind {
	case "blob":
		return ref{repo: repo, kind: kind, path: tail}, nil
	case "issues", "pulls":
		n, err := strconv.Atoi(tail)
		if err != nil || n <= 0 {
			return ref{}, fmt.Errorf("github: bad document id %q", id)
		}
		return ref{repo: repo, kind: kind, n: n}, nil
	}
	return ref{}, fmt.Errorf("github: bad document id %q", id)
}

func (r ref) id() string {
	if r.kind == "blob" {
		return r.repo + "/blob/" + r.path
	}
	return r.repo + "/" + r.kind + "/" + strconv.Itoa(r.n)
}

// isDoc says whether a file is indexed: READMEs, docs/, Markdown, and with source opted
// in any other file under MaxFile.
func isDoc(p string, size int, source bool) bool {
	base := path.Base(p)
	switch {
	case size > MaxFile:
		return false
	case strings.HasPrefix(strings.ToUpper(base), "README"), strings.HasPrefix(p, "docs/"), strings.Contains(p, "/docs/"),
		strings.HasSuffix(strings.ToLower(p), ".md"):
		return true
	}
	return source
}

func (c *Connector) source(repo string) bool {
	if c.SourceRepos == nil {
		return false
	}
	for _, r := range c.SourceRepos() {
		if r == repo {
			return true
		}
	}
	return false
}

func micros(ts string) (int, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0, err
	}
	return int(t.UnixMicro()), nil
}

type repoInfo struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type issue struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	UpdatedAt   string          `json:"updated_at"`
	PullRequest json.RawMessage `json:"pull_request"`
}

func (i issue) kind() string {
	if len(i.PullRequest) > 0 && string(i.PullRequest) != "null" {
		return "pulls"
	}
	return "issues"
}

// Fetch reads one file, issue or pull request.
func (c *Connector) Fetch(ctx context.Context, sr ports.SourceRef) (ports.Document, error) {
	r, err := parse(sr.ID)
	if err != nil {
		return ports.Document{}, err
	}
	k, err := c.creds(ctx)
	if err != nil {
		return ports.Document{}, err
	}
	repoPath := "/repos/" + url.PathEscape(k.Org) + "/" + url.PathEscape(r.repo)
	var info repoInfo
	if err := c.get(ctx, k, repoPath, nil, &info); err != nil {
		return ports.Document{}, err
	}
	pre, err := c.prefilter(ctx, k, info)
	if err != nil {
		return ports.Document{}, err
	}
	doc := ports.Document{Ref: sr, ACLPrefilter: pre, Location: r.repo + "/" + r.path}
	if r.kind != "blob" {
		var is issue
		if err := c.get(ctx, k, repoPath+"/issues/"+strconv.Itoa(r.n), nil, &is); err != nil {
			return ports.Document{}, err
		}
		if is.kind() != r.kind {
			return ports.Document{}, fmt.Errorf("github: %s is not a %s", sr.ID, r.kind)
		}
		v, err := micros(is.UpdatedAt)
		if err != nil {
			return ports.Document{}, fmt.Errorf("github: bad updated_at on %s", sr.ID)
		}
		doc.Version, doc.Title, doc.Text = v, "#"+strconv.Itoa(is.Number)+" "+is.Title, is.Body
		doc.Location = r.repo + "/" + r.kind + "/" + strconv.Itoa(r.n)
		return doc, nil
	}
	var file struct {
		Type, Encoding, Content string
	}
	if err := c.get(ctx, k, repoPath+"/contents/"+escapePath(r.path), nil, &file); err != nil {
		return ports.Document{}, err
	}
	text, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil || file.Type != "file" {
		return ports.Document{}, fmt.Errorf("github: %s is not a readable file", sr.ID)
	}
	if !isDoc(r.path, len(text), c.source(r.repo)) {
		return ports.Document{}, fmt.Errorf("github: %s is source code, and no space opts %s in", sr.ID, r.repo)
	}
	var commits []struct {
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := c.get(ctx, k, repoPath+"/commits", url.Values{"path": {r.path}, "per_page": {"1"}}, &commits); err != nil {
		return ports.Document{}, err
	}
	if len(commits) == 0 {
		return ports.Document{}, fmt.Errorf("github: no commit touches %s", sr.ID)
	}
	v, err := micros(commits[0].Commit.Committer.Date)
	if err != nil {
		return ports.Document{}, fmt.Errorf("github: bad commit date on %s", sr.ID)
	}
	doc.Version, doc.Title, doc.Text = v, r.path, string(text)
	return doc, nil
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, x := range parts {
		parts[i] = url.PathEscape(x)
	}
	return strings.Join(parts, "/")
}

// prefilter is a stale-able copy of who may read the repository: any member for a
// public one, else the SSO emails of its collaborators with at least read.
func (c *Connector) prefilter(ctx context.Context, k creds, info repoInfo) ([]string, error) {
	if info.Visibility == "public" {
		return []string{PublicToken}, nil
	}
	var collab []struct {
		Login string `json:"login"`
	}
	if err := c.get(ctx, k, "/repos/"+url.PathEscape(k.Org)+"/"+url.PathEscape(info.Name)+"/collaborators",
		url.Values{"permission": {"pull"}, "per_page": {"100"}}, &collab); err != nil {
		return nil, err
	}
	emails, err := c.identities(ctx, k, "")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, cl := range collab {
		if e, ok := emails[cl.Login]; ok {
			out = append(out, strings.ToLower(e))
		}
	}
	sort.Strings(out)
	return out, nil
}

const identityQuery = `query($org: String!, $email: String) { organization(login: $org) { samlIdentityProvider {
  externalIdentities(first: 100, userName: $email) { nodes { samlIdentity { nameId } user { login } } } } } }`

// identities maps login -> SSO email, for one email or for everyone ("").
func (c *Connector) identities(ctx context.Context, k creds, email string) (map[string]string, error) {
	vars := map[string]any{"org": k.Org}
	if email != "" {
		vars["email"] = email
	}
	var out struct {
		Data struct {
			Organization struct {
				SamlIdentityProvider struct {
					ExternalIdentities struct {
						Nodes []struct {
							SamlIdentity struct {
								NameID string `json:"nameId"`
							} `json:"samlIdentity"`
							User struct {
								Login string `json:"login"`
							} `json:"user"`
						} `json:"nodes"`
					} `json:"externalIdentities"`
				} `json:"samlIdentityProvider"`
			} `json:"organization"`
		} `json:"data"`
	}
	if err := c.call(ctx, k, http.MethodPost, "/graphql", nil, map[string]any{"query": identityQuery, "variables": vars}, &out); err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, n := range out.Data.Organization.SamlIdentityProvider.ExternalIdentities.Nodes {
		if n.User.Login != "" {
			m[n.User.Login] = n.SamlIdentity.NameID
		}
	}
	return m, nil
}

// ACL answers per repository, live. The zero verdict is Unknown (LAW-03).
func (c *Connector) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	if u.Email == "" {
		return out, nil
	}
	k, err := c.creds(ctx)
	if err != nil {
		return out, nil //nolint:nilerr // an ACL that can't ask answers Unknown, never an error (LAW-03)
	}
	ids, err := c.identities(ctx, k, u.Email)
	if err != nil {
		return out, nil //nolint:nilerr // an ACL that can't ask answers Unknown, never an error (LAW-03)
	}
	// Only an identity whose SSO name is the asker's email counts, and it must be exactly
	// one: another member's identity never answers for the asker (SB-1401).
	login, matches := "", 0
	for l, nameID := range ids {
		if strings.EqualFold(strings.TrimSpace(nameID), strings.TrimSpace(u.Email)) {
			login, matches = l, matches+1
		}
	}
	switch {
	case matches > 1:
		return out, nil // ambiguous: Unknown
	case matches == 0 && len(ids) > 0:
		return out, nil // the filter returned someone else: Unknown, never their access
	case login == "":
		for i := range out {
			out[i] = ports.VerdictDeny // no SSO identity in the organisation
		}
		return out, nil
	}
	byRepo := map[string][]int{}
	for i, sr := range refs {
		if r, err := parse(sr.ID); err == nil && sr.Kind == "code" {
			byRepo[r.repo] = append(byRepo[r.repo], i)
		}
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for repo, idx := range byRepo {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			v := c.repoVerdict(ctx, k, repo, login)
			mu.Lock()
			for _, i := range idx {
				out[i] = v
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out, nil
}

func (c *Connector) repoVerdict(ctx context.Context, k creds, repo, login string) ports.Verdict {
	repoPath := "/repos/" + url.PathEscape(k.Org) + "/" + url.PathEscape(repo)
	var info repoInfo
	if err := c.get(ctx, k, repoPath, nil, &info); err != nil {
		return ports.VerdictUnknown // gone, or GitHub can't say
	}
	if info.Visibility == "public" {
		err := c.get(ctx, k, "/orgs/"+url.PathEscape(k.Org)+"/members/"+url.PathEscape(login), nil, nil)
		switch {
		case err == nil:
			return ports.VerdictAllow
		case notFound(err):
			return ports.VerdictDeny
		}
		return ports.VerdictUnknown
	}
	var p struct {
		Permission string `json:"permission"`
	}
	if err := c.get(ctx, k, repoPath+"/collaborators/"+url.PathEscape(login)+"/permission", nil, &p); err != nil {
		return ports.VerdictUnknown
	}
	switch p.Permission {
	case "read", "triage", "write", "maintain", "admin":
		return ports.VerdictAllow
	case "none":
		return ports.VerdictDeny
	}
	return ports.VerdictUnknown
}

type cursor struct {
	T string `json:"t"`
	// At are the changes already reported at exactly T, so a change in the cursor's own
	// second is reported once and never lost (SB-1401's follow-up, as the Linear connector).
	At []string `json:"at,omitempty"`
	// Source are the repositories whose source was opted in at the last sync: one opted in
	// since gets its existing files listed (SB-1401's follow-up).
	Source []string `json:"source,omitempty"`
}

// Changes reports what changed strictly after the cursor's time; "" lists everything.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	k, err := c.creds(ctx)
	if err != nil {
		return nil, cur, err
	}
	var from time.Time
	var cr cursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &cr); err != nil {
			return nil, cur, fmt.Errorf("github: bad cursor")
		}
		if from, err = time.Parse(time.RFC3339, cr.T); err != nil {
			return nil, cur, fmt.Errorf("github: bad cursor")
		}
	}
	reported := map[string]bool{}
	for _, k := range cr.At {
		reported[k] = true
	}
	repos, err := pages[repoInfo](ctx, c, k, "/orgs/"+url.PathEscape(k.Org)+"/repos", url.Values{})
	if err != nil {
		return nil, cur, err
	}
	latest := from
	at := append([]string(nil), cr.At...) // what is reported at latest
	seen := func(ts, key string) bool {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return false
		}
		fresh := cur == "" || t.After(from) || (t.Equal(from) && !reported[key])
		switch {
		case t.After(latest):
			latest, at = t, []string{key}
		case t.Equal(latest) && fresh:
			at = append(at, key)
		}
		return fresh
	}
	var set ports.ChangeSet
	add := func(ch ports.Change) { set = append(set, ch) }
	for _, rp := range repos {
		if err := c.repoChanges(ctx, k, rp, cur, from, cr, seen, add); err != nil {
			return nil, cur, err
		}
	}
	var src []string
	for _, rp := range repos {
		if c.source(rp.Name) {
			src = append(src, rp.Name)
		}
	}
	sort.Strings(src)
	sort.Strings(at)
	b, _ := json.Marshal(cursor{T: latest.UTC().Format(time.RFC3339), At: at, Source: src})
	return set, ports.Cursor(b), nil
}

// docs lists a repository's indexed files on its default branch.
func (c *Connector) docs(ctx context.Context, k creds, repo string) ([]string, error) {
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			Size int    `json:"size"`
		} `json:"tree"`
	}
	if err := c.get(ctx, k, "/repos/"+url.PathEscape(k.Org)+"/"+url.PathEscape(repo)+"/git/trees/HEAD", url.Values{"recursive": {"1"}}, &tree); err != nil {
		return nil, err
	}
	var out []string
	for _, t := range tree.Tree {
		if t.Type == "blob" && isDoc(t.Path, t.Size, c.source(repo)) {
			out = append(out, t.Path)
		}
	}
	return out, nil
}

// allDocs is every document id of a repository: its indexed files, issues and pulls.
func (c *Connector) allDocs(ctx context.Context, k creds, repo string) ([]string, error) {
	files, err := c.docs(ctx, k, repo)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range files {
		out = append(out, ref{repo: repo, kind: "blob", path: p}.id())
	}
	var issues []issue
	if err := c.get(ctx, k, "/repos/"+url.PathEscape(k.Org)+"/"+url.PathEscape(repo)+"/issues",
		url.Values{"state": {"all"}, "per_page": {"100"}}, &issues); err != nil {
		return nil, err
	}
	for _, is := range issues {
		out = append(out, ref{repo: repo, kind: is.kind(), n: is.Number}.id())
	}
	return out, nil
}

type commitInfo struct {
	SHA    string `json:"sha"`
	Commit struct {
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

// pagesMax bounds a listing, so a server that ignores page numbers can't loop us forever.
const pagesMax = 100

// pages reads every page of a GitHub listing, 100 at a time (SB-1401: one page lost the
// rest).
func pages[T any](ctx context.Context, c *Connector, k creds, p string, q url.Values) ([]T, error) {
	var all []T
	for page := 1; page <= pagesMax; page++ {
		pq := url.Values{}
		for key, v := range q {
			pq[key] = v
		}
		pq.Set("per_page", "100")
		pq.Set("page", strconv.Itoa(page))
		var batch []T
		if err := c.get(ctx, k, p, pq, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			return all, nil
		}
	}
	return nil, fmt.Errorf("github: %s has more than %d pages", p, pagesMax)
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// repoChanges adds one repository's changes since from: issues and pull requests, commits
// (or, on a first sync, its indexed files), a newly opted-in repository's source, and an
// ACL-only change for everything when its members or visibility changed.
func (c *Connector) repoChanges(ctx context.Context, k creds, rp repoInfo, cur ports.Cursor, from time.Time, cr cursor,
	seen func(ts, key string) bool, add func(ports.Change)) error {
	repoPath := "/repos/" + url.PathEscape(k.Org) + "/" + url.PathEscape(rp.Name)
	// Sorted by update, oldest first, every page: an update on page 2 is never lost
	// behind a cursor that moved past it (SB-1401).
	q := url.Values{"state": {"all"}, "sort": {"updated"}, "direction": {"asc"}}
	if cur != "" {
		q.Set("since", from.Format(time.RFC3339))
	}
	issues, err := pages[issue](ctx, c, k, repoPath+"/issues", q)
	if err != nil {
		return err
	}
	for _, is := range issues {
		if seen(is.UpdatedAt, "issue:"+rp.Name+"/"+strconv.Itoa(is.Number)) {
			add(ports.Change{Ref: ports.SourceRef{Kind: "code", ID: ref{repo: rp.Name, kind: is.kind(), n: is.Number}.id()}})
		}
	}
	cq := url.Values{}
	if cur != "" {
		cq.Set("since", from.Format(time.RFC3339))
	}
	commits, err := pages[commitInfo](ctx, c, k, repoPath+"/commits", cq)
	if err != nil {
		return err
	}
	if cur == "" {
		for _, cm := range commits {
			seen(cm.Commit.Committer.Date, "commit:"+cm.SHA)
		}
		files, err := c.docs(ctx, k, rp.Name)
		if err != nil {
			return err
		}
		for _, p := range files {
			add(ports.Change{Ref: ports.SourceRef{Kind: "code", ID: ref{repo: rp.Name, kind: "blob", path: p}.id()}})
		}
	} else {
		if c.source(rp.Name) && !contains(cr.Source, rp.Name) {
			// Opted in since the last sync: its existing source files, not only new ones.
			files, err := c.docs(ctx, k, rp.Name)
			if err != nil {
				return err
			}
			for _, p := range files {
				add(ports.Change{Ref: ports.SourceRef{Kind: "code", ID: ref{repo: rp.Name, kind: "blob", path: p}.id()}})
			}
		}
		for i := len(commits) - 1; i >= 0; i-- { // oldest first
			cm := commits[i]
			if !seen(cm.Commit.Committer.Date, "commit:"+cm.SHA) {
				continue
			}
			var detail struct {
				Files []struct {
					Filename string `json:"filename"`
					Status   string `json:"status"`
				} `json:"files"`
			}
			if err := c.get(ctx, k, repoPath+"/commits/"+url.PathEscape(cm.SHA), nil, &detail); err != nil {
				return err
			}
			for _, f := range detail.Files {
				if !isDoc(f.Filename, 0, c.source(rp.Name)) {
					continue
				}
				add(ports.Change{Ref: ports.SourceRef{Kind: "code", ID: ref{repo: rp.Name, kind: "blob", path: f.Filename}.id()},
					Deleted: f.Status == "removed"})
			}
		}
	}
	var events []struct {
		Type      string `json:"type"`
		CreatedAt string `json:"created_at"`
	}
	if err := c.get(ctx, k, repoPath+"/events", url.Values{"per_page": {"100"}}, &events); err != nil {
		return err
	}
	permChanged := false
	for _, e := range events {
		if e.Type != "MemberEvent" && e.Type != "PublicEvent" {
			continue
		}
		if seen(e.CreatedAt, "event:"+e.Type+"@"+e.CreatedAt) && cur != "" {
			permChanged = true
		}
	}
	if permChanged {
		all, err := c.allDocs(ctx, k, rp.Name)
		if err != nil {
			return err
		}
		for _, id := range all {
			add(ports.Change{Ref: ports.SourceRef{Kind: "code", ID: id}, ACLOnly: true})
		}
	}
	return nil
}

// Link opens a cited chunk on GitHub (sb-connector(3), Links; SB-3006): a file on the
// repository's default branch at the chunk's lines, an issue or a pull request. It reads
// only the organisation from the credentials; nothing is fetched. "" when it can't.
func (c *Connector) Link(ctx context.Context, ch ports.Chunk) string {
	r, err := parse(ch.SourceID)
	if err != nil {
		return ""
	}
	k, err := c.creds(ctx)
	if err != nil {
		return ""
	}
	web := "https://github.com"
	if b := strings.TrimSuffix(c.base(), "/"); b != "https://api.github.com" {
		web = strings.TrimSuffix(b, "/api/v3")
	}
	base := web + "/" + url.PathEscape(k.Org) + "/" + url.PathEscape(r.repo)
	switch r.kind {
	case "issues":
		return base + "/issues/" + strconv.Itoa(r.n)
	case "pulls":
		return base + "/pull/" + strconv.Itoa(r.n)
	}
	u := base + "/blob/HEAD/" + escapePath(r.path)
	switch from, to := ch.Lines[0], ch.Lines[1]; {
	case from > 0 && to > from:
		u += fmt.Sprintf("#L%d-L%d", from, to)
	case from > 0:
		u += fmt.Sprintf("#L%d", from)
	}
	return u
}

// Package githubtest is a recorded fake of the parts of GitHub's REST and GraphQL APIs the
// code connector uses (SB-1103): repositories, trees, contents, commits, issues and pull
// requests, collaborator permissions, organisation membership, repository events, and the
// organisation's SSO identities. Faults can be injected on the next permission call.
package githubtest

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Repo is a fake repository.
type Repo struct {
	Name       string
	Visibility string            // "public", "private" or "internal"
	Perms      map[string]string // login -> "read", "write", "admin" or "none"
	Files      map[string]*File  // path -> file on the default branch
	Gone       bool
}

// File is one file on the default branch.
type File struct {
	Text    string
	Changed time.Time // the last commit that touched it
}

// Issue is an issue or a pull request.
type Issue struct {
	Repo    string
	Number  int
	Title   string
	Body    string
	PR      bool
	Updated time.Time
}

type commit struct {
	sha     string
	repo    string
	at      time.Time
	files   []string
	removed []string
}

type event struct {
	repo string
	typ  string
	at   time.Time
}

// Fake is one organisation.
type Fake struct {
	// LooseIdentities makes the identity query ignore its email filter (SB-1401).
	LooseIdentities bool
	Server          *httptest.Server
	Org             string
	Token           string

	mu      sync.Mutex
	clock   time.Time
	repos   map[string]*Repo
	issues  map[string]*Issue // "repo#n"
	commits []commit
	events  []event
	members map[string]bool   // login -> org member
	emails  map[string]string // SSO email -> login
	faults  []string
	n       int
}

// New starts a fake over TLS for the organisation acme.
func New() *Fake {
	f := &Fake{Org: "acme", Token: "ghs_fake_token_0123456789", clock: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
		repos: map[string]*Repo{}, issues: map[string]*Issue{}, members: map[string]bool{}, emails: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /orgs/{org}/repos", f.auth(f.listRepos))
	mux.HandleFunc("GET /orgs/{org}/members/{login}", f.auth(f.member))
	mux.HandleFunc("GET /repos/{org}/{repo}", f.auth(f.getRepo))
	mux.HandleFunc("GET /repos/{org}/{repo}/collaborators", f.auth(f.collaborators))
	mux.HandleFunc("GET /repos/{org}/{repo}/git/trees/{ref}", f.auth(f.tree))
	mux.HandleFunc("GET /repos/{org}/{repo}/contents/{path...}", f.auth(f.contents))
	mux.HandleFunc("GET /repos/{org}/{repo}/commits", f.auth(f.listCommits))
	mux.HandleFunc("GET /repos/{org}/{repo}/commits/{sha}", f.auth(f.getCommit))
	mux.HandleFunc("GET /repos/{org}/{repo}/issues", f.auth(f.listIssues))
	mux.HandleFunc("GET /repos/{org}/{repo}/issues/{n}", f.auth(f.getIssue))
	mux.HandleFunc("GET /repos/{org}/{repo}/collaborators/{login}/permission", f.auth(f.permission))
	mux.HandleFunc("GET /repos/{org}/{repo}/events", f.auth(f.listEvents))
	mux.HandleFunc("POST /graphql", f.auth(f.graphql))
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the API's base URL.
func (f *Fake) Base() string { return f.Server.URL }

// CredsJSON is what the vault holds for the connector.
func (f *Fake) CredsJSON() string { return `{"token":"` + f.Token + `","org":"` + f.Org + `"}` }

func (f *Fake) tick() time.Time { f.clock = f.clock.Add(time.Second); return f.clock }

// AddMember adds an organisation member with an SSO email.
func (f *Fake) AddMember(login, email string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.members[login] = true
	f.emails[strings.ToLower(email)] = login
}

// AddRepo creates a repository.
func (f *Fake) AddRepo(name, visibility string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[name] = &Repo{Name: name, Visibility: visibility, Perms: map[string]string{}, Files: map[string]*File{}}
}

// Grant sets a login's permission on a repository: a MemberEvent, as in GitHub.
func (f *Fake) Grant(repo, login, perm string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[repo].Perms[login] = perm
	f.events = append(f.events, event{repo: repo, typ: "MemberEvent", at: f.tick()})
}

// Publish makes a repository public: a PublicEvent.
func (f *Fake) Publish(repo string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[repo].Visibility = "public"
	f.events = append(f.events, event{repo: repo, typ: "PublicEvent", at: f.tick()})
}

// Commit writes (text non-nil) or removes files in one commit on the default branch.
func (f *Fake) Commit(repo string, files map[string]*string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	at := f.tick()
	c := commit{sha: "c" + strconv.Itoa(f.n), repo: repo, at: at}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if t := files[p]; t != nil {
			f.repos[repo].Files[p] = &File{Text: *t, Changed: at}
			c.files = append(c.files, p)
		} else {
			delete(f.repos[repo].Files, p)
			c.removed = append(c.removed, p)
		}
	}
	f.commits = append(f.commits, c)
}

// PutIssue creates or edits an issue or pull request.
func (f *Fake) PutIssue(repo string, n int, title, body string, pr bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issues[repo+"#"+strconv.Itoa(n)] = &Issue{Repo: repo, Number: n, Title: title, Body: body, PR: pr, Updated: f.tick()}
}

// Remove deletes a repository.
func (f *Fake) Remove(repo string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[repo].Gone = true
}

// FailNext makes the next permission call fail: "500", "timeout" or "404".
func (f *Fake) FailNext(kind string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.faults = append(f.faults, kind)
}

func (f *Fake) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+f.Token {
			http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
			return
		}
		if o := r.PathValue("org"); o != "" && o != f.Org {
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
			return
		}
		next(w, r)
	}
}

func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func notFound(w http.ResponseWriter) { http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound) }

func (f *Fake) repo(w http.ResponseWriter, r *http.Request) (*Repo, bool) {
	x, ok := f.repos[r.PathValue("repo")]
	if !ok || x.Gone {
		notFound(w)
		return nil, false
	}
	return x, true
}

func (f *Fake) listRepos(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]any
	names := make([]string, 0, len(f.repos))
	for n := range f.repos {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		x := f.repos[n]
		if !x.Gone {
			out = append(out, map[string]any{"name": x.Name, "visibility": x.Visibility, "private": x.Visibility != "public", "default_branch": "main"})
		}
	}
	write(w, paged(r, out))
}

// paged serves one page of a listing as GitHub does, by page and per_page (SB-1401); a
// request without page gets the whole listing.
func paged(r *http.Request, out []map[string]any) []map[string]any {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 || per < 1 {
		return out
	}
	start := (page - 1) * per
	if start >= len(out) {
		return []map[string]any{}
	}
	end := start + per
	if end > len(out) {
		end = len(out)
	}
	return out[start:end]
}

func (f *Fake) getRepo(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	write(w, map[string]any{"name": x.Name, "visibility": x.Visibility, "private": x.Visibility != "public", "default_branch": "main"})
}

// collaborators lists logins with at least read (permission=pull), as GitHub does.
func (f *Fake) collaborators(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	var logins []string
	for l, p := range x.Perms {
		if p != "none" && p != "" {
			logins = append(logins, l)
		}
	}
	sort.Strings(logins)
	out := []map[string]any{}
	for _, l := range logins {
		out = append(out, map[string]any{"login": l})
	}
	write(w, out)
}

func (f *Fake) member(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.members[r.PathValue("login")] {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	notFound(w)
}

func (f *Fake) tree(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	paths := make([]string, 0, len(x.Files))
	for p := range x.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var tree []map[string]any
	for _, p := range paths {
		tree = append(tree, map[string]any{"path": p, "type": "blob", "size": len(x.Files[p].Text)})
	}
	write(w, map[string]any{"tree": tree, "truncated": false})
}

func (f *Fake) contents(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	file, ok := x.Files[r.PathValue("path")]
	if !ok {
		notFound(w)
		return
	}
	write(w, map[string]any{"type": "file", "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(file.Text))})
}

func since(r *http.Request) time.Time {
	t, _ := time.Parse(time.RFC3339, r.URL.Query().Get("since"))
	return t
}

func (f *Fake) listCommits(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	path, from := r.URL.Query().Get("path"), since(r)
	var out []map[string]any
	for i := len(f.commits) - 1; i >= 0; i-- { // newest first, as GitHub lists them
		c := f.commits[i]
		if c.repo != x.Name || c.at.Before(from) {
			continue
		}
		if path != "" && !containsStr(c.files, path) && !containsStr(c.removed, path) {
			continue
		}
		out = append(out, map[string]any{"sha": c.sha, "commit": map[string]any{"committer": map[string]any{"date": c.at.Format(time.RFC3339)}}})
		if r.URL.Query().Get("per_page") == "1" {
			break
		}
	}
	write(w, paged(r, out))
}

func (f *Fake) getCommit(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.commits {
		if c.sha == r.PathValue("sha") && c.repo == r.PathValue("repo") {
			var files []map[string]any
			for _, p := range c.files {
				files = append(files, map[string]any{"filename": p, "status": "modified"})
			}
			for _, p := range c.removed {
				files = append(files, map[string]any{"filename": p, "status": "removed"})
			}
			write(w, map[string]any{"sha": c.sha, "files": files})
			return
		}
	}
	notFound(w)
}

func (f *Fake) issueJSON(x *Issue) map[string]any {
	m := map[string]any{"number": x.Number, "title": x.Title, "body": x.Body, "updated_at": x.Updated.Format(time.RFC3339)}
	if x.PR {
		m["pull_request"] = map[string]any{"url": "x"}
	}
	return m
}

func (f *Fake) listIssues(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	from := since(r)
	var list []*Issue
	for _, is := range f.issues {
		if is.Repo == x.Name && !is.Updated.Before(from) {
			list = append(list, is)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Number < list[j].Number })
	if r.URL.Query().Get("sort") == "updated" {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Updated.Before(list[j].Updated) })
	}
	out := []map[string]any{}
	for _, is := range list {
		out = append(out, f.issueJSON(is))
	}
	write(w, paged(r, out))
}

func (f *Fake) getIssue(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.repo(w, r); !ok {
		return
	}
	is, ok := f.issues[r.PathValue("repo")+"#"+r.PathValue("n")]
	if !ok {
		notFound(w)
		return
	}
	write(w, f.issueJSON(is))
}

func (f *Fake) permission(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	var fault string
	if len(f.faults) > 0 {
		fault, f.faults = f.faults[0], f.faults[1:]
	}
	f.mu.Unlock()
	switch fault {
	case "500":
		http.Error(w, `{"message":"Server Error"}`, http.StatusInternalServerError)
		return
	case "timeout":
		time.Sleep(time.Second)
		http.Error(w, `{"message":"Server Error"}`, http.StatusBadGateway)
		return
	case "404":
		notFound(w)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	login := r.PathValue("login")
	if !f.members[login] {
		notFound(w) // not a user of this organisation
		return
	}
	p := x.Perms[login]
	if p == "" {
		p = "none"
	}
	write(w, map[string]any{"permission": p})
}

func (f *Fake) listEvents(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.repo(w, r)
	if !ok {
		return
	}
	var out []map[string]any
	for i := len(f.events) - 1; i >= 0; i-- {
		e := f.events[i]
		if e.repo == x.Name {
			out = append(out, map[string]any{"type": e.typ, "created_at": e.at.Format(time.RFC3339)})
		}
	}
	write(w, out)
}

// graphql answers only the organisation's SSO identity lookup by email.
func (f *Fake) graphql(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || !strings.Contains(req.Query, "externalIdentities") {
		http.Error(w, `{"message":"unsupported query"}`, http.StatusBadRequest)
		return
	}
	email, _ := req.Variables["email"].(string)
	f.mu.Lock()
	if f.LooseIdentities {
		email = "" // a filter the server ignored: every identity comes back
	}
	nodes := []any{}
	if email != "" {
		if login := f.emails[strings.ToLower(email)]; login != "" {
			nodes = append(nodes, map[string]any{"samlIdentity": map[string]any{"nameId": email}, "user": map[string]any{"login": login}})
		}
	} else { // every identity, for the prefilter
		es := make([]string, 0, len(f.emails))
		for e := range f.emails {
			es = append(es, e)
		}
		sort.Strings(es)
		for _, e := range es {
			nodes = append(nodes, map[string]any{"samlIdentity": map[string]any{"nameId": e}, "user": map[string]any{"login": f.emails[e]}})
		}
	}
	f.mu.Unlock()
	write(w, map[string]any{"data": map[string]any{"organization": map[string]any{"samlIdentityProvider": map[string]any{
		"externalIdentities": map[string]any{"nodes": nodes}}}}})
}

func containsStr(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

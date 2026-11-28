// Package atlassiantest is a recorded fake of the Atlassian Cloud endpoints the atlassian
// adapters call (ADR-0024): Confluence spaces, pages, labels, restrictions, space
// permissions and the permission check; Jira search, issues, the permission search,
// myself and mypermissions; and users and groups. Its shapes are the v2 and v3 APIs'; a run
// against a real site stays In review until sb prove passes on one.
package atlassiantest

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Creds are credentials the fake accepts.
const Creds = `{"email":"brain@acme.example","api_token":"atl-token","webhook_secret":"hook-secret"}`

// Page is one Confluence page or blog post.
type Page struct {
	ID, Space, Title, Body string
	Blog                   bool
	Version                int
	Labels                 []string
	Users, Groups          []string // read restrictions: emails and group names
	Deleted                bool
}

// Issue is one Jira issue.
type Issue struct {
	Key, Project, Summary, Description, Status, Updated, Created string
	Done                                                         bool
	Labels, Comments                                             []string
	// Security is the issue's security level's members (emails); nil for none.
	Security []string
	Deleted  bool
}

// Fake is the recorded Atlassian site.
type Fake struct {
	Server *httptest.Server

	mu sync.Mutex
	// Spaces are space key -> anonymous read; Pages by id.
	Spaces map[string]bool
	Pages  map[string]*Page
	// Users are email -> account id; Groups are group name -> member emails.
	Users  map[string]string
	Groups map[string][]string
	// Projects are Jira project keys -> the emails that may browse them (nil: every user).
	Projects map[string][]string
	Issues   map[string]*Issue
	// Writes are the brain account's Jira permissions it shouldn't have (CREATE_ISSUES …).
	Writes []string
	// Calls are the paths asked for, in order.
	Calls []string
}

// New starts an empty site.
func New() *Fake {
	f := &Fake{Spaces: map[string]bool{}, Pages: map[string]*Page{}, Users: map[string]string{"brain@acme.example": "acc-brain"}, Groups: map[string][]string{},
		Projects: map[string][]string{}, Issues: map[string]*Issue{}}
	f.Server = httptest.NewServer(f)
	return f
}

// Base is the site's address.
func (f *Fake) Base() string { return f.Server.URL }

// User adds a person with an account.
func (f *Fake) User(email string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Users[email] = "acc-" + strings.Split(email, "@")[0]
}

func (f *Fake) emailOf(acct string) string {
	for e, a := range f.Users {
		if a == acct {
			return e
		}
	}
	return ""
}

func (f *Fake) inGroup(email string, groups []string) bool {
	for _, g := range groups {
		for _, m := range f.Groups[g] {
			if m == email {
				return true
			}
		}
	}
	return false
}

// CanRead is whether email may read page id: an account, and a restriction naming them or a
// group of theirs, or none.
func (f *Fake) CanRead(email, id string) bool {
	p := f.Pages[id]
	if p == nil || p.Deleted || f.Users[email] == "" {
		return false
	}
	if len(p.Users) == 0 && len(p.Groups) == 0 {
		return true
	}
	for _, u := range p.Users {
		if u == email {
			return true
		}
	}
	return f.inGroup(email, p.Groups)
}

// CanBrowse is whether email may browse issue key.
func (f *Fake) CanBrowse(email, key string) bool {
	is := f.Issues[key]
	if is == nil || is.Deleted || f.Users[email] == "" {
		return false
	}
	if who, ok := f.Projects[is.Project]; ok && who != nil && !contains(who, email) {
		return false
	}
	return is.Security == nil || contains(is.Security, email)
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

type route struct {
	method string
	re     *regexp.Regexp
	h      func(f *Fake, w http.ResponseWriter, r *http.Request, m []string)
}

var routes = []route{
	{"GET", regexp.MustCompile(`^/wiki/api/v2/spaces$`), (*Fake).spaces},
	{"GET", regexp.MustCompile(`^/wiki/api/v2/spaces/([^/]+)$`), (*Fake).spaceByID},
	{"GET", regexp.MustCompile(`^/wiki/api/v2/spaces/([^/]+)/(pages|blogposts)$`), (*Fake).pages},
	{"GET", regexp.MustCompile(`^/wiki/api/v2/spaces/([^/]+)/permissions$`), (*Fake).spacePerms},
	{"GET", regexp.MustCompile(`^/wiki/api/v2/(?:pages|blogposts)/([^/]+)$`), (*Fake).page},
	{"GET", regexp.MustCompile(`^/wiki/api/v2/(?:pages|blogposts)/([^/]+)/labels$`), (*Fake).labels},
	{"GET", regexp.MustCompile(`^/wiki/rest/api/content/([^/]+)/restriction/byOperation/read$`), (*Fake).restrictions},
	{"POST", regexp.MustCompile(`^/wiki/rest/api/content/([^/]+)/permission/check$`), (*Fake).check},
	{"GET", regexp.MustCompile(`^/rest/api/3/user/search$`), (*Fake).userSearch},
	{"GET", regexp.MustCompile(`^/rest/api/3/user/groups$`), (*Fake).userGroups},
	{"GET", regexp.MustCompile(`^/rest/api/3/group/member$`), (*Fake).groupMembers},
	{"GET", regexp.MustCompile(`^/rest/api/3/myself$`), (*Fake).myself},
	{"GET", regexp.MustCompile(`^/rest/api/3/mypermissions$`), (*Fake).mypermissions},
	{"GET", regexp.MustCompile(`^/rest/api/3/project/([^/]+)$`), (*Fake).project},
	{"GET", regexp.MustCompile(`^/rest/api/3/search/jql$`), (*Fake).search},
	{"GET", regexp.MustCompile(`^/rest/api/3/issue/([^/]+)$`), (*Fake).issue},
	{"GET", regexp.MustCompile(`^/rest/api/3/user/permission/search$`), (*Fake).permSearch},
}

func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, r.URL.Path)
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("brain@acme.example:atl-token"))
	if r.Header.Get("Authorization") != want {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	for _, rt := range routes {
		if m := rt.re.FindStringSubmatch(r.URL.Path); m != nil && rt.method == r.Method {
			rt.h(f, w, r, m[1:])
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func spaceID(key string) string { return "sp-" + strings.ToLower(key) }

func (f *Fake) spaceKey(id string) string {
	for k := range f.Spaces {
		if spaceID(k) == id {
			return k
		}
	}
	return ""
}

func (f *Fake) spaces(w http.ResponseWriter, r *http.Request, _ []string) {
	var out []map[string]string
	for _, k := range strings.Split(r.URL.Query().Get("keys"), ",") {
		if _, ok := f.Spaces[k]; ok {
			out = append(out, map[string]string{"id": spaceID(k), "key": k})
		}
	}
	writeJSON(w, map[string]any{"results": out})
}

func (f *Fake) spaceByID(w http.ResponseWriter, _ *http.Request, m []string) {
	k := f.spaceKey(m[0])
	if k == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"id": m[0], "key": k})
}

func (f *Fake) sortedPages() []*Page {
	var ps []*Page
	for _, p := range f.Pages {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })
	return ps
}

func (f *Fake) pages(w http.ResponseWriter, _ *http.Request, m []string) {
	key := f.spaceKey(m[0])
	var out []map[string]any
	for _, p := range f.sortedPages() {
		if p.Space == key && !p.Deleted && p.Blog == (m[1] == "blogposts") {
			out = append(out, map[string]any{"id": p.ID, "title": p.Title, "spaceId": m[0], "version": map[string]int{"number": p.Version}})
		}
	}
	writeJSON(w, map[string]any{"results": out, "_links": map[string]string{}})
}

func (f *Fake) spacePerms(w http.ResponseWriter, _ *http.Request, m []string) {
	out := []map[string]any{{"principal": map[string]string{"type": "group", "id": "g-users"}, "operation": map[string]string{"key": "read", "targetType": "space"}}}
	if f.Spaces[f.spaceKey(m[0])] {
		out = append(out, map[string]any{"principal": map[string]string{"type": "anonymous"}, "operation": map[string]string{"key": "read", "targetType": "space"}})
	}
	writeJSON(w, map[string]any{"results": out})
}

func (f *Fake) page(w http.ResponseWriter, _ *http.Request, m []string) {
	p := f.Pages[m[0]]
	if p == nil || p.Deleted {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"id": p.ID, "title": p.Title, "spaceId": spaceID(p.Space), "version": map[string]int{"number": p.Version},
		"body": map[string]any{"storage": map[string]string{"value": p.Body, "representation": "storage"}}})
}

func (f *Fake) labels(w http.ResponseWriter, _ *http.Request, m []string) {
	p := f.Pages[m[0]]
	if p == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var out []map[string]string
	for _, l := range p.Labels {
		out = append(out, map[string]string{"name": l, "prefix": "global"})
	}
	writeJSON(w, map[string]any{"results": out})
}

func (f *Fake) restrictions(w http.ResponseWriter, _ *http.Request, m []string) {
	p := f.Pages[m[0]]
	if p == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var us []map[string]any
	for _, e := range p.Users {
		us = append(us, map[string]any{"accountId": f.Users[e], "emailAddress": e, "active": true, "accountType": "atlassian"})
	}
	var gs []map[string]string
	for _, g := range p.Groups {
		gs = append(gs, map[string]string{"name": g, "id": "gid-" + g})
	}
	writeJSON(w, map[string]any{"operation": "read", "restrictions": map[string]any{"user": map[string]any{"results": us}, "group": map[string]any{"results": gs}}})
}

func (f *Fake) check(w http.ResponseWriter, r *http.Request, m []string) {
	var in struct {
		Subject   struct{ Type, Identifier string } `json:"subject"`
		Operation string                            `json:"operation"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Operation != "read" || in.Subject.Type != "user" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if p := f.Pages[m[0]]; p == nil || p.Deleted {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]bool{"hasPermission": f.CanRead(f.emailOf(in.Subject.Identifier), m[0])})
}

func (f *Fake) userJSON(email string) map[string]any {
	return map[string]any{"accountId": f.Users[email], "emailAddress": email, "active": true, "accountType": "atlassian"}
}

func (f *Fake) userSearch(w http.ResponseWriter, r *http.Request, _ []string) {
	q := r.URL.Query().Get("query")
	out := []map[string]any{}
	if _, ok := f.Users[q]; ok {
		out = append(out, f.userJSON(q))
	}
	writeJSON(w, out)
}

func (f *Fake) userGroups(w http.ResponseWriter, r *http.Request, _ []string) {
	email := f.emailOf(r.URL.Query().Get("accountId"))
	out := []map[string]string{}
	var names []string
	for g := range f.Groups {
		if f.inGroup(email, []string{g}) {
			names = append(names, g)
		}
	}
	sort.Strings(names)
	for _, g := range names {
		out = append(out, map[string]string{"name": g, "groupId": "gid-" + g})
	}
	writeJSON(w, out)
}

func (f *Fake) groupMembers(w http.ResponseWriter, r *http.Request, _ []string) {
	g, ok := f.Groups[r.URL.Query().Get("groupname")]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	start, _ := strconv.Atoi(r.URL.Query().Get("startAt"))
	var vals []map[string]any
	for _, e := range g[min(start, len(g)):min(start+50, len(g))] {
		vals = append(vals, f.userJSON(e))
	}
	writeJSON(w, map[string]any{"values": vals, "isLast": start+50 >= len(g)})
}

func (f *Fake) myself(w http.ResponseWriter, _ *http.Request, _ []string) {
	writeJSON(w, f.userJSON("brain@acme.example"))
}

func (f *Fake) mypermissions(w http.ResponseWriter, r *http.Request, _ []string) {
	out := map[string]any{}
	for _, p := range strings.Split(r.URL.Query().Get("permissions"), ",") {
		out[p] = map[string]any{"key": p, "havePermission": contains(f.Writes, p) || p == "BROWSE_PROJECTS"}
	}
	writeJSON(w, map[string]any{"permissions": out})
}

func (f *Fake) project(w http.ResponseWriter, _ *http.Request, m []string) {
	if _, ok := f.Projects[m[0]]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"key": m[0], "id": "pid-" + m[0]})
}

var projectsIn = regexp.MustCompile(`project in \(([^)]*)\)`)

func (f *Fake) search(w http.ResponseWriter, r *http.Request, _ []string) {
	jql := r.URL.Query().Get("jql")
	want := map[string]bool{}
	if m := projectsIn.FindStringSubmatch(jql); m != nil {
		for _, p := range strings.Split(m[1], ",") {
			want[strings.Trim(strings.TrimSpace(p), `"`)] = true
		}
	}
	var keys []string
	for k, is := range f.Issues {
		if want[is.Project] && !is.Deleted {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []map[string]any
	for _, k := range keys {
		out = append(out, map[string]any{"key": k, "fields": map[string]string{"updated": f.Issues[k].Updated, "created": f.Issues[k].Created}})
	}
	writeJSON(w, map[string]any{"issues": out, "isLast": true})
}

func adf(text string) map[string]any {
	return map[string]any{"type": "doc", "version": 1, "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]string{"type": "text", "text": text}}}}}
}

func (f *Fake) issue(w http.ResponseWriter, _ *http.Request, m []string) {
	is := f.Issues[m[0]]
	if is == nil || is.Deleted {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var cs []map[string]any
	for _, c := range is.Comments {
		cs = append(cs, map[string]any{"body": adf(c)})
	}
	cat := "indeterminate"
	if is.Done {
		cat = "done"
	}
	fields := map[string]any{"summary": is.Summary, "description": adf(is.Description), "labels": is.Labels, "updated": is.Updated, "created": is.Created,
		"project": map[string]string{"key": is.Project}, "status": map[string]any{"name": is.Status, "statusCategory": map[string]string{"key": cat}},
		"comment": map[string]any{"comments": cs}}
	if is.Security != nil {
		fields["security"] = map[string]string{"id": "sec-1", "name": "Restricted"}
	}
	writeJSON(w, map[string]any{"key": is.Key, "fields": fields})
}

func (f *Fake) permSearch(w http.ResponseWriter, r *http.Request, _ []string) {
	key, q := r.URL.Query().Get("issueKey"), r.URL.Query().Get("query")
	out := []map[string]any{}
	var emails []string
	for e := range f.Users {
		emails = append(emails, e)
	}
	sort.Strings(emails)
	for _, e := range emails {
		if (q == "" || q == e) && f.CanBrowse(e, key) {
			out = append(out, f.userJSON(e))
		}
	}
	writeJSON(w, out)
}

// Package lineartest is a recorded fake of the Linear GraphQL operations the connector
// sends, dispatched by operationName, with a clock that only moves forward.
package lineartest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Fake is one Linear workspace.
type Fake struct {
	Server *httptest.Server
	Key    string

	mu      sync.Mutex
	clock   time.Time
	users   map[string]*User
	teams   map[string]*Team
	issues  map[string]*Issue
	members []membership
	faults  []string
	limited int
	// Comments made through commentCreate, by comment id.
	Comments map[string]Comment
}

// Comment is a comment the connector created (SB-410).
type Comment struct {
	ID, IssueID, Body string
}

// User is a workspace member.
type User struct {
	ID, Email     string
	Active, Guest bool
}

// Team owns issues; private teams are visible to members only.
type Team struct {
	ID, Key   string
	Private   bool
	Members   map[string]bool
	UpdatedAt time.Time
}

type membership struct {
	team string
	at   time.Time
}

// Issue is a ticket.
type Issue struct {
	ID, Identifier, Title, Description, Team string
	Comments                                 []string
	UpdatedAt                                time.Time
	Archived, Trashed                        bool
}

// New starts a fake over TLS.
func New() *Fake {
	f := &Fake{Key: "lin_api_fakekey0123456789", clock: time.Now().UTC(), users: map[string]*User{},
		teams: map[string]*Team{}, issues: map[string]*Issue{}, Comments: map[string]Comment{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /graphql", f.serve)
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Endpoint is the GraphQL URL.
func (f *Fake) Endpoint() string { return f.Server.URL + "/graphql" }

func (f *Fake) tick() time.Time { f.clock = f.clock.Add(time.Second); return f.clock }

// AddUser adds a member.
func (f *Fake) AddUser(id, email string, guest bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[id] = &User{ID: id, Email: email, Active: true, Guest: guest}
}

// Deactivate deactivates a user.
func (f *Fake) Deactivate(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[id].Active = false
}

// AddTeam adds a team with members.
func (f *Fake) AddTeam(id, key string, private bool, members ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ms := map[string]bool{}
	for _, m := range members {
		ms[m] = true
	}
	f.teams[id] = &Team{ID: id, Key: key, Private: private, Members: ms, UpdatedAt: f.tick()}
}

// Join adds a member to a team (a permission change).
func (f *Fake) Join(team, user string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.teams[team].Members[user] = true
	f.members = append(f.members, membership{team, f.tick()})
}

// Leave removes a member from a team.
func (f *Fake) Leave(team, user string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.teams[team].Members, user)
	f.members = append(f.members, membership{team, f.tick()})
}

// SetPrivate changes a team's privacy.
func (f *Fake) SetPrivate(team string, private bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.teams[team].Private, f.teams[team].UpdatedAt = private, f.tick()
}

// Put creates or edits an issue.
func (f *Fake) Put(id, team, title, desc string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	is, ok := f.issues[id]
	if !ok {
		n := 0
		for _, x := range f.issues {
			if x.Team == team {
				n++
			}
		}
		is = &Issue{ID: id, Team: team, Identifier: f.teams[team].Key + "-" + strconv.Itoa(n+1)}
		f.issues[id] = is
	}
	is.Title, is.Description, is.UpdatedAt = title, desc, f.tick()
}

// Archive archives an issue (a delete for the connector).
func (f *Fake) Archive(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issues[id].Archived, f.issues[id].UpdatedAt = true, f.tick()
}

// FailNext queues faults: "500"/"timeout" on UserByEmail, "404" on IssueByID.
func (f *Fake) FailNext(kinds ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.faults = append(f.faults, kinds...)
}

// RateLimitNext makes the next n requests answer 429.
func (f *Fake) RateLimitNext(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.limited += n
}

func (f *Fake) take(kinds ...string) string {
	if len(f.faults) == 0 {
		return ""
	}
	for _, k := range kinds {
		if f.faults[0] == k {
			f.faults = f.faults[1:]
			return k
		}
	}
	return ""
}

func ts(t time.Time) string { return t.Format(time.RFC3339Nano) }

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if r.Header.Get("Authorization") != f.Key {
		w.WriteHeader(401)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		return
	}
	f.mu.Lock()
	if f.limited > 0 {
		f.limited--
		f.mu.Unlock()
		w.WriteHeader(429)
		return
	}
	var fault string
	switch req.OperationName {
	case "UserByEmail":
		fault = f.take("500", "timeout")
	case "IssueByID":
		fault = f.take("404")
	case "CommentCreate":
		fault = f.take("500", "timeout", "lost", "noid", "noop")
	}
	f.mu.Unlock()
	switch fault {
	case "500":
		w.WriteHeader(500)
		return
	case "timeout":
		time.Sleep(time.Second)
		w.WriteHeader(503)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	str := func(k string) string { s, _ := req.Variables[k].(string); return s }
	var data any
	switch req.OperationName {
	case "UserByEmail":
		var nodes []any
		for _, u := range f.users {
			if u.Email == str("email") {
				nodes = append(nodes, map[string]any{"id": u.ID, "active": u.Active, "guest": u.Guest})
			}
		}
		data = map[string]any{"users": map[string]any{"nodes": nodes}}
	case "CommentByID":
		c, ok := f.Comments[str("id")]
		if !ok {
			data = map[string]any{"comment": nil}
			break
		}
		data = map[string]any{"comment": map[string]any{"id": c.ID}}
	case "CommentCreate":
		in, _ := req.Variables["input"].(map[string]any)
		id, _ := in["id"].(string)
		issue, _ := in["issueId"].(string)
		body, _ := in["body"].(string)
		var target *Issue
		for _, is := range f.issues {
			if is.ID == issue || is.Identifier == issue {
				target = is
			}
		}
		if target == nil || id == "" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": []any{map[string]any{"message": "Entity not found", "extensions": map[string]any{"code": "INVALID_INPUT"}}}})
			return
		}
		if _, dup := f.Comments[id]; dup {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": []any{map[string]any{"message": "Entity already exists", "extensions": map[string]any{"code": "INVALID_INPUT"}}}})
			return
		}
		if fault == "noop" { // success, no id, and nothing was made
			data = map[string]any{"commentCreate": map[string]any{"success": true, "comment": nil}}
			break
		}
		f.Comments[id] = Comment{ID: id, IssueID: target.ID, Body: body}
		target.Comments = append(target.Comments, body)
		target.UpdatedAt = f.tick()
		if fault == "lost" { // the comment was made, but the answer never arrives
			w.WriteHeader(504)
			return
		}
		if fault == "noid" { // made, but the answer carries no id
			data = map[string]any{"commentCreate": map[string]any{"success": true, "comment": nil}}
			break
		}
		data = map[string]any{"commentCreate": map[string]any{"success": true, "comment": map[string]any{"id": id}}}
	case "IssueByID":
		is, ok := f.issues[str("id")]
		if !ok || fault == "404" {
			data = map[string]any{"issue": nil}
			break
		}
		data = map[string]any{"issue": f.issueJSON(is)}
	case "IssuesSince":
		since, _ := time.Parse(time.RFC3339Nano, str("since"))
		var list []*Issue
		for _, is := range f.issues {
			if !is.UpdatedAt.Before(since) { // gte
				list = append(list, is)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt.Before(list[j].UpdatedAt) })
		from, _ := strconv.Atoi(str("after"))
		if from > len(list) {
			from = len(list)
		}
		to := from + 100
		if to > len(list) {
			to = len(list)
		}
		var nodes []any
		for _, is := range list[from:to] {
			nodes = append(nodes, map[string]any{"id": is.ID, "updatedAt": ts(is.UpdatedAt), "archivedAt": archived(is), "trashed": is.Trashed})
		}
		data = map[string]any{"issues": map[string]any{"nodes": nodes,
			"pageInfo": map[string]any{"hasNextPage": to < len(list), "endCursor": strconv.Itoa(to)}}}
	case "TeamsChangedSince":
		since, _ := time.Parse(time.RFC3339Nano, str("since"))
		var teams, mems []any
		for _, t := range f.teams {
			if t.UpdatedAt.After(since) {
				teams = append(teams, map[string]any{"id": t.ID, "updatedAt": ts(t.UpdatedAt)})
			}
		}
		for _, m := range f.members {
			if m.at.After(since) {
				mems = append(mems, map[string]any{"updatedAt": ts(m.at), "team": map[string]any{"id": m.team}})
			}
		}
		data = map[string]any{"teams": map[string]any{"nodes": teams}, "teamMemberships": map[string]any{"nodes": mems}}
	case "TeamIssues":
		var nodes []any
		var ids []string
		for _, is := range f.issues {
			if is.Team == str("id") {
				ids = append(ids, is.ID)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			nodes = append(nodes, map[string]any{"id": id})
		}
		data = map[string]any{"team": map[string]any{"issues": map[string]any{"nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false}}}}
	default:
		w.WriteHeader(400)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func archived(is *Issue) any {
	if is.Archived {
		return ts(is.UpdatedAt)
	}
	return nil
}

func (f *Fake) issueJSON(is *Issue) map[string]any {
	t := f.teams[is.Team]
	var members []any
	var ids []string
	for id := range t.Members {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		members = append(members, map[string]any{"id": id, "email": f.users[id].Email})
	}
	var comments []any
	for _, c := range is.Comments {
		comments = append(comments, map[string]any{"body": c})
	}
	return map[string]any{"id": is.ID, "identifier": is.Identifier, "title": is.Title, "description": is.Description,
		"updatedAt": ts(is.UpdatedAt), "archivedAt": archived(is), "trashed": is.Trashed,
		"team":     map[string]any{"id": t.ID, "key": t.Key, "private": t.Private, "members": map[string]any{"nodes": members}},
		"comments": map[string]any{"nodes": comments}}
}

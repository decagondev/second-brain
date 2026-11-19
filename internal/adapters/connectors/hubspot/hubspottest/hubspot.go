// Package hubspottest is a recorded fake of the HubSpot CRM v3 endpoints the connector
// calls (objects, search, owners, settings users), with a clock that only moves forward.
package hubspottest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Fake is one HubSpot portal.
type Fake struct {
	Server *httptest.Server
	Token  string

	mu      sync.Mutex
	clock   time.Time
	users   map[string]*User // by email
	records map[string]*Record
	faults  []string
	limited int
}

// User is a HubSpot user and its owner record.
type User struct {
	ID, OwnerID, Email, Role string
	SuperAdmin, Deactivated  bool
	Teams                    []string
}

// Record is a company or a deal.
type Record struct {
	Type, ID, Name, Stage, Amount, Description   string
	Owner, Team                                  string // owner id, team id
	Created, Modified, OwnerAssigned, ArchivedAt time.Time
	Archived                                     bool
}

// New starts a fake over TLS.
func New() *Fake {
	f := &Fake{Token: "pat-na1-fake0123456789", clock: time.Now().UTC().Truncate(time.Millisecond),
		users: map[string]*User{}, records: map[string]*Record{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", f.serve)
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the API base URL.
func (f *Fake) Base() string { return f.Server.URL }

func (f *Fake) tick() time.Time { f.clock = f.clock.Add(time.Second); return f.clock }

// AddUser adds a user (and its owner record) with a role and teams.
func (f *Fake) AddUser(id, email, role string, superAdmin bool, teams ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[email] = &User{ID: id, OwnerID: "o" + id, Email: email, Role: role, SuperAdmin: superAdmin, Teams: teams}
}

// Deactivate archives a user's owner record.
func (f *Fake) Deactivate(email string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[email].Deactivated = true
}

// SetRole changes a user's role and super admin flag.
func (f *Fake) SetRole(email, role string, superAdmin bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[email].Role, f.users[email].SuperAdmin = role, superAdmin
}

// SetTeams changes a user's teams.
func (f *Fake) SetTeams(email string, teams ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[email].Teams = append([]string(nil), teams...)
}

// Owner is a user's owner id.
func (f *Fake) Owner(email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.users[email].OwnerID
}

// Put creates or edits a record ("companies" or "deals"); key is "TYPE/ID".
func (f *Fake) Put(key, name, desc, owner, team string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	typ, id, _ := strings.Cut(key, "/")
	r, ok := f.records[key]
	now := f.tick()
	if !ok {
		r = &Record{Type: typ, ID: id, Owner: owner, Team: team, Created: now, OwnerAssigned: now}
		f.records[key] = r
	}
	r.Name, r.Description, r.Modified = name, desc, now
}

// Reassign changes a record's owner and team (a permission-only change).
func (f *Fake) Reassign(key, owner, team string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.records[key]
	now := f.tick()
	r.Owner, r.Team, r.Modified, r.OwnerAssigned = owner, team, now, now
}

// Archive archives a record (a delete for the connector).
func (f *Fake) Archive(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.records[key]
	r.Archived, r.ArchivedAt = true, f.tick()
}

// FailNext queues faults: "500"/"timeout" on the user lookup, "404" on a record read.
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

func iso(t time.Time) string { return t.Format("2006-01-02T15:04:05.000Z") }

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func notFound(w http.ResponseWriter) {
	reply(w, 404, map[string]any{"status": "error", "category": "OBJECT_NOT_FOUND"})
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+f.Token {
		reply(w, 401, map[string]any{"status": "error", "category": "INVALID_AUTHENTICATION"})
		return
	}
	path := r.URL.Path
	f.mu.Lock()
	if f.limited > 0 {
		f.limited--
		f.mu.Unlock()
		reply(w, 429, map[string]any{"status": "error", "category": "RATE_LIMITS"})
		return
	}
	var fault string
	switch {
	case strings.HasPrefix(path, "/settings/v3/users/"):
		fault = f.take("500", "timeout")
	case r.Method == http.MethodGet && strings.Count(path, "/") == 5 && strings.HasPrefix(path, "/crm/v3/objects/"):
		fault = f.take("404")
	}
	f.mu.Unlock()
	switch fault {
	case "500":
		reply(w, 500, map[string]any{"status": "error"})
		return
	case "timeout":
		time.Sleep(time.Second)
		reply(w, 503, map[string]any{"status": "error"})
		return
	case "404":
		notFound(w)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	q := r.URL.Query()
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case len(parts) == 4 && parts[0] == "settings" && parts[2] == "users" && q.Get("idProperty") == "EMAIL":
		u, ok := f.users[parts[3]]
		if !ok {
			notFound(w)
			return
		}
		reply(w, 200, map[string]any{"id": u.ID, "email": u.Email, "roleId": u.Role, "superAdmin": u.SuperAdmin})
	case len(parts) == 3 && parts[0] == "crm" && parts[2] == "owners":
		var res []any
		u, ok := f.users[q.Get("email")]
		archived := q.Get("archived") == "true"
		if ok && u.Deactivated == archived {
			var teams []any
			for _, t := range u.Teams {
				teams = append(teams, map[string]any{"id": t})
			}
			res = append(res, map[string]any{"id": u.OwnerID, "email": u.Email, "userId": u.ID, "archived": u.Deactivated, "teams": teams})
		}
		reply(w, 200, map[string]any{"results": res})
	case len(parts) == 4 && parts[0] == "crm" && parts[2] == "owners":
		for _, u := range f.users {
			if u.OwnerID == parts[3] {
				reply(w, 200, map[string]any{"id": u.OwnerID, "email": u.Email, "userId": u.ID, "archived": u.Deactivated})
				return
			}
		}
		notFound(w)
	case len(parts) == 5 && parts[0] == "crm" && parts[2] == "objects" && r.Method == http.MethodGet:
		rec, ok := f.records[parts[3]+"/"+parts[4]]
		if !ok || rec.Archived {
			notFound(w)
			return
		}
		reply(w, 200, f.recordJSON(rec))
	case len(parts) == 5 && parts[0] == "crm" && parts[2] == "objects" && parts[4] == "search" && r.Method == http.MethodPost:
		f.search(w, r, parts[3])
	case len(parts) == 4 && parts[0] == "crm" && parts[2] == "objects" && r.Method == http.MethodGet && q.Get("archived") == "true":
		var list []*Record
		for _, rec := range f.records {
			if rec.Type == parts[3] && rec.Archived {
				list = append(list, rec)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
		page(w, q.Get("after"), len(list), func(i int) any {
			return map[string]any{"id": list[i].ID, "archived": true, "archivedAt": iso(list[i].ArchivedAt)}
		})
	default:
		notFound(w)
	}
}

func page(w http.ResponseWriter, after string, n int, item func(int) any) {
	from, _ := strconv.Atoi(after)
	if from > n {
		from = n
	}
	to := from + 100
	if to > n {
		to = n
	}
	res := []any{}
	for i := from; i < to; i++ {
		res = append(res, item(i))
	}
	out := map[string]any{"results": res}
	if to < n {
		out["paging"] = map[string]any{"next": map[string]any{"after": strconv.Itoa(to)}}
	}
	reply(w, 200, out)
}

func (f *Fake) search(w http.ResponseWriter, r *http.Request, typ string) {
	var body struct {
		FilterGroups []struct {
			Filters []struct {
				PropertyName, Operator, Value string
			} `json:"filters"`
		} `json:"filterGroups"`
		After string `json:"after"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.FilterGroups) != 1 || len(body.FilterGroups[0].Filters) != 1 {
		reply(w, 400, map[string]any{"status": "error", "category": "VALIDATION_ERROR"})
		return
	}
	flt := body.FilterGroups[0].Filters[0]
	ms, err := strconv.ParseInt(flt.Value, 10, 64)
	if flt.PropertyName != "hs_lastmodifieddate" || flt.Operator != "GTE" || err != nil {
		reply(w, 400, map[string]any{"status": "error", "category": "VALIDATION_ERROR"})
		return
	}
	since := time.UnixMilli(ms).UTC()
	var list []*Record
	for _, rec := range f.records {
		if rec.Type == typ && !rec.Archived && !rec.Modified.Before(since) {
			list = append(list, rec)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if !list[i].Modified.Equal(list[j].Modified) {
			return list[i].Modified.Before(list[j].Modified)
		}
		return list[i].ID < list[j].ID
	})
	page(w, body.After, len(list), func(i int) any {
		return map[string]any{"id": list[i].ID, "properties": map[string]any{
			"createdate": iso(list[i].Created), "hs_lastmodifieddate": iso(list[i].Modified), "hubspot_owner_assigneddate": iso(list[i].OwnerAssigned)}}
	})
}

func (f *Fake) recordJSON(r *Record) map[string]any {
	props := map[string]any{"hs_object_id": r.ID, "hs_lastmodifieddate": iso(r.Modified),
		"hubspot_owner_id": r.Owner, "hubspot_team_id": r.Team, "description": r.Description}
	if r.Type == "deals" {
		props["dealname"], props["dealstage"], props["amount"] = r.Name, r.Stage, r.Amount
	} else {
		props["name"] = r.Name
	}
	return map[string]any{"id": r.ID, "properties": props, "archived": false, "updatedAt": iso(r.Modified)}
}

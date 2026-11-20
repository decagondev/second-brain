// Package scim is a cell's SCIM 2.0 endpoint (sb-identity(7), SB-107): the only way users
// and groups enter the directory. Deactivating a user revokes their sessions before the
// response is sent (LAW-12).
package scim

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

const (
	userSchema  = "urn:ietf:params:scim:schemas:core:2.0:User"
	groupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
	errSchema   = "urn:ietf:params:scim:api:messages:2.0:Error"
)

// Endpoint serves /scim/v2 for one tenant.
type Endpoint struct {
	Tenant    string
	Directory ports.DirectoryStore
	Sessions  ports.Sessions
	// Token returns the SCIM bearer token from vault://tenants/TENANT/scim at call time.
	Token func(ctx context.Context) (string, error)
}

// Handler routes the SCIM requests in sb-identity(7).
func (e *Endpoint) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /scim/v2/Users", e.createUser)
	m.HandleFunc("GET /scim/v2/Users/{id}", e.getUser)
	m.HandleFunc("PUT /scim/v2/Users/{id}", e.replaceUser)
	m.HandleFunc("PATCH /scim/v2/Users/{id}", e.patchUser)
	m.HandleFunc("DELETE /scim/v2/Users/{id}", e.deleteUser)
	m.HandleFunc("POST /scim/v2/Groups", e.createGroup)
	m.HandleFunc("PUT /scim/v2/Groups/{id}", e.replaceGroup)
	m.HandleFunc("PATCH /scim/v2/Groups/{id}", e.patchGroup)
	m.HandleFunc("DELETE /scim/v2/Groups/{id}", e.deleteGroup)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !e.authorised(r) {
			fail(w, http.StatusUnauthorized, "invalid SCIM token")
			return
		}
		m.ServeHTTP(w, r)
	})
}

func (e *Endpoint) authorised(r *http.Request) bool {
	want, err := e.Token(r.Context())
	got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return err == nil && ok && want != "" && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

type userDoc struct {
	Schemas    []string `json:"schemas"`
	ID         string   `json:"id,omitempty"`
	ExternalID string   `json:"externalId,omitempty"`
	UserName   string   `json:"userName"`
	Active     *bool    `json:"active,omitempty"`
	Emails     []struct {
		Value string `json:"value"`
	} `json:"emails,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
	Enterprise *enterprise       `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
}

// enterprise is the SCIM enterprise extension's part the cell keeps: the manager (SB-1204).
type enterprise struct {
	Manager *struct {
		Value string `json:"value"`
	} `json:"manager,omitempty"`
}

// managerPath is the PATCH path of the manager attribute.
const managerPath = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager"

func toDoc(u ports.User) userDoc {
	a := u.Active
	d := userDoc{Schemas: []string{userSchema}, ID: u.ID, ExternalID: u.ExternalID, UserName: u.UserName, Active: &a,
		Meta: map[string]string{"resourceType": "User"}}
	for _, m := range u.Emails {
		d.Emails = append(d.Emails, struct {
			Value string `json:"value"`
		}{m})
	}
	if u.Manager != "" {
		d.Enterprise = &enterprise{Manager: &struct {
			Value string `json:"value"`
		}{u.Manager}}
	}
	return d
}

func fromDoc(d userDoc, u ports.User) ports.User {
	u.UserName, u.ExternalID = d.UserName, d.ExternalID
	u.Active = d.Active == nil || *d.Active // SCIM default is active
	u.Emails = nil
	for _, m := range d.Emails {
		u.Emails = append(u.Emails, m.Value)
	}
	u.Manager = ""
	if d.Enterprise != nil && d.Enterprise.Manager != nil {
		u.Manager = d.Enterprise.Manager.Value
	}
	return u
}

func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("scim: no randomness: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func read(r *http.Request, v any) bool {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v) == nil
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, detail string) {
	reply(w, status, map[string]any{"schemas": []string{errSchema}, "status": strconv.Itoa(status), "detail": detail})
}

func (e *Endpoint) createUser(w http.ResponseWriter, r *http.Request) {
	var d userDoc
	if !read(r, &d) || d.UserName == "" {
		fail(w, http.StatusBadRequest, "userName is required")
		return
	}
	ctx := r.Context()
	if _, dup, err := e.Directory.UserByName(ctx, e.Tenant, d.UserName); err != nil {
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
		return
	} else if dup {
		fail(w, http.StatusConflict, "userName already exists")
		return
	}
	u := fromDoc(d, ports.User{ID: newID(), Tenant: e.Tenant})
	if err := e.Directory.PutUser(ctx, u); err != nil {
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
		return
	}
	reply(w, http.StatusCreated, toDoc(u))
}

func (e *Endpoint) load(w http.ResponseWriter, r *http.Request) (ports.User, bool) {
	u, found, err := e.Directory.User(r.Context(), e.Tenant, r.PathValue("id"))
	switch {
	case err != nil:
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
	case !found:
		fail(w, http.StatusNotFound, "no such user")
	}
	return u, err == nil && found
}

func (e *Endpoint) getUser(w http.ResponseWriter, r *http.Request) {
	if u, ok := e.load(w, r); ok {
		reply(w, http.StatusOK, toDoc(u))
	}
}

// save writes u and, if it went from active to inactive, revokes its sessions first
// in the response path (sb-identity(7)): a failed revocation is a failed request.
func (e *Endpoint) save(w http.ResponseWriter, r *http.Request, before, u ports.User, status int) {
	ctx := r.Context()
	if err := e.Directory.PutUser(ctx, u); err != nil {
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
		return
	}
	if before.Active && !u.Active {
		if _, err := e.Sessions.RevokeUser(ctx, e.Tenant, u.ID); err != nil {
			fail(w, http.StatusServiceUnavailable, "session revocation failed; retry")
			return
		}
	}
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	reply(w, status, toDoc(u))
}

func (e *Endpoint) replaceUser(w http.ResponseWriter, r *http.Request) {
	before, ok := e.load(w, r)
	if !ok {
		return
	}
	var d userDoc
	if !read(r, &d) || d.UserName == "" {
		fail(w, http.StatusBadRequest, "userName is required")
		return
	}
	e.save(w, r, before, fromDoc(d, before), http.StatusOK)
}

type patchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}

// boolValue accepts true/false and the "True"/"False" strings some IdPs send.
func boolValue(raw json.RawMessage) (bool, bool) {
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if v, err := strconv.ParseBool(s); err == nil {
			return v, true
		}
	}
	return false, false
}

func (e *Endpoint) patchUser(w http.ResponseWriter, r *http.Request) {
	before, ok := e.load(w, r)
	if !ok {
		return
	}
	var body struct{ Operations []patchOp }
	if !read(r, &body) || len(body.Operations) == 0 {
		fail(w, http.StatusBadRequest, "Operations are required")
		return
	}
	u := before
	for _, op := range body.Operations {
		if !strings.EqualFold(op.Op, "replace") && !strings.EqualFold(op.Op, "add") {
			fail(w, http.StatusBadRequest, "only replace and add are supported on users")
			return
		}
		fields := map[string]json.RawMessage{}
		if op.Path != "" {
			fields[op.Path] = op.Value
		} else if json.Unmarshal(op.Value, &fields) != nil {
			fail(w, http.StatusBadRequest, "value must be an object when path is empty")
			return
		}
		for path, v := range fields {
			switch path {
			case "active":
				b, ok := boolValue(v)
				if !ok {
					fail(w, http.StatusBadRequest, "active must be a boolean")
					return
				}
				u.Active = b
			case "userName":
				_ = json.Unmarshal(v, &u.UserName)
			case "externalId":
				_ = json.Unmarshal(v, &u.ExternalID)
			case managerPath:
				var m struct {
					Value string `json:"value"`
				}
				if json.Unmarshal(v, &m) != nil {
					fail(w, http.StatusBadRequest, "manager must be {\"value\": \"<user id>\"}")
					return
				}
				u.Manager = m.Value
			default:
				fail(w, http.StatusBadRequest, "unsupported path "+path)
				return
			}
		}
	}
	e.save(w, r, before, u, http.StatusOK)
}

func (e *Endpoint) deleteUser(w http.ResponseWriter, r *http.Request) {
	before, ok := e.load(w, r)
	if !ok {
		return
	}
	u := before
	u.Active = false // kept for the audit trail
	e.save(w, r, before, u, http.StatusNoContent)
}

type groupDoc struct {
	Schemas     []string `json:"schemas"`
	ID          string   `json:"id,omitempty"`
	DisplayName string   `json:"displayName"`
	Members     []struct {
		Value string `json:"value"`
	} `json:"members,omitempty"`
}

func members(d groupDoc) []string {
	var out []string
	for _, m := range d.Members {
		out = append(out, m.Value)
	}
	return out
}

func (e *Endpoint) createGroup(w http.ResponseWriter, r *http.Request) {
	var d groupDoc
	if !read(r, &d) || d.DisplayName == "" {
		fail(w, http.StatusBadRequest, "displayName is required")
		return
	}
	g := ports.Group{ID: newID(), Tenant: e.Tenant, Name: d.DisplayName, Members: members(d)}
	if err := e.Directory.PutGroup(r.Context(), g); err != nil {
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
		return
	}
	d.Schemas, d.ID = []string{groupSchema}, g.ID
	reply(w, http.StatusCreated, d)
}

func (e *Endpoint) loadGroup(w http.ResponseWriter, r *http.Request) (ports.Group, bool) {
	g, found, err := e.Directory.Group(r.Context(), e.Tenant, r.PathValue("id"))
	switch {
	case err != nil:
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
	case !found:
		fail(w, http.StatusNotFound, "no such group")
	}
	return g, err == nil && found
}

func (e *Endpoint) replaceGroup(w http.ResponseWriter, r *http.Request) {
	g, ok := e.loadGroup(w, r)
	if !ok {
		return
	}
	var d groupDoc
	if !read(r, &d) || d.DisplayName == "" {
		fail(w, http.StatusBadRequest, "displayName is required")
		return
	}
	g.Name, g.Members = d.DisplayName, members(d)
	e.putGroup(w, r, g)
}

func (e *Endpoint) patchGroup(w http.ResponseWriter, r *http.Request) {
	g, ok := e.loadGroup(w, r)
	if !ok {
		return
	}
	var body struct{ Operations []patchOp }
	if !read(r, &body) || len(body.Operations) == 0 {
		fail(w, http.StatusBadRequest, "Operations are required")
		return
	}
	for _, op := range body.Operations {
		var vals []struct {
			Value string `json:"value"`
		}
		path := op.Path
		// remove with a filter: members[value eq "ID"]
		if strings.HasPrefix(path, `members[value eq "`) && strings.HasSuffix(path, `"]`) {
			id := strings.TrimSuffix(strings.TrimPrefix(path, `members[value eq "`), `"]`)
			vals = append(vals, struct {
				Value string `json:"value"`
			}{id})
			path = "members"
		} else if len(op.Value) > 0 && json.Unmarshal(op.Value, &vals) != nil {
			fail(w, http.StatusBadRequest, "members value must be a list")
			return
		}
		if path != "members" {
			fail(w, http.StatusBadRequest, "only members can be patched on groups")
			return
		}
		switch strings.ToLower(op.Op) {
		case "add":
			for _, v := range vals {
				if !contains(g.Members, v.Value) {
					g.Members = append(g.Members, v.Value)
				}
			}
		case "remove":
			var kept []string
			for _, m := range g.Members {
				drop := false
				for _, v := range vals {
					drop = drop || v.Value == m
				}
				if !drop {
					kept = append(kept, m)
				}
			}
			g.Members = kept
		default:
			fail(w, http.StatusBadRequest, "only add and remove are supported on members")
			return
		}
	}
	e.putGroup(w, r, g)
}

func (e *Endpoint) putGroup(w http.ResponseWriter, r *http.Request, g ports.Group) {
	if err := e.Directory.PutGroup(r.Context(), g); err != nil {
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
		return
	}
	d := groupDoc{Schemas: []string{groupSchema}, ID: g.ID, DisplayName: g.Name}
	for _, m := range g.Members {
		d.Members = append(d.Members, struct {
			Value string `json:"value"`
		}{m})
	}
	reply(w, http.StatusOK, d)
}

func (e *Endpoint) deleteGroup(w http.ResponseWriter, r *http.Request) {
	if _, ok := e.loadGroup(w, r); !ok {
		return
	}
	if err := e.Directory.DeleteGroup(r.Context(), e.Tenant, r.PathValue("id")); err != nil {
		fail(w, http.StatusServiceUnavailable, "directory unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

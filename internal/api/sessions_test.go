package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/session"
)

type activeDir struct{}

func (activeDir) Active(context.Context, string, string) (bool, bool, error) { return true, true, nil }

// SB-1906: signing out everywhere ends every session of the caller, and only theirs; a
// revoked session is refused at its next request.
func TestSignOutEverywhere(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	store := sessmem.New()
	for _, s := range []ports.Session{
		{ID: "laptop", Tenant: "acme-logistics", User: "usr_ann", SignedIn: now, Validated: now},
		{ID: "phone", Tenant: "acme-logistics", User: "usr_ann", SignedIn: now, Validated: now},
		{ID: "bob", Tenant: "acme-logistics", User: "usr_bob", SignedIn: now, Validated: now},
	} {
		_ = store.Put(ctx, s)
	}
	rows := auditmem.New()
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: rows, Now: time.Now}
	cell := auth.WithIssuer(&auth.Cell{Tenant: "acme-logistics", Sessions: store, Directory: activeDir{}, Now: time.Now,
		GroupsOf: func(context.Context, string, string) ([]string, error) { return nil, nil }})
	mux := http.NewServeMux()
	Sessions{Revoker: session.Revoker{Tenant: "acme-logistics", Store: store, Audit: log}}.Register(mux)
	mux.HandleFunc("GET /v1/me", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	h := cell.Middleware(mux)
	call := func(method, path, sid string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{Name: auth.Cookie, Value: sid})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if w := call("GET", "/v1/me", "phone"); w.Code != 204 {
		t.Fatalf("before: %d", w.Code)
	}
	w := call("POST", "/v1/sessions/revoke", "laptop")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"revoked":2`) || !strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("revoke: %d %s %q", w.Code, w.Body, w.Header().Get("Set-Cookie"))
	}
	for _, sid := range []string{"laptop", "phone"} {
		if w := call("GET", "/v1/me", sid); w.Code != 401 {
			t.Fatalf("%s after revoke: %d", sid, w.Code)
		}
	}
	if w := call("GET", "/v1/me", "bob"); w.Code != 204 {
		t.Fatalf("someone else's session ended: %d", w.Code)
	}
	all, _ := rows.Range(ctx, 1, 10)
	if len(all) != 2 || all[0].State != "revoke_requested" || all[1].State != "revoked" || all[0].Actor != "usr_ann" || all[0].Subject != "sessions:usr_ann" {
		t.Fatalf("audit %+v", all)
	}
}

// SB-1906: an operator revokes a named person's sessions, policy-checked and audited first.
func TestAdminRevokesSessions(t *testing.T) {
	mux, rows, store := adminServerWithSessions(t)
	if w := adminCall(t, mux, []string{"grp_support"}, "POST", "/v1/admin/sessions/revoke", `{"user":"usr_ann","reason":"lost laptop"}`); w.Code != 403 {
		t.Fatalf("without the role: %d", w.Code)
	}
	op := []string{"sb-operators"}
	for _, bad := range []string{`{"user":"usr_ann"}`, `{"user":"ann","reason":"r"}`, `x`} {
		if w := adminCall(t, mux, op, "POST", "/v1/admin/sessions/revoke", bad); w.Code != 400 {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}
	w := adminCall(t, mux, op, "POST", "/v1/admin/sessions/revoke", `{"user":"usr_ann","reason":"lost laptop"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"revoked":1`) {
		t.Fatalf("revoke: %d %s", w.Code, w.Body)
	}
	if _, found, _ := store.Get(context.Background(), "laptop"); found {
		t.Fatal("the session is still there")
	}
	all, _ := rows.Range(context.Background(), 1, 100)
	var states []string
	for _, r := range all {
		if r.Subject == "sessions:usr_ann" && r.Actor == "usr_olga" {
			states = append(states, r.State)
		}
	}
	if strings.Join(states, ",") != "revoke_requested,revoked" {
		t.Fatalf("audit %v", states)
	}
}

func adminServerWithSessions(t *testing.T) (*http.ServeMux, *auditmem.Store, *sessmem.Store) {
	t.Helper()
	h, rows := adminHandler(t)
	store := sessmem.New()
	now := time.Now()
	_ = store.Put(context.Background(), ports.Session{ID: "laptop", Tenant: "acme-logistics", User: "usr_ann", SignedIn: now, Validated: now})
	h.Sessions = session.Revoker{Tenant: "acme-logistics", Store: store, Audit: h.Audit}
	mux := http.NewServeMux()
	h.Register(mux)
	return mux, rows, store
}

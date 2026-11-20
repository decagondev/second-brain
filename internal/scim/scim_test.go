package scim

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/core/ports"
)

type fixture struct {
	h        http.Handler
	dir      *dirmem.Dir
	sessions *sessmem.Store
}

func setup() fixture {
	f := fixture{dir: dirmem.New(), sessions: sessmem.New()}
	f.h = (&Endpoint{Tenant: "acme-logistics", Directory: f.dir, Sessions: f.sessions,
		Token: func(context.Context) (string, error) { return "tok", nil }}).Handler()
	return f
}

func (f fixture) do(t *testing.T, method, path, body string) (int, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	var v map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	return w.Code, v
}

func (f fixture) user(t *testing.T, name string) string {
	t.Helper()
	code, v := f.do(t, "POST", "/scim/v2/Users", `{"userName":"`+name+`","emails":[{"value":"`+name+`@acme.example"}]}`)
	if code != 201 {
		t.Fatalf("create %s: %d %v", name, code, v)
	}
	return v["id"].(string)
}

func (f fixture) session(t *testing.T, user string) {
	t.Helper()
	now := time.Now()
	_ = f.sessions.Put(context.Background(), ports.Session{ID: "s-" + user, Tenant: "acme-logistics", User: user, SignedIn: now, Validated: now})
}

func (f fixture) hasSession(user string) bool {
	_, found, _ := f.sessions.Get(context.Background(), "s-"+user)
	return found
}

func TestUsers(t *testing.T) {
	f := setup()
	id := f.user(t, "ann")
	if code, _ := f.do(t, "POST", "/scim/v2/Users", `{"userName":"ann"}`); code != 409 {
		t.Fatalf("duplicate: %d", code)
	}
	if code, v := f.do(t, "GET", "/scim/v2/Users/"+id, ""); code != 200 || v["userName"] != "ann" || v["active"] != true {
		t.Fatalf("get: %d %v", code, v)
	}
	if code, _ := f.do(t, "GET", "/scim/v2/Users/nope", ""); code != 404 {
		t.Fatalf("missing: %d", code)
	}
	if code, v := f.do(t, "PUT", "/scim/v2/Users/"+id, `{"userName":"ann.b","active":true}`); code != 200 || v["userName"] != "ann.b" {
		t.Fatalf("put: %d %v", code, v)
	}
	if code, _ := f.do(t, "POST", "/scim/v2/Users", `{}`); code != 400 {
		t.Fatalf("no userName: %d", code)
	}
}

// Every way of deactivating revokes sessions before the response, and only the leaver's.
func TestDeactivateRevokes(t *testing.T) {
	bodies := map[string][2]string{
		"patch path":        {"PATCH", `{"Operations":[{"op":"replace","path":"active","value":false}]}`},
		"patch object":      {"PATCH", `{"Operations":[{"op":"Replace","value":{"active":false}}]}`},
		"patch string bool": {"PATCH", `{"Operations":[{"op":"Replace","path":"active","value":"False"}]}`},
		"put inactive":      {"PUT", `{"userName":"ann","active":false}`},
		"delete":            {"DELETE", ``},
	}
	for name, b := range bodies {
		f := setup()
		ann, bob := f.user(t, "ann"), f.user(t, "bob")
		f.session(t, ann)
		f.session(t, bob)
		code, _ := f.do(t, b[0], "/scim/v2/Users/"+ann, b[1])
		if code != 200 && code != 204 {
			t.Fatalf("%s: %d", name, code)
		}
		if f.hasSession(ann) || !f.hasSession(bob) {
			t.Errorf("%s: ann's session kept=%v, bob's session kept=%v", name, f.hasSession(ann), f.hasSession(bob))
		}
		if active, known, _ := f.dir.Active(context.Background(), "acme-logistics", ann); active || !known {
			t.Errorf("%s: ann must be kept but inactive", name)
		}
	}
}

type failingSessions struct{ *sessmem.Store }

func (failingSessions) RevokeUser(context.Context, string, string) (int, error) {
	return 0, errors.New("store down")
}

func TestRevocationFailureIsAFailedRequest(t *testing.T) {
	f := setup()
	f.h = (&Endpoint{Tenant: "acme-logistics", Directory: f.dir, Sessions: failingSessions{f.sessions},
		Token: func(context.Context) (string, error) { return "tok", nil }}).Handler()
	ann := f.user(t, "ann")
	if code, _ := f.do(t, "DELETE", "/scim/v2/Users/"+ann, ""); code != 503 {
		t.Fatalf("got %d, want 503 so the IdP retries", code)
	}
}

func TestBadToken(t *testing.T) {
	f := setup()
	for _, h := range []string{"", "Bearer nope", "tok", "Basic tok"} {
		r := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(`{"userName":"x"}`))
		if h != "" {
			r.Header.Set("Authorization", h)
		}
		w := httptest.NewRecorder()
		f.h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Errorf("%q: %d", h, w.Code)
		}
	}
}

// Group changes reach policy at once: GroupsOf reads the directory at decision time.
func TestGroupsReachPolicyAtOnce(t *testing.T) {
	f := setup()
	ann, bob := f.user(t, "ann"), f.user(t, "bob")
	code, v := f.do(t, "POST", "/scim/v2/Groups", `{"displayName":"it-leads","members":[{"value":"`+ann+`"}]}`)
	if code != 201 {
		t.Fatalf("group: %d", code)
	}
	gid := v["id"].(string)
	groups := func(u string) []string { g, _ := f.dir.GroupsOf(context.Background(), "acme-logistics", u); return g }
	if !reflect.DeepEqual(groups(ann), []string{"it-leads"}) || groups(bob) != nil {
		t.Fatal("initial membership wrong")
	}
	f.do(t, "PATCH", "/scim/v2/Groups/"+gid, `{"Operations":[{"op":"add","path":"members","value":[{"value":"`+bob+`"}]}]}`)
	f.do(t, "PATCH", "/scim/v2/Groups/"+gid, `{"Operations":[{"op":"remove","path":"members[value eq \"`+ann+`\"]"}]}`)
	if groups(ann) != nil || !reflect.DeepEqual(groups(bob), []string{"it-leads"}) {
		t.Fatalf("after patch: ann %v, bob %v", groups(ann), groups(bob))
	}
	if code, _ := f.do(t, "DELETE", "/scim/v2/Groups/"+gid, ""); code != 204 || groups(bob) != nil {
		t.Fatalf("delete group: %d", code)
	}
}

// SB-1204: the enterprise extension's manager is kept, read back, and patched.
func TestManager(t *testing.T) {
	f := setup()
	boss := f.user(t, "boss")
	code, v := f.do(t, "POST", "/scim/v2/Users", `{"userName":"ana","urn:ietf:params:scim:schemas:extension:enterprise:2.0:User":{"manager":{"value":"`+boss+`"}}}`)
	if code != 201 {
		t.Fatalf("create: %d %v", code, v)
	}
	id := v["id"].(string)
	u, _, _ := f.dir.User(context.Background(), "acme-logistics", id)
	if u.Manager != boss {
		t.Fatalf("manager %q, want %q", u.Manager, boss)
	}
	code, _ = f.do(t, "PATCH", "/scim/v2/Users/"+id, `{"Operations":[{"op":"replace","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager","value":{"value":"usr_other"}}]}`)
	if u, _, _ := f.dir.User(context.Background(), "acme-logistics", id); code != 200 || u.Manager != "usr_other" {
		t.Fatalf("patch: %d manager %q", code, u.Manager)
	}
}

// SB-1205: title and the enterprise organisation attributes are kept and patched.
func TestEnterpriseAttributes(t *testing.T) {
	f := setup()
	code, v := f.do(t, "POST", "/scim/v2/Users", `{"userName":"bo","title":"Controller","urn:ietf:params:scim:schemas:extension:enterprise:2.0:User":{"department":"Finance","division":"EMEA","costCenter":"CC-100"}}`)
	if code != 201 {
		t.Fatalf("create: %d %v", code, v)
	}
	id := v["id"].(string)
	u, _, _ := f.dir.User(context.Background(), "acme-logistics", id)
	if want := (map[string]string{"title": "Controller", "department": "Finance", "division": "EMEA", "cost_center": "CC-100"}); !reflect.DeepEqual(u.Attributes(), want) {
		t.Fatalf("attributes %v", u.Attributes())
	}
	code, _ = f.do(t, "PATCH", "/scim/v2/Users/"+id, `{"Operations":[{"op":"replace","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department","value":"Sales"},{"op":"replace","path":"title","value":"Lead"}]}`)
	if u, _, _ := f.dir.User(context.Background(), "acme-logistics", id); code != 200 || u.Department != "Sales" || u.Title != "Lead" {
		t.Fatalf("patch: %d %+v", code, u)
	}
}

// SB-1401: a manager sent as a plain string, alongside active:false, is applied; a
// remove clears it.
func TestManagerForms(t *testing.T) {
	f := setup()
	id := f.user(t, "eve")
	code, v := f.do(t, "PATCH", "/scim/v2/Users/"+id, `{"Operations":[{"op":"replace","path":"active","value":false},{"op":"replace","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager","value":"usr_boss"}]}`)
	u, _, _ := f.dir.User(context.Background(), "acme-logistics", id)
	if code != 200 || u.Manager != "usr_boss" || u.Active {
		t.Fatalf("patch: %d %v %+v", code, v, u)
	}
	code, _ = f.do(t, "PATCH", "/scim/v2/Users/"+id, `{"Operations":[{"op":"remove","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager"}]}`)
	if u, _, _ := f.dir.User(context.Background(), "acme-logistics", id); code != 200 || u.Manager != "" {
		t.Fatalf("remove: %d %q", code, u.Manager)
	}
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

type fakeSearch struct {
	got  ask.SearchRequest
	user string
	err  error
}

func (f *fakeSearch) Search(_ context.Context, s tenancy.Scope, r ask.SearchRequest) (ask.SearchResponse, error) {
	f.got, f.user = r, s.User()
	if f.err != nil {
		return ask.SearchResponse{}, f.err
	}
	return ask.SearchResponse{Trace: "tr_s", Release: "rel_x", Results: []ask.Result{{Cite: "docs:a@1", Snippet: "s", Flags: []string{}}}}, nil
}

type mapQuestions struct {
	saved  map[string]ask.SavedQuestion
	shares map[string]ask.Share
}

func (m *mapQuestions) PutSaved(_ context.Context, q ask.SavedQuestion) error {
	m.saved[q.ID] = q
	return nil
}
func (m *mapQuestions) SavedOf(_ context.Context, t, u string) ([]ask.SavedQuestion, error) {
	var out []ask.SavedQuestion
	for _, q := range m.saved {
		if q.Tenant == t && q.User == u {
			out = append(out, q)
		}
	}
	return out, nil
}
func (m *mapQuestions) DeleteSaved(_ context.Context, t, u, id string) (bool, error) {
	q, ok := m.saved[id]
	if !ok || q.User != u {
		return false, nil
	}
	delete(m.saved, id)
	return true, nil
}
func (m *mapQuestions) PutShare(_ context.Context, s ask.Share) error { m.shares[s.ID] = s; return nil }
func (m *mapQuestions) GetShare(_ context.Context, t, id string) (ask.Share, bool, error) {
	s, ok := m.shares[id]
	return s, ok && s.Tenant == t, nil
}

type noteRows struct{ n int }

func (r *noteRows) Write(context.Context, audit.Event) (audit.Row, error) {
	r.n++
	return audit.Row{}, nil
}

func knowledge(f *fakeSearch) (http.Handler, *noteRows) {
	mux := http.NewServeMux()
	rows := &noteRows{}
	n := 0
	Knowledge{Search: f, Release: "rel_x", NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Questions: ask.Questions{Tenant: "acme-logistics", Store: &mapQuestions{map[string]ask.SavedQuestion{}, map[string]ask.Share{}},
			Audit: rows, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}}.Register(mux)
	return mux, rows
}

func kcall(t *testing.T, h http.Handler, method, path, body, user string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", user, nil)
		if err != nil {
			t.Fatal(err)
		}
		req = req.WithContext(tenancy.WithScope(req.Context(), s))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestSearchEndpoint(t *testing.T) {
	f := &fakeSearch{}
	h, _ := knowledge(f)
	code, body := kcall(t, h, "GET", "/v1/search?q=refund+limit&sources=docs,chat", "", "alice")
	if code != 200 || f.got.Query != "refund limit" || len(f.got.Sources) != 2 || f.user != "alice" || len(body["results"].([]any)) != 1 {
		t.Fatalf("%d %v %+v", code, body, f.got)
	}
	if code, _ := kcall(t, h, "GET", "/v1/search?q=x", "", ""); code != 401 {
		t.Fatalf("signed out: %d", code)
	}
	f.err = ask.ErrInvalid
	if code, body := kcall(t, h, "GET", "/v1/search", "", "alice"); code != 400 || body["error"].(map[string]any)["code"] != "EINVAL" {
		t.Fatalf("%d %v", code, body)
	}
}

func TestSavedAndShareEndpoints(t *testing.T) {
	h, rows := knowledge(&fakeSearch{})
	code, body := kcall(t, h, "POST", "/v1/saved", `{"question":"refund limit?"}`, "alice")
	if code != 200 {
		t.Fatalf("%d %v", code, body)
	}
	id := body["id"].(string)
	if _, body := kcall(t, h, "GET", "/v1/saved", "", "bob"); len(body["items"].([]any)) != 0 {
		t.Fatalf("bob sees %v", body)
	}
	if code, _ := kcall(t, h, "DELETE", "/v1/saved/"+id, "", "bob"); code != 403 {
		t.Fatalf("bob deleting alice's question: %d", code)
	}
	if code, _ := kcall(t, h, "DELETE", "/v1/saved/"+id, "", "alice"); code != 200 {
		t.Fatalf("alice deleting: %d", code)
	}
	code, body = kcall(t, h, "POST", "/v1/shares", `{"question":"refund limit?"}`, "alice")
	if code != 200 || !strings.HasPrefix(body["url"].(string), "/q/sh_") || rows.n != 1 {
		t.Fatalf("%d %v rows %d", code, body, rows.n)
	}
	code, got := kcall(t, h, "GET", "/v1/shares/"+body["id"].(string), "", "bob")
	if code != 200 || got["question"] != "refund limit?" || got["answer"] != nil {
		t.Fatalf("%d %v", code, got)
	}
	if code, _ := kcall(t, h, "GET", "/v1/shares/sh_nope", "", "bob"); code != 403 {
		t.Fatalf("unknown share: %d", code)
	}
	if code, _ := kcall(t, h, "POST", "/v1/shares", `{"question":""}`, "bob"); code != 400 {
		t.Fatalf("empty question: %d", code)
	}
}

// SB-2404: type and since reach the search; a bad since is EINVAL.
func TestSearchFiltersEndpoint(t *testing.T) {
	f := &fakeSearch{}
	h, _ := knowledge(f)
	if code, _ := kcall(t, h, "GET", "/v1/search?q=x&type=pdf,sheet&since=30d", "", "alice"); code != 200 || strings.Join(f.got.Types, ",") != "pdf,sheet" || f.got.SinceDays != 30 {
		t.Fatalf("%d %+v", code, f.got)
	}
	for _, bad := range []string{"30", "0d", "366d", "xd"} {
		if code, _ := kcall(t, h, "GET", "/v1/search?q=x&since="+bad, "", "alice"); code != 400 {
			t.Errorf("since=%s: %d", bad, code)
		}
	}
}

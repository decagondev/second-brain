package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

type fakeAgent struct {
	err  error
	got  ask.Request
	user string
}

func (f *fakeAgent) Ask(_ context.Context, s tenancy.Scope, r ask.Request) (ask.Response, error) {
	f.got, f.user = r, s.User()
	if f.err != nil {
		return ask.Response{}, f.err
	}
	return ask.Response{Trace: "tr_1", Release: "rel_x", Conversation: "cv_1",
		Answer: ask.Answer{Claims: []answer.Claim{{Text: "t", Cites: []answer.Cite{{Cite: "docs:a@1", Flags: []string{}}}}}}}, nil
}

func (f *fakeAgent) Trace(_ context.Context, _ tenancy.Scope, id string) (ask.Trace, error) {
	if id != "tr_1" {
		return ask.Trace{}, ask.ErrNotFound
	}
	return ask.Trace{ID: "tr_1", Decisions: []retrieve.Decision{{Cite: "docs:a@1", Verdict: "allow", Reason: "allowed"}}}, nil
}

func serve(t *testing.T, a Asker, method, path, body string, signedIn bool) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Ask{Agent: a, Release: "rel_x", NewID: func(p string) string { return p + "err" }}.Register(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if signedIn {
		s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "alice", nil)
		if err != nil {
			t.Fatal(err)
		}
		req = req.WithContext(tenancy.WithScope(req.Context(), s))
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestAskOK(t *testing.T) {
	f := &fakeAgent{}
	w := serve(t, f, "POST", "/v1/ask", `{"question":"q","sources":["docs"],"future_field":1}`, true)
	if w.Code != 200 || w.Header().Get("X-SB-Trace") != "tr_1" || w.Header().Get("X-SB-Release") != "rel_x" {
		t.Fatalf("%d %v %s", w.Code, w.Header(), w.Body)
	}
	if f.got.Question != "q" || f.user != "alice" || len(f.got.Sources) != 1 {
		t.Errorf("agent got %+v as %s", f.got, f.user)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["notice"]; !ok {
		t.Error("notice must be present (null) in every answer")
	}
}

func TestAskErrors(t *testing.T) {
	cases := []struct {
		name, body string
		signedIn   bool
		err        error
		code       string
		status     int
	}{
		{"not signed in", `{"question":"q"}`, false, nil, "EAUTH", 401},
		{"not JSON", `nope`, true, nil, "EINVAL", 400},
		{"invalid", `{"question":""}`, true, errors.Join(ask.ErrInvalid), "EINVAL", 400},
		{"killed", `{"question":"q"}`, true, ask.ErrKilled, "EKILLED", 503},
		{"down", `{"question":"q"}`, true, ask.ErrUnavailable, "EUNAVAIL", 503},
	}
	for _, c := range cases {
		w := serve(t, &fakeAgent{err: c.err}, "POST", "/v1/ask", c.body, c.signedIn)
		var body struct {
			Error struct{ Code, Message, Trace, Remedy string } `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if w.Code != c.status || body.Error.Code != c.code || body.Error.Trace == "" || body.Error.Remedy == "" {
			t.Errorf("%s: %d %+v", c.name, w.Code, body.Error)
		}
	}
}

func TestTraces(t *testing.T) {
	w := serve(t, &fakeAgent{}, "GET", "/v1/traces/tr_1", "", true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"decisions":[{"cite":"docs:a@1","verdict":"allow","reason":"allowed"}]`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if w := serve(t, &fakeAgent{}, "GET", "/v1/traces/tr_2", "", true); w.Code != 403 {
		t.Errorf("unknown trace: %d, want 403 EPERM", w.Code)
	}
}

// The request and response types match sb-ask(3)'s field tables and example exactly.
func TestContractMatchesManPage(t *testing.T) {
	b, err := os.ReadFile("../../manual/sb-ask.3.md")
	if err != nil {
		t.Fatal(err)
	}
	man := string(b)
	section := func(from, to string) string {
		i := strings.Index(man, from)
		j := strings.Index(man[i:], to)
		return man[i : i+j]
	}
	fieldRe := regexp.MustCompile("(?m)^\\| `([a-z_\\[\\].]+)` \\|")
	fields := func(s string) []string {
		var out []string
		for _, m := range fieldRe.FindAllStringSubmatch(s, -1) {
			out = append(out, m[1])
		}
		sort.Strings(out)
		return out
	}
	tags := func(v any) []string {
		var out []string
		rt := reflect.TypeOf(v)
		for i := 0; i < rt.NumField(); i++ {
			out = append(out, strings.Split(rt.Field(i).Tag.Get("json"), ",")[0])
		}
		sort.Strings(out)
		return out
	}
	if got, want := tags(ask.Request{}), fields(section("### Request", "### Response")); !reflect.DeepEqual(got, want) {
		t.Errorf("request fields %v, man page %v", got, want)
	}
	// Every top-level field of the example response, and nothing else.
	ex := section("### Response `200`", "| Field |")
	ex = ex[strings.Index(ex, "{") : strings.LastIndex(ex, "}")+1]
	ex = regexp.MustCompile(`"[^"]*…"`).ReplaceAllString(ex, `"x"`)
	var example map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ex), &example); err != nil {
		t.Fatalf("man page example isn't JSON: %v\n%s", err, ex)
	}
	var keys []string
	for k := range example {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if got := tags(ask.Response{}); !reflect.DeepEqual(got, keys) {
		t.Errorf("response fields %v, man page example %v", got, keys)
	}
	var claims struct {
		Claims []map[string]json.RawMessage `json:"claims"`
	}
	if err := json.Unmarshal(example["answer"], &claims); err != nil {
		t.Fatal(err)
	}
	var cite map[string]json.RawMessage
	if err := json.Unmarshal(claims.Claims[0]["cites"], &[]*map[string]json.RawMessage{&cite}); err != nil {
		t.Fatal(err)
	}
	for k := range cite {
		if !strings.Contains(strings.Join(tags(answer.Cite{}), ","), k) {
			t.Errorf("man page cite field %q missing from answer.Cite", k)
		}
	}
}

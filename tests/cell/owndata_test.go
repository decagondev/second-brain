package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-3501 to SB-3503 through a running cell: a person's export holds their own records,
// with a cite they lost left out; forgetting empties their history and nobody else's; and
// they list and end their own sessions only.
func TestOwnDataThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(share, "Ops", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("cutoff.md", "The zebra cutoff is 3pm on weekdays.")
	write(".acl", "read domain:acme.example\n")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	r.signIn(t, "lee")
	lee := r.idp.User
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	if !r.asks(t, "When is the zebra cutoff?", "3pm") {
		t.Fatal("lee's answer")
	}
	_, leeSessions := r.do(t, "GET", "/v1/me/sessions", "")
	r.signIn(t, "sam")
	if !r.asks(t, "What is the zebra cutoff on weekdays?", "3pm") {
		t.Fatal("sam's answer")
	}

	export := func(other ...string) (map[string][]json.RawMessage, int) {
		t.Helper()
		code, body := r.do(t, "GET", "/v1/me/export", "")
		var x struct {
			User    string
			Records map[string][]json.RawMessage
			LeftOut int `json:"left_out"`
		}
		if err := json.Unmarshal([]byte(body), &x); err != nil || code != 200 || x.User != r.idp.User {
			t.Fatalf("export: %d %s", code, body)
		}
		for _, o := range other {
			if strings.Contains(body, o) {
				t.Fatalf("someone else's records in the export: %s", body)
			}
		}
		return x.Records, x.LeftOut
	}
	recs, left := export("When is the zebra cutoff?", lee)
	if len(recs["traces"]) != 1 || !strings.Contains(string(recs["traces"][0]), "docs:Ops/cutoff.md") || left != 0 ||
		len(recs["conversations"]) != 1 || !strings.Contains(string(recs["conversations"][0]), "on weekdays") {
		t.Fatalf("records %s, %d left out", recs, left)
	}
	// The document is taken from sam: the cite, and the claim it backed, are left out.
	write(".acl", "read lee@acme.example\n")
	recs, left = export()
	if strings.Contains(string(recs["traces"][0]), "cutoff.md") || strings.Contains(string(recs["traces"][0]), "3pm") || left < 1 {
		t.Fatalf("a lost cite in the export: %s, %d left out", recs["traces"], left)
	}

	// Sessions: sam's own only, and lee's can't be ended by sam.
	if code, _ := r.do(t, "POST", "/v1/tokens", `{"name":"nightly","days":7}`); code != 200 {
		t.Fatalf("token: %d", code)
	}
	code, body := r.do(t, "GET", "/v1/me/sessions", "")
	var ss struct {
		Items []struct {
			ID, Kind, Name string
			Current        bool
		}
	}
	if err := json.Unmarshal([]byte(body), &ss); err != nil || code != 200 || len(ss.Items) != 2 || strings.Contains(body, "sb_session") || strings.Contains(body, "sbt_") {
		t.Fatalf("sessions: %d %s", code, body)
	}
	var token, current string
	for _, s := range ss.Items {
		if s.Kind == "token" && s.Name == "nightly" {
			token = s.ID
		}
		if s.Kind == "web" && s.Current {
			current = s.ID
		}
	}
	if token == "" || current == "" {
		t.Fatalf("sessions %+v", ss.Items)
	}
	var leeS struct{ Items []struct{ ID string } }
	_ = json.Unmarshal([]byte(leeSessions), &leeS)
	if len(leeS.Items) != 1 {
		t.Fatalf("lee's sessions %s", leeSessions)
	}
	if code, _ := r.do(t, "DELETE", "/v1/me/sessions/"+leeS.Items[0].ID, ""); code != http.StatusForbidden {
		t.Fatalf("sam ended lee's session: %d", code)
	}
	if code, _ := r.do(t, "DELETE", "/v1/me/sessions/"+token, ""); code != 200 {
		t.Fatalf("sign out the token: %d", code)
	}
	if code, body := r.do(t, "GET", "/v1/me/sessions", ""); code != 200 || strings.Contains(body, token) || !strings.Contains(body, current) {
		t.Fatalf("after signing the token out: %d %s", code, body)
	}

	// Forget: a typed confirmation, then sam's history is empty and lee's isn't.
	if code, _ := r.do(t, "POST", "/v1/me/forget", `{"confirm":"yes"}`); code != http.StatusBadRequest {
		t.Fatalf("forget without the phrase: %d", code)
	}
	code, body = r.do(t, "POST", "/v1/me/forget", `{"confirm":"forget my questions"}`)
	if code != 200 || !strings.Contains(body, `"traces":1`) || !strings.Contains(body, `"held":0`) {
		t.Fatalf("forget: %d %s", code, body)
	}
	if recs, _ = export(); len(recs["traces"]) != 0 || len(recs["conversations"]) != 0 {
		t.Fatalf("after forgetting: %s", recs)
	}
	r.idp.User = lee
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if recs, _ = export(); len(recs["traces"]) != 1 {
		t.Fatalf("lee's history went too: %s", recs)
	}
}

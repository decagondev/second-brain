package cell

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/ask"
)

// SB-5803 to SB-5805 through a running cell: Keep my conversations on every device lists a
// web conversation only once it is on, keeps no answer, and off deletes it; a conversation
// shared with a link opens with each answer checked as the reader and ends when revoked;
// and where you left off lists a question waiting on a steward and a research plan.
func TestCarryTheConversationThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Finance"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, text := range map[string]string{
		"Finance/close.md": "The quokka close moves to Tuesday, and the AP review is still open for the whole quarter.",
		"Finance/.acl":     "read domain:acme.example\n",
	} {
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	r.signIn(t, "sam")
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	list := func() ask.Conversations {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/me/conversations", "")
		var c ask.Conversations
		if err := json.Unmarshal([]byte(out), &c); err != nil || code != 200 {
			t.Fatalf("conversations: %d %s", code, out)
		}
		return c
	}
	ask1 := func(q string) ask.Response {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/ask", `{"question":"`+q+`"}`)
		var resp ask.Response
		if err := json.Unmarshal([]byte(out), &resp); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, out)
		}
		return resp
	}
	first := ask1("When does the quokka close move?")
	if c := list(); len(c.Conversations) != 0 || c.Sync {
		t.Fatalf("a web conversation listed before sync: %+v", c)
	}
	if code, out := r.do(t, "PUT", "/v1/me/conversations/sync", `{"on":true}`); code != 200 {
		t.Fatalf("sync on: %d %s", code, out)
	}
	second := ask1("Is the AP review still open?")
	c := list()
	if !c.Sync || len(c.Conversations) != 1 || c.Conversations[0].ID != second.Conversation || !c.Conversations[0].Synced {
		t.Fatalf("synced: %+v", c)
	}
	code, out := r.do(t, "GET", "/v1/me/export", "")
	if code != 200 || !strings.Contains(out, `"sync":true`) || !strings.Contains(out, second.Conversation) {
		t.Fatalf("export: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/me/conversations/"+second.Conversation, ""); code != 200 || !strings.Contains(out, "docs:Finance/close.md@") {
		t.Fatalf("open: %d %s", code, out)
	}

	// Share the first conversation; it opens checked as the reader, and ends when revoked.
	code, out = r.do(t, "POST", "/v1/me/conversations/"+first.Conversation+"/share", "")
	var sh struct{ ID, URL string }
	if err := json.Unmarshal([]byte(out), &sh); err != nil || code != 200 || !strings.HasPrefix(sh.URL, "/c/sh_") {
		t.Fatalf("share: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/conversation-shares/"+sh.ID, ""); code != 200 || !strings.Contains(out, "quokka close") {
		t.Fatalf("open share: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/shares/"+sh.ID, ""); code == 200 {
		t.Fatalf("a conversation share opened as a question: %s", out)
	}
	if code, out := r.do(t, "DELETE", "/v1/conversation-shares/"+sh.ID, ""); code != 200 {
		t.Fatalf("revoke: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/conversation-shares/"+sh.ID, ""); code != 403 {
		t.Fatalf("a revoked share: %d", code)
	}

	if code, out := r.do(t, "PUT", "/v1/me/conversations/sync", `{"on":false}`); code != 200 {
		t.Fatalf("sync off: %d %s", code, out)
	}
	if c := list(); c.Sync || len(c.Conversations) != 0 {
		t.Fatalf("after off: %+v", c)
	}

	// Where you left off: a research plan waiting for approval.
	if code, out := r.do(t, "POST", "/v1/research", `{"question":"When does the quokka close move?"}`); code != 200 {
		t.Fatalf("plan: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/me/left-off", "")
	var lo ask.LeftOff
	if err := json.Unmarshal([]byte(out), &lo); err != nil || code != 200 || len(lo.Items) == 0 || lo.Items[0].Kind != ask.InboxResearch {
		t.Fatalf("left off: %d %s", code, out)
	}
}

package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/notify"
)

type agentsTold struct{ got []string }

func (s *agentsTold) Send(_ context.Context, to notify.Person, m notify.Message) error {
	s.got = append(s.got, m.Kind+" "+to.ID+": "+m.Text)
	return nil
}

// SB-4004 to SB-4006 through a running cell: the weekly brief and a new transcript's
// decisions wait as drafts for the steward they ran as, a new member gets one welcome
// brief, and the registry counts the runs.
func TestAgentsOnAScheduleThroughARunningCell(t *testing.T) {
	day := strings.ToLower(time.Now().Weekday().String())
	share := t.TempDir()
	for path, text := range map[string]string{
		"Finance/Start/guide.md": "This week in this space the quokka close happened; still open is the AP review.",
		"Finance/standup.vtt":    "WEBVTT\n\n00:01:00.000 --> 00:01:04.000\n<v Ann>We decided the quokka close moves to Tuesday.</v>\n\n",
		"Finance/Start/.acl":     "read domain:acme.example\n",
		"Finance/.acl":           "read domain:acme.example\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
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
spaces:
  finance:
    template: finance
    label: confidential
    members: [group:finance]
    stewards: [group:finance-leads]
    start_here: ["docs:/Finance/Start/"]
    brief: { agent: ask, weekly: ` + day + ` }
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, ids := range map[string][]string{"finance": {sam, lee}, "finance-leads": {sam}} {
		var ms []string
		for _, id := range ids {
			ms = append(ms, `{"value":"`+id+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	s := &agentsTold{}
	if _, err := r.cell.NotifyJob(s).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	welcomes := 0
	for _, g := range s.got {
		if strings.HasPrefix(g, "welcome ") {
			welcomes++
			if !strings.Contains(g, "Welcome to finance. Start here: ") || !strings.Contains(g, "guide") {
				t.Fatalf("welcome %q", g)
			}
		}
	}
	if welcomes != 2 {
		t.Fatalf("welcomes: %v", s.got)
	}
	code, out := r.do(t, "GET", "/v1/spaces/finance/questions", "")
	var q struct{ Items []ask.StewardItem }
	if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
		t.Fatalf("queue: %d %s", code, out)
	}
	kinds := map[string]bool{}
	for _, it := range q.Items {
		if it.Thread != nil && it.Thread.Agent == "ask" && it.Thread.Text != "" && !it.Thread.Hidden {
			kinds[it.Thread.Kind] = true
		}
	}
	if !kinds["brief"] || !kinds["decisions"] {
		t.Fatalf("drafts: %s", out)
	}
	// Run again: nothing is welcomed, briefed or drafted twice.
	s.got = nil
	if _, err := r.cell.NotifyJob(s).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, g := range s.got {
		if strings.HasPrefix(g, "welcome ") {
			t.Fatalf("welcomed twice: %v", s.got)
		}
	}
	_, out = r.do(t, "GET", "/v1/spaces/finance/questions", "")
	if err := json.Unmarshal([]byte(out), &q); err != nil || len(q.Items) != 2 {
		t.Fatalf("queue again: %s", out)
	}
	// SB-4004: the steward's registry counts the two runs in finance.
	code, out = r.do(t, "GET", "/v1/spaces/finance/agents", "")
	if code != 200 || !strings.Contains(out, `"name":"ask"`) || !strings.Contains(out, `"runs":2`) {
		t.Fatalf("registry: %d %s", code, out)
	}
}

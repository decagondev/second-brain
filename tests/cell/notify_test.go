package cell

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/jobsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/notify"
)

type told struct {
	mu  sync.Mutex
	got []string
}

func (t *told) Send(_ context.Context, to notify.Person, m notify.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.got = append(t.got, to.Email+": "+m.Text+" "+m.Link)
	return nil
}

func (t *told) take() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.got
	t.got = nil
	return out
}

// S20.1 through a running cell (SB-2101, SB-2102): the company's answerers are told of a
// new question, the asker of its reply, and nobody of its text; a person's settings stop it.
func TestNotificationsThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, notesManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	out := &told{}
	jobs := &jobsvc.Service{Cell: r.cell, Send: out}

	r.signIn(t, "sam")
	sam := r.idp.User
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"answerers","members":[{"value":"`+sam+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != 201 {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	r.signIn(t, "ann")
	const q = "What is the zebra shipping cutoff?"
	_, body := r.do(t, "POST", "/v1/ask", `{"question":"`+q+`"}`)
	var a struct{ Trace string }
	_ = json.Unmarshal([]byte(body), &a)
	if code, body := r.do(t, "POST", "/v1/questions", `{"question":"`+q+`","trace":"`+a.Trace+`"}`); code != 200 {
		t.Fatalf("send: %d %s", code, body)
	}
	if err := jobs.Run(ctx, "notify"); err != nil {
		t.Fatal(err)
	}
	got := out.take()
	if len(got) != 1 || !strings.HasPrefix(got[0], "sam") || !strings.Contains(got[0], "1 new question sent company-wide") ||
		!strings.HasSuffix(got[0], "/?steward=company") || strings.Contains(got[0], "zebra") {
		t.Fatalf("stewards were told %q", got)
	}

	if code, body := r.do(t, "PUT", "/v1/me/notifications", `{"off":["everything"]}`); code != 400 {
		t.Fatalf("an unknown kind: %d %s", code, body)
	}
	if code, body := r.do(t, "PUT", "/v1/me/notifications", `{"off":["replies"]}`); code != 200 || !strings.Contains(body, `"off":["replies"]`) {
		t.Fatalf("settings: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/me/notifications", ""); code != 200 || !strings.Contains(body, `"off":["replies"]`) {
		t.Fatalf("settings: %d %s", code, body)
	}
	if code, body := r.do(t, "PUT", "/v1/me/notifications", `{"off":[]}`); code != 200 || !strings.Contains(body, `"off":[]`) {
		t.Fatalf("settings: %d %s", code, body)
	}

	r.idp.User = sam
	if code, _ := r.do(t, "GET", "/login", ""); code != 404 {
		t.Fatalf("sam's sign-in: %d", code)
	}
	_, body = r.do(t, "GET", "/v1/spaces/company/questions", "")
	var queue struct{ Items []struct{ ID string } }
	_ = json.Unmarshal([]byte(body), &queue)
	if len(queue.Items) != 1 {
		t.Fatalf("queue: %s", body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/company/questions/"+queue.Items[0].ID+"/reply", `{"text":"3pm on weekdays.","cites":[]}`); code != 200 {
		t.Fatalf("reply: %d %s", code, body)
	}
	if err := jobs.Run(ctx, "notify"); err != nil {
		t.Fatal(err)
	}
	got = out.take()
	if len(got) != 1 || !strings.HasPrefix(got[0], "ann") || !strings.Contains(got[0], "1 of your questions has a reply") ||
		!strings.HasSuffix(got[0], "/?tab=questions") || strings.Contains(got[0], "3pm") {
		t.Fatalf("the asker was told %q", got)
	}
	if err := jobs.Run(ctx, "notify"); err != nil || len(out.take()) != 0 {
		t.Fatalf("a run told something twice: %v", err)
	}
}

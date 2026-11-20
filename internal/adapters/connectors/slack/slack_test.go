package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/slack/slacktest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

func setup(t *testing.T) (*slacktest.Fake, *Connector) {
	t.Helper()
	f := slacktest.New()
	t.Cleanup(f.Server.Close)
	c := &Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(), Inbox: &MemoryInbox{},
		Creds: func(context.Context) (string, error) { return f.CredsJSON(), nil }}
	f.AddUser("U1", "ann@acme.example", false)
	f.AddUser("U2", "bob@acme.example", false)
	f.AddUser("U3", "cy@acme.example", false)
	f.AddUser("U9", "guest@partner.example", true)
	return f, c
}

func deliver(t *testing.T, f *slacktest.Fake, c *Connector, body []byte) int {
	t.Helper()
	w := httptest.NewRecorder()
	c.EventsHandler(time.Now).ServeHTTP(w, f.Deliver(body, time.Now()))
	return w.Code
}

type sandbox struct {
	t    *testing.T
	f    *slacktest.Fake
	c    *Connector
	refs []ports.SourceRef
}

func (s *sandbox) Connector() ports.Connector { return s.c }
func (s *sandbox) Doc() ports.SourceRef       { return s.refs[0] }
func (s *sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (s *sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s *sandbox) Refs(n int) []ports.SourceRef { return s.refs[:n] }
func (s *sandbox) InjectACLFault(kind string)   { s.f.FailNext(kind) }
func (s *sandbox) Action() (ports.Action, bool) { return ports.Action{}, false }
func (s *sandbox) Effects() int                 { return 0 }
func (s *sandbox) Edit(ref ports.SourceRef) {
	_, _, ts, _ := splitID(ref.ID)
	deliver(s.t, s.f, s.c, s.f.Edit("C1", ts, "edited"))
}
func (s *sandbox) ChangePermissions(ports.SourceRef) { deliver(s.t, s.f, s.c, s.f.Join("C1", "U3")) }

func TestKit(t *testing.T) {
	f, c := setup(t)
	f.AddChannel("C1", "finance", true, true, "U1")
	s := &sandbox{t: t, f: f, c: c}
	for i := 0; i < 200; i++ {
		ts, _ := f.Post("C1", "U1", "", "message")
		s.refs = append(s.refs, ports.SourceRef{Kind: "chat", ID: docID("C1", ts, ts)})
	}
	connectorkit.Run(t, s)
}

func verdict(t *testing.T, c *Connector, email, id string) ports.Verdict {
	t.Helper()
	vs, _ := c.ACL(context.Background(), ports.UserRef{Email: email}, []ports.SourceRef{{Kind: "chat", ID: id}})
	return vs[0]
}

// Public channels are open to full members; guests and private channels need membership.
func TestMembershipMapping(t *testing.T) {
	f, c := setup(t)
	f.AddChannel("C1", "general", false, true, "U9")
	f.AddChannel("C2", "random", false, true)
	f.AddChannel("C3", "leads", true, true, "U1")
	pub, _ := f.Post("C1", "U1", "", "hi")
	pub2, _ := f.Post("C2", "U1", "", "hi")
	priv, _ := f.Post("C3", "U1", "", "secret")
	cases := []struct {
		email, id string
		want      ports.Verdict
	}{
		{"bob@acme.example", docID("C1", pub, pub), ports.VerdictAllow},       // public, full member
		{"guest@partner.example", docID("C1", pub, pub), ports.VerdictAllow},  // guest in the channel
		{"guest@partner.example", docID("C2", pub2, pub2), ports.VerdictDeny}, // guest not in it
		{"ann@acme.example", docID("C3", priv, priv), ports.VerdictAllow},
		{"bob@acme.example", docID("C3", priv, priv), ports.VerdictDeny},
		{"nobody@acme.example", docID("C3", priv, priv), ports.VerdictDeny},  // no Slack account
		{"ann@acme.example", docID("C3", priv, "1.0"), ports.VerdictUnknown}, // no such message
	}
	for _, x := range cases {
		if got := verdict(t, c, x.email, x.id); got != x.want {
			t.Errorf("%s on %s: %v, want %v", x.email, x.id, got, x.want)
		}
	}
}

// Direct messages and excluded channels are never fetched or answered for.
func TestExclusions(t *testing.T) {
	f, c := setup(t)
	c.Exclude = []string{"#hr-*"}
	f.AddChannel("D1", "", true, true, "U1", "U2")
	f.AddChannel("C4", "hr-cases", true, true, "U1")
	f.AddChannel("C5", "ops", false, true)
	dm, _ := f.Post("D1", "U1", "", "private chat")
	hr, _ := f.Post("C4", "U1", "", "case notes")
	ops, _ := f.Post("C5", "U1", "", "deploy at 5")
	for _, id := range []string{docID("D1", dm, dm), docID("C4", hr, hr)} {
		if _, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "chat", ID: id}); err == nil {
			t.Errorf("%s was fetched", id)
		}
		if v := verdict(t, c, "ann@acme.example", id); v != ports.VerdictUnknown {
			t.Errorf("%s: %v, want unknown", id, v)
		}
	}
	set, _, err := c.Changes(context.Background(), "")
	if err != nil || len(set) != 1 || set[0].Ref.ID != docID("C5", ops, ops) {
		t.Fatalf("first sync lists %v %v; want only #ops", set, err)
	}
}

// Deliveries must be signed and fresh; Slack's URL check is answered.
func TestEventsAreVerified(t *testing.T) {
	f, c := setup(t)
	f.AddChannel("C1", "general", false, true)
	_, body := f.Post("C1", "U1", "", "x")
	h := c.EventsHandler(time.Now)
	bad := f.Deliver(body, time.Now())
	bad.Header.Set("X-Slack-Signature", "v0=00")
	stale := f.Deliver(body, time.Now().Add(-10*time.Minute))
	for name, r := range map[string]*http.Request{"bad signature": bad, "stale": stale} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
	if n, _ := c.Inbox.Len(context.Background()); n != 0 {
		t.Fatal("a refused delivery reached the inbox")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, f.Deliver([]byte(`{"type":"url_verification","challenge":"abc"}`), time.Now()))
	if !strings.Contains(w.Body.String(), "abc") {
		t.Fatalf("challenge: %s", w.Body.String())
	}
	if deliver(t, f, c, body) != 200 {
		t.Fatal("a valid delivery was refused")
	}
}

func TestDeleteIsADelete(t *testing.T) {
	f, c := setup(t)
	f.AddChannel("C1", "general", false, true)
	ts, _ := f.Post("C1", "U1", "", "oops")
	_, cur, _ := c.Changes(context.Background(), "")
	deliver(t, f, c, f.Delete("C1", ts))
	set, _, err := c.Changes(context.Background(), cur)
	if err != nil || len(set) != 1 || !set[0].Deleted {
		t.Fatalf("%v %v", set, err)
	}
}

// SB-1005: the location is the channel; a private channel's messages are confidential.
func TestLocationAndLabel(t *testing.T) {
	f, c := setup(t)
	f.AddChannel("C1", "eng-platform", false, true)
	f.AddChannel("C2", "leads", true, true, "U1")
	pub, _ := f.Post("C1", "U1", "", "hi")
	priv, _ := f.Post("C2", "U1", "", "plans")
	d, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "chat", ID: docID("C1", pub, pub)})
	if err != nil || d.Location != "#eng-platform" || d.Label != "" {
		t.Fatalf("public: %q %q %v", d.Location, d.Label, err)
	}
	d, err = c.Fetch(context.Background(), ports.SourceRef{Kind: "chat", ID: docID("C2", priv, priv)})
	if err != nil || d.Location != "#leads" || d.Label != "confidential" {
		t.Fatalf("private: %q %q %v", d.Location, d.Label, err)
	}
}

package gdrive

import (
	"context"
	"fmt"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/gdrive/drivetest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

const gdoc = "application/vnd.google-apps.document"

func setup(t *testing.T) (*drivetest.Fake, *Connector, *obs.Redactor) {
	t.Helper()
	f := drivetest.New()
	t.Cleanup(f.Server.Close)
	red := obs.NewRedactor()
	sa := &ServiceAccount{Key: func(context.Context) (string, error) { return f.KeyJSON(), nil },
		HTTP: f.Server.Client(), Redact: red, Now: time.Now}
	return f, &Connector{Base: f.Base(), HTTP: f.Server.Client(), Tokens: sa}, red
}

type sandbox struct {
	f *drivetest.Fake
	c *Connector
}

func (s sandbox) Connector() ports.Connector { return s.c }
func (s sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "docs", ID: "f0"} }
func (s sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (s sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "docs", ID: fmt.Sprintf("f%d", i)}
	}
	return out
}
func (s sandbox) InjectACLFault(kind string) { s.f.FailNext(kind) }
func (s sandbox) Edit(ref ports.SourceRef)   { s.f.Put(ref.ID, "Bonus policy", gdoc, "edited text") }
func (s sandbox) ChangePermissions(ref ports.SourceRef) {
	s.f.Share(ref.ID, "ann@acme.example", "cy@acme.example")
}
func (s sandbox) Action() (ports.Action, bool) { return ports.Action{}, false }
func (s sandbox) Effects() int                 { return 0 }

func TestKit(t *testing.T) {
	f, c, _ := setup(t)
	for i := 0; i < 200; i++ {
		f.Put(fmt.Sprintf("f%d", i), "Bonus policy", gdoc, "text", "ann@acme.example")
	}
	connectorkit.Run(t, sandbox{f, c})
}

func TestFetchAndPrefilter(t *testing.T) {
	f, c, _ := setup(t)
	f.Put("a", "Plan", gdoc, "exported text", "ann@acme.example")
	f.Put("b", "notes.txt", "text/plain", "raw text", "bob@acme.example")
	f.Put("c", "deck.pdf", "application/pdf", "binary", "ann@acme.example")
	ctx := context.Background()
	for id, want := range map[string]string{"a": "exported text", "b": "raw text", "c": ""} {
		d, err := c.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: id})
		if err != nil || d.Text != want || d.Version != 1 || len(d.ACLPrefilter) != 1 {
			t.Errorf("%s: %+v %v", id, d, err)
		}
	}
}

func TestRateLimitsAreRetried(t *testing.T) {
	f, c, _ := setup(t)
	f.Put("a", "Plan", gdoc, "x", "ann@acme.example")
	ctx := context.Background()
	if _, err := c.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: "a"}); err != nil {
		t.Fatal(err) // warms the token cache
	}
	f.RateLimitNext(2)
	vs, _ := c.ACL(ctx, ports.UserRef{Email: "ann@acme.example"}, []ports.SourceRef{{Kind: "docs", ID: "a"}})
	if vs[0] != ports.VerdictAllow {
		t.Fatalf("after two 429s within the deadline: %v", vs[0])
	}
	f.RateLimitNext(1000)
	vs, _ = c.ACL(ctx, ports.UserRef{Email: "ann@acme.example"}, []ports.SourceRef{{Kind: "docs", ID: "a"}})
	if vs[0] != ports.VerdictUnknown {
		t.Fatalf("rate-limited past the deadline: %v, want unknown", vs[0])
	}
}

func TestTrashedIsUnknownThenDeleted(t *testing.T) {
	f, c, _ := setup(t)
	f.Put("a", "Plan", gdoc, "x", "ann@acme.example")
	ctx := context.Background()
	_, cur, err := c.Changes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	f.Trash("a")
	if vs, _ := c.ACL(ctx, ports.UserRef{Email: "bob@acme.example"}, []ports.SourceRef{{Kind: "docs", ID: "a"}}); vs[0] != ports.VerdictUnknown {
		t.Fatalf("trashed file: %v, want unknown", vs[0])
	}
	set, _, err := c.Changes(ctx, cur)
	if err != nil || len(set) != 1 || !set[0].Deleted {
		t.Fatalf("trash must be a delete: %v %v", set, err)
	}
}

// Access tokens are secrets: they are registered for redaction, and cached per subject.
func TestTokensAreRedactedAndCached(t *testing.T) {
	_, c, red := setup(t)
	sa := c.Tokens.(*ServiceAccount)
	a, err := sa.Token(context.Background(), "ann@acme.example")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := sa.Token(context.Background(), "ann@acme.example")
	if a != b || red.Scrub("Bearer "+a) != "Bearer "+obs.Redacted {
		t.Fatal("token not cached or not registered with the redactor")
	}
}

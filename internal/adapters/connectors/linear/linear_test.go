package linear

import (
	"context"
	"fmt"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/linear/lineartest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

func setup(t *testing.T) (*lineartest.Fake, *Connector) {
	t.Helper()
	f := lineartest.New()
	t.Cleanup(f.Server.Close)
	f.AddUser("U1", "ann@acme.example", false)
	f.AddUser("U2", "bob@acme.example", false)
	f.AddUser("U3", "cy@acme.example", false)
	f.AddUser("U9", "guest@partner.example", true)
	return f, &Connector{Endpoint: f.Endpoint(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Key: func(context.Context) (string, error) { return f.Key, nil }}
}

type sandbox struct {
	f *lineartest.Fake
	c *Connector
}

func (s sandbox) Connector() ports.Connector { return s.c }
func (s sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "tickets", ID: "i0"} }
func (s sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (s sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "tickets", ID: fmt.Sprintf("i%d", i)}
	}
	return out
}
func (s sandbox) InjectACLFault(kind string)        { s.f.FailNext(kind) }
func (s sandbox) Edit(ref ports.SourceRef)          { s.f.Put(ref.ID, "T1", "Payroll", "edited") }
func (s sandbox) ChangePermissions(ports.SourceRef) { s.f.Join("T1", "U3") }
func (s sandbox) Action() (ports.Action, bool)      { return ports.Action{}, false }
func (s sandbox) Effects() int                      { return 0 }

func TestKit(t *testing.T) {
	f, c := setup(t)
	f.AddTeam("T1", "FIN", true, "U1")
	for i := 0; i < 200; i++ {
		f.Put(fmt.Sprintf("i%d", i), "T1", "Payroll", "description")
	}
	connectorkit.Run(t, sandbox{f, c})
}

func TestTeamPermissions(t *testing.T) {
	f, c := setup(t)
	f.AddTeam("T1", "OPS", false, "U9")
	f.AddTeam("T2", "ENG", false)
	f.AddTeam("T3", "FIN", true, "U1")
	f.Put("a", "T1", "x", "x")
	f.Put("b", "T2", "x", "x")
	f.Put("c", "T3", "x", "x")
	f.Put("d", "T3", "x", "x")
	f.Archive("d")
	f.AddUser("U8", "gone@acme.example", false)
	f.Deactivate("U8")
	cases := []struct {
		email, id string
		want      ports.Verdict
	}{
		{"bob@acme.example", "a", ports.VerdictAllow},      // public team, full member
		{"guest@partner.example", "a", ports.VerdictAllow}, // guest in the team
		{"guest@partner.example", "b", ports.VerdictDeny},  // guest not in it
		{"ann@acme.example", "c", ports.VerdictAllow},      // private, member
		{"bob@acme.example", "c", ports.VerdictDeny},       // private, not a member
		{"nobody@acme.example", "c", ports.VerdictDeny},    // no Linear account
		{"gone@acme.example", "a", ports.VerdictDeny},      // deactivated
		{"ann@acme.example", "d", ports.VerdictUnknown},    // archived
		{"ann@acme.example", "zz", ports.VerdictUnknown},   // missing
	}
	for _, x := range cases {
		vs, _ := c.ACL(context.Background(), ports.UserRef{Email: x.email}, []ports.SourceRef{{Kind: "tickets", ID: x.id}})
		if vs[0] != x.want {
			t.Errorf("%s on %s: %v, want %v", x.email, x.id, vs[0], x.want)
		}
	}
}

func TestChanges(t *testing.T) {
	f, c := setup(t)
	f.AddTeam("T1", "OPS", true, "U1")
	f.Put("a", "T1", "x", "x")
	f.Put("b", "T1", "x", "x")
	ctx := context.Background()
	first, cur, err := c.Changes(ctx, "")
	if err != nil || len(first) != 2 {
		t.Fatalf("first sync: %v %v", first, err)
	}
	f.Archive("a")
	f.SetPrivate("T1", false)
	set, _, err := c.Changes(ctx, cur)
	if err != nil {
		t.Fatal(err)
	}
	var del, acl int
	for _, ch := range set {
		if ch.Deleted && ch.Ref.ID == "a" {
			del++
		}
		if ch.ACLOnly {
			acl++
		}
	}
	if del != 1 || acl != 2 {
		t.Fatalf("archive must be a delete and a privacy change permission-only for both issues: %v", set)
	}
}

func TestRateLimitsAreRetried(t *testing.T) {
	f, c := setup(t)
	f.AddTeam("T1", "OPS", false)
	f.Put("a", "T1", "x", "x")
	f.RateLimitNext(2)
	vs, _ := c.ACL(context.Background(), ports.UserRef{Email: "bob@acme.example"}, []ports.SourceRef{{Kind: "tickets", ID: "a"}})
	if vs[0] != ports.VerdictAllow {
		t.Fatalf("after two 429s: %v", vs[0])
	}
}

package hubspot

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/hubspot/hubspottest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

const roles = `"roles": {"r_sales": "owned", "r_lead": "team", "r_exec": "all"}`

func setup(t *testing.T) (*hubspottest.Fake, *Connector) {
	t.Helper()
	f := hubspottest.New()
	t.Cleanup(f.Server.Close)
	f.AddUser("U1", "ann@acme.example", "r_sales", false, "T1")
	f.AddUser("U2", "bob@acme.example", "r_sales", false, "T1")
	f.AddUser("U3", "cy@acme.example", "r_lead", false, "T1")
	f.AddUser("U4", "dee@acme.example", "r_exec", false)
	f.AddUser("U5", "eve@acme.example", "r_lead", false, "T2")
	f.AddUser("U6", "root@acme.example", "r_unmapped", true)
	f.AddUser("U7", "new@acme.example", "r_unmapped", false)
	return f, &Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return `{"token": "` + f.Token + `", ` + roles + `}`, nil }}
}

type sandbox struct {
	f *hubspottest.Fake
	c *Connector
}

func (s sandbox) Connector() ports.Connector { return s.c }
func (s sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "crm", ID: "deals/0"} }
func (s sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (s sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "crm", ID: fmt.Sprintf("deals/%d", i)}
	}
	return out
}
func (s sandbox) InjectACLFault(kind string) { s.f.FailNext(kind) }
func (s sandbox) Edit(ref ports.SourceRef) {
	s.f.Put(ref.ID, "Renewal", "edited", s.f.Owner("ann@acme.example"), "T1")
}
func (s sandbox) ChangePermissions(ref ports.SourceRef) {
	s.f.Reassign(ref.ID, s.f.Owner("ann@acme.example"), "T2")
}
func (s sandbox) Action() (ports.Action, bool) { return ports.Action{}, false }
func (s sandbox) Effects() int                 { return 0 }

func TestKit(t *testing.T) {
	f, c := setup(t)
	for i := 0; i < 200; i++ {
		f.Put(fmt.Sprintf("deals/%d", i), "Renewal", "description", f.Owner("ann@acme.example"), "T1")
	}
	connectorkit.Run(t, sandbox{f, c})
}

// SB-610: record-level permissions map to the ACL.
func TestRecordPermissions(t *testing.T) {
	f, c := setup(t)
	f.Put("deals/a", "Renewal", "x", f.Owner("ann@acme.example"), "T1")
	f.Put("companies/b", "Gamma Parts", "x", f.Owner("eve@acme.example"), "T2")
	f.Put("deals/c", "Old", "x", f.Owner("ann@acme.example"), "T1")
	f.Archive("deals/c")
	f.AddUser("U8", "gone@acme.example", "r_exec", false)
	f.Deactivate("gone@acme.example")
	cases := []struct {
		email, id string
		want      ports.Verdict
	}{
		{"ann@acme.example", "deals/a", ports.VerdictAllow},        // owned, owner
		{"bob@acme.example", "deals/a", ports.VerdictDeny},         // owned, not the owner
		{"cy@acme.example", "deals/a", ports.VerdictAllow},         // team, same team
		{"cy@acme.example", "companies/b", ports.VerdictDeny},      // team, other team
		{"eve@acme.example", "companies/b", ports.VerdictAllow},    // team, owner
		{"dee@acme.example", "companies/b", ports.VerdictAllow},    // all
		{"root@acme.example", "deals/a", ports.VerdictAllow},       // super admin
		{"new@acme.example", "deals/a", ports.VerdictUnknown},      // role not in the role map
		{"nobody@acme.example", "deals/a", ports.VerdictDeny},      // no HubSpot user
		{"gone@acme.example", "deals/a", ports.VerdictDeny},        // deactivated
		{"dee@acme.example", "deals/c", ports.VerdictUnknown},      // archived
		{"dee@acme.example", "deals/zz", ports.VerdictUnknown},     // missing
		{"dee@acme.example", "contacts/1", ports.VerdictUnknown},   // contacts are never read
		{"dee@acme.example", "deals/a/../x", ports.VerdictUnknown}, // a malformed id
	}
	for _, x := range cases {
		vs, _ := c.ACL(context.Background(), ports.UserRef{Email: x.email}, []ports.SourceRef{{Kind: "crm", ID: x.id}})
		if vs[0] != x.want {
			t.Errorf("%s on %s: %v, want %v", x.email, x.id, vs[0], x.want)
		}
	}
}

func TestFetch(t *testing.T) {
	f, c := setup(t)
	f.Put("deals/a", "Renewal", "Three-year renewal", f.Owner("ann@acme.example"), "T1")
	d, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "crm", ID: "deals/a"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Renewal" || !strings.Contains(d.Text, "Description: Three-year renewal") {
		t.Fatalf("document: %+v", d)
	}
	if strings.Join(d.ACLPrefilter, ",") != "hubspot:users,ann@acme.example" {
		t.Fatalf("prefilter: %v", d.ACLPrefilter)
	}
	if _, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "crm", ID: "contacts/1"}); err == nil {
		t.Fatal("fetched a contact")
	}
}

func TestChanges(t *testing.T) {
	f, c := setup(t)
	f.Put("deals/a", "x", "x", f.Owner("ann@acme.example"), "T1")
	f.Put("companies/b", "x", "x", f.Owner("ann@acme.example"), "T1")
	f.Put("deals/old", "x", "x", f.Owner("ann@acme.example"), "T1")
	f.Archive("deals/old")
	ctx := context.Background()
	first, cur, err := c.Changes(ctx, "")
	if err != nil || len(first) != 2 {
		t.Fatalf("first sync must list the two live records only: %v %v", first, err)
	}
	f.Archive("deals/a")
	f.Reassign("companies/b", f.Owner("bob@acme.example"), "T2")
	f.Put("deals/new", "x", "x", f.Owner("ann@acme.example"), "T1")
	set, _, err := c.Changes(ctx, cur)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, ch := range set {
		switch {
		case ch.Deleted:
			got[ch.Ref.ID] = "deleted"
		case ch.ACLOnly:
			got[ch.Ref.ID] = "acl"
		default:
			got[ch.Ref.ID] = "content"
		}
	}
	want := map[string]string{"deals/a": "deleted", "companies/b": "acl", "deals/new": "content"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("changes: %v, want %v", got, want)
	}
	if _, _, err := c.Changes(ctx, "{}"); err == nil {
		t.Fatal("an empty cursor object was accepted")
	}
}

func TestRateLimitsAreRetried(t *testing.T) {
	f, c := setup(t)
	f.Put("deals/a", "x", "x", f.Owner("ann@acme.example"), "T1")
	f.RateLimitNext(2)
	vs, _ := c.ACL(context.Background(), ports.UserRef{Email: "ann@acme.example"}, []ports.SourceRef{{Kind: "crm", ID: "deals/a"}})
	if vs[0] != ports.VerdictAllow {
		t.Fatalf("after two 429s: %v", vs[0])
	}
}

func TestBadCredentials(t *testing.T) {
	f, c := setup(t)
	f.Put("deals/a", "x", "x", f.Owner("ann@acme.example"), "T1")
	for _, raw := range []string{`not json`, `{"roles": {}}`, `{"token": "` + f.Token + `", "roles": {"r_sales": "everything"}}`} {
		c.Creds = func(context.Context) (string, error) { return raw, nil }
		vs, _ := c.ACL(context.Background(), ports.UserRef{Email: "ann@acme.example"}, []ports.SourceRef{{Kind: "crm", ID: "deals/a"}})
		if vs[0] != ports.VerdictUnknown {
			t.Errorf("credentials %q: %v, want unknown", raw, vs[0])
		}
	}
}

func TestTokenIsRedacted(t *testing.T) {
	f, c := setup(t)
	f.Put("deals/a", "x", "x", f.Owner("ann@acme.example"), "T1")
	r := obs.NewRedactor()
	c.Redact = r
	if _, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "crm", ID: "deals/a"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Scrub("token "+f.Token), f.Token) {
		t.Fatal("the private app token was not registered with the redactor")
	}
}

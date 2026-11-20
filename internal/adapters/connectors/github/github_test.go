package github

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/github/githubtest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

func setup(t *testing.T) (*githubtest.Fake, *Connector) {
	t.Helper()
	f := githubtest.New()
	t.Cleanup(f.Server.Close)
	f.AddMember("ann", "ann@acme.example")
	f.AddMember("bob", "bob@acme.example")
	f.AddMember("cy", "cy@acme.example")
	return f, &Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return f.CredsJSON(), nil }}
}

func s(x string) *string { return &x }

type sandbox struct {
	f *githubtest.Fake
	c *Connector
}

func (b sandbox) Connector() ports.Connector { return b.c }
func (b sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "code", ID: "payments/issues/1"} }
func (b sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (b sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (b sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "code", ID: fmt.Sprintf("payments/issues/%d", i+1)}
	}
	return out
}
func (b sandbox) InjectACLFault(kind string)        { b.f.FailNext(kind) }
func (b sandbox) Edit(ports.SourceRef)              { b.f.PutIssue("payments", 1, "Queue stalls", "edited", false) }
func (b sandbox) ChangePermissions(ports.SourceRef) { b.f.Grant("payments", "cy", "read") }
func (b sandbox) Action() (ports.Action, bool)      { return ports.Action{}, false }
func (b sandbox) Effects() int                      { return 0 }

func TestKit(t *testing.T) {
	f, c := setup(t)
	f.AddRepo("payments", "private")
	f.Grant("payments", "ann", "write")
	f.Grant("payments", "bob", "none")
	for i := 1; i <= 200; i++ {
		f.PutIssue("payments", i, "Queue stalls", "body", false)
	}
	connectorkit.Run(t, sandbox{f, c})
}

// SB-1103: docs, READMEs and Markdown by default, source code only when opted in, and
// issues and pull requests with their own kinds and locations.
func TestWhatIsIndexed(t *testing.T) {
	f, c := setup(t)
	ctx := context.Background()
	f.AddRepo("payments", "private")
	f.Grant("payments", "ann", "read")
	f.Commit("payments", map[string]*string{"README.md": s("# Payments"), "docs/runbook.txt": s("Restart the consumer."),
		"src/queue.go": s("package queue"), "notes/design.md": s("Design")})
	f.PutIssue("payments", 7, "Stalls", "The queue stalls.", false)
	f.PutIssue("payments", 8, "Fix stalls", "Restart on stall.", true)
	set, _, err := c.Changes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, ch := range set {
		ids = append(ids, ch.Ref.ID)
	}
	got := strings.Join(ids, ",")
	for _, want := range []string{"payments/issues/7", "payments/pulls/8", "payments/blob/README.md", "payments/blob/docs/runbook.txt", "payments/blob/notes/design.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("%s not listed in %s", want, got)
		}
	}
	if strings.Contains(got, "src/queue.go") {
		t.Fatal("source code was listed without a space opting in")
	}
	if _, err := c.Fetch(ctx, ports.SourceRef{Kind: "code", ID: "payments/blob/src/queue.go"}); err == nil {
		t.Fatal("source code was fetched without a space opting in")
	}
	c.SourceRepos = func() []string { return []string{"payments"} }
	d, err := c.Fetch(ctx, ports.SourceRef{Kind: "code", ID: "payments/blob/src/queue.go"})
	if err != nil || d.Text != "package queue" || d.Location != "payments/src/queue.go" || d.Version == 0 {
		t.Fatalf("opted-in source: %+v %v", d, err)
	}
	if !strings.Contains(fmt.Sprint(d.ACLPrefilter), "ann@acme.example") {
		t.Fatalf("prefilter %v", d.ACLPrefilter)
	}
	pr, err := c.Fetch(ctx, ports.SourceRef{Kind: "code", ID: "payments/pulls/8"})
	if err != nil || pr.Title != "#8 Fix stalls" || pr.Location != "payments/pulls/8" {
		t.Fatalf("pull: %+v %v", pr, err)
	}
	if _, err := c.Fetch(ctx, ports.SourceRef{Kind: "code", ID: "payments/issues/8"}); err == nil {
		t.Fatal("a pull request fetched as an issue")
	}
	_, cur, _ := c.Changes(ctx, "")
	f.Commit("payments", map[string]*string{"docs/runbook.txt": nil, "README.md": s("# Payments v2")})
	set, _, err = c.Changes(ctx, cur)
	if err != nil {
		t.Fatal(err)
	}
	gone, edited := false, false
	for _, ch := range set {
		gone = gone || (ch.Ref.ID == "payments/blob/docs/runbook.txt" && ch.Deleted)
		edited = edited || (ch.Ref.ID == "payments/blob/README.md" && !ch.Deleted)
	}
	if !gone || !edited {
		t.Fatalf("changes after a commit: %+v", set)
	}
}

// SB-1103: public repositories are open to members, private ones by permission, and
// anyone GitHub can't place is denied or unknown, never allowed.
func TestPermissions(t *testing.T) {
	f, c := setup(t)
	ctx := context.Background()
	f.AddRepo("site", "public")
	f.AddRepo("payments", "private")
	f.AddRepo("infra", "internal")
	f.Grant("payments", "ann", "admin")
	f.Grant("infra", "bob", "triage")
	for _, r := range []string{"site", "payments", "infra"} {
		f.PutIssue(r, 1, "x", "x", false)
	}
	f.AddRepo("gone", "private")
	f.Remove("gone")
	refs := []ports.SourceRef{{Kind: "code", ID: "site/issues/1"}, {Kind: "code", ID: "payments/issues/1"}, {Kind: "code", ID: "infra/issues/1"}, {Kind: "code", ID: "gone/issues/1"}}
	for _, tc := range []struct {
		email string
		want  []ports.Verdict
	}{
		{"ann@acme.example", []ports.Verdict{ports.VerdictAllow, ports.VerdictAllow, ports.VerdictDeny, ports.VerdictUnknown}},
		{"bob@acme.example", []ports.Verdict{ports.VerdictAllow, ports.VerdictDeny, ports.VerdictAllow, ports.VerdictUnknown}},
		{"stranger@else.example", []ports.Verdict{ports.VerdictDeny, ports.VerdictDeny, ports.VerdictDeny, ports.VerdictDeny}},
	} {
		got, err := c.ACL(ctx, ports.UserRef{ID: "u", Email: tc.email}, refs)
		if err != nil || fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("%s: %v %v, want %v", tc.email, got, err, tc.want)
		}
	}
	got, _ := c.ACL(ctx, ports.UserRef{ID: "u"}, refs)
	for _, v := range got {
		if v != ports.VerdictUnknown {
			t.Fatal("no email must be Unknown")
		}
	}
}

// SB-1401: if the identity filter is ignored, another member's identity never answers
// for the asker: the verdict is Unknown, not their access.
func TestACLNeverUsesAnotherMembersIdentity(t *testing.T) {
	f, c := setup(t)
	f.AddRepo("payments", "private")
	f.Grant("payments", "ann", "write")
	f.Grant("payments", "bob", "none")
	f.LooseIdentities = true
	vs, err := c.ACL(context.Background(), ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}, []ports.SourceRef{{Kind: "code", ID: "payments/issues/1"}})
	if err != nil || len(vs) != 1 || vs[0] == ports.VerdictAllow {
		t.Fatalf("bob's verdict with every identity returned: %v %v", vs, err)
	}
}

// SB-1401: issues past the first page are listed too.
func TestIssuesPastTheFirstPage(t *testing.T) {
	f, c := setup(t)
	f.AddRepo("payments", "private")
	for i := 1; i <= 230; i++ {
		f.PutIssue("payments", i, "Issue", "body", false)
	}
	set, _, err := c.Changes(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ch := range set {
		if strings.Contains(ch.Ref.ID, "/issues/") {
			n++
		}
	}
	if n != 230 {
		t.Fatalf("%d issues listed, want 230", n)
	}
}

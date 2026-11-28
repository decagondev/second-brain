package atlassian

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/atlassian/atlassiantest"
	"example.com/second-brain/internal/core/ports"
)

func client(f *atlassiantest.Fake) *Client {
	return &Client{Base: f.Base(), HTTP: f.Server.Client(), Creds: func(context.Context) (string, error) { return atlassiantest.Creds, nil }}
}

func html(_ string, b []byte) (string, bool) {
	s := string(b)
	for _, t := range []string{"<p>", "</p>", "<h1>", "</h1>"} {
		s = strings.ReplaceAll(s, t, "\n")
	}
	return strings.TrimSpace(s), true
}

// SB-6201, SB-6202: a space syncs with its pages, blog posts, labels and restrictions; a
// restricted page is open only to its people and groups, live; an anonymous space is wide;
// a deleted page is gone at the next sync; someone with no account is denied; a group's
// members and a person's groups resolve.
func TestConfluence(t *testing.T) {
	f := atlassiantest.New()
	t.Cleanup(f.Server.Close)
	for _, e := range []string{"sam@acme.example", "lee@acme.example", "kim@acme.example"} {
		f.User(e)
	}
	f.Groups["finance"] = []string{"sam@acme.example"}
	f.Spaces["FIN"], f.Spaces["PUB"] = false, true
	f.Pages["1"] = &atlassiantest.Page{ID: "1", Space: "FIN", Title: "Close calendar", Body: "<h1>Close</h1><p>The close moves to Tuesday.</p>", Version: 3, Labels: []string{"finance"}, Groups: []string{"finance"}}
	f.Pages["2"] = &atlassiantest.Page{ID: "2", Space: "FIN", Title: "Team lunch", Body: "<p>Lunch is at noon.</p>", Version: 1}
	f.Pages["3"] = &atlassiantest.Page{ID: "3", Space: "PUB", Title: "Press kit", Body: "<p>Logos.</p>", Version: 1, Blog: true}
	c := client(f)
	dir := &Directory{Client: c}
	cf := &Confluence{Client: c, Spaces: []string{"FIN", "PUB"}, Dir: dir, Text: html}
	ctx := context.Background()
	set, cur, err := cf.Changes(ctx, "")
	if err != nil || len(set) != 3 {
		t.Fatalf("first sync: %v %+v", err, set)
	}
	d, err := cf.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: "page:1"})
	if err != nil || !strings.HasPrefix(d.Text, "Labels: finance") || !strings.Contains(d.Text, "Tuesday") || d.Version != 3 || d.Location != "/FIN/Close calendar" {
		t.Fatalf("fetch: %v %+v", err, d)
	}
	if len(d.ACLPrefilter) != 1 || d.ACLPrefilter[0] != "group:finance" {
		t.Fatalf("restricted prefilter: %v", d.ACLPrefilter)
	}
	if d, _ := cf.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: "page:2"}); d.ACLPrefilter[0] != ConfluenceUsers {
		t.Fatalf("open page: %v", d.ACLPrefilter)
	}
	if d, _ := cf.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: "blog:3"}); d.ACLPrefilter[0] != "anyone" {
		t.Fatalf("anonymous space: %v", d.ACLPrefilter)
	}
	refs := []ports.SourceRef{{Kind: "docs", ID: "page:1"}, {Kind: "docs", ID: "page:2"}}
	for who, want := range map[string][2]ports.Verdict{
		"sam@acme.example":    {ports.VerdictAllow, ports.VerdictAllow},
		"lee@acme.example":    {ports.VerdictDeny, ports.VerdictAllow},
		"nobody@acme.example": {ports.VerdictDeny, ports.VerdictDeny},
	} {
		vs, _ := cf.ACL(ctx, ports.UserRef{Email: who}, refs)
		if vs[0] != want[0] || vs[1] != want[1] {
			t.Errorf("%s: %v, want %v", who, vs, want)
		}
	}
	// Taken out of the group: denied at once.
	f.Groups["finance"] = nil
	if vs, _ := cf.ACL(ctx, ports.UserRef{Email: "sam@acme.example"}, refs[:1]); vs[0] != ports.VerdictDeny {
		t.Fatalf("after leaving the group: %v", vs)
	}
	f.Pages["2"].Deleted = true
	f.Pages["1"].Version = 4
	set, _, err = cf.Changes(ctx, cur)
	if err != nil || len(set) != 2 || set[0].Ref.ID != "page:1" || set[0].Deleted || set[1].Ref.ID != "page:2" || !set[1].Deleted {
		t.Fatalf("next sync: %v %+v", err, set)
	}
	f.Groups["finance"] = []string{"sam@acme.example", "kim@acme.example"}
	if ms, err := dir.Members(ctx, "finance"); err != nil || len(ms) != 2 {
		t.Fatalf("members: %v %v", ms, err)
	}
	if gs, err := dir.GroupsOf(ctx, "kim@acme.example"); err != nil || len(gs) != 1 || gs[0] != "finance" {
		t.Fatalf("groups: %v %v", gs, err)
	}
	if _, err := dir.GroupsOf(ctx, "nobody@acme.example"); !errors.Is(err, ErrNoUser) {
		t.Fatalf("no account: %v", err)
	}
	big := make([]string, MaxMembers+60)
	for i := range big {
		big[i] = "u" + strings.Repeat("x", i%7) + "@acme.example"
	}
	f.Groups["everyone"] = big
	for _, e := range big {
		f.Users[e] = "a"
	}
	if _, err := dir.Members(ctx, "everyone"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("a huge group: %v", err)
	}
}

// SB-6204: a project's issues sync; an issue reads with its labels and status first, like
// Linear's; one under a security level is open only to its people, live; a closed one says
// so; a deleted one is gone at the next sync.
func TestJira(t *testing.T) {
	f := atlassiantest.New()
	t.Cleanup(f.Server.Close)
	for _, e := range []string{"sam@acme.example", "lee@acme.example"} {
		f.User(e)
	}
	f.Projects["OPS"] = nil
	f.Issues["OPS-1"] = &atlassiantest.Issue{Key: "OPS-1", Project: "OPS", Summary: "Queue stopped", Description: "The queue stopped at 9.", Status: "Done", Done: true,
		Labels: []string{"sev-1"}, Comments: []string{"Restarted."}, Updated: "2026-11-27T09:00:00.000+0000", Created: "2026-11-27T08:00:00.000+0000"}
	f.Issues["OPS-2"] = &atlassiantest.Issue{Key: "OPS-2", Project: "OPS", Summary: "Payroll export", Description: "Salaries.", Status: "To Do",
		Security: []string{"sam@acme.example"}, Updated: "2026-11-27T10:00:00.000+0000", Created: "2026-11-27T10:00:00.000+0000"}
	j := &Jira{Client: client(f), Projects: []string{"OPS"}}
	ctx := context.Background()
	set, cur, err := j.Changes(ctx, "")
	if err != nil || len(set) != 2 {
		t.Fatalf("sync: %v %+v", err, set)
	}
	d, err := j.Fetch(ctx, ports.SourceRef{Kind: "tickets", ID: "OPS-1"})
	if err != nil || !strings.HasPrefix(d.Text, "Labels: sev-1\nStatus: Done (closed)\n\nThe queue stopped at 9.") || !strings.Contains(d.Text, "Restarted.") || d.Title != "OPS-1 Queue stopped" {
		t.Fatalf("fetch: %v %q", err, d.Text)
	}
	if d, _ := j.Fetch(ctx, ports.SourceRef{Kind: "tickets", ID: "OPS-2"}); len(d.ACLPrefilter) != 1 || d.ACLPrefilter[0] != "sam@acme.example" {
		t.Fatalf("security level prefilter: %v", d.ACLPrefilter)
	}
	refs := []ports.SourceRef{{Kind: "tickets", ID: "OPS-1"}, {Kind: "tickets", ID: "OPS-2"}}
	if vs, _ := j.ACL(ctx, ports.UserRef{Email: "lee@acme.example"}, refs); vs[0] != ports.VerdictAllow || vs[1] != ports.VerdictDeny {
		t.Fatalf("lee: %v", vs)
	}
	if vs, _ := j.ACL(ctx, ports.UserRef{Email: "sam@acme.example"}, refs); vs[1] != ports.VerdictAllow {
		t.Fatalf("sam: %v", vs)
	}
	f.Issues["OPS-1"].Deleted = true
	if set, _, _ := j.Changes(ctx, cur); len(set) != 1 || !set[0].Deleted {
		t.Fatalf("deleted: %+v", set)
	}
	if _, err := j.Act(ctx, ports.Action{}, ""); err == nil {
		t.Fatal("Jira took a write")
	}
}

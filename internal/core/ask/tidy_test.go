package ask_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

type retired map[string]ask.Retirement

func (r retired) PutRetirement(_ context.Context, x ask.Retirement) error {
	r[x.Space+"|"+x.Doc] = x
	return nil
}
func (r retired) Retirements(_ context.Context, _, space string) ([]ask.Retirement, error) {
	var out []ask.Retirement
	for _, x := range r {
		if x.Space == space {
			out = append(out, x)
		}
	}
	return out, nil
}
func (r retired) DeleteRetirement(_ context.Context, _, space, doc string) (bool, error) {
	_, ok := r[space+"|"+doc]
	delete(r, space+"|"+doc)
	return ok, nil
}

type reviewSrc []ask.Trace

func (r reviewSrc) Between(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	return r, nil
}
func (reviewSrc) FeedbackBetween(context.Context, string, time.Time, time.Time) ([]ask.Feedback, error) {
	return nil, nil
}

func stewardsOf(a ask.Agent, now time.Time) ask.Stewards {
	n := 0
	return ask.Stewards{Agent: a, Store: &stq{m: map[string]ask.StewardQuestion{}}, Audit: &rows{},
		Policy: policy.Kernel{Engine: allowAll{}, Recorder: rec{}},
		Facts:  func(string) ([]string, string, bool) { return []string{"group:ops"}, "", true },
		NewID:  func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: func() time.Time { return now }}
}

func inOps(t *testing.T, user string) tenancy.Scope {
	s, err := tenancy.NewIssuer(tenant).IssueIn(tenant, user, nil, "spc_ops")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func cited(r ask.Response) string {
	var out []string
	for _, c := range r.Answer.Claims {
		for _, ci := range c.Cites {
			out = append(out, ci.Cite+"<"+ci.Replaces+">")
		}
	}
	return strings.Join(out, " ")
}

// SB-2801: a retired document leaves its own space's answers, stays in another's, names
// itself on its replacement's cite, and comes back.
func TestRetireOnlyNarrowsItsSpace(t *testing.T) {
	ctx := context.Background()
	a := freshAgent(t, `{"claims":[{"text":"x","cites":["C1","C2","C3"]}]}`)
	ix := a.Index.(index)
	for i := range ix {
		ix[i].Title = strings.ToUpper(ix[i].SourceID)
	}
	ix[0].Spaces = []string{"spc_ops", "spc_hr"} // a: also held by HR
	a.Retired = retired{}
	x := stewardsOf(a, time.Now())
	sam := scope(t, "alice")
	if _, err := x.Retire(ctx, sam, "ops", "docs:a@3", " ", ""); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("no reason: %v", err)
	}
	if _, err := x.Retire(ctx, sam, "ops", "docs:salaries@3", "old", ""); !errors.Is(err, ask.ErrNotSteward) {
		t.Fatalf("a document the steward can't open: %v", err)
	}
	if _, err := x.Retire(ctx, sam, "ops", "docs:a@3", "superseded by the C rota", "docs:c"); err != nil {
		t.Fatal(err)
	}
	x.Agent = a
	resp, err := a.Ask(ctx, inOps(t, "alice"), ask.Request{Question: "refund?"})
	if err != nil {
		t.Fatal(err)
	}
	if got := cited(resp); strings.Contains(got, "docs:a@") || !strings.Contains(got, "docs:c@3<A>") {
		t.Fatalf("in ops: %s", got)
	}
	resp, _ = a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund?"})
	if got := cited(resp); !strings.Contains(got, "docs:a@3") {
		t.Fatalf("company-wide, HR still holds it: %s", got)
	}
	list, err := x.RetiredOf(ctx, sam, "ops")
	if err != nil || len(list.Items) != 1 || list.Items[0].Title != "A" || list.Items[0].ReplacedBy != "docs:c" {
		t.Fatalf("%+v %v", list, err)
	}
	if err := x.Restore(ctx, sam, "ops", "docs:a"); err != nil {
		t.Fatal(err)
	}
	if err := x.Restore(ctx, sam, "ops", "docs:a"); !errors.Is(err, ask.ErrNotSteward) {
		t.Fatalf("a second restore: %v", err)
	}
	resp, _ = a.Ask(ctx, inOps(t, "alice"), ask.Request{Question: "refund?"})
	if got := cited(resp); !strings.Contains(got, "docs:a@3") {
		t.Fatalf("brought back: %s", got)
	}
}

// SB-2802, SB-2803, SB-2804: same name with different text is grouped; never-cited old
// documents are listed, hidden ones only by owner; a note whose source moved comes back.
func TestTidyTheList(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	a := freshAgent(t, `{"claims":[]}`)
	ix := a.Index.(index)
	ix[0].Title, ix[0].TextSHA256, ix[0].Seen = "Leave policy", "h1", now.Add(-200*24*time.Hour)
	ix[1].Title, ix[1].TextSHA256, ix[1].Seen = "leave  Policy", "h2", now.Add(-100*24*time.Hour)
	ix[2].Title, ix[2].TextSHA256, ix[2].Seen = "Rota", "h3", now.Add(-24*time.Hour)
	ix[3].Title, ix[3].TextSHA256, ix[3].Seen, ix[3].Owner = "Leave policy", "h4", now.Add(-300*24*time.Hour), "ann@acme.example"
	note := ports.Chunk{ID: "chk_note", Tenant: tenant, Kind: "docs", SourceID: "notes/ops/sq_1.md", Version: 1, Cite: "docs:notes/ops/sq_1.md@1",
		Text: "Finance signs.", Location: "/notes/ops/sq_1", Spaces: []string{"spc_ops"}, Seen: now}
	newer := ix[2]
	newer.ID, newer.Version, newer.Cite, newer.Seen = "chk_c4", 4, "docs:c@4", now.Add(-time.Hour)
	ix = append(ix, note, newer)
	a.Index = ix
	a.Auth.Connectors["docs"].(source)["notes/ops/sq_1.md|alice"] = ports.VerdictAllow
	x := stewardsOf(a, now)
	x.Store.(*stq).m["sq_1"] = ask.StewardQuestion{ID: "sq_1", Space: "spc_ops", Reply: &ask.StewardReply{Text: "Finance signs.", Cites: []string{"docs:c@3"}, Note: "r"}}
	alice := scope(t, "alice")
	cites := reviewSrc{{Response: ask.Response{Answer: ask.Answer{Claims: nil}}}}
	res, err := x.Review(ctx, alice, "ops", cites, 365)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.SameName) != 1 || len(res.SameName[0].Items) != 2 || res.SameName[0].Hidden != 1 || res.SameName[0].Items[0].Cite != "docs:b@3" {
		t.Fatalf("same name: %+v", res.SameName)
	}
	moved := 0
	for _, it := range res.Items {
		if it.Cite == note.Cite {
			moved = it.SourcesMoved
		}
	}
	if moved != 1 {
		t.Fatalf("the note whose source moved: %+v", res.Items)
	}
	un, err := x.Uncited(ctx, alice, "ops", cites)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, r := range un.Items {
		got = append(got, r.Cite)
	}
	if fmt.Sprint(got) != "[docs:a@3 docs:b@3]" || len(un.Hidden) != 1 || un.Hidden[0].Owner != "ann@acme.example" {
		t.Fatalf("never cited: %v %+v", got, un.Hidden)
	}
}

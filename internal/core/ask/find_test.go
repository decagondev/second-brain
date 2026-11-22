package ask_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// SB-1701: a search returns allowed documents, each once, with no model call.
func TestSearchNeedsNoModel(t *testing.T) {
	m := &model{err: errors.New("the model is down")}
	a, _ := setup(t, m)
	seen := time.Now().Add(-50 * time.Hour)
	ix := a.Index.(index)
	long := strings.Repeat("word ", 100)
	ix = append(ix, ports.Chunk{ID: "chk_refunds2", Tenant: tenant, Kind: "docs", SourceID: "refunds", Version: 2, Cite: "docs:refunds@2", Text: "second chunk"})
	ix[0].Text, ix[0].Title, ix[0].Seen, ix[0].Owner = long, "Refund policy", seen, "ann@acme-logistics.example"
	a.Index = ix
	a.Owner = func(_ context.Context, e string) (answer.Person, bool) {
		return answer.Person{Email: e, Title: "Support lead"}, true
	}
	res, err := a.Search(context.Background(), scope(t, "alice"), ask.SearchRequest{Query: "refund"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.got) != 0 {
		t.Fatal("search called the model")
	}
	if len(res.Results) != 1 || res.Refused != 1 {
		t.Fatalf("results %+v refused %d", res.Results, res.Refused)
	}
	r := res.Results[0]
	if r.Cite != "docs:refunds@2" || r.Title != "Refund policy" || r.AgeDays == nil || *r.AgeDays != 2 || r.Owner == nil || r.Owner.Title != "Support lead" {
		t.Fatalf("result %+v", r)
	}
	if n := len([]rune(r.Snippet)); n > ask.SnippetLen+1 || !strings.HasSuffix(r.Snippet, "…") {
		t.Fatalf("snippet of %d characters: %q", n, r.Snippet)
	}
	tr, err := a.Trace(context.Background(), scope(t, "alice"), res.Trace)
	if err != nil || tr.Response.Agent != ask.AgentSearch || tr.QuerySHA256 == "" || len(tr.Decisions) != 2 {
		t.Fatalf("trace %+v %v", tr, err)
	}
}

func TestSearchRefusesLikeAsk(t *testing.T) {
	a, _ := setup(t, &model{})
	ctx := context.Background()
	for _, q := range []string{"", strings.Repeat("x", ask.MaxQuestion+1)} {
		if _, err := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Query: q}); !errors.Is(err, ask.ErrInvalid) {
			t.Errorf("query of %d: %v", len(q), err)
		}
	}
	if _, err := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Query: "x", Sources: []string{"mail"}}); !errors.Is(err, ask.ErrInvalid) {
		t.Errorf("unknown kind: %v", err)
	}
	a.Killed = func(context.Context) bool { return true }
	if _, err := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Query: "x"}); !errors.Is(err, ask.ErrKilled) {
		t.Errorf("killed: %v", err)
	}
	a.Killed = nil
	if _, err := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Query: "x"}); !errors.Is(err, ask.ErrKilled) {
		t.Errorf("no kill reader: %v", err)
	}
}

// SB-1702: owners on allowed cites only, active ones only, most cited first.
func TestAskNamesOwnersOfAllowedCites(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]},{"text":"Again.","cites":["C1","C2"]}]}`}
	a, _ := setup(t, m)
	ix := a.Index.(index)
	ix[0].Owner, ix[1].Owner = "ann@acme-logistics.example", "bob@acme-logistics.example"
	ix = append(ix, ports.Chunk{ID: "chk_faq", Tenant: tenant, Kind: "docs", SourceID: "faq", Version: 1, Cite: "docs:faq@1", Text: "faq", Owner: "gone@acme-logistics.example"})
	a.Index = ix
	a.Auth.Connectors["docs"].(source)["faq|alice"] = ports.VerdictAllow
	users := map[string][]ports.User{
		"ann@acme-logistics.example":  {{ID: "usr_a", Active: true, Title: "Support lead"}},
		"bob@acme-logistics.example":  {{ID: "usr_b", Active: true}},
		"gone@acme-logistics.example": {{ID: "usr_g", Active: false}},
	}
	a.Owner = ask.ActiveOwner(func(_ context.Context, e string) ([]ports.User, error) { return users[e], nil })
	resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund?"})
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(resp.People); got != "[{ann@acme-logistics.example Support lead 2}]" {
		t.Fatalf("people %s", got) // bob's document was refused; gone's owner is deprovisioned
	}
	for _, c := range resp.Answer.Claims {
		for _, ci := range c.Cites {
			if ci.Cite == "docs:faq@1" && ci.Owner != nil {
				t.Fatal("a deprovisioned owner was shown")
			}
		}
	}
}

func TestActiveOwnerNeedsExactlyOne(t *testing.T) {
	two := []ports.User{{Active: true}, {Active: true}}
	for name, c := range map[string]struct {
		us  []ports.User
		err error
	}{"none": {}, "two": {us: two}, "error": {err: errors.New("down")}} {
		if _, ok := ask.ActiveOwner(func(context.Context, string) ([]ports.User, error) { return c.us, c.err })(context.Background(), "x@y"); ok {
			t.Errorf("%s: an owner was found", name)
		}
	}
}

type rows struct{ got []audit.Event }

func (r *rows) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	r.got = append(r.got, e)
	return audit.Row{}, nil
}

// SB-1703: saved questions are the person's own; a share keeps the question only.
func TestSavedAndShared(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1_800_000_000, 0)
	st := &qstore{saved: map[string]ask.SavedQuestion{}, shares: map[string]ask.Share{}}
	au := &rows{}
	n := 0
	q := ask.Questions{Tenant: tenant, Store: st, Audit: au, Now: func() time.Time { return now },
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }}
	alice, bob := scope(t, "alice"), scope(t, "bob")
	sq, err := q.Save(ctx, alice, "  what is the refund limit?  ")
	if err != nil || sq.Question != "what is the refund limit?" {
		t.Fatalf("%+v %v", sq, err)
	}
	if mine, _ := q.Saved(ctx, alice); len(mine) != 1 {
		t.Fatalf("alice's saved %v", mine)
	}
	if theirs, _ := q.Saved(ctx, bob); len(theirs) != 0 {
		t.Fatalf("bob sees %v", theirs)
	}
	if err := q.Unsave(ctx, bob, sq.ID); !errors.Is(err, ask.ErrNoShare) {
		t.Fatalf("bob removed alice's question: %v", err)
	}
	sh, err := q.Share(ctx, alice, "what is the refund limit?")
	if err != nil || len(sh.ID) != 35 {
		t.Fatalf("%+v %v", sh, err)
	}
	if len(au.got) != 1 || au.got[0].Kind != "access" || au.got[0].Subject != sh.ID || au.got[0].PayloadSHA256 == "" ||
		strings.Contains(fmt.Sprint(au.got[0]), "refund") {
		t.Fatalf("audit %+v", au.got)
	}
	got, err := q.Open(ctx, bob, sh.ID)
	if err != nil || got.Question != "what is the refund limit?" {
		t.Fatalf("%+v %v", got, err)
	}
	other, _ := tenancy.NewIssuer("beta-freight").Issue("beta-freight", "carol", nil)
	if _, err := q.Open(ctx, other, sh.ID); !errors.Is(err, ask.ErrNoShare) {
		t.Fatalf("another tenant opened a share: %v", err)
	}
	if _, err := q.Open(ctx, bob, "sh_"+strings.Repeat("0", 32)); !errors.Is(err, ask.ErrNoShare) {
		t.Fatalf("unknown share: %v", err)
	}
}

// qstore is a QuestionStore in a map; the adapters' own tests cover retention and erasure.
type qstore struct {
	saved  map[string]ask.SavedQuestion
	shares map[string]ask.Share
}

func (q *qstore) PutSaved(_ context.Context, s ask.SavedQuestion) error {
	q.saved[s.ID] = s
	return nil
}
func (q *qstore) SavedOf(_ context.Context, tenant, user string) ([]ask.SavedQuestion, error) {
	var out []ask.SavedQuestion
	for _, s := range q.saved {
		if s.Tenant == tenant && s.User == user {
			out = append(out, s)
		}
	}
	return out, nil
}
func (q *qstore) DeleteSaved(_ context.Context, tenant, user, id string) (bool, error) {
	s, ok := q.saved[id]
	if !ok || s.Tenant != tenant || s.User != user {
		return false, nil
	}
	delete(q.saved, id)
	return true, nil
}
func (q *qstore) PutShare(_ context.Context, sh ask.Share) error { q.shares[sh.ID] = sh; return nil }
func (q *qstore) GetShare(_ context.Context, tenant, id string) (ask.Share, bool, error) {
	sh, ok := q.shares[id]
	return sh, ok && sh.Tenant == tenant, nil
}

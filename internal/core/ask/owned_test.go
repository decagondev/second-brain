package ask_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

func citing(at time.Time, space string, stale bool, docs ...string) ask.Trace {
	var cs []answer.Cite
	for _, d := range docs {
		cs = append(cs, answer.Cite{Cite: d})
	}
	tr := ask.Trace{At: at, Space: space, Response: ask.Response{Agent: "ask", Answer: ask.Answer{Claims: []answer.Claim{{Text: "x", Cites: cs}}}}}
	if stale {
		tr.Response.NoticeStale = &ask.StaleNotice{Stale: 1, Of: 1}
	}
	return tr
}

// SB-2806: an owner's often-cited stale documents are counted as they can open them; one
// not often cited, a fresh one, and one they can't open aren't; someone gone counts nothing.
func TestOwnedPastReview(t *testing.T) {
	a := freshAgent(t, `{"claims":[]}`)
	ix := a.Index.(index)
	for i := range ix {
		ix[i].Owner = "alice@acme.example"
	}
	ix[1].Owner = "gone@acme.example"
	a.Index = ix
	now := time.Now()
	var traces []ask.Trace
	for i := 0; i < ask.ReviewCites; i++ {
		traces = append(traces, citing(now, "spc_ops", false, "docs:a@3", "docs:b@3", "docs:c@3", "docs:salaries@3"))
	}
	scopeOf := func(_ context.Context, email string) (tenancy.Scope, bool, error) {
		if email != "alice@acme.example" {
			return tenancy.Scope{}, false, nil
		}
		return scope(t, "alice"), true, nil
	}
	got, err := a.OwnedPastReview(context.Background(), traces, scopeOf)
	if err != nil || !reflect.DeepEqual(got, map[string]int{"alice": 1}) {
		t.Fatalf("%v %v", got, err)
	}
	if got, _ := a.OwnedPastReview(context.Background(), traces[1:], scopeOf); len(got) != 0 {
		t.Fatalf("cited fewer than %d times: %v", ask.ReviewCites, got)
	}
}

// SB-2805, SB-2807: the never-cited section names terms no question used; the review list
// counts the month's stale answers, and last month's, in its own space only.
func TestUnusedTermsAndFreshness(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	a := freshAgent(t, `{"claims":[]}`)
	a.Glossary = func(string) map[string]string {
		return map[string]string{"COD": "cash on delivery", "ETA": "estimated arrival", "POD": "proof of delivery"}
	}
	used := citing(now.Add(-24*time.Hour), "spc_ops", true, "docs:c@3")
	used.Response.Terms = []ask.Term{{Term: "cod", Means: "cash on delivery"}}
	elsewhere := citing(now.Add(-24*time.Hour), "spc_hr", true, "docs:c@3")
	elsewhere.Response.Terms = []ask.Term{{Term: "ETA"}}
	src := reviewSrc{used, elsewhere, citing(now.Add(-48*time.Hour), "spc_ops", false, "docs:c@3"),
		citing(now.AddDate(0, -1, 0), "spc_ops", true, "docs:c@3"),
		{At: now, Space: "spc_ops", Response: ask.Response{Agent: "ask"}}} // cites nothing: not counted
	x := stewardsOf(a, now)
	alice := scope(t, "alice")
	un, err := x.Uncited(ctx, alice, "ops", src)
	if err != nil || !reflect.DeepEqual(un.UnusedTerms, []string{"ETA", "POD"}) {
		t.Fatalf("unused terms: %v %v", un.UnusedTerms, err)
	}
	res, err := x.Review(ctx, alice, "ops", src, 365)
	if err != nil {
		t.Fatal(err)
	}
	want := ask.Freshness{ThisMonth: ask.StaleCount{Answers: 2, Stale: 1}, LastMonth: ask.StaleCount{Answers: 1, Stale: 1}}
	if res.Freshness != want {
		t.Fatalf("freshness: %+v", res.Freshness)
	}
	if got := ask.StaleShare(src, ""); got != (ask.StaleCount{Answers: 4, Stale: 3}) {
		t.Fatalf("the whole cell: %+v", got)
	}
}

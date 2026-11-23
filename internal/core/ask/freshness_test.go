package ask_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

type marks []ask.ReviewMark

func (m marks) PutReviewMark(context.Context, ask.ReviewMark) error { return nil }
func (m marks) ReviewMarks(_ context.Context, _, space string) ([]ask.ReviewMark, error) {
	var out []ask.ReviewMark
	for _, x := range m {
		if x.Space == space {
			out = append(out, x)
		}
	}
	return out, nil
}

// freshAgent answers from three allowed documents, a and b old, c new, and a hidden old one.
func freshAgent(t *testing.T, reply string) ask.Agent {
	a, _ := setup(t, &model{reply: reply})
	now := time.Now()
	old, fresh := now.Add(-400*24*time.Hour), now.Add(-time.Hour)
	var ix index
	for _, d := range []struct {
		id   string
		seen time.Time
	}{{"a", old}, {"b", old}, {"c", fresh}, {"salaries", old}} {
		ix = append(ix, ports.Chunk{ID: "chk_" + d.id, Tenant: tenant, Kind: "docs", SourceID: d.id, Version: 3, Cite: "docs:" + d.id + "@3",
			Text: "refund " + d.id, Seen: d.seen, Spaces: []string{"spc_ops"}})
	}
	a.Index = ix
	src := a.Auth.Connectors["docs"].(source)
	for _, id := range []string{"a", "b", "c"} {
		src[id+"|alice"] = ports.VerdictAllow
	}
	return a
}

// SB-2701: more than half of the cited documents stale warns; a hidden one changes nothing.
func TestOldSourceWarning(t *testing.T) {
	ctx := context.Background()
	two := freshAgent(t, `{"claims":[{"text":"x","cites":["C1","C2"]},{"text":"y","cites":["C1","C3"]}]}`)
	resp, err := two.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund?"})
	if err != nil {
		t.Fatal(err)
	}
	n := resp.NoticeStale
	if n == nil || n.Stale != 2 || n.Of != 3 || n.OldestDays < 399 {
		t.Fatalf("2 of 3 stale: %+v (claims %+v)", n, resp.Answer.Claims)
	}
	if !strings.Contains(ask.StaleLine(*n), "Check with their owner") {
		t.Fatal(ask.StaleLine(*n))
	}
	one := freshAgent(t, `{"claims":[{"text":"x","cites":["C1","C2","C3"]}]}`)
	all := one.Index.(index)
	one.Index = index{all[0], all[2], all[3]} // a (old), c (new), salaries (old, hidden)
	resp, err = one.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund?"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NoticeStale != nil || resp.Refused != 1 {
		t.Fatalf("1 of 2 stale, a hidden old one: %+v refused %d", resp.NoticeStale, resp.Refused)
	}
}

// SB-2702: a review mark at the cite's version shows and takes it off stale; another
// version's mark doesn't.
func TestReviewedOnTheCite(t *testing.T) {
	a := freshAgent(t, `{"claims":[{"text":"x","cites":["C1","C2"]}]}`)
	at := time.Now().Add(-48 * time.Hour)
	a.Marks = marks{{Space: "spc_ops", Doc: "docs:a", Version: 3, At: at}, {Space: "spc_ops", Doc: "docs:b", Version: 2, At: at}}
	resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund?"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, ci := range resp.Answer.Claims[0].Cites {
		got[ci.Cite] = ci.Reviewed + map[bool]string{true: " stale", false: ""}[ci.Stale]
	}
	if got["docs:a@3"] != at.UTC().Format("2006-01-02") || got["docs:b@3"] != " stale" {
		t.Fatalf("%v", got)
	}
	if resp.NoticeStale != nil {
		t.Fatalf("1 of 2 stale after the mark: %+v", resp.NoticeStale)
	}
}

// SB-2704: a connector behind is named; one the cell can't read about is unknown, never fresh.
func TestSyncLagNotice(t *testing.T) {
	a := freshAgent(t, `{"claims":[{"text":"x","cites":["C1"]}]}`)
	at := time.Date(2027, 3, 10, 14, 5, 0, 0, time.UTC)
	a.SyncLag = func(_ context.Context, kinds []string) ([]ask.SyncNotice, error) {
		if len(kinds) != 1 || kinds[0] != "docs" {
			t.Fatalf("kinds %v", kinds)
		}
		return []ask.SyncNotice{{Kind: "docs", LastSuccess: &at}}, nil
	}
	resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund?"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.NoticeSync) != 1 || ask.SyncLine(resp.NoticeSync[0]) != "docs last synced at 14:05 UTC; newer changes may be missing." {
		t.Fatalf("%+v", resp.NoticeSync)
	}
	a.SyncLag = func(context.Context, []string) ([]ask.SyncNotice, error) { return nil, errors.New("unreadable") }
	resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund?"})
	if len(resp.NoticeSync) != 1 || resp.NoticeSync[0].LastSuccess != nil || !strings.Contains(ask.SyncLine(resp.NoticeSync[0]), "unknown") {
		t.Fatalf("an unreadable record: %+v", resp.NoticeSync)
	}
}

// SB-2705: a past review-by date makes a new document stale, a mark or not; 2099 changes nothing.
func TestReviewByMakesStaleSooner(t *testing.T) {
	a := freshAgent(t, `{"claims":[{"text":"x","cites":["C1"]}]}`)
	ix := a.Index.(index)
	c := ix[2] // "c", seen an hour ago
	c.ReviewBy = time.Now().Add(-time.Hour)
	far := ix[2]
	far.ID, far.SourceID, far.Cite, far.ReviewBy = "chk_far", "far", "docs:far@3", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	a.Index = index{c, far}
	a.Auth.Connectors["docs"].(source)["far|alice"] = ports.VerdictAllow
	a.Marks = marks{{Space: "spc_ops", Doc: "docs:c", Version: 3, At: time.Now()}}
	for q, want := range map[string]bool{"refund c": true, "refund far": false} {
		a.Index = index{map[bool]ports.Chunk{true: c, false: far}[want]}
		resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: q})
		if err != nil {
			t.Fatal(err)
		}
		if got := resp.Answer.Claims[0].Cites[0].Stale; got != want {
			t.Errorf("%s: stale %v, want %v", q, got, want)
		}
	}
}

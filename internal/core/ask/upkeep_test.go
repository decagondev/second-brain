package ask_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

type markStore struct {
	mu sync.Mutex
	m  []ask.ReviewMark
}

func (s *markStore) PutReviewMark(_ context.Context, m ask.ReviewMark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.m {
		if x.Space == m.Space && x.Doc == m.Doc {
			s.m[i] = m
			return nil
		}
	}
	s.m = append(s.m, m)
	return nil
}

func (s *markStore) ReviewMarks(_ context.Context, _, space string) ([]ask.ReviewMark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.ReviewMark
	for _, x := range s.m {
		if x.Space == space {
			out = append(out, x)
		}
	}
	return out, nil
}

func upkeepAgent(t *testing.T, now time.Time) (ask.Agent, reviewSrc) {
	a, _ := setup(t, &model{reply: `{"claims":[]}`})
	day := 24 * time.Hour
	var ix index
	var src reviewSrc
	for _, d := range []struct {
		id       string
		seen     time.Time
		reviewBy time.Time
		cites    int
	}{
		{"a", now.Add(-20 * day), time.Time{}, ask.ReviewCites},         // due in 10
		{"b", now.Add(-10 * day), time.Time{}, ask.ReviewCites},         // due in 20
		{"c", now.Add(-40 * day), time.Time{}, ask.ReviewCites},         // stale already
		{"e", now.Add(-day), now.Add(3 * day), ask.ReviewCites},         // its own date, in 3
		{"f", now.Add(-20 * day), time.Time{}, 1},                       // hardly cited
		{"salaries", now.Add(-20 * day), time.Time{}, ask.ReviewCites},  // alice can't open it
		{"elsewhere", now.Add(-20 * day), time.Time{}, ask.ReviewCites}, // another space's
	} {
		sp := "spc_ops"
		if d.id == "elsewhere" {
			sp = "spc_hr"
		}
		ix = append(ix, ports.Chunk{ID: "chk_" + d.id, Tenant: tenant, Kind: "docs", SourceID: d.id, Version: 3, Cite: "docs:" + d.id + "@3",
			Text: "refund " + d.id, Seen: d.seen, ReviewBy: d.reviewBy, Spaces: []string{sp}, Owner: "alice@acme.example"})
		for i := 0; i < d.cites; i++ {
			src = append(src, citing(now.Add(-time.Hour), sp, false, "docs:"+d.id+"@3"))
		}
	}
	ix[1].Owner = "bob@acme.example"
	a.Index = ix
	allow := a.Auth.Connectors["docs"].(source)
	for _, id := range []string{"a", "b", "c", "e", "f", "elsewhere"} {
		allow[id+"|alice"] = ports.VerdictAllow
	}
	a.Emails = func(_ context.Context, u string) ([]string, error) { return []string{u + "@ACME.example"}, nil }
	return a, src
}

// SB-3105: due in 10 days is listed, 20 isn't; a document's own date counts; soonest
// first; one the steward can't open is counted only; a mark moves the date on.
func TestComingDue(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	a, src := upkeepAgent(t, now)
	x := stewardsOf(a, now)
	x.Marks = &markStore{}
	alice := scope(t, "alice")
	got, err := x.ComingDue(ctx, alice, "ops", src, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.Items[0].Cite != "docs:e@3" || got.Items[0].DueInDays != 3 ||
		got.Items[1].Cite != "docs:a@3" || got.Items[1].DueInDays != 10 || len(got.Hidden) != 1 {
		t.Fatalf("due: %+v hidden %+v", got.Items, got.Hidden)
	}
	if got.Items[0].Snippet != "" {
		t.Fatal("a snippet on the review list")
	}
	if _, err := x.MarkReviewed(ctx, alice, "ops", "docs:a@3", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := x.MarkReviewed(ctx, alice, "ops", "docs:e@3", 0); err != nil {
		t.Fatal(err)
	}
	got, _ = x.ComingDue(ctx, alice, "ops", src, 30)
	if len(got.Items) != 1 || got.Items[0].Cite != "docs:e@3" {
		t.Fatalf("after the marks (a moves on, e's own date doesn't): %+v", got.Items)
	}
	if _, err := x.ComingDue(ctx, scope(t, "bob"), "company", src, 30); !errors.Is(err, ask.ErrNotSteward) {
		t.Fatalf("company-wide: %v", err)
	}
}

// SB-3106: the owner marks their own document in each space that holds it, audited as
// them; someone else's, one they can't open and an owner the directory no longer knows
// are EPERM; a version that isn't current is EINVAL.
func TestOwnerMarksTheirOwn(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	a, _ := upkeepAgent(t, now)
	x := stewardsOf(a, now)
	st := &markStore{}
	x.Marks = st
	alice := scope(t, "alice")
	m, err := x.MarkOwn(ctx, alice, "docs:a@3", 0)
	if err != nil || m.Doc != "docs:a" || len(m.Spaces) != 1 || m.Spaces[0] != "ops" {
		t.Fatalf("%+v %v", m, err)
	}
	if len(st.m) != 1 || st.m[0].By != "alice" || st.m[0].Space != "spc_ops" {
		t.Fatalf("marks: %+v", st.m)
	}
	rs := x.Audit.(*rows).got
	if len(rs) != 1 || rs[0].Actor != "alice" || rs[0].State != "reviewed" || rs[0].Reason != "reviewed by its owner" {
		t.Fatalf("audit: %+v", rs)
	}
	for _, c := range []string{"docs:b@3", "docs:salaries@3", "docs:nope@1"} {
		if _, err := x.MarkOwn(ctx, alice, c, 0); !errors.Is(err, ask.ErrNotOwner) {
			t.Fatalf("%s: %v", c, err)
		}
	}
	if _, err := x.MarkOwn(ctx, alice, "docs:a@2", 0); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("an old version: %v", err)
	}
	x.Agent.Emails = func(context.Context, string) ([]string, error) { return nil, errors.New("directory down") }
	if _, err := x.MarkOwn(ctx, alice, "docs:a@3", 0); !errors.Is(err, ask.ErrNotOwner) {
		t.Fatalf("an unknown directory: %v", err)
	}
	if len(st.m) != 1 {
		t.Fatalf("a refused mark was kept: %+v", st.m)
	}
}

// SB-3106: a search result is mine only when its owner is one of my addresses now.
func TestSearchSaysMine(t *testing.T) {
	a, _ := upkeepAgent(t, time.Now().UTC())
	res, err := a.Search(context.Background(), scope(t, "alice"), ask.SearchRequest{Query: "refund"})
	if err != nil {
		t.Fatal(err)
	}
	mine := map[string]bool{}
	for _, r := range res.Results {
		mine[r.Cite] = r.Mine
	}
	if len(mine) == 0 || !mine["docs:a@3"] || mine["docs:b@3"] {
		t.Fatalf("%v", mine)
	}
}

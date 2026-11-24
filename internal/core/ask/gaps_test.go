package ask_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

type fbSrc struct {
	traces []ask.Trace
	fbs    []ask.Feedback
}

func (f fbSrc) Between(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	return f.traces, nil
}
func (f fbSrc) FeedbackBetween(_ context.Context, _ string, from, to time.Time) ([]ask.Feedback, error) {
	var out []ask.Feedback
	for _, x := range f.fbs {
		if !x.At.Before(from) && !x.At.After(to) {
			out = append(out, x)
		}
	}
	return out, nil
}

func noticed(at time.Time, space string, n string, terms ...string) ask.Trace {
	tr := ask.Trace{At: at, Space: space, Response: ask.Response{Agent: "ask", Notice: &n}}
	for _, term := range terms {
		tr.Response.Terms = append(tr.Response.Terms, ask.Term{Term: term})
	}
	return tr
}

// SB-3101: answers that found nothing are counted by week in their own space, with the
// glossary terms used, most used first; other notices and other spaces don't count.
func TestFoundNothing(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	src := fbSrc{traces: []ask.Trace{
		noticed(now.Add(-day), "spc_ops", ask.NoticeNoSources, "rota", "cod"),
		noticed(now.Add(-2*day), "spc_ops", ask.NoticeMostUncited, "cod"),
		noticed(now.Add(-9*day), "spc_ops", ask.NoticeNoSources),
		noticed(now.Add(-20*day), "spc_ops", ask.NoticeNoSources), // older than two weeks
		noticed(now.Add(-day), "spc_hr", ask.NoticeNoSources, "pay"),
		citing(now.Add(-day), "spc_ops", false, "docs:a@3"),
	}}
	a := freshAgent(t, `{"claims":[]}`)
	res, err := stewardsOf(a, now).Review(ctx, scope(t, "alice"), "ops", src, 365)
	if err != nil {
		t.Fatal(err)
	}
	want := ask.FoundNothing{ThisWeek: 2, LastWeek: 1, Terms: []string{"cod", "rota"}}
	if !reflect.DeepEqual(res.FoundNothing, want) {
		t.Fatalf("found nothing: %+v", res.FoundNothing)
	}
	by := ask.FoundNothingBySpace(src.traces, now.Add(-7*day), now)
	if !reflect.DeepEqual(by, map[string]int{"spc_ops": 2, "spc_hr": 1}) {
		t.Fatalf("by space: %v", by)
	}
}

// SB-3104: ratings this month and last count only answers asked in the space, and the
// documents unhelpful answers cited are checked as the steward: one they can't open is
// counted, never named.
func TestStewardFeedback(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	t1 := citing(now.Add(-time.Hour), "spc_ops", false, "docs:a@3", "docs:salaries@3")
	t1.ID = "tr_1"
	t2 := citing(now.Add(-2*time.Hour), "spc_ops", false, "docs:a@3")
	t2.ID = "tr_2"
	t3 := citing(now.AddDate(0, -1, 0), "spc_ops", false, "docs:b@3")
	t3.ID = "tr_3"
	t4 := citing(now.Add(-time.Hour), "spc_hr", false, "docs:c@3")
	t4.ID = "tr_4"
	src := fbSrc{traces: []ask.Trace{t1, t2, t3, t4}, fbs: []ask.Feedback{
		{Trace: "tr_1", Rating: "down", At: now.Add(-time.Hour)},
		{Trace: "tr_2", Rating: "up", At: now.Add(-time.Hour)},
		{Trace: "tr_3", Rating: "down", At: now.AddDate(0, -1, 0)},
		{Trace: "tr_4", Rating: "down", At: now.Add(-time.Hour)},
	}}
	a := freshAgent(t, `{"claims":[]}`)
	res, err := stewardsOf(a, now).Review(ctx, scope(t, "alice"), "ops", src, 365)
	if err != nil {
		t.Fatal(err)
	}
	f := res.Feedback
	if f.ThisMonth != (ask.Ratings{Helpful: 1, NotHelpful: 1}) || f.LastMonth != (ask.Ratings{NotHelpful: 1}) {
		t.Fatalf("ratings: %+v", f)
	}
	if len(f.Documents) != 1 || f.Documents[0].Cite != "docs:a@3" || f.DocumentsHidden != 1 {
		t.Fatalf("documents: %+v hidden %d", f.Documents, f.DocumentsHidden)
	}
}

// SB-3102: a question from a space with no active steward goes to the company's stewards
// and says where it came from; a directory that can't answer keeps it in its space.
func TestNoStewardReroute(t *testing.T) {
	ctx := context.Background()
	a, _ := setup(t, &model{reply: `{"claims":[]}`})
	notice := ask.NoticeNoSources
	_ = a.Store.PutTrace(ctx, ask.Trace{ID: "tr_q", Tenant: tenant, User: "alice", Space: "spc_ops", Response: ask.Response{Notice: &notice}, At: time.Now()})
	alice := inOps(t, "alice")
	for _, c := range []struct {
		name        string
		none, known bool
		err         error
		space, from string
	}{
		{"nobody", true, true, nil, "company", "ops"},
		{"someone", false, true, nil, "spc_ops", ""},
		{"unknown", true, false, nil, "spc_ops", ""},
		{"failed", true, true, errors.New("directory down"), "spc_ops", ""},
	} {
		x := stewardsOf(a, time.Now())
		x.Unstewarded = func(context.Context, string) (bool, bool, error) { return c.none, c.known, c.err }
		sq, err := x.Send(ctx, alice, "who covers nights?", "tr_q")
		if err != nil || sq.Space != c.space || sq.FromSpace != c.from {
			t.Fatalf("%s: %+v %v", c.name, sq, err)
		}
	}
}

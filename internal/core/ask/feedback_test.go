package ask_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

type fbStore struct{ got map[string]ask.Feedback }

func (s *fbStore) PutFeedback(_ context.Context, f ask.Feedback) error {
	s.got[f.Trace] = f
	return nil
}

func (s *fbStore) FeedbackBetween(_ context.Context, tenant string, from, to time.Time) ([]ask.Feedback, error) {
	var out []ask.Feedback
	for _, f := range s.got {
		if f.Tenant == tenant && !f.At.Before(from) && f.At.Before(to) {
			out = append(out, f)
		}
	}
	return out, nil
}

func TestFeedback(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]},{"text":"Ask finance.","cites":[]}]}`}
	a, _ := setup(t, m)
	ctx := context.Background()
	resp, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	st := &fbStore{got: map[string]ask.Feedback{}}
	f := ask.Feedbacks{Agent: a, Store: st}
	bad := []struct {
		user string
		req  ask.FeedbackRequest
		want error
	}{
		{"alice", ask.FeedbackRequest{Trace: resp.Trace, Rating: "meh"}, ask.ErrInvalid},
		{"alice", ask.FeedbackRequest{Trace: resp.Trace, Rating: "down", WrongCitation: "docs:salaries@2"}, ask.ErrInvalid},
		{"alice", ask.FeedbackRequest{Trace: resp.Trace, Rating: "down", Comment: strings.Repeat("x", ask.MaxComment+1)}, ask.ErrInvalid},
		{"bob", ask.FeedbackRequest{Trace: resp.Trace, Rating: "up"}, ask.ErrNotFound},
		{"alice", ask.FeedbackRequest{Trace: "tr_nope", Rating: "up"}, ask.ErrNotFound},
	}
	for _, b := range bad {
		if err := f.Give(ctx, scope(t, b.user), b.req); !errors.Is(err, b.want) {
			t.Errorf("%s %+v: got %v, want %v", b.user, b.req, err, b.want)
		}
	}
	if len(st.got) != 0 {
		t.Fatal("refused feedback was stored")
	}
	if err := f.Give(ctx, scope(t, "alice"), ask.FeedbackRequest{Trace: resp.Trace, Rating: "up"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Give(ctx, scope(t, "alice"), ask.FeedbackRequest{Trace: resp.Trace, Rating: "down", WrongCitation: "docs:refunds@2", Comment: "wrong doc"}); err != nil {
		t.Fatal(err)
	}
	got := st.got[resp.Trace]
	if len(st.got) != 1 || got.Rating != "down" || got.WrongCitation != "docs:refunds@2" || got.Claims != 2 || got.Uncited != 1 || got.Refused != 1 || got.Release != "rel_test" {
		t.Fatalf("stored %+v", got)
	}

	var buf bytes.Buffer
	from, to := got.At.Add(-time.Hour), got.At.Add(time.Hour)
	n, err := f.Export(ctx, "acme-logistics", from, to, &buf)
	if err != nil || n != 1 {
		t.Fatalf("export: %d %v", n, err)
	}
	line := buf.String()
	for _, banned := range []string{"refund limit?", "Limit is 500", "Ask finance"} {
		if strings.Contains(line, banned) {
			t.Errorf("export carries question or answer text %q: %s", banned, line)
		}
	}
	if !strings.Contains(line, `"wrong_citation":"docs:refunds@2"`) || !strings.Contains(line, `"comment":"wrong doc"`) {
		t.Errorf("export: %s", line)
	}
}

func TestWeek(t *testing.T) {
	// Wednesday 2026-11-18 15:00 UTC: the window is Mon 2026-11-09 to Mon 2026-11-16.
	from, to := ask.Week(time.Date(2026, 11, 18, 15, 0, 0, 0, time.UTC))
	if !from.Equal(time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)) || !to.Equal(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("%v %v", from, to)
	}
	// On a Monday the window ends that midnight.
	_, to = ask.Week(time.Date(2026, 11, 16, 0, 0, 1, 0, time.UTC))
	if !to.Equal(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("monday: %v", to)
	}
}

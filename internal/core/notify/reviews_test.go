package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/reviews"
)

// SB-3504: the stewards of a restricted space hear once when its access review opens and
// once when it becomes overdue; a complete review tells no one, and a message holds counts
// and dates, never who.
func TestAccessReviewsTellTheStewardsOnce(t *testing.T) {
	opened := time.Date(2027, 1, 1, 5, 0, 0, 0, time.UTC)
	j, s, _ := world(opened.Add(time.Hour))
	j.Questions = questions{}
	rs := []reviews.Review{
		{ID: "rev_1", Space: "spc_finance", Stewards: []string{"group:finance-leads"}, State: reviews.Open,
			Opened: opened.Format(time.RFC3339), Due: opened.Add(reviews.Window).Format(time.RFC3339),
			Items: []reviews.Item{{User: "usr_ann"}, {User: "usr_bob"}}},
		{ID: "rev_2", Space: "spc_hr", Stewards: []string{"group:finance-leads"}, State: reviews.Complete,
			Opened: opened.Format(time.RFC3339), Due: opened.Add(reviews.Window).Format(time.RFC3339)},
	}
	j.Reviews = func(context.Context) ([]reviews.Review, error) { return rs, nil }
	tell := func(at time.Time) string {
		t.Helper()
		j.Now = func() time.Time { return at }
		s.got = nil
		if _, err := j.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, m := range s.got {
			if strings.Contains(m.msg.Text, "usr_") {
				t.Fatalf("a person in a message: %q", m.msg.Text)
			}
			got = append(got, m.to.ID+": "+m.msg.Kind+" "+m.msg.Text+" "+m.msg.Link)
		}
		return strings.Join(got, "|")
	}
	link := " https://brain.acme.example/?steward=finance"
	if got := tell(opened.Add(time.Hour)); got != "usr_sam: questions The finance access review is open: 2 people, due 2027-01-15"+link {
		t.Fatalf("opened: %q", got)
	}
	if got := tell(opened.Add(2 * time.Hour)); got != "" {
		t.Fatalf("told twice: %q", got)
	}
	if got := tell(opened.Add(reviews.Window + time.Hour)); got != "usr_sam: questions The finance access review is overdue"+link {
		t.Fatalf("overdue: %q", got)
	}
	if got := tell(opened.Add(reviews.Window + 2*time.Hour)); got != "" {
		t.Fatalf("overdue told twice: %q", got)
	}
	j.Reviews = func(context.Context) ([]reviews.Review, error) { return nil, context.DeadlineExceeded }
	j.Now = func() time.Time { return opened.Add(reviews.Window + 3*time.Hour) }
	if _, err := j.Run(context.Background()); err == nil {
		t.Fatal("reviews that can't be read were treated as none")
	}
}

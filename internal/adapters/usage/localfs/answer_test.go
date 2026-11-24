package localfs

import (
	"context"
	"reflect"
	"testing"
	"time"
)

type observed struct {
	counts
	seen []float64
}

func (o *observed) Observe(_, _ string, _ map[string]string, v float64) { o.seen = append(o.seen, v) }

// SB-3205: each answer's duration is kept in milliseconds and forwarded; other histograms
// are forwarded only.
func TestAnswerMillis(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	next := &observed{counts: counts{}}
	l := &Ledger{Root: t.TempDir(), Tenant: "acme-logistics", Next: next, Now: func() time.Time { return now }}
	l.Observe("sb_ask_seconds", "", nil, 1.2345)
	l.Observe("sb_other_seconds", "", nil, 9)
	now = now.Add(8 * 24 * time.Hour)
	l.Observe("sb_ask_seconds", "", nil, 2)
	if !reflect.DeepEqual(next.seen, []float64{1.2345, 9, 2}) {
		t.Fatalf("not forwarded: %v", next.seen)
	}
	got, err := l.AnswerMillis(context.Background(), "acme-logistics", now.Add(-10*24*time.Hour), now.Add(-7*24*time.Hour))
	if err != nil || !reflect.DeepEqual(got, []int{1235}) {
		t.Fatalf("%v %v", got, err)
	}
}

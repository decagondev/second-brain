package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/retention"
)

// LAW-13 under retention (SB-611): the nightly retention job and subject erasure delete
// derived records (traces, conversations, feedback) exactly as sb-tenant(5) says, and never
// touch the index: after a sweep and an erasure the index's (cite, ACL) set is unchanged
// and a rebuild from the sources still gives the same set.

// RetentionWorld is an ingest world plus the cell's stores of derived data and its job.
type RetentionWorld interface {
	Ingest() IngestWorld
	Record(user string, at time.Time) // one answer, its conversation and its feedback
	Sweep(now time.Time, days int) error
	Erase(user string) error
	// Held returns the times of the records the stores still hold for a user.
	Held(user string) []time.Time
}

// NewRetentionWorld builds the system under test: the real job over the in-memory
// derived stores, beside the real ingest pipeline and index.
var NewRetentionWorld = func() RetentionWorld {
	return newRetentionWorld(func(*retentionWorld) []retention.Store { return nil })
}

type retentionWorld struct {
	in     *ingestWorld
	traces *tracemem.Store
	extra  func(*retentionWorld) []retention.Store
	recs   map[string][]time.Time // user -> times recorded (the model)
	ids    map[string][]string    // user -> trace ids
	n      int
}

func newRetentionWorld(extra func(*retentionWorld) []retention.Store) *retentionWorld {
	return &retentionWorld{in: NewIngestWorld().(*ingestWorld), traces: tracemem.New(), extra: extra,
		recs: map[string][]time.Time{}, ids: map[string][]string{}}
}

func (w *retentionWorld) Ingest() IngestWorld { return w.in }

func (w *retentionWorld) Record(user string, at time.Time) {
	w.n++
	ctx := context.Background()
	id := fmt.Sprintf("tr_%d", w.n)
	w.traces.Now = func() time.Time { return at }
	_ = w.traces.PutTrace(ctx, ask.Trace{ID: id, Tenant: "acme-logistics", User: user, At: at})
	_ = w.traces.AppendQuestion(ctx, "acme-logistics", user, "cv_"+id, "q")
	_ = w.traces.PutFeedback(ctx, ask.Feedback{Tenant: "acme-logistics", User: user, Trace: id, At: at})
	w.recs[user] = append(w.recs[user], at)
	w.ids[user] = append(w.ids[user], id)
}

func (w *retentionWorld) job(now time.Time, days int) retention.Job {
	rows := &auditSink{}
	return retention.Job{Tenant: "acme-logistics", Days: days, Audit: rows, Now: func() time.Time { return now },
		Stores: append(w.traces.Retention(), w.extra(w)...)}
}

func (w *retentionWorld) Sweep(now time.Time, days int) error {
	_, err := w.job(now, days).Sweep(context.Background())
	return err
}

func (w *retentionWorld) Erase(user string) error {
	_, err := w.job(time.Now(), 365).Erase(context.Background(), "usr_op", user, "subject request")
	return err
}

func (w *retentionWorld) Held(user string) []time.Time {
	ctx := context.Background()
	var out []time.Time
	for i, id := range w.ids[user] {
		t, found, _ := w.traces.GetTrace(ctx, "acme-logistics", id)
		_, conv, _ := w.traces.Questions(ctx, "acme-logistics", user, "cv_"+id)
		fb, _ := w.traces.FeedbackBetween(ctx, "acme-logistics", w.recs[user][i], w.recs[user][i].Add(time.Nanosecond))
		mine := false
		for _, f := range fb {
			mine = mine || f.Trace == id
		}
		if found != conv || found != mine {
			return append(out, time.Time{}) // the stores disagree: a half-deleted record
		}
		if found {
			out = append(out, t.At)
		}
	}
	return out
}

type auditSink struct{ n int }

func (a *auditSink) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	a.n++
	return audit.Row{ID: fmt.Sprintf("aud_%012d", a.n), Kind: e.Kind}, nil
}

func law13Retention(r *rand.Rand, w RetentionWorld) error {
	var docs []SourceDoc
	for i := 0; i < 2+r.Intn(3); i++ {
		docs = append(docs, SourceDoc{ID: fmt.Sprintf("d%d", i), Text: strings.Repeat("freight invoice approval. ", 1+r.Intn(30)),
			Viewers: []string{"ann@acme.example", "bob@acme.example"}[:1+r.Intn(2)]})
	}
	in := w.Ingest()
	if err := in.Load(docs); err != nil {
		return err
	}
	if err := in.IngestAll(); err != nil {
		return err
	}
	before, err := in.Snapshot()
	if err != nil {
		return err
	}
	now := time.Date(2027, 3, 10, 3, 0, 0, 0, time.UTC)
	days := 30 + r.Intn(400)
	cut := now.Add(-time.Duration(days) * 24 * time.Hour)
	users := []string{"usr_ann", "usr_bob", "usr_cy"}
	want := map[string][]time.Time{}
	for i := 0; i < 3+r.Intn(10); i++ {
		u := users[r.Intn(len(users))]
		at := now.Add(-time.Duration(r.Intn(900*24)) * time.Hour)
		if r.Intn(6) == 0 {
			at = cut // exactly retention_days old: kept
		}
		w.Record(u, at)
		if !at.Before(cut) {
			want[u] = append(want[u], at)
		}
	}
	if err := w.Sweep(now, days); err != nil {
		return err
	}
	for _, u := range users {
		if got := w.Held(u); fmt.Sprint(got) != fmt.Sprint(want[u]) {
			return fmt.Errorf("after a sweep with retention_days %d, %s holds %v; want %v", days, u, got, want[u])
		}
	}
	gone := users[r.Intn(len(users))]
	if err := w.Erase(gone); err != nil {
		return err
	}
	want[gone] = nil
	for _, u := range users {
		if got := w.Held(u); fmt.Sprint(got) != fmt.Sprint(want[u]) {
			return fmt.Errorf("after erasing %s, %s holds %v; want %v", gone, u, got, want[u])
		}
	}
	after, err := in.Snapshot()
	if err != nil {
		return err
	}
	if !sameSet(before, after) {
		return fmt.Errorf("retention changed the index:\n%v\n%v", before, after)
	}
	if err := in.Drop(); err != nil {
		return err
	}
	if err := in.Rebuild(); err != nil {
		return err
	}
	rebuilt, err := in.Snapshot()
	if err != nil {
		return err
	}
	if !sameSet(before, rebuilt) {
		return fmt.Errorf("after retention, a rebuild differs from the index:\n%v\n%v", before, rebuilt)
	}
	return nil
}

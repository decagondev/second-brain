package schedule

import (
	"context"
	"errors"
	"testing"
	"time"
)

type runs map[string]time.Time

func (r runs) LastFinish(_ context.Context, job string) (time.Time, bool, error) {
	if job == "broken" {
		return time.Time{}, false, errors.New("unreadable")
	}
	at, ok := r[job]
	return at, ok, nil
}

type gauge map[string]float64

func (g gauge) Set(_, _ string, l map[string]string, v float64) { g[l["job"]] = v }

func TestWatch(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	r := runs{}
	for _, j := range Jobs {
		r[j.Name] = now.Add(-j.Interval / 2)
	}
	g := gauge{}
	if as := Watch(context.Background(), r, now, g); len(as) != 0 {
		t.Fatalf("on time: %+v", as)
	}
	r["anchor"] = now.Add(-3 * time.Hour)
	delete(r, "retention")
	as := Watch(context.Background(), r, now, g)
	if len(as) != 2 || as[0].Job != "anchor" || as[0].Severity != "sev1" || as[0].Law != "LAW-11" || as[1].Job != "retention" || as[1].Severity != "sev2" {
		t.Fatalf("%+v", as)
	}
	if g["anchor"] != 1 || g["sync"] != 0 {
		t.Fatalf("gauge %v", g)
	}
	if j, _ := Find("access"); j.Grace() != 32*24*time.Hour {
		t.Fatalf("monthly grace %v", j.Grace())
	}
}

package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/schedule"
)

// LAW-11 on GCP (SB-904): anchors and backups written to the cell's WORM bucket can't be
// overwritten or deleted inside their retention, so a tamperer with the cell's own rights
// can't replace the evidence a restore checks against.
// LAW-11, LAW-13 and LAW-17 watched (SB-904): every job that carries a law is scheduled, and
// any missed run raises a Sev-1 alert; a job whose last run is unknown counts as missed.

func law11WORM(r *rand.Rand) error {
	f := gcptest.New()
	defer f.Server.Close()
	f.Grant("tok", rtProjectA, gcp.BucketPermissions...)
	f.Bucket(rtProjectA, "sb-gamma-parts-worm", gcp.MinRetention, true)
	b := gcp.Bucket{Project: gcp.Project{ID: rtProjectA, HTTP: f.Server.Client(), Token: func(context.Context) (string, error) { return "tok", nil }},
		Bucket: "sb-gamma-parts-worm", Base: f.URL()}
	ctx := context.Background()
	n := 1 + r.Intn(6)
	for i := 0; i < n; i++ {
		if err := b.Put(ctx, fmt.Sprintf("gamma-parts/anchors/%04d.json", i), []byte(fmt.Sprintf("anchor %d", i)), time.Time{}); err != nil {
			return err
		}
	}
	target := fmt.Sprintf("gamma-parts/anchors/%04d.json", r.Intn(n))
	if r.Intn(2) == 0 {
		if err := b.Put(ctx, target, []byte("forged"), time.Time{}); err == nil {
			return fmt.Errorf("anchor %s was overwritten", target)
		}
	} else if err := f.DeleteObject("sb-gamma-parts-worm", target, time.Now().Add(time.Duration(r.Intn(365*24))*time.Hour)); err == nil {
		return fmt.Errorf("anchor %s was deleted inside its retention", target)
	}
	got, err := b.Get(ctx, target)
	if err != nil || len(got) == 0 || string(got) == "forged" {
		return fmt.Errorf("anchor %s after the attempt: %q %v", target, got, err)
	}
	return nil
}

// WatchFunc is the system under test for missed runs.
type WatchFunc func(ctx context.Context, runs schedule.Runs, now time.Time) []schedule.Alert

type lawRuns map[string]time.Time

func (l lawRuns) LastFinish(_ context.Context, job string) (time.Time, bool, error) {
	if at, ok := l[job]; ok {
		if at.IsZero() {
			return time.Time{}, false, fmt.Errorf("unreadable")
		}
		return at, true, nil
	}
	return time.Time{}, false, nil
}

func lawWatched(law string) func(r *rand.Rand, watch WatchFunc) error {
	return func(r *rand.Rand, watch WatchFunc) error {
		now := time.Unix(1_800_000_000, 0).UTC()
		runs := lawRuns{}
		late := map[string]bool{}
		for _, j := range schedule.Jobs {
			switch r.Intn(5) {
			case 0: // never ran
				late[j.Name] = true
			case 1: // unreadable
				runs[j.Name] = time.Time{}
				late[j.Name] = true
			case 2: // late
				runs[j.Name] = now.Add(-j.Grace() - time.Duration(1+r.Intn(3600))*time.Second)
				late[j.Name] = true
			default: // on time
				runs[j.Name] = now.Add(-time.Duration(r.Int63n(int64(j.Grace()))))
			}
		}
		carries := false
		for _, j := range schedule.Jobs {
			carries = carries || j.Law == law
		}
		if !carries {
			return fmt.Errorf("no scheduled job carries %s", law)
		}
		alerted := map[string]schedule.Alert{}
		for _, a := range watch(context.Background(), runs, now) {
			alerted[a.Job] = a
		}
		for _, j := range schedule.Jobs {
			a, ok := alerted[j.Name]
			if late[j.Name] != ok {
				return fmt.Errorf("job %s late=%v but alerted=%v", j.Name, late[j.Name], ok)
			}
			if ok && j.Law == law && a.Severity != "sev1" {
				return fmt.Errorf("a missed %s run (%s) alerted as %s, not Sev-1", j.Name, law, a.Severity)
			}
		}
		return nil
	}
}

// NewWatch builds the system under test: the schedule watchdog (SB-904).
var NewWatch = func() WatchFunc {
	return func(ctx context.Context, runs schedule.Runs, now time.Time) []schedule.Alert {
		return schedule.Watch(ctx, runs, now, nil)
	}
}

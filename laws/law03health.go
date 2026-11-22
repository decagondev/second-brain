package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/core/health"
	"example.com/second-brain/internal/core/schedule"
)

// LAW-03 for the console's health (SB-1607): a record the cell can't read is never shown
// as healthy. An unreadable or missing job record is overdue; an unreadable connector
// record fails the whole report; a connector that never synced is stale.

type lawJobRuns map[string]int // 0 on time, 1 late, 2 missing, 3 unreadable

func (l lawJobRuns) at(now time.Time, j schedule.Job) time.Time {
	if l[j.Name] == 1 {
		return now.Add(-j.Grace() - time.Minute)
	}
	return now.Add(-time.Second)
}

type lawRunsView struct {
	l   lawJobRuns
	now time.Time
}

func (v lawRunsView) LastFinish(_ context.Context, job string) (time.Time, bool, error) {
	for _, j := range schedule.Jobs {
		if j.Name == job {
			switch v.l[job] {
			case 2:
				return time.Time{}, false, nil
			case 3:
				return time.Time{}, false, errors.New("unreadable")
			}
			return v.l.at(v.now, j), true, nil
		}
	}
	return time.Time{}, false, nil
}

type lawSyncs map[string]int // 0 fresh, 1 old, 2 never, 3 unreadable

func (s lawSyncs) Sync(_ context.Context, kind string) (health.Sync, bool, error) {
	switch s[kind] {
	case 1:
		return health.Sync{LastSuccess: time.Unix(1_800_000_000, 0).Add(-time.Hour)}, true, nil
	case 2:
		return health.Sync{}, false, nil
	case 3:
		return health.Sync{}, false, errors.New("unreadable")
	}
	return health.Sync{LastSuccess: time.Unix(1_800_000_000, 0).Add(-time.Minute)}, true, nil
}

func law03Health(r *rand.Rand) error {
	now := time.Unix(1_800_000_000, 0)
	jobs := lawJobRuns{}
	for _, j := range schedule.Jobs {
		jobs[j.Name] = r.Intn(4)
	}
	kinds := []string{"docs", "chat", "tickets"}[:1+r.Intn(3)]
	syncs := lawSyncs{}
	broken := false
	for _, k := range kinds {
		syncs[k] = r.Intn(4)
		broken = broken || syncs[k] == 3
	}
	rep, err := health.Build(context.Background(), lawRunsView{jobs, now}, syncs, kinds, now)
	if broken {
		if !errors.Is(err, health.ErrUnavailable) {
			return fmt.Errorf("an unreadable connector record gave a report (%v)", err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	for _, j := range rep.Jobs {
		if want := jobs[j.Name] != 0; j.Overdue != want {
			return fmt.Errorf("job %s (state %d) overdue=%v", j.Name, jobs[j.Name], j.Overdue)
		}
	}
	for _, c := range rep.Connectors {
		if want := syncs[c.Kind] != 0; c.Stale != want {
			return fmt.Errorf("connector %s (state %d) stale=%v", c.Kind, syncs[c.Kind], c.Stale)
		}
	}
	return nil
}

package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/spaces"
)

// LAW-13 and LAW-11 under legal holds (SB-1207), through the real hold service, retention
// job, trace store and space sync: no record a hold covers is ever deleted, by the sweep,
// an erasure or a space removal; records nothing covers age out as before; a released
// hold protects nothing; and every hold and release is audited before it takes effect.

// HoldCell is the parts under test (the real ones in the shipped system).
type HoldCell struct {
	// Held turns the active holds into retention's filter (holds.Set.Held).
	Held func(holds.Set) retention.Held
	// Sync is the space sync with holds (spaces.SyncHeld).
	Sync func(ctx context.Context, st ports.SpaceStore, t manifest.Tenant, held func(string) bool) error
}

// RealHoldCell is the shipped wiring.
var RealHoldCell = HoldCell{
	Held: func(s holds.Set) retention.Held { return s.Held },
	Sync: func(ctx context.Context, st ports.SpaceStore, t manifest.Tenant, held func(string) bool) error {
		return spaces.SyncHeld(ctx, st, nil, t, held)
	},
}

// flakyAudit fails when told to; it records every row it wrote.
type flakyAudit struct {
	fail bool
	rows []audit.Event
}

func (a *flakyAudit) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	if a.fail {
		return audit.Row{}, errors.New("audit store down")
	}
	a.rows = append(a.rows, e)
	return audit.Row{ID: fmt.Sprintf("aud_%d", len(a.rows)), Kind: e.Kind}, nil
}

var holdCounts struct{ kept, aged int }

func law13Holds(r *rand.Rand, c HoldCell) error {
	ctx := context.Background()
	const tenant = "acme-logistics"
	now := time.Date(2027, 6, 1, 3, 0, 0, 0, time.UTC)
	users := []string{"usr_ann", "usr_bob", "usr_cy"}
	names := []string{"legal", "finance"}
	store := tracemem.New()
	type rec struct {
		id, user, space string
		at              time.Time
	}
	var recs []rec
	for i := 0; i < 6+r.Intn(10); i++ {
		x := rec{id: fmt.Sprintf("tr_%d", i), user: users[r.Intn(len(users))], space: spaces.ID(names[r.Intn(len(names))]),
			at: now.Add(-time.Duration(1+r.Intn(120)) * 24 * time.Hour)}
		if r.Intn(4) == 0 {
			x.space = "company"
		}
		store.Now = func() time.Time { return x.at }
		_ = store.PutTrace(ctx, ask.Trace{ID: x.id, Tenant: tenant, User: x.user, Space: x.space, At: x.at})
		recs = append(recs, x)
	}
	aud := &flakyAudit{}
	hst := &holds.MemoryStore{}
	n := 0
	svc := holds.Service{Store: hst, Audit: aud, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: func() time.Time { return now }}
	var placed []holds.Hold
	for i := 0; i < 1+r.Intn(3); i++ {
		space, us := "", []string(nil)
		switch r.Intn(3) {
		case 0:
			space = names[r.Intn(len(names))]
		case 1:
			us = []string{users[r.Intn(len(users))]}
		default:
			space, us = names[r.Intn(len(names))], []string{users[r.Intn(len(users))]}
		}
		aud.fail = r.Intn(5) == 0
		before := len(aud.rows)
		h, err := svc.Place(ctx, "usr_counsel", "orion", space, us, "litigation")
		if aud.fail {
			all, _ := hst.List(ctx)
			if err == nil || len(all) != len(placed) {
				return fmt.Errorf("a hold was placed although its audit row failed (LAW-11)")
			}
			aud.fail = false
			continue
		}
		if err != nil {
			return err
		}
		if len(aud.rows) != before+1 || aud.rows[before].Subject != "hold:"+h.ID || aud.rows[before].State != "placed" {
			return fmt.Errorf("hold %s has no placed audit row (LAW-11)", h.ID)
		}
		placed = append(placed, h)
	}
	if len(placed) > 0 && r.Intn(3) == 0 {
		h := placed[r.Intn(len(placed))]
		if _, err := svc.Release(ctx, "usr_counsel", h.ID, "matter closed"); err != nil {
			return err
		}
		if last := aud.rows[len(aud.rows)-1]; last.Subject != "hold:"+h.ID || last.State != "released" {
			return fmt.Errorf("release of %s has no audit row (LAW-11)", h.ID)
		}
	}
	active, err := holds.Active(ctx, hst)
	if err != nil {
		return err
	}
	covered := func(x rec) bool {
		sp := x.space
		if sp == "company" {
			sp = ""
		}
		for _, h := range active {
			if h.Covers(sp, x.user) {
				return true
			}
		}
		return false
	}
	// The sweep.
	job := retention.Job{Tenant: tenant, Days: 30, Stores: store.Retention(), Audit: aud, Now: func() time.Time { return now }, Held: c.Held(active)}
	if _, err := job.Sweep(ctx); err != nil {
		return err
	}
	cut := job.Cutoff()
	for _, x := range recs {
		_, found, _ := store.GetTrace(ctx, tenant, x.id)
		switch {
		case covered(x) && !found:
			return fmt.Errorf("the sweep deleted %s (user %s, space %s), which a hold covers (holds %+v)", x.id, x.user, x.space, active)
		case !covered(x) && x.at.Before(cut) && found:
			return fmt.Errorf("the sweep kept %s, which nothing holds and is %v old", x.id, now.Sub(x.at))
		case covered(x) && x.at.Before(cut):
			holdCounts.kept++
		case x.at.Before(cut):
			holdCounts.aged++
		}
	}
	// An erasure.
	u := users[r.Intn(len(users))]
	held := false
	for _, h := range active {
		held = held || h.Covers("", u)
	}
	_, err = job.Erase(ctx, "usr_op", u, "subject request")
	switch {
	case held && !errors.Is(err, retention.ErrHeld):
		return fmt.Errorf("erasure of %s went ahead under a hold (err %v)", u, err)
	case !held && err != nil:
		return fmt.Errorf("erasure of %s, held by nothing, failed: %v", u, err)
	}
	// Removing a space.
	reg := spacemem.New()
	full := manifest.Tenant{Tenant: tenant, Spaces: map[string]manifest.Space{}}
	for _, nm := range names {
		full.Spaces[nm] = manifest.Space{Template: "custom", Members: []string{"group:" + nm}, Label: "internal"}
	}
	if err := c.Sync(ctx, reg, full, nil); err != nil {
		return err
	}
	drop := names[r.Intn(len(names))]
	less := manifest.Tenant{Tenant: tenant, Spaces: map[string]manifest.Space{}}
	for _, nm := range names {
		if nm != drop {
			less.Spaces[nm] = full.Spaces[nm]
		}
	}
	err = c.Sync(ctx, reg, less, active.HeldSpace)
	_, still, _ := reg.Space(ctx, tenant, spaces.ID(drop))
	switch {
	case active.HeldSpace(spaces.ID(drop)) && (!errors.Is(err, spaces.ErrHeld) || !still):
		return fmt.Errorf("held space %s was removed (err %v, still there %v)", drop, err, still)
	case !active.HeldSpace(spaces.ID(drop)) && (err != nil || still):
		return fmt.Errorf("space %s, held by nothing, wasn't removed (err %v)", drop, err)
	}
	return nil
}

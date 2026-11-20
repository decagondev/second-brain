// Package retention is the nightly retention job and subject deletion (sb-tenant(5),
// Retention and deletion; SB-611). It runs over the cell's stores of derived data only:
// audit and the index are never stores here.
package retention

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// Bounds on retention_days (sb-tenant(5)).
const (
	MinDays = 30
	MaxDays = 3650
)

// Store is one store of derived data.
type Store interface {
	Name() string
	// Expire deletes the tenant's records older than before, except those held reports as
	// held (by the record's space id and user, "" for one the store doesn't know), and
	// returns how many it deleted and how many it kept for a hold (SB-1207).
	Expire(ctx context.Context, tenant string, before time.Time, held Held) (deleted, kept int, err error)
	// Erase deletes everything the store holds about one user and returns how many.
	Erase(ctx context.Context, tenant, user string) (int, error)
}

// Held reports whether a record in space (an id, or "") about user (or "") is under a legal
// hold (holds.Set.Held). Nil holds nothing.
type Held func(space, user string) bool

// Is is h(space, user), false for a nil h.
func (h Held) Is(space, user string) bool { return h != nil && h(space, user) }

// Auditor writes audit rows (audit.Log).
type Auditor interface {
	Write(ctx context.Context, e audit.Event) (audit.Row, error)
}

// Errors.
var (
	ErrDays    = errors.New("retention: retention_days must be 30 to 3650")
	ErrInvalid = errors.New("retention: EINVAL")
	ErrAudit   = errors.New("retention: the audit row can't be written")
	ErrStore   = errors.New("retention: a store failed")
	ErrHeld    = errors.New("retention: EHOLD a legal hold covers this")
)

// Job is one cell's retention.
type Job struct {
	Tenant string
	Days   int
	Stores []Store
	Audit  Auditor
	Now    func() time.Time
	// Held is the cell's active legal holds (SB-1207); nil is none.
	Held Held
}

// Report is what a run did: records deleted per store, and the stores that failed.
type Report struct {
	Cutoff string         `json:"cutoff,omitempty"`
	Counts map[string]int `json:"erased"`
	Failed []string       `json:"failed,omitempty"`
	Audit  []string       `json:"audit"`
	Held   int            `json:"held,omitempty"` // records a legal hold kept (SB-1207)
}

func summary(r Report) string {
	names := make([]string, 0, len(r.Counts))
	for n := range r.Counts {
		names = append(names, n)
	}
	sort.Strings(names)
	var parts []string
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", n, r.Counts[n]))
	}
	if len(r.Failed) > 0 {
		parts = append(parts, "failed="+strings.Join(r.Failed, ","))
	}
	if len(parts) == 0 {
		return "no stores"
	}
	return strings.Join(parts, " ")
}

// Cutoff is the oldest moment kept: a record exactly Days old is kept.
func (j Job) Cutoff() time.Time { return j.Now().UTC().Add(-time.Duration(j.Days) * 24 * time.Hour) }

// Sweep deletes everything older than the cutoff from every store and audits the counts.
func (j Job) Sweep(ctx context.Context) (Report, error) {
	if j.Days < MinDays || j.Days > MaxDays {
		return Report{}, ErrDays
	}
	cut := j.Cutoff()
	rep := Report{Cutoff: cut.Format(time.RFC3339), Counts: map[string]int{}}
	held := 0
	for _, s := range j.Stores {
		n, k, err := s.Expire(ctx, j.Tenant, cut, j.Held)
		if err != nil {
			rep.Failed = append(rep.Failed, s.Name())
			continue
		}
		rep.Counts[s.Name()] = n
		held += k
	}
	rep.Held = held
	state := "done"
	if len(rep.Failed) > 0 {
		state = "failed"
	}
	row, err := j.Audit.Write(ctx, audit.Event{Actor: "sb-retention", Kind: "retention", Subject: "sweep", State: state,
		Reason: "cutoff=" + rep.Cutoff + " " + summary(rep) + fmt.Sprintf(" held=%d", held)})
	if err != nil {
		return rep, fmt.Errorf("%w: %v", ErrAudit, err)
	}
	rep.Audit = []string{row.ID}
	if len(rep.Failed) > 0 {
		return rep, fmt.Errorf("%w: %s", ErrStore, strings.Join(rep.Failed, ", "))
	}
	return rep, nil
}

// UserID is the form of a user id an erasure takes.
var UserID = regexp.MustCompile(`^usr_[a-z0-9_]{1,64}$`)

// Erase removes everything the stores hold about one user. It audits the request before
// erasing anything, and the outcome after.
func (j Job) Erase(ctx context.Context, actor, user, reason string) (Report, error) {
	if !UserID.MatchString(user) || strings.TrimSpace(reason) == "" {
		return Report{}, fmt.Errorf("%w: a usr_ id and a reason are required", ErrInvalid)
	}
	if j.Held.Is("", user) {
		return Report{}, fmt.Errorf("%w: %s is a custodian, or in a held space; nothing was erased", ErrHeld, user)
	}
	subject := "erase:" + user
	first, err := j.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "retention", Subject: subject, State: "requested", Reason: reason})
	if err != nil {
		return Report{}, fmt.Errorf("%w, so nothing was erased: %v", ErrAudit, err)
	}
	rep := Report{Counts: map[string]int{}, Audit: []string{first.ID}}
	for _, s := range j.Stores {
		n, err := s.Erase(ctx, j.Tenant, user)
		if err != nil {
			rep.Failed = append(rep.Failed, s.Name())
			continue
		}
		rep.Counts[s.Name()] = n
	}
	state := "done"
	if len(rep.Failed) > 0 {
		state = "failed"
	}
	last, err := j.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "retention", Subject: subject, State: state, Reason: summary(rep)})
	if err != nil {
		return rep, fmt.Errorf("%w: the outcome row: %v", ErrAudit, err)
	}
	rep.Audit = append(rep.Audit, last.ID)
	if len(rep.Failed) > 0 {
		return rep, fmt.Errorf("%w: %s", ErrStore, strings.Join(rep.Failed, ", "))
	}
	return rep, nil
}

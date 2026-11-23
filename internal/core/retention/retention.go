// Package retention is the nightly retention job and subject deletion (sb-tenant(5),
// Retention and deletion; SB-611). It runs over the cell's stores of derived data only:
// audit and the index are never stores here.
package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	// Plan counts what Expire with the same arguments would delete and keep for a hold,
	// by the same selection, and deletes nothing (SB-1905).
	Plan(ctx context.Context, tenant string, before time.Time, held Held) (would, kept int, err error)
	// Export returns, one JSON value each, every record Erase would delete for the user,
	// by the same selection (SB-1903).
	Export(ctx context.Context, tenant, user string) ([]json.RawMessage, error)
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
	// SpaceDays are spaces whose retention_days is below Days, by space id (spc_NAME):
	// their records age out sooner (sb-space(5); SB-1401). A space can keep less, never more.
	SpaceDays map[string]int
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
	// A space that keeps less: a second pass with its own cutoff, over its records only.
	for _, id := range sortedKeys(j.SpaceDays) {
		days := j.SpaceDays[id]
		if days < MinDays || days >= j.Days {
			continue
		}
		spaceCut := j.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		only := func(space, user string) bool { return space != id || j.Held.Is(space, user) }
		for _, s := range j.Stores {
			n, _, err := s.Expire(ctx, j.Tenant, spaceCut, only)
			if err != nil {
				rep.Failed = append(rep.Failed, s.Name()+"@"+id)
				continue
			}
			rep.Counts[s.Name()] += n
		}
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

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Plan is what Sweep would do now, deleting nothing: the count per store, and the records
// a legal hold keeps counted apart (SB-1905). in moves the clock forward: the plan of the
// sweep that runs in days from now. It runs each store's own selection (Store.Plan), the
// tenant's cutoff and then each shorter space's, as Sweep does.
func (j Job) Plan(ctx context.Context, in time.Duration) (Report, error) {
	if j.Days < MinDays || j.Days > MaxDays {
		return Report{}, ErrDays
	}
	if in < 0 {
		return Report{}, fmt.Errorf("%w: days must be 0 or more", ErrInvalid)
	}
	now := j.Now
	later := j
	later.Now = func() time.Time { return now().Add(in) }
	cut := later.Cutoff()
	rep := Report{Cutoff: cut.Format(time.RFC3339), Counts: map[string]int{}}
	for _, s := range j.Stores {
		n, k, err := s.Plan(ctx, j.Tenant, cut, j.Held)
		if err != nil {
			rep.Failed = append(rep.Failed, s.Name())
			continue
		}
		rep.Counts[s.Name()] = n
		rep.Held += k
	}
	for _, id := range sortedKeys(j.SpaceDays) {
		days := j.SpaceDays[id]
		if days < MinDays || days >= j.Days {
			continue
		}
		spaceCut := later.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		only := func(space, user string) bool { return space != id || j.Held.Is(space, user) }
		for _, s := range j.Stores {
			// Sweep's first pass has already deleted the space's records older than the
			// tenant's cutoff, so only those between the two cutoffs are added.
			n, _, err := s.Plan(ctx, j.Tenant, spaceCut, only)
			var m int
			if err == nil {
				m, _, err = s.Plan(ctx, j.Tenant, cut, only)
			}
			if err != nil {
				rep.Failed = append(rep.Failed, s.Name()+"@"+id)
				continue
			}
			rep.Counts[s.Name()] += n - m
		}
	}
	if len(rep.Failed) > 0 {
		return rep, fmt.Errorf("%w: %s", ErrStore, strings.Join(rep.Failed, ", "))
	}
	return rep, nil
}

// Export is a copy of everything Erase would remove for one user, by store (SB-1903). It
// audits the request before reading anything, and the outcome after: the outcome row holds
// the counts and the SHA-256 of the export, never its text (LAW-07). A store that fails
// fails the export: a partial copy is never handed over as the whole.
func (j Job) Export(ctx context.Context, actor, user, reason string) (Export, error) {
	if !UserID.MatchString(user) || strings.TrimSpace(reason) == "" {
		return Export{}, fmt.Errorf("%w: a usr_ id and a reason are required", ErrInvalid)
	}
	subject := "export:" + user
	first, err := j.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "retention", Subject: subject, State: "requested", Reason: reason})
	if err != nil {
		return Export{}, fmt.Errorf("%w, so nothing was exported: %v", ErrAudit, err)
	}
	out := Export{User: user, Records: map[string][]json.RawMessage{}, Audit: []string{first.ID}}
	rep := Report{Counts: map[string]int{}}
	for _, s := range j.Stores {
		rs, err := s.Export(ctx, j.Tenant, user)
		if err != nil {
			rep.Failed = append(rep.Failed, s.Name())
			continue
		}
		if rs == nil {
			rs = []json.RawMessage{}
		}
		out.Records[s.Name()] = rs
		rep.Counts[s.Name()] = len(rs)
	}
	state, hash := "done", ""
	if len(rep.Failed) > 0 {
		state = "failed"
		out.Records = nil
	} else {
		hash = out.Hash()
	}
	reasonOut := summary(rep)
	if hash != "" {
		reasonOut += " sha256=" + hash
	}
	last, err := j.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "retention", Subject: subject, State: state, Reason: reasonOut})
	if err != nil {
		return Export{}, fmt.Errorf("%w: the outcome row, so nothing is handed over: %v", ErrAudit, err)
	}
	out.Audit = append(out.Audit, last.ID)
	if len(rep.Failed) > 0 {
		return Export{Audit: out.Audit}, fmt.Errorf("%w: %s", ErrStore, strings.Join(rep.Failed, ", "))
	}
	out.SHA256 = hash
	return out, nil
}

// Export is one person's records, by store.
type Export struct {
	User    string                       `json:"user"`
	Records map[string][]json.RawMessage `json:"records"`
	SHA256  string                       `json:"sha256"`
	Audit   []string                     `json:"audit"`
}

// Count is the number of records in the export.
func (e Export) Count() int {
	n := 0
	for _, rs := range e.Records {
		n += len(rs)
	}
	return n
}

// Hash is the SHA-256 of the records, as the JSON of the map from store to records (keys
// sorted, as encoding/json writes them): the value the audit row holds.
func (e Export) Hash() string {
	b, _ := json.Marshal(e.Records)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

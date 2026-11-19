// Package apply enacts a manifest's plan (sb-tenant(1), SB-104): stamp the cell if it is
// new, then record each change, with a write-ahead journal. Every step is idempotent, so
// an interrupted apply resumes by running again.
package apply

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/ports"
)

// Deps are the ports apply needs. NewID may be nil (a random id is used).
type Deps struct {
	Registry  ports.Registry
	Writer    ports.RegistryWriter
	Placement ports.Placement
	Keys      ports.Keys
	Journal   ports.Journal
	Cells     ports.CellsWriter
	// BaseDomain gives a new cell its default host, TENANT.BaseDomain (sb-tenancy(7)).
	BaseDomain string
	NewID      func() string
}

// Record is the registry record of a new cell: v1 custody is always factory
// (sb-factory(7)), and the placement is whatever account the cell was stamped into.
func Record(t manifest.Tenant, p ports.Placement, baseDomain string) ports.CellRecord {
	return ports.CellRecord{
		Tenant: t.Tenant, Region: t.Region,
		Endpoint: p.Account() + "/cells/" + t.Tenant,
		Hosts:    []string{t.Tenant + "." + baseDomain},
		Custody:  cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory,
	}
}

// StepError is a failure while enacting one change; changes before it stay recorded.
type StepError struct {
	Step   int
	Change plan.Change
	Err    error
}

func (e *StepError) Error() string {
	return fmt.Sprintf("step %d (%s %s %s): %v", e.Step, e.Change.Op, e.Change.Kind, e.Change.Name, e.Err)
}

func (e *StepError) Unwrap() error { return e.Err }

// UnavailableError means the registry couldn't say what is applied. Apply never treats
// that as "no cell".
type UnavailableError struct{ Err error }

func (e *UnavailableError) Error() string { return "tenant registry unavailable: " + e.Err.Error() }
func (e *UnavailableError) Unwrap() error { return e.Err }

// Run enacts t's plan and returns it. done is called after each change is recorded.
// With nothing to do it writes nothing at all. A region change returns *plan.ImmutableError.
func Run(ctx context.Context, d Deps, t manifest.Tenant, done func(plan.Change)) (plan.Plan, error) {
	applied, found, err := d.Registry.Applied(ctx, t.Tenant)
	if err != nil {
		return plan.Plan{}, &UnavailableError{err}
	}
	p, err := plan.Make(t, applied, found)
	if err != nil || len(p.Changes) == 0 {
		return p, err
	}
	id, err := newID(d.NewID)
	if err != nil {
		return p, err
	}
	state := applied
	if !found {
		state = nil
	}
	for i, c := range p.Changes {
		step := i + 1
		entry := ports.JournalEntry{Tenant: t.Tenant, ApplyID: id, Step: step, Op: c.Op, Kind: c.Kind, Name: c.Name}
		fail := func(err error) (plan.Plan, error) { return p, &StepError{Step: step, Change: c, Err: err} }
		entry.Phase = "start"
		if err := d.Journal.Record(ctx, entry); err != nil {
			return fail(err)
		}
		if c.Kind == plan.KindCell && c.Op == plan.OpAdd {
			// The cell is recorded only after every resource exists (sb-tenant(1)).
			if _, err := cell.Stamp(ctx, d.Placement, d.Keys, t); err != nil {
				return fail(err)
			}
			if err := d.Cells.PutCell(ctx, Record(t, d.Placement, d.BaseDomain)); err != nil {
				return fail(err)
			}
		}
		state = plan.Apply(state, plan.Plan{Changes: []plan.Change{c}})
		if err := d.Writer.Put(ctx, t.Tenant, state); err != nil {
			return fail(err)
		}
		entry.Phase = "done"
		if err := d.Journal.Record(ctx, entry); err != nil {
			return fail(err)
		}
		if done != nil {
			done(c)
		}
	}
	return p, nil
}

func newID(f func() string) (string, error) {
	if f != nil {
		return f(), nil
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "apply-" + hex.EncodeToString(b), nil
}

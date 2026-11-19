// Package localfs is a tenant registry kept in a local directory, for development and
// tests until the control plane exists (SB-105). Layout: DIR/tenants/<tenant>.json.
//
// It answers "no cell" only when DIR/tenants exists and has no file for the tenant.
// A missing DIR, an unreadable file or bad JSON is an error, never "no cell".
package localfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/ports"
)

// Dir is a registry rooted at a directory.
type Dir struct{ Root string }

// Record is the on-disk form of one tenant's applied state.
type Record struct {
	Tenant    string           `json:"tenant"`
	Resources []ports.Resource `json:"resources"`
}

var _ ports.Registry = Dir{}

func (d Dir) path(tenant string) (string, error) {
	if tenant == "" || strings.ContainsAny(tenant, `/\.`) {
		return "", fmt.Errorf("registry: invalid tenant name %q", tenant)
	}
	return filepath.Join(d.Root, "tenants", tenant+".json"), nil
}

// Applied reads the tenant's applied resources.
func (d Dir) Applied(_ context.Context, tenant string) ([]ports.Resource, bool, error) {
	p, err := d.path(tenant)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		if st, serr := os.Stat(filepath.Dir(p)); serr != nil || !st.IsDir() {
			return nil, false, fmt.Errorf("registry: %s is not a registry directory", d.Root)
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("registry: read %s: %w", tenant, err)
	}
	var rec Record
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rec); err != nil {
		return nil, false, fmt.Errorf("registry: %s.json: %w", tenant, err)
	}
	if rec.Tenant != tenant {
		return nil, false, fmt.Errorf("registry: %s.json records tenant %q", tenant, rec.Tenant)
	}
	return rec.Resources, true, nil
}

// Put records the tenant's applied resources (used by tests now, by SB-104 later).
func (d Dir) Put(_ context.Context, tenant string, rs []ports.Resource) error {
	p, err := d.path(tenant)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(Record{Tenant: tenant, Resources: rs}, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

var (
	_ ports.RegistryWriter = Dir{}
	_ ports.Journal        = Dir{}
)

// Record appends one apply step to DIR/journal/TENANT.ndjson and syncs it to disk.
func (d Dir) Record(_ context.Context, e ports.JournalEntry) error {
	if _, err := d.path(e.Tenant); err != nil {
		return err
	}
	dir := filepath.Join(d.Root, "journal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, e.Tenant+".ndjson"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

var (
	_ ports.Cells       = Dir{}
	_ ports.CellsWriter = Dir{}
)

func (d Dir) cellPath(tenant string) (string, error) {
	if _, err := d.path(tenant); err != nil {
		return "", err
	}
	return filepath.Join(d.Root, "cells", tenant+".json"), nil
}

func (d Dir) cells() ([]ports.CellRecord, error) {
	dir := filepath.Join(d.Root, "cells")
	if st, err := os.Stat(d.Root); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("registry: %s is not a registry directory", d.Root)
	}
	ents, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ports.CellRecord
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var c ports.CellRecord
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil || c.Tenant+".json" != e.Name() {
			return nil, fmt.Errorf("registry: cells/%s is unreadable", e.Name()) // unknown, not absent
		}
		out = append(out, c)
	}
	return out, nil
}

// CellByHost finds the cell that serves host.
func (d Dir) CellByHost(_ context.Context, host string) (ports.CellRecord, bool, error) {
	cs, err := d.cells()
	if err != nil {
		return ports.CellRecord{}, false, err
	}
	for _, c := range cs {
		for _, h := range c.Hosts {
			if h == host {
				return c, true, nil
			}
		}
	}
	return ports.CellRecord{}, false, nil
}

// CellByTenant finds the tenant's cell.
func (d Dir) CellByTenant(_ context.Context, tenant string) (ports.CellRecord, bool, error) {
	cs, err := d.cells()
	if err != nil {
		return ports.CellRecord{}, false, err
	}
	for _, c := range cs {
		if c.Tenant == tenant {
			return c, true, nil
		}
	}
	return ports.CellRecord{}, false, nil
}

// PutCell records c. It refuses a host that another cell already serves.
func (d Dir) PutCell(_ context.Context, c ports.CellRecord) error {
	p, err := d.cellPath(c.Tenant)
	if err != nil {
		return err
	}
	cs, err := d.cells()
	if err != nil {
		return err
	}
	for _, o := range cs {
		if o.Tenant == c.Tenant {
			continue
		}
		for _, h := range o.Hosts {
			for _, mine := range c.Hosts {
				if h == mine {
					return fmt.Errorf("registry: host %s already serves %s", h, o.Tenant)
				}
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// EvalRecord implements ports.EvalGate: DIR/evals/TENANT/WORKFLOW.json, as written by
// python3 -m evals run --record (sb-tenant(1), SB-410).
func (d Dir) EvalRecord(_ context.Context, tenant, workflow string) (ports.EvalRecord, bool, error) {
	if _, err := d.path(tenant); err != nil {
		return ports.EvalRecord{}, false, err
	}
	if !workflowName.MatchString(workflow) {
		return ports.EvalRecord{}, false, fmt.Errorf("localfs: bad workflow %q", workflow)
	}
	b, err := os.ReadFile(filepath.Join(d.Root, "evals", tenant, workflow+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return ports.EvalRecord{}, false, nil
	}
	if err != nil {
		return ports.EvalRecord{}, false, err
	}
	var r ports.EvalRecord
	if err := json.Unmarshal(b, &r); err != nil {
		return ports.EvalRecord{}, false, fmt.Errorf("localfs: eval record is not valid JSON: %w", err)
	}
	return r, true, nil
}

var workflowName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)

// CustodyChange is one entry of a cell's custody history (sb-factory(7)).
type CustodyChange struct {
	At           string `json:"at"`
	Custody      string `json:"custody"`
	Placement    string `json:"placement"`
	KeyCustodian string `json:"key_custodian"`
}

func (d Dir) history(tenant string) (string, error) {
	if _, err := d.path(tenant); err != nil {
		return "", err
	}
	return filepath.Join(d.Root, "custody", tenant+".ndjson"), nil
}

func (d Dir) appendHistory(tenant string, c CustodyChange) error {
	p, err := d.history(tenant)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.Marshal(c)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// SetCustody changes a cell's custody record and appends it to the custody history
// (handoff.Custody). The history is written first, so no change goes unrecorded.
func (d Dir) SetCustody(ctx context.Context, tenant, custody, placement, keyCustodian string) error {
	c, found, err := d.CellByTenant(ctx, tenant)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("registry: no cell for %s", tenant)
	}
	if err := d.appendHistory(tenant, CustodyChange{At: time.Now().UTC().Format(time.RFC3339), Custody: custody, Placement: placement, KeyCustodian: keyCustodian}); err != nil {
		return err
	}
	c.Custody, c.Placement, c.KeyCustodian = custody, placement, keyCustodian
	return d.PutCell(ctx, c)
}

// Tombstone removes a detached cell's record, so it stops routing, and keeps only its
// custody history (sb-factory(7), Detach, exactly).
func (d Dir) Tombstone(ctx context.Context, tenant string) error {
	c, found, err := d.CellByTenant(ctx, tenant)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("registry: no cell for %s", tenant)
	}
	if c.Custody != cell.CustodyDetached {
		return fmt.Errorf("registry: %s is in %s custody; only a detached cell is tombstoned", tenant, c.Custody)
	}
	p, _ := d.cellPath(tenant)
	return os.Remove(p)
}

// History returns a cell's custody history, oldest first.
func (d Dir) History(_ context.Context, tenant string) ([]CustodyChange, error) {
	p, err := d.history(tenant)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []CustodyChange
	for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
		var c CustodyChange
		if err := json.Unmarshal(line, &c); err != nil {
			return nil, fmt.Errorf("registry: custody history of %s is unreadable", tenant)
		}
		out = append(out, c)
	}
	return out, nil
}

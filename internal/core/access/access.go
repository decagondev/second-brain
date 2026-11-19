// Package access enumerates the factory's access paths to a cell and checks them against
// the hand-off record (sb-factory(7), Access paths, exactly; SB-706, LAW-17). Paths come from
// where they live (IAM policies, the wrap inventory), never from a kept list, and an
// enumeration that can't finish never passes.
package access

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/keywrap"
)

// Path is one way the factory could reach a cell.
type Path struct {
	Source string `json:"source"` // e.g. "gcp-iam", "key-wraps"
	Kind   string `json:"kind"`   // e.g. "iam-binding", "impersonation", "key-wrap"
	ID     string `json:"id"`     // unique within the source
	Grant  string `json:"grant"`  // the retained grant it serves, or "" for none
}

// Source enumerates paths from one place.
type Source interface {
	Name() string
	Paths(ctx context.Context, tenant string) ([]Path, error)
}

// Result is a check's outcome.
type Result struct {
	Paths    []Path `json:"paths"`
	Unlisted []Path `json:"unlisted"`
}

// ErrUnavailable is an enumeration that couldn't finish; the check fails, never passes.
var ErrUnavailable = errors.New("access: EUNAVAIL a source can't be read")

// Check enumerates every source and returns the paths not served by a retained grant.
func Check(ctx context.Context, tenant string, sources []Source, retain []string) (Result, error) {
	if len(sources) == 0 {
		return Result{}, fmt.Errorf("%w: no sources", ErrUnavailable)
	}
	keep := map[string]bool{}
	for _, g := range retain {
		keep[g] = true
	}
	var res Result
	for _, s := range sources {
		ps, err := s.Paths(ctx, tenant)
		if err != nil {
			return Result{}, fmt.Errorf("%w: %s: %v", ErrUnavailable, s.Name(), err)
		}
		for _, p := range ps {
			res.Paths = append(res.Paths, p)
			if p.Grant == "" || !keep[p.Grant] {
				res.Unlisted = append(res.Unlisted, p)
			}
		}
	}
	sort.Slice(res.Paths, func(i, j int) bool { return res.Paths[i].ID < res.Paths[j].ID })
	sort.Slice(res.Unlisted, func(i, j int) bool { return res.Unlisted[i].ID < res.Unlisted[j].ID })
	return res, nil
}

// Wraps is the wrap-inventory source: a factory wrap of any data key is a path.
type Wraps struct{ Store keywrap.Store }

// Name implements Source.
func (Wraps) Name() string { return "key-wraps" }

// Paths implements Source.
func (w Wraps) Paths(ctx context.Context, tenant string) ([]Path, error) {
	ws, err := w.Store.List(ctx, tenant)
	if err != nil {
		return nil, err
	}
	var out []Path
	for _, x := range ws {
		if x.Custodian == cell.KeyCustodianFactory {
			out = append(out, Path{Source: "key-wraps", Kind: "key-wrap", ID: "key-wrap:" + x.Purpose + ":" + x.KeyID})
		}
	}
	return out, nil
}

// Setter sets a gauge (obs.Metrics).
type Setter interface {
	Set(name, help string, labels map[string]string, v float64)
}

// Watch is the monthly re-check for one tenant-custody cell. It sets
// sb_unlisted_access_paths to the number of unlisted paths, or to 1 when the check can't
// finish, so silence never looks like a pass.
func Watch(ctx context.Context, tenant string, sources []Source, retain []string, m Setter) (Result, error) {
	res, err := Check(ctx, tenant, sources, retain)
	v := float64(len(res.Unlisted))
	if err != nil {
		v = 1
	}
	m.Set("sb_unlisted_access_paths", "Factory access paths not in the hand-off record (LAW-17).", map[string]string{"tenant": tenant}, v)
	return res, err
}

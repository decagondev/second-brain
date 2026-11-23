package laws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/barrier"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// LAW-22 and LAW-04 for notes in a cell with barriers (SB-2001): a steward publishes a
// note in a space on their side of a barrier; a note in a space across it is refused, and
// nothing is written; a cell that can't place the note now (its spaces unreadable) refuses
// too. The real bus, Cedar policies, barrier placement and folder connector.

// NotePlacer places a call's target in a barrier cell (barrier.Cell, as the bus uses it).
type NotePlacer func(spaces func(context.Context) ([]ports.Space, [][2]string, error)) toolbus.Barriers

// RealNotePlacer is the real one.
var RealNotePlacer NotePlacer = func(sp func(context.Context) ([]ports.Space, [][2]string, error)) toolbus.Barriers {
	return barrier.Cell{Spaces: sp, GroupsOf: func(context.Context, string) ([]string, error) { return nil, nil },
		Owners: func(context.Context, string) ([]string, error) { return nil, nil }}
}

func law22Notes(r *rand.Rand, dir string, place NotePlacer) error {
	if askPoliciesErr != nil {
		return askPoliciesErr
	}
	ctx := context.Background()
	share := filepath.Join(dir, fmt.Sprint(r.Int63()))
	if err := os.MkdirAll(share, 0o755); err != nil {
		return err
	}
	c, err := folder.Open(share, "")
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.EnableNotes(share); err != nil {
		return err
	}
	all := []ports.Space{
		{ID: "spc_support", Name: "support", Members: []string{"group:support"}, Stewards: []string{"group:support-leads"}, Scope: map[string][]string{"docs": {"/notes/support/**"}}},
		{ID: "spc_deals", Name: "deals", Members: []string{"group:deal-team"}, Stewards: []string{"group:deal-leads"}, Scope: map[string][]string{"docs": {"/notes/deals/**"}}},
	}
	bars := [][2]string{{"group:deal-team", "group:support"}}
	down := r.Intn(6) == 0
	spacesNow := func(context.Context) ([]ports.Space, [][2]string, error) {
		if down {
			return nil, nil, fmt.Errorf("the registry is down")
		}
		return all, bars, nil
	}
	n := 0
	rows := &keepRows{}
	bus := &toolbus.Bus{Tenant: "acme-logistics", Tools: map[string]toolbus.Tool{ask.NoteTool: {Name: ask.NoteTool, Connector: "docs"}},
		Kill: lawKill{}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}, Audit: rows, Actions: actmem.New(),
		Connectors: map[string]ports.Connector{"docs": c}, Barriers: place(spacesNow),
		Now: time.Now, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }}
	// A steward of each side, who (to test the barrier, not stewardship) is listed as a
	// steward of whichever space they publish in.
	side := []struct {
		groups []string
		own    string
	}{{[]string{"support", "support-leads"}, "support"}, {[]string{"deal-team", "deal-leads"}, "deals"}}[r.Intn(2)]
	who, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_s", side.groups)
	space := []string{"support", "deals"}[r.Intn(2)]
	target := fmt.Sprintf("notes/%s/qs_%d.md", space, r.Intn(1000))
	payload := `{"markdown":"# Refunds\n\nOver 500 needs a lead.\n","readers":["ann@acme.example"],"owner":"sam@acme.example","label":"internal"}`
	a, err := bus.Invoke(ctx, who, toolbus.Call{Agent: "steward", Tool: ask.NoteTool, Target: target, Payload: []byte(payload),
		Context: map[string]any{"stewards": []string{"group:" + side.groups[1]}, "groups": groupNames(who)}})
	_, statErr := os.Stat(filepath.Join(share, filepath.FromSlash(target)))
	written := statErr == nil
	switch {
	case down && (err == nil || written):
		return fmt.Errorf("a note was published while the cell couldn't place it")
	case down:
		return nil
	case space != side.own && (err == nil || written):
		return fmt.Errorf("a steward of %s published %s across the barrier (LAW-22)", side.own, target)
	case space != side.own:
		return nil
	case err != nil:
		return fmt.Errorf("a steward of %s couldn't publish %s in their own space: %v", side.own, target, err)
	case a.State != approvals.Ran || !written:
		return fmt.Errorf("the note in %s didn't run (%s), or isn't on the share", space, a.State)
	}
	for _, row := range rows.rows {
		if row.State == string(approvals.Executing) {
			return nil // LAW-04: the publish was recorded before the write
		}
	}
	return fmt.Errorf("the note was published with no Executing audit row")
}

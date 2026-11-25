package wire

import (
	"context"
	"fmt"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/tenancy"
)

// agentHistory is agent name's versions (SB-5104): its settings in every approved proposal,
// and as applied now.
func (c *Cell) agentHistory(ctx context.Context, name string) ([]proposals.Version, error) {
	if ask.AgentOwner(name) == "" {
		return nil, fmt.Errorf("%w: no agent %q", ask.ErrInvalid, name)
	}
	ps, err := c.Proposals.List(ctx)
	if err != nil {
		return nil, err
	}
	return proposals.History(ps, c.Manifest(), func(t manifest.Tenant) []string { return proposals.AgentLines(t, name) }), nil
}

// restore makes an open proposal of what change does to the applied manifest, audited
// first, like a steward's (SB-5104). Another operator approves it; nothing is applied here.
func (c *Cell) restore(ctx context.Context, s tenancy.Scope, space string, change func(manifest.Tenant) manifest.Tenant, reason string) (proposals.Proposal, error) {
	cur := c.Manifest()
	next, ds := manifest.Check(manifest.Format(change(cur)))
	if len(ds) > 0 {
		return proposals.Proposal{}, fmt.Errorf("%w: the restored version doesn't check: %s", ask.ErrInvalid, ds[0].Msg)
	}
	diff := manifest.Diff(cur, next)
	if len(diff) == 0 {
		return proposals.Proposal{}, fmt.Errorf("%w: that version is what is applied now", ask.ErrInvalid)
	}
	p := proposals.Proposal{ID: c.NewID("prop_"), Space: space, Requester: s.User(), Base: proposals.Hash(cur), Hash: proposals.Hash(next),
		Manifest: string(manifest.Format(next)), Diff: diff, State: proposals.Open, At: time.Now().UTC()}
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: p.ID, State: "proposed",
		PayloadSHA256: release.SHA256([]byte(p.Manifest)), Reason: reason}); err != nil {
		return proposals.Proposal{}, fmt.Errorf("%w: %v", ask.ErrUnavailable, err)
	}
	if err := c.Proposals.Put(ctx, p); err != nil {
		return proposals.Proposal{}, fmt.Errorf("%w: %v", ask.ErrUnavailable, err)
	}
	return p, nil
}

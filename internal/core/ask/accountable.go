package ask

import (
	"context"
	"fmt"

	"example.com/second-brain/internal/core/tenancy"
)

// Pause here and spend caps (sb-agents(7), Pause here, Spend caps; SB-4702, SB-4703): an
// agent paused in a space runs nothing more there, and one at its monthly cap runs no draft
// or routine until the month turns; on-demand asks go on under each space's budget.

// pausedIn says agent is paused in s's space.
func (a Agent) pausedIn(agent string, s tenancy.Scope) bool {
	return a.AgentPaused != nil && s.Space() != "company" && a.AgentPaused(agent, spaceName(s.Space()))
}

// runsStop is why agent may not run on its own in s's space now: paused there (ErrKilled),
// or at its cap or with spend that can't be read (ErrBudget).
func (a Agent) runsStop(ctx context.Context, s tenancy.Scope, agent string) error {
	if agent == "" {
		agent = "ask"
	}
	if a.pausedIn(agent, s) {
		return fmt.Errorf("%w: %s is paused in this space", ErrKilled, agent)
	}
	if a.AgentCapped == nil {
		return nil
	}
	capped, err := a.AgentCapped(ctx, agent)
	if err != nil || capped {
		return fmt.Errorf("%w: %s has spent its monthly budget", ErrBudget, agent) // unread spend pauses too
	}
	return nil
}

// MaxPauseReason is the longest reason a pause takes.
const MaxPauseReason = 300

// OwnedAgents are the agents s owns: their email, or a group they're in, is the agent's
// owner (SB-4402).
func (a Agent) OwnedAgents(ctx context.Context, s tenancy.Scope) ([]string, error) {
	var emails []string
	if a.Emails != nil {
		es, err := a.Emails(ctx, s.User())
		if err != nil {
			return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		emails = es
	}
	return AgentsOwnedBy(emails, s.Groups()), nil
}

// MayPause checks that s may pause or resume agent in space name: a shipped agent, and s a
// steward of the space or the agent's owner (SB-4702).
func (x Stewards) MayPause(ctx context.Context, s tenancy.Scope, name, agent string) error {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return ErrNotSteward
	}
	if AgentOwner(agent) == "" {
		return fmt.Errorf("%w: no agent %q", ErrInvalid, agent)
	}
	if x.Facts != nil {
		if _, _, ok := x.Facts(SpaceKey(name)); !ok {
			return fmt.Errorf("%w: no space %q", ErrInvalid, name)
		}
	}
	owned, err := x.Agent.OwnedAgents(ctx, s)
	if err != nil {
		return err
	}
	if contains(owned, agent) {
		return nil
	}
	_, _, err = x.steward(ctx, s, name)
	return err
}

// Package kill is the kill switches (sb-kill-switches(7), How it works, exactly; SB-403):
// an audited setter, and a per-cell checker that polls the store every 2 seconds, accepts
// pushes, and fails safe after 30 seconds without a successful load.
package kill

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// Timings from sb-kill-switches(7).
const (
	Poll     = 2 * time.Second
	FailSafe = 30 * time.Second
)

// Switch is one set kill switch.
type Switch struct {
	Scope  string    `json:"scope"` // "agent:NAME", "agent:NAME@SPACE", "connector:KIND", "space:NAME" or "tenant"
	Reason string    `json:"reason"`
	Actor  string    `json:"actor"`
	At     time.Time `json:"at"`
}

// Store is the control plane's kill store, per tenant.
type Store interface {
	Set(ctx context.Context, tenant string, s Switch) error
	Clear(ctx context.Context, tenant, scope string) error
	List(ctx context.Context, tenant string) ([]Switch, error)
}

// Errors.
var (
	ErrKilled  = errors.New("kill: EKILLED")
	ErrScope   = errors.New("kill: bad scope")
	ErrNoAudit = errors.New("kill: no audit log; refusing to change a switch untraced")
)

var scopeRe = regexp.MustCompile(`^(agent:[a-z][a-z0-9-]{0,40}(@[a-z][a-z0-9-]{1,30})?|connector:(docs|chat|tickets|crm|code|hris|accounting)|space:[a-z][a-z0-9-]{1,30}|tenant)$`)

// ValidScope reports whether s is a scope sb-kill(1) accepts.
func ValidScope(s string) bool { return scopeRe.MatchString(s) }

// Control sets and clears switches, each audited first (kind kill).
type Control struct {
	Store Store
	Audit *audit.Log // the tenant's audit log
	Now   func() time.Time
}

// Kill sets scope for tenant. The audit row is written before the switch.
func (c Control) Kill(ctx context.Context, tenant, scope, reason, actor string) error {
	return c.change(ctx, tenant, scope, reason, actor, "killed")
}

// Unkill clears scope for tenant, audited the same way.
func (c Control) Unkill(ctx context.Context, tenant, scope, reason, actor string) error {
	return c.change(ctx, tenant, scope, reason, actor, "cleared")
}

func (c Control) change(ctx context.Context, tenant, scope, reason, actor, state string) error {
	if !ValidScope(scope) {
		return fmt.Errorf("%w: %q", ErrScope, scope)
	}
	if reason == "" || actor == "" {
		return fmt.Errorf("kill: --reason and an actor are required")
	}
	if c.Audit == nil || c.Audit.Tenant != tenant {
		return ErrNoAudit
	}
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "kill", Subject: scope, State: state, Reason: reason}); err != nil {
		return fmt.Errorf("kill: audit: %w", err) // no trace, no change
	}
	if state == "killed" {
		return c.Store.Set(ctx, tenant, Switch{Scope: scope, Reason: reason, Actor: actor, At: c.Now().UTC()})
	}
	return c.Store.Clear(ctx, tenant, scope)
}

// Op is something a cell is about to start.
type Op struct {
	Agent         string // "" for none
	Connector     string // "" for none
	Consequential bool   // a tool call with effects outside the platform
	Admin         bool   // admin, audit read or unkill: allowed even when the tenant is killed
	Space         string // the space it runs in, by name ("" or "company" for none; SB-1008)
}

// Checker is one cell's view of its tenant's switches.
type Checker struct {
	Tenant string
	Store  Store
	Now    func() time.Time

	mu       sync.Mutex
	scopes   map[string]bool
	loaded   bool
	lastLoad time.Time
}

// Refresh loads the switch list; on failure the old list stays and the clock runs on.
func (c *Checker) Refresh(ctx context.Context) error {
	list, err := c.Store.List(ctx, c.Tenant)
	if err != nil {
		return err
	}
	c.Push(list)
	return nil
}

// Push replaces the list with one the control plane sent (or Refresh loaded).
func (c *Checker) Push(list []Switch) {
	m := map[string]bool{}
	for _, s := range list {
		m[s.Scope] = true
	}
	c.mu.Lock()
	c.scopes, c.loaded, c.lastLoad = m, true, c.Now()
	c.mu.Unlock()
}

// Run refreshes every Poll until ctx ends.
func (c *Checker) Run(ctx context.Context) {
	t := time.NewTicker(Poll)
	defer t.Stop()
	_ = c.Refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = c.Refresh(ctx)
		}
	}
}

// Check returns nil if op may start now, or an error wrapping ErrKilled with the reason.
func (c *Checker) Check(op Op) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case !c.loaded:
		if op.Admin || (!op.Consequential && op.Agent == "" && op.Connector == "") {
			return nil
		}
		return fmt.Errorf("%w: kill_list_not_loaded", ErrKilled)
	case op.Consequential && c.Now().Sub(c.lastLoad) >= FailSafe:
		return fmt.Errorf("%w: kill_store_unreachable", ErrKilled)
	case c.scopes["tenant"] && !op.Admin:
		return fmt.Errorf("%w: tenant", ErrKilled)
	case op.Agent != "" && c.scopes["agent:"+op.Agent]:
		return fmt.Errorf("%w: agent:%s", ErrKilled, op.Agent)
	case op.Agent != "" && op.Space != "" && c.scopes["agent:"+op.Agent+"@"+op.Space]: // SB-4702
		return fmt.Errorf("%w: agent:%s@%s", ErrKilled, op.Agent, op.Space)
	case op.Connector != "" && c.scopes["connector:"+op.Connector]:
		return fmt.Errorf("%w: connector:%s", ErrKilled, op.Connector)
	case op.Space != "" && op.Space != "company" && c.scopes["space:"+op.Space]:
		return fmt.Errorf("%w: space:%s", ErrKilled, op.Space)
	}
	return nil
}

// SpaceKilled reports whether a space (by name) is killed, or the list isn't loaded yet:
// retrieval drops every chunk such a space holds (sb-kill-switches(7); SB-1008).
func (c *Checker) SpaceKilled(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.loaded || c.scopes["space:"+name] || c.scopes["tenant"]
}

// Killed lists the scopes the checker believes are set, sorted (sb kill list).
func (c *Checker) Killed() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for s := range c.scopes {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// AgentKilled is the ask agent's Killed hook for one agent name.
func (c *Checker) AgentKilled(agent string) func(context.Context) bool {
	return func(context.Context) bool { return c.Check(Op{Agent: agent}) != nil }
}

// PauseScope is the scope of one agent paused in one space (SB-4702).
func PauseScope(agent, space string) string { return "agent:" + agent + "@" + space }

// Paused reports whether agent is paused in space (by name), or the list isn't loaded yet.
func (c *Checker) Paused(agent, space string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.loaded || c.scopes[PauseScope(agent, space)]
}

// PausedIn are the spaces agent is paused in, sorted.
func (c *Checker) PausedIn(agent string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := []string{}
	for s := range c.scopes {
		if sp, ok := strings.CutPrefix(s, "agent:"+agent+"@"); ok {
			out = append(out, sp)
		}
	}
	sort.Strings(out)
	return out
}

// ConnectorKilled is the live check's Killed hook (retrieve.Authoriser.Killed).
func (c *Checker) ConnectorKilled(kind string) bool { return c.Check(Op{Connector: kind}) != nil }

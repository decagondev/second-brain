package laws

import (
	"fmt"
	"math/rand"
	"time"
)

// LAW-23 Grants are consented and revocable (SB-804; ADR-0014, W13): a parent-org grant on
// a company space answers only the aggregates the company approved, never content, and
// stops answering within 5 minutes of being revoked.

// GrantSystem is a company space with a parent-org grant, on a fake clock.
type GrantSystem interface {
	// Grant approves aggregates (metric names) from space to the parent; it returns the grant id.
	Grant(space string, aggregates []string) (string, error)
	Revoke(id string) error
	Advance(d time.Duration)
	// Aggregate asks for one metric; answered is false when refused.
	Aggregate(space, metric string) (answered bool, err error)
	// Content asks for documents of the space; answered is false when refused.
	Content(space string) (answered bool, err error)
}

// NewGrantSystem builds the system under test: aggregate-only grants (SB-1302).
var NewGrantSystem = func() GrantSystem { return pendingGrants{} }

type pendingGrants struct{}

func (pendingGrants) Grant(string, []string) (string, error) {
	return "", fmt.Errorf("%w: parent-org space grants arrive with SB-1302 (and only if Q-01 is yes)", ErrPending)
}
func (pendingGrants) Revoke(string) error                    { return nil }
func (pendingGrants) Advance(time.Duration)                  {}
func (pendingGrants) Aggregate(string, string) (bool, error) { return false, nil }
func (pendingGrants) Content(string) (bool, error)           { return false, nil }

var grantMetrics = []string{"headcount", "attrition_rate", "revenue_month", "open_roles", "budget_variance"}

func law23(r *rand.Rand, sys GrantSystem) error {
	var approved []string
	for _, m := range grantMetrics {
		if r.Intn(2) == 0 {
			approved = append(approved, m)
		}
	}
	id, err := sys.Grant("finance", approved)
	if err != nil {
		return err
	}
	for _, m := range grantMetrics {
		ok, err := sys.Aggregate("finance", m)
		if err != nil {
			return err
		}
		if ok != contains(approved, m) {
			return fmt.Errorf("the grant approved %v, and %s was answered: %v", approved, m, ok)
		}
	}
	if ok, _ := sys.Content("finance"); ok {
		return fmt.Errorf("a grant answered with content, not aggregates")
	}
	if ok, _ := sys.Aggregate("people", "headcount"); ok {
		return fmt.Errorf("a space with no grant answered")
	}
	if err := sys.Revoke(id); err != nil {
		return err
	}
	sys.Advance(MoveWindow + time.Second)
	for _, m := range approved {
		if ok, _ := sys.Aggregate("finance", m); ok {
			return fmt.Errorf("%s still answered more than 5 minutes after the grant was revoked", m)
		}
	}
	return nil
}

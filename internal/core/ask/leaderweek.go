package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/tenancy"
)

// The leader's week (sb-agents(7), The leader's week; SB-4503): each Monday a department's
// stewards get a draft of its branch. Its claims are the leader-week agent's, every one
// cited, from a branch ask as the steward; then one line per team with counts only, never
// a title, a question or a person (LAW-24). A steward publishes it as the space's update.

// AgentLeaderWeek is the leader's week agent, and DraftLeaderWeek its draft's kind.
const (
	AgentLeaderWeek = "leader-week"
	DraftLeaderWeek = "leader-week"
)

// LeaderWeekQuestion is the fixed question of the leader's week.
const LeaderWeekQuestion = "What did this department and its teams publish, decide and check in on this week, and which goals are at risk?"

var leaderAgent = Department{Name: AgentLeaderWeek, Prompt: "leader-week", Lead: "Week: "}

// leaderDept checks that the leader-week agent may run in the scope's space.
func (a Agent) leaderDept(ctx context.Context, s tenancy.Scope) (Department, error) {
	return a.spaceAgent(ctx, s, leaderAgent, a.LeaderWeek, "a space with teams and its leader's week on")
}

// teamCounts are the lines of a leader's week after its claims: each team of space name
// with its open and overdue requests, its open questions and, with a budget, the share its
// branch has spent this month. Counts only, whoever reads them.
func (x Stewards) teamCounts(ctx context.Context, name string) (string, error) {
	if x.Children == nil {
		return "", nil
	}
	now := x.Now().UTC()
	var b strings.Builder
	for _, team := range x.Children(name) {
		qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(team))
		if err != nil {
			return "", fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		reqs, late, open := 0, 0, 0
		for _, q := range qs {
			if q.Reply != nil || q.Thread != nil {
				continue // answered, or a saved thread or an agent's draft
			}
			if q.Request != nil {
				reqs++
				if now.After(q.Request.Due) {
					late++
				}
			} else {
				open++
			}
		}
		fmt.Fprintf(&b, "\n- %s: %d open requests (%d overdue), %d open questions", team, reqs, late, open)
		if x.BudgetShare != nil {
			share, ok, err := x.BudgetShare(ctx, team)
			if err != nil {
				return "", fmt.Errorf("%w: budgets: %v", ErrUnavailable, err)
			}
			if ok {
				fmt.Fprintf(&b, ", %d%% of its budget spent", int(share*100+1e-9))
			}
		}
	}
	if b.Len() == 0 {
		return "", nil
	}
	return "\nBy team (counts only):" + b.String(), nil
}

// withCounts is a draft's cited lines with counts after them, dropping cited lines from the
// end until both fit in MaxReply.
func withCounts(text, counts string) string {
	if counts == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for len(lines) > 1 && utf8.RuneCountInString(strings.Join(lines, "\n")+"\n"+counts) > MaxReply {
		lines = lines[:len(lines)-1]
	}
	out := strings.Join(lines, "\n") + "\n" + counts
	if rs := []rune(out); len(rs) > MaxReply {
		out = string(rs[:MaxReply])
	}
	return out
}

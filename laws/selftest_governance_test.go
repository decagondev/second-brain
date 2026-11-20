package laws

import (
	"context"
	"sync"
	"testing"

	"example.com/second-brain/internal/core/proposals"
)

// SB-1401: two decisions at once on one proposal: exactly one wins, and a stale proposal
// is audited when it is closed.
func TestProposalDecisionsAreAtomicAndStaleIsAudited(t *testing.T) {
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	c, err := newStewardCell(ev, nil)
	if err != nil {
		t.Fatal(err)
	}
	sp := map[string]any{"template": "finance", "members": []string{"group:finance", "group:execs"}, "stewards": []string{"group:finance-leads"},
		"scope": map[string][]string{"docs": {"/Finance/**"}}}
	code, out := c.call("usr_lead", []string{"finance-leads"}, "POST", "/v1/spaces/finance/proposals", map[string]any{"space": sp})
	if code != 200 {
		t.Fatalf("propose: %d %v", code, out)
	}
	id, hash := out["id"].(string), out["hash"].(string)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i, who := range []string{"usr_op1", "usr_op2"} {
		wg.Add(1)
		go func(i int, who string) {
			defer wg.Done()
			if i == 0 {
				codes[i], _ = c.call(who, []string{"sb-operators"}, "POST", "/v1/admin/proposals/"+id+"/approve", map[string]any{"hash": hash})
			} else {
				codes[i], _ = c.call(who, []string{"sb-operators"}, "POST", "/v1/admin/proposals/"+id+"/reject", map[string]any{"reason": "no"})
			}
		}(i, who)
	}
	wg.Wait()
	if (codes[0] == 200) == (codes[1] == 200) {
		t.Fatalf("both or neither decision won: %v", codes)
	}
	// A stale proposal is closed with an audit row.
	code, out = c.call("usr_lead", []string{"finance-leads"}, "POST", "/v1/spaces/finance/proposals", map[string]any{"space": sp})
	if code != 200 {
		t.Fatalf("propose again: %d %v", code, out)
	}
	t2 := c.applied.t
	t2.RetentionDays = 300
	c.applied.t = t2
	code, _ = c.call("usr_op1", []string{"sb-operators"}, "POST", "/v1/admin/proposals/"+out["id"].(string)+"/approve", map[string]any{"hash": out["hash"]})
	p, _, _ := c.h.Proposals.Get(context.Background(), out["id"].(string))
	if code != 409 || p.State != proposals.Stale {
		t.Fatalf("stale approve: %d %s", code, p.State)
	}
	rows, _ := c.h.Audit.Store.Range(context.Background(), 1, 100)
	stale := false
	for _, r := range rows {
		stale = stale || (r.Subject == p.ID && r.State == proposals.Stale)
	}
	if !stale {
		t.Fatal("the stale proposal has no audit row")
	}
}

package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

func TestLAW14_DepartmentAgentsCiteEveryClaimInTheirOwnSpaces(t *testing.T) {
	Check(t, "LAW-14", 40, func(r *rand.Rand) error { return law14Departments(r, newDeptCell) })
}

func TestLAW09_DepartmentAgentsCantAct(t *testing.T) {
	Check(t, "LAW-09", 1, func(*rand.Rand) error { return law09Departments(ask.Departments) })
}

// The department law tests catch an agent that keeps uncited claims, one that runs in
// any space that lists it whatever the template, and a department agent with a tool.
func TestSelf_DepartmentMutationsAreCaught(t *testing.T) {
	keepsUncited := func(sp []deptSpace) (DeptAsk, error) {
		run, err := newDeptCell(sp)
		return func(a, s string) (ask.Response, error) {
			resp, err := run(a, s)
			if err == nil {
				resp.Answer.Claims = append(resp.Answer.Claims, resp.Answer.Claims[0])
				resp.Answer.Claims[len(resp.Answer.Claims)-1].Uncited = true
				resp.Answer.Claims[len(resp.Answer.Claims)-1].Cites = nil
			}
			return resp, err
		}, err
	}
	if !violates(40, func(r *rand.Rand) error { return law14Departments(r, keepsUncited) }) {
		t.Error("LAW-14 (departments): an agent that keeps uncited claims was not caught")
	}
	anyTemplate := func(sp []deptSpace) (DeptAsk, error) {
		seen := append([]deptSpace(nil), sp...)
		for i := range seen {
			seen[i].template = "engineering" // every space looks like Engineering to the agent
		}
		return newDeptCell(seen)
	}
	if !violates(40, func(r *rand.Rand) error { return law14Departments(r, anyTemplate) }) {
		t.Error("departments: an agent that ignores the space's template was not caught")
	}
	armed := map[string]ask.Department{}
	for k, v := range ask.Departments {
		armed[k] = v
	}
	d := armed["runbook-answer"]
	d.Tools = []string{"ticket-update.comment"}
	armed["runbook-answer"] = d
	if law09Departments(armed) == nil {
		t.Error("LAW-09 (departments): a department agent with a tool was not caught")
	}
}

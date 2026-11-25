package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
)

// LAW-14's table test goes red when a made-up number on the sheet is kept, and when the
// tool's arithmetic is wrong.
func TestSelf_LAW14CatchesAMadeUpNumber(t *testing.T) {
	keeps := func(sh TableSheet, plan, madeUp string) (ask.Response, error) {
		resp, err := RealTables(sh, plan, madeUp)
		resp.Answer.Claims = append(resp.Answer.Claims, answer.Claim{Text: "The invoices come to " + madeUp + ".",
			Cites: []answer.Cite{{Cite: "docs:invoices.csv@1"}}})
		return resp, err
	}
	wrong := func(sh TableSheet, plan, madeUp string) (ask.Response, error) {
		resp, err := RealTables(sh, plan, madeUp)
		if resp.Table != nil {
			resp.Table.Value++
		}
		return resp, err
	}
	for name, sys := range map[string]TableAsk{"keeps a made-up number": keeps, "adds up wrong": wrong} {
		caught := false
		for seed := int64(1); seed <= runs && !caught; seed++ {
			caught = law14Tables(rand.New(rand.NewSource(seed)), sys) != nil
		}
		if !caught {
			t.Fatalf("LAW-14 stayed green with a system that %s", name)
		}
	}
}

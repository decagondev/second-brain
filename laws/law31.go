package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-31 Lent access lasts one run, names the lender, and its result reaches only people who
// can open every cite (SB-5303; sb-laws(7)). Random lent drafts with random cites, and random
// readers each able to open a random part of them, through the real visibility gate: a
// reader sees a draft exactly when they can open all of its cites. And a loan, once
// answered, either way, can't be answered again, through the real loan service and store,
// so one approval is never a second run.

// LentGate is what the law checks: whether a reader sees a lent draft, given what they can
// open.
type LentGate func(cites []string, canOpen func(string) bool) bool

// RealLentGate is the real one.
var RealLentGate LentGate = ask.LentVisible

func law31(r *rand.Rand, gate LentGate) error {
	for i := 0; i < 20; i++ {
		var cites []string
		for j := 0; j < r.Intn(5); j++ {
			cites = append(cites, fmt.Sprintf("docs:d%d@1", r.Intn(8)))
		}
		open := map[string]bool{}
		for j := 0; j < 8; j++ {
			open[fmt.Sprintf("docs:d%d@1", j)] = r.Intn(3) > 0
		}
		want := true
		for _, c := range cites {
			want = want && open[c]
		}
		if got := gate(cites, func(c string) bool { return open[c] }); got != want {
			return fmt.Errorf("a reader who can open %v sees a draft citing %v: %v, want %v", open, cites, got, want)
		}
	}
	// A loan is answered once.
	ctx := context.Background()
	store := tracemem.New()
	n := 0
	x := ask.Stewards{Agent: ask.Agent{Tenant: "acme-logistics"}, Store: store, Audit: &keepRows{},
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}
	lender, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_lee", nil)
	if err != nil {
		return err
	}
	q := ask.StewardQuestion{ID: "qs_loan", Tenant: "acme-logistics", Asker: "usr_sam", Space: ask.PersonKey("usr_lee"), At: time.Now(),
		Question: "Lend finance for one run", Thread: &ask.SavedThread{Agent: "ask", Kind: ask.DraftLoan, Asked: "close status", Doc: "spc_finance", Link: "spc_ops"}}
	if err := store.PutStewardQuestion(ctx, q); err != nil {
		return err
	}
	runs := 0
	as := func(string) (tenancy.Scope, error) { runs++; return tenancy.Scope{}, errors.New("no run in this law") }
	first := r.Intn(2) == 0
	if _, err := x.Lend(ctx, lender, "qs_loan", first, as); first && err == nil || !first && err != nil {
		return fmt.Errorf("the first answer (%v): %v", first, err)
	}
	if _, err := x.Lend(ctx, lender, "qs_loan", true, as); !errors.Is(err, ask.ErrInvalid) {
		return fmt.Errorf("a loan was answered twice: %v", err)
	}
	if want := map[bool]int{true: 1, false: 0}[first]; runs != want {
		return fmt.Errorf("the loan ran %d times, want %d", runs, want)
	}
	return nil
}

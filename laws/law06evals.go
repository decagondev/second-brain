package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/runtime"
)

// A tenant's cases from wrong citations (SB-2204) under LAW-06: a release that fails one of
// the tenant's cases that an earlier release passed isn't run for that tenant; its updater
// runs the release the cases last passed on, admitted like any release. A case that never
// passed holds nothing. The real case runner and the runtime updater over the GCP fake.

// HoldSystem turns the cell's hold into the updater's Hold.
type HoldSystem interface {
	Hold(h *ask.Hold) func(ctx context.Context, target string) (string, error)
}

// RealHold is the cell's: the hold's Instead.
type RealHold struct{}

// Hold implements HoldSystem.
func (RealHold) Hold(h *ask.Hold) func(context.Context, string) (string, error) {
	return func(_ context.Context, target string) (string, error) {
		if h == nil {
			return "", nil
		}
		return h.Instead(target), nil
	}
}

func law06Evals(r *rand.Rand, w *runtimeWorld, sys HoldSystem) error {
	ctx := context.Background()
	r1, err := w.Release(true, true)
	if err != nil {
		return err
	}
	r2, err := w.Release(true, true)
	if err != nil {
		return err
	}
	// Cases; on each release an answer cites the expected document or the wrong one.
	n := 1 + r.Intn(4)
	var cases []ask.EvalCase
	for i := 0; i < n; i++ {
		cases = append(cases, ask.EvalCase{ID: fmt.Sprintf("ev_%d", i), Tenant: "gamma-parts", Space: "spc_ops", Question: "q",
			Expect: fmt.Sprintf("docs:right%d", i), Wrong: fmt.Sprintf("docs:wrong%d", i), By: "usr_sam", At: time.Unix(0, 0)})
	}
	right := map[string]map[string]bool{r1: {}, r2: {}} // release -> case -> answered right
	for _, c := range cases {
		right[r1][c.ID] = r.Intn(4) > 0 // some cases are still wrong on r1: they never passed
		right[r2][c.ID] = r.Intn(2) == 0
	}
	runOn := func(rel string) func(ask.EvalCase) (bool, error) {
		return func(c ask.EvalCase) (bool, error) {
			cite := c.Wrong
			if right[rel][c.ID] {
				cite = c.Expect
			}
			resp := ask.Response{Answer: ask.Answer{Claims: []answer.Claim{{Text: "x", Cites: []answer.Cite{{Cite: cite + "@1"}}}}}}
			return c.Passes(resp), nil
		}
	}
	update := func(target string, h *ask.Hold) (runtime.Result, error) {
		u := w.updater(target)
		u.Hold = sys.Hold(h)
		return u.Run(ctx)
	}
	if _, err := update(r1, nil); err != nil {
		return err
	}
	cases, hold, err := ask.RunCases(cases, r1, runOn(r1), nil)
	if err != nil {
		return err
	}
	if hold != nil {
		return fmt.Errorf("the first release the cases ran on is held: %+v", hold)
	}
	if _, err := update(r2, nil); err != nil { // the channel names r2; the cell runs it
		return err
	}
	_, hold, err = ask.RunCases(cases, r2, runOn(r2), nil)
	if err != nil {
		return err
	}
	regressed := false
	for _, c := range cases {
		if right[r1][c.ID] && !right[r2][c.ID] {
			regressed = true
		}
	}
	if regressed != (hold != nil) {
		return fmt.Errorf("r2 regresses a case that passed on r1: %v; held: %+v", regressed, hold)
	}
	res, err := update(r2, hold) // the next updater run, with the channel still on r2
	if err != nil {
		return err
	}
	m1, err := w.rw.manifest(r1)
	if err != nil {
		return err
	}
	want := r2
	if regressed {
		want = r1
	}
	for svc, img := range w.Running() {
		u := w.updater(r1)
		for _, s := range runtime.Services {
			if s.Name == svc && (img == u.Image(s, m1.Code[s.Image])) != (want == r1) {
				return fmt.Errorf("LAW-06: with r2 held %v, %s runs %s (want %s)", regressed, svc, img, want)
			}
		}
	}
	if regressed && (res.Release != r1 || res.Held != r2) {
		return fmt.Errorf("the updater ran %s, held %q; want %s held %s", res.Release, res.Held, r1, r2)
	}
	return nil
}

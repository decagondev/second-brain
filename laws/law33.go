package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
)

// LAW-33 A model call stays in its region (SB-5705; sb-laws(7)): with model.region set, no
// call of any tier, through the fallback or a batch, reaches a provider of another region
// or of none. Random regions for a primary and a fallback, random failures, gates and
// batchable calls, through the real gateways and fallback: every provider that answered a
// call was in the region, and a call nobody in the region could take answered nothing.

// RegionGateway is what the law checks: a gateway over m for a provider of region, under
// allowed.
type RegionGateway func(m ports.Model, region, allowed string) ports.Model

// RealRegionGateway is the real one.
var RealRegionGateway RegionGateway = func(m ports.Model, region, allowed string) ports.Model {
	return gateway.Gateway{Provider: "anthropic", Model: m, Secrets: passScrub{}, Region: region, Allowed: allowed,
		Sleep: func(context.Context, time.Duration) error { return nil }, BatchWait: time.Millisecond}
}

// regionModel records every call it was sent, and can fail.
type regionModel struct {
	fail  *bool
	calls int
}

func (m *regionModel) Complete(context.Context, ports.ModelRequest) (ports.ModelResponse, error) {
	m.calls++
	if *m.fail {
		return ports.ModelResponse{}, ports.ErrModelDown
	}
	return ports.ModelResponse{Text: "ok"}, nil
}

func (m *regionModel) SubmitBatch(context.Context, ports.ModelRequest) (string, error) {
	m.calls++
	return "b", nil
}

func (m *regionModel) CollectBatch(context.Context, string) (ports.ModelResponse, bool, error) {
	return ports.ModelResponse{Text: "ok"}, true, nil
}

func law33(r *rand.Rand, mk RegionGateway) error {
	regions := []string{"", "eu", "us", "jp"}
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		allowed := regions[r.Intn(len(regions))]
		pr, sr := regions[r.Intn(len(regions))], regions[r.Intn(len(regions))]
		pfail, sfail := r.Intn(2) == 0, r.Intn(3) == 0
		p, s := &regionModel{fail: &pfail}, &regionModel{fail: &sfail}
		gated := r.Intn(2) == 0
		f := gateway.NewFallback(mk(p, pr, allowed), mk(s, sr, allowed), func() bool { return gated }, func() time.Time { return now })
		for j := 0; j < 3; j++ {
			_, err := f.Complete(context.Background(), ports.ModelRequest{Batch: r.Intn(2) == 0})
			in := func(reg string) bool { return allowed == "" || (reg != "" && reg == allowed) }
			if p.calls > 0 && !in(pr) {
				return fmt.Errorf("run %d: a primary in %q was called under model.region %q", i, pr, allowed)
			}
			if s.calls > 0 && !in(sr) {
				return fmt.Errorf("run %d: a fallback in %q was called under model.region %q", i, sr, allowed)
			}
			if err == nil && !in(pr) && !in(sr) {
				return fmt.Errorf("run %d: answered with no provider in %q", i, allowed)
			}
		}
	}
	return nil
}

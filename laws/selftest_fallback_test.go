package laws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

type anyFallback struct{ p, s ports.Model }

func (f anyFallback) Complete(ctx context.Context, r ports.ModelRequest) (ports.ModelResponse, error) {
	if resp, err := f.p.Complete(ctx, r); err == nil {
		return resp, nil
	}
	return f.s.Complete(ctx, r)
}

// SB-5601: a fallback that ignores the gate is caught.
func TestSelfUngatedFallbackIsCaught(t *testing.T) {
	failsSomewhere(t, "a fallback that ignores the gate", func(r *rand.Rand) error {
		return law06Fallback(r, func(p, s ports.Model, _ func() bool, _ func() time.Time) ports.Model { return anyFallback{p, s} })
	})
}

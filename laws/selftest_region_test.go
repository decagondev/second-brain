package laws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
)

// SB-5705: a gateway that doesn't know its region (so calls go anywhere) is caught.
func TestSelfRegionBlindGatewayIsCaught(t *testing.T) {
	failsSomewhere(t, "a gateway that ignores model.region", func(r *rand.Rand) error {
		return law33(r, func(m ports.Model, _, _ string) ports.Model {
			return gateway.Gateway{Provider: "anthropic", Model: m, Secrets: passScrub{}, Sleep: func(context.Context, time.Duration) error { return nil }, BatchWait: time.Millisecond}
		})
	})
}

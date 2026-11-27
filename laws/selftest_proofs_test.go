package laws

import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/cell/prove"
)

// SB-6001: a proof maker that keeps a person's address as a seen id, unsealed, is caught.
func TestSelfProofsAreCaught(t *testing.T) {
	leaky := func(ctx context.Context, ticket string, env prove.Env) ([]byte, error) {
		p, err := prove.Run(ctx, ticket, env)
		if err != nil {
			return nil, err
		}
		p.Criteria[0].Seen = []string{env.Person}
		return json.Marshal(p)
	}
	failsSomewhere(t, "a proof with an unhashed address", func(r *rand.Rand) error { return law07Proofs(r, leaky) })
}

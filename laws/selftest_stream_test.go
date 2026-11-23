package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/auditstream"
)

// hopeful counts a row as taken whether or not the SIEM took it.
type hopeful struct{}

type hopefulSender struct{ real auditstream.Sender }

func (h hopefulSender) Send(ctx context.Context, msg []byte) error {
	_ = h.real.Send(ctx, msg)
	return nil
}

func (hopeful) Sender(real auditstream.Sender) auditstream.Sender { return hopefulSender{real} }

func TestSelf_StreamLawCatchesLostRows(t *testing.T) {
	if !violates(60, func(r *rand.Rand) error { return law11Stream(r, hopeful{}) }) {
		t.Fatal("the stream law passes a stream that loses the rows a down SIEM didn't take")
	}
}

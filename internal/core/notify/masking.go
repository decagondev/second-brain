package notify

import (
	"context"

	"example.com/second-brain/internal/core/ingest"
)

// Masking is a Sender that masks each message's text before Next sends it (sb-ingest(7),
// Nothing sensitive goes out; SB-5102). Masked is told how many strings changed; nil is
// told nothing.
type Masking struct {
	Next   Sender
	Masked func(ctx context.Context, n int)
}

// Send masks the message's text, then sends it.
func (m Masking) Send(ctx context.Context, to Person, msg Message) error {
	n := 0
	var changed bool
	if msg.Text, changed = ingest.MaskOut(msg.Text); changed {
		n++
	}
	if n > 0 && m.Masked != nil {
		m.Masked(ctx, n)
	}
	return m.Next.Send(ctx, to, msg)
}

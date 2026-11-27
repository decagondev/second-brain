package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// inlineReader is a connector that reads its pictures and recordings itself and hands the
// text over as the document's own: what the law forbids.
type inlineReader struct {
	ports.Connector
	vision, transcribe func(context.Context, ports.Part) (string, error)
}

func (c inlineReader) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	d, err := c.Connector.Fetch(ctx, ref)
	for _, p := range d.Images {
		t, _ := c.vision(ctx, p)
		d.Text += "\n\n" + t
	}
	for _, p := range d.Media {
		t, _ := c.transcribe(ctx, p)
		d.Text += "\n\n" + t
	}
	d.Images, d.Media = nil, nil
	return d, err
}

// SB-6101: text a model read, passed off as the document's own, is caught.
func TestSelfReadTextIsCaught(t *testing.T) {
	failsSomewhere(t, "a connector that reads pictures itself", func(r *rand.Rand) error {
		return law28Read(r, t.TempDir(), func(conn ports.Connector, v, tr func(context.Context, ports.Part) (string, error)) ingest.Pipeline {
			p := RealReadIngest(inlineReader{Connector: conn, vision: v, transcribe: tr}, nil, nil)
			return p
		})
	})
}

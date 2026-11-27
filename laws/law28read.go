package laws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// LAW-28 for what a model read (SB-6101, SB-6102; sb-laws(7)): text the vision or the
// transcribe tier read from a picture or a recording is marked wherever it is held and
// cited, never shown as the document's own words. Random folders of text, pictures and
// recordings through the real folder connector, ingest pipeline, index and composer, with
// readers that answer with recognisable text.

// ReadIngest is the system under test: the pipeline for a connector, with readers.
type ReadIngest func(conn ports.Connector, vision, transcribe func(context.Context, ports.Part) (string, error)) ingest.Pipeline

// RealReadIngest is the real pipeline.
var RealReadIngest ReadIngest = func(conn ports.Connector, vision, transcribe func(context.Context, ports.Part) (string, error)) ingest.Pipeline {
	return ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_law", Connector: conn, Embedder: hashing.Embedder{}, Index: memindex.New(),
		Vision: vision, Transcribe: transcribe}
}

func law28Read(r *rand.Rand, dir string, mk ReadIngest) error {
	root := filepath.Join(dir, fmt.Sprintf("read-%d", r.Int63()))
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(root) }()
	n := 1 + r.Intn(6)
	for i := 0; i < n; i++ {
		var name string
		var data []byte
		switch r.Intn(3) {
		case 0:
			name, data = fmt.Sprintf("note%d.md", i), []byte(strings.Repeat(fmt.Sprintf("Plain text %d of the person who wrote it. ", i), 1+r.Intn(30)))
		case 1:
			name, data = fmt.Sprintf("scan%d.png", i), []byte(fmt.Sprintf("\x89PNG picture %d", i))
		default:
			name, data = fmt.Sprintf("call%d.wav", i), []byte(fmt.Sprintf("RIFF recording %d", i))
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			return err
		}
	}
	read := func(tag string) func(context.Context, ports.Part) (string, error) {
		return func(_ context.Context, p ports.Part) (string, error) {
			return strings.Repeat(tag+" words a model read from "+p.Name+". ", 1+r.Intn(40)), nil
		}
	}
	conn, err := folder.Open(root, filepath.Join(root, ".state"))
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	p := mk(conn, read("MODELREADVISION"), read("MODELREADVOICE"))
	if _, err := p.Sync(context.Background(), ""); err != nil {
		return err
	}
	chunks, err := p.Index.All(context.Background(), "acme-logistics")
	if err != nil {
		return err
	}
	ctx := answer.NewContext(chunks)
	for label, ch := range ctx.Labels {
		read := strings.Contains(ch.Text, "MODELREADVISION") || strings.Contains(ch.Text, "MODELREADVOICE")
		if read && ch.ReadBy == "" {
			return fmt.Errorf("a chunk with text a model read isn't marked: %s", ch.Cite)
		}
		res := answer.Compose(`{"claims":[{"text":"x","cites":["`+label+`"]}]}`, ctx, nil)
		if read && (len(res.Claims) != 1 || len(res.Claims[0].Cites) != 1 || res.Claims[0].Cites[0].ReadBy == "") {
			return fmt.Errorf("a cite of what a model read isn't marked: %s", ch.Cite)
		}
	}
	return nil
}

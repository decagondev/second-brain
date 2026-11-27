package ingest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// partsFeed is one document with a picture on page 2 and a recording.
type partsFeed struct{ feed }

func (f *partsFeed) Fetch(_ context.Context, ref ports.SourceRef) (ports.Document, error) {
	return ports.Document{Ref: ref, Version: 1, Title: "Contract", Text: "The signed contract follows.",
		Images: []ports.Part{{Name: "contract.pdf", Page: 2, Data: []byte("jpeg")}}, Media: []ports.Part{{Name: "call.wav", Data: []byte("wav")}}}, nil
}

// SB-6101, SB-6102: what the readers read joins the document after its marker, and every
// chunk of it is marked with its reader and page; the source's own text isn't. With no
// readers, or a reader that fails, the parts are skipped and counted, and only the source's
// text is indexed.
func TestPartsAreReadAndMarked(t *testing.T) {
	f := &partsFeed{feed{changes: []ports.Change{{Ref: ports.SourceRef{Kind: "docs", ID: "c"}}}}}
	idx := &chunkIndex{m: map[string][]ports.Chunk{}}
	unread := map[string]int{}
	p := Pipeline{Tenant: "acme-logistics", Connector: f, Embedder: flatEmbed{}, Index: idx,
		Vision: func(context.Context, ports.Part) (string, error) {
			return strings.Repeat("Total due 1,200 by 31 March. ", 3), nil
		},
		Transcribe: func(context.Context, ports.Part) (string, error) { return "", errors.New("down") },
		Unread:     func(kind, what string) { unread[what]++ }}
	if _, err := p.Sync(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	var own, read int
	for _, cs := range idx.m {
		for _, c := range cs {
			switch {
			case strings.Contains(c.Text, "Total due"):
				read++
				if c.ReadBy != ReadByVision || c.Page != 2 {
					t.Fatalf("a chunk a model read: %+v", c)
				}
			case c.ReadBy != "":
				t.Fatalf("the source's own text is marked: %q", c.Text)
			default:
				own++
			}
		}
	}
	if read == 0 || unread["recording"] != 1 {
		t.Fatalf("read %d own %d unread %v", read, own, unread)
	}
	idx.m = map[string][]ports.Chunk{}
	p.Vision, p.Transcribe = nil, nil
	if _, err := p.Sync(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	for _, cs := range idx.m {
		for _, c := range cs {
			if c.ReadBy != "" || strings.Contains(c.Text, "Read by the") {
				t.Fatalf("no reader, but: %+v", c)
			}
		}
	}
	if unread["image"] != 1 || unread["recording"] != 2 {
		t.Fatalf("unread: %v", unread)
	}
}

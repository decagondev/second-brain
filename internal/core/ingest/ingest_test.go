package ingest

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

func TestChunksAreStableAndBounded(t *testing.T) {
	long := strings.Repeat("word ", 700) // 3,500 characters in one paragraph
	text := Normalise("Title\r\n\r\nfirst paragraph  \n\n" + long + "\n\nlast paragraph")
	a, b := Chunks(text), Chunks(text)
	if len(a) == 0 || len(a) != len(b) {
		t.Fatalf("chunks %v vs %v", a, b)
	}
	rs := []rune(text)
	for i, sp := range a {
		if sp != b[i] {
			t.Fatal("same text, different boundaries")
		}
		if sp.End-sp.Start > MaxChunk || sp.End <= sp.Start {
			t.Fatalf("span %v of %d chars", sp, sp.End-sp.Start)
		}
		if s := string(rs[sp.Start:sp.End]); s != strings.TrimSpace(s) {
			t.Fatalf("span %v has untrimmed edges", sp)
		}
	}
}

// An edit to the last paragraph leaves earlier chunks' boundaries alone.
func TestEditIsLocal(t *testing.T) {
	ps := []string{strings.Repeat("alpha ", 150), strings.Repeat("beta ", 150), strings.Repeat("gamma ", 150)}
	before := Chunks(Normalise(strings.Join(ps, "\n\n")))
	ps[2] = strings.Repeat("delta ", 150)
	after := Chunks(Normalise(strings.Join(ps, "\n\n")))
	if len(before) < 2 || before[0] != after[0] {
		t.Fatalf("first chunk moved: %v -> %v", before, after)
	}
}

func TestIDsAreDerived(t *testing.T) {
	ref := ports.SourceRef{Kind: "docs", ID: "d1"}
	if Cite(ref, 3) != "docs:d1@3" {
		t.Fatal(Cite(ref, 3))
	}
	sp := Span{0, 10}
	if ChunkID("acme-logistics", "docs:d1@3", sp) != ChunkID("acme-logistics", "docs:d1@3", sp) ||
		ChunkID("acme-logistics", "docs:d1@3", sp) == ChunkID("beta-freight", "docs:d1@3", sp) {
		t.Fatal("chunk ids must be derived from tenant, cite and span")
	}
}

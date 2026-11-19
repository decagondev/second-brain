// Package ingest turns source changes into indexed chunks with provenance (sb-ingest(7),
// SB-206). Chunking and provenance are pure; Fetch, Embed and the index write go through
// ports. Ingest is idempotent per (source, version), and the index can be rebuilt from
// the sources alone (LAW-13).
package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// MaxChunk is the largest chunk, in characters.
const MaxChunk = 1200

// Span is a chunk's position in the normalised text, in runes.
type Span struct{ Start, End int }

// Normalise fixes line endings and trailing spaces so the same text always chunks the same.
func Normalise(text string) string {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// Chunks splits normalised text into spans: paragraphs packed up to MaxChunk; a longer
// paragraph is cut at the last space before the limit. Boundaries depend only on text.
func Chunks(text string) []Span {
	rs := []rune(text)
	var paras []Span
	start := 0
	for i := 0; i <= len(rs); i++ {
		end := i == len(rs) || (rs[i] == '\n' && i+1 < len(rs) && rs[i+1] == '\n')
		if !end {
			continue
		}
		if s, e := trim(rs, start, i); e > s {
			paras = append(paras, cut(rs, s, e)...)
		}
		start = i + 1
	}
	var out []Span
	for _, p := range paras {
		if n := len(out); n > 0 && p.End-out[n-1].Start <= MaxChunk {
			out[n-1].End = p.End // pack into the previous chunk
			continue
		}
		out = append(out, p)
	}
	return out
}

func trim(rs []rune, s, e int) (int, int) {
	for s < e && (rs[s] == '\n' || rs[s] == ' ') {
		s++
	}
	for e > s && (rs[e-1] == '\n' || rs[e-1] == ' ') {
		e--
	}
	return s, e
}

// cut splits one paragraph longer than MaxChunk at the last space before the limit.
func cut(rs []rune, s, e int) []Span {
	var out []Span
	for e-s > MaxChunk {
		at := s + MaxChunk
		for at > s && rs[at] != ' ' {
			at--
		}
		if at == s {
			at = s + MaxChunk // no space: hard cut
		}
		out = append(out, Span{s, at})
		s = at
		for s < e && rs[s] == ' ' {
			s++
		}
	}
	return append(out, Span{s, e})
}

// Cite is a chunk's citation key: kind:id@version.
func Cite(ref ports.SourceRef, version int) string {
	return fmt.Sprintf("%s:%s@%d", ref.Kind, ref.ID, version)
}

// ChunkID is derived from tenant, cite and span, so re-ingest writes identical ids.
func ChunkID(tenant, cite string, sp Span) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%d", tenant, cite, sp.Start, sp.End)))
	return "chk_" + hex.EncodeToString(h[:12])
}

// Pipeline ingests one tenant's documents from one connector.
type Pipeline struct {
	Tenant, Release string
	Connector       ports.Connector
	Embedder        ports.Embedder
	Index           ports.Index
}

// Ingest fetches ref and replaces its chunks, unless that version (or newer) is indexed.
// force re-ingests the same version (permission-only changes).
func (p Pipeline) Ingest(ctx context.Context, ref ports.SourceRef, force bool) error {
	doc, err := p.Connector.Fetch(ctx, ref)
	if err != nil {
		return fmt.Errorf("ingest: fetch %s:%s: %w", ref.Kind, ref.ID, err) // ref only, never content
	}
	if have, found, err := p.Index.Indexed(ctx, p.Tenant, ref); err != nil {
		return err
	} else if found && (have > doc.Version || (have == doc.Version && !force)) {
		return nil
	}
	text := Normalise(doc.Title + "\n\n" + doc.Text)
	spans := Chunks(text)
	rs := []rune(text)
	cite := Cite(ref, doc.Version)
	acl := append([]string(nil), doc.ACLPrefilter...)
	sort.Strings(acl)
	chunks := make([]ports.Chunk, len(spans))
	texts := make([]string, len(spans))
	for i, sp := range spans {
		t := string(rs[sp.Start:sp.End])
		h := sha256.Sum256([]byte(t))
		texts[i] = t
		chunks[i] = ports.Chunk{ID: ChunkID(p.Tenant, cite, sp), Tenant: p.Tenant, Cite: cite, Kind: ref.Kind, SourceID: ref.ID,
			Version: doc.Version, Start: sp.Start, End: sp.End, Text: t, ContentSHA256: hex.EncodeToString(h[:]),
			ACLPrefilter: acl, Release: p.Release}
	}
	if len(texts) > 0 {
		vecs, err := p.Embedder.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("ingest: embed %s: %w", cite, err)
		}
		if len(vecs) != len(texts) {
			return fmt.Errorf("ingest: embedder returned %d vectors for %d chunks", len(vecs), len(texts))
		}
		for i := range chunks {
			chunks[i].Vector = vecs[i]
		}
	}
	return p.Index.Replace(ctx, p.Tenant, ref, doc.Version, chunks)
}

// Sync handles every change after cursor (the last change per document wins) and
// returns the next cursor. A failure returns the old cursor, so a retry loses nothing.
func (p Pipeline) Sync(ctx context.Context, cursor ports.Cursor) (ports.Cursor, error) {
	set, next, err := p.Connector.Changes(ctx, cursor)
	if err != nil {
		return cursor, err
	}
	type last struct {
		ch    ports.Change
		force bool
	}
	latest := map[string]last{}
	var order []string
	for _, ch := range set {
		k := ch.Ref.Kind + "\x00" + ch.Ref.ID
		l, seen := latest[k]
		if !seen {
			order = append(order, k)
		}
		latest[k] = last{ch: ch, force: l.force || ch.ACLOnly}
	}
	for _, k := range order {
		l := latest[k]
		if l.ch.Deleted {
			if err := p.Index.Delete(ctx, p.Tenant, l.ch.Ref); err != nil {
				return cursor, err
			}
			continue
		}
		if err := p.Ingest(ctx, l.ch.Ref, l.force); err != nil {
			return cursor, err
		}
	}
	return next, nil
}

// Rebuild drops the tenant's index and replays every source document from the start.
func (p Pipeline) Rebuild(ctx context.Context) error {
	if err := p.Index.Drop(ctx, p.Tenant); err != nil {
		return err
	}
	_, err := p.Sync(ctx, "")
	return err
}

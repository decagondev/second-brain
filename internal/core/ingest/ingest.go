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
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
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
	Sanitiser       Sanitiser // strip and flag rules (SB-207)
	// Spaces returns the cell's spaces now (SB-1005); nil is a cell without spaces. An
	// error stops ingest: a document is never written without its routing.
	Spaces func(ctx context.Context) ([]ports.Space, error)
	// Route is the routing rule; nil is route.Route, the only one a cell ships.
	Route func(spaces []ports.Space, kind, location, sourceLabel string) ([]string, string)
	// SecretDropped counts a chunk dropped for holding a secret (sb_ingest_secrets_dropped_total,
	// by connector kind and pattern name); nil counts nothing, and the chunk is dropped anyway.
	SecretDropped func(kind, pattern string)
	// BadReviewBy counts a document whose review-by date can't be read, which is ignored
	// (sb_ingest_review_by_ignored_total; SB-2705). nil counts nothing.
	BadReviewBy func(kind string)
	// FieldsDropped reports a structured document's dropped fields by name, never value
	// (sb_ingest_fields_dropped_total); "(undeclared)" is a source that can't declare them.
	FieldsDropped func(kind string, names []string)
	// Secrets is the secret scanner; nil is SecretIn, the only one a cell ships.
	Secrets func(text string) string
	// RerouteDropped counts a document Reroute took out of the index because it couldn't
	// be fetched (sb_ingest_reroute_dropped_total{connector}); nil counts nothing.
	RerouteDropped func(kind string)
	// Sharing is told when a document's ACL copy changes from the version indexed before
	// (SB-5403); nil records nothing. It never changes the source.
	Sharing func(ctx context.Context, ch SharingChange)
	// Kind is the connector kind this pipeline ingests; Reroute touches only its documents.
	// Empty means the index holds this connector's documents only.
	Kind string
	// Now stamps when the index first saw a version (Chunk.Seen, SB-1701); nil leaves it
	// unknown. Re-ingesting the same version keeps the first time (the index does that).
	Now func() time.Time
	// Pace spaces reads to sb-cell(5) sync.max_per_minute (SB-6004); nil is unpaced. A 429
	// is waited out either way, and Throttled counts each wait by kind (nil counts nothing).
	Pace      *Pace
	Throttled func(kind string)

	// During a rebuild: each version's first-seen time from before the drop, and whether
	// versions it doesn't know are marked rebuilt (SB-1708).
	carry      map[string]time.Time
	rebuilding bool
}

func (p Pipeline) spaces(ctx context.Context) ([]ports.Space, error) {
	if p.Spaces == nil {
		return nil, nil
	}
	sp, err := p.Spaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("ingest: spaces: %w", err)
	}
	return sp, nil
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
	spaces, err := p.spaces(ctx)
	if err != nil {
		return err
	}
	var holders []string
	label, routing := "", route.Fingerprint(spaces)
	if len(spaces) > 0 {
		rt := p.Route
		if rt == nil {
			rt = route.Route
		}
		holders, label = rt(spaces, ref.Kind, doc.Location, doc.Label)
	}
	if doc.Fields != nil {
		text, sensitive, narrowed, ok := p.fieldText(ref.Kind, doc, spaces, holders)
		if !ok {
			// A structured source that can't declare its fields is never ingested (SB-1106).
			return p.Index.Delete(ctx, p.Tenant, ref)
		}
		doc.Text = text
		if sensitive {
			label = "restricted" // an allow-listed sensitive field (LAW-20)
			holders = narrowed   // only the spaces that allow it (SB-1401)
		}
	}
	reviewBy := p.reviewBy(ref.Kind, doc)
	mark := MarkOf(doc.Text, doc.Owner) // SB-4706
	clean, docFlags := p.Sanitiser.Sanitise(doc.Title + "\n\n" + doc.Text)
	text := Normalise(clean)
	spans := Chunks(text)
	rs := []rune(text)
	hs := headings(text)
	lineOf := lines(rs, doc.Title, doc.Text)
	cite := Cite(ref, doc.Version)
	acl := append([]string(nil), doc.ACLPrefilter...)
	sort.Strings(acl)
	title := Title(doc.Title, text)
	var seen time.Time
	if p.Now != nil {
		seen = p.Now().UTC()
	}
	rebuilt := false
	if t, ok := p.carry[cite]; ok {
		seen = t // the version was indexed before the rebuild: it keeps its time
	} else if p.rebuilding {
		rebuilt = true
	}
	textHash := "" // copies share it (SB-2202); never the text itself
	if body := Normalise(doc.Text); body != "" {
		h := sha256.Sum256([]byte(body))
		textHash = hex.EncodeToString(h[:])
	}
	chunks := make([]ports.Chunk, 0, len(spans))
	texts := make([]string, 0, len(spans))
	scan := p.Secrets
	whole := []SecretSpan(nil)
	if scan == nil {
		scan = SecretIn
		whole = SecretSpans(text) // secrets a chunk boundary would split (SB-1401)
	}
	pii := PIISpans(text) // numbers a chunk boundary would cut still flag both chunks (SB-1904)
	secretRemoved := false
	for _, sp := range spans {
		t := string(rs[sp.Start:sp.End])
		pat := overlaps(whole, sp.Start, sp.End)
		if pat == "" {
			pat = scan(t)
		}
		if pat != "" {
			// A secret is never embedded, indexed or logged; only the pattern is counted (LAW-07).
			if p.SecretDropped != nil {
				p.SecretDropped(ref.Kind, pat)
			}
			secretRemoved = true
			continue
		}
		h := sha256.Sum256([]byte(t))
		texts = append(texts, t)
		chunks = append(chunks, ports.Chunk{ID: ChunkID(p.Tenant, cite, sp), Tenant: p.Tenant, Cite: cite, Kind: ref.Kind, SourceID: ref.ID,
			Version: doc.Version, Start: sp.Start, End: sp.End, Text: t, ContentSHA256: hex.EncodeToString(h[:]),
			ACLPrefilter: acl, Release: p.Release, Flags: merge(merge(docFlags, p.Sanitiser.Flag(t)), personal(pii, sp)),
			Label: label, Spaces: holders, Routing: routing, Owner: strings.ToLower(strings.TrimSpace(doc.Owner)),
			Title: title, Seen: seen, Location: doc.Location, Rebuilt: rebuilt, TextSHA256: textHash, ReviewBy: reviewBy, Section: sectionAt(hs, sp.Start), Lines: [2]int{lineOf(sp.Start), lineOf(sp.End - 1)}, Page: PageAt(doc.Pages, sp.Start),
			GeneratedBy: mark.GeneratedBy, ApprovedBy: mark.ApprovedBy, Edited: mark.Edited, AskedBy: mark.AskedBy, EditedBy: mark.EditedBy})
	}
	if secretRemoved { // the document's other chunks say so, for sharing checks (SB-5401)
		for i := range chunks {
			chunks[i].Flags = merge(chunks[i].Flags, []string{FlagSecretRemoved})
		}
	}
	if fm := Sensitivity(doc.Text); fm != "" { // SB-5405: front matter only ever raises a label
		for i := range chunks {
			chunks[i].Label = Highest(chunks[i].Label, fm)
		}
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
	var before []string
	if p.Sharing != nil {
		before = p.aclBefore(ctx, ref)
	}
	if err := p.Index.Replace(ctx, p.Tenant, ref, doc.Version, chunks); err != nil {
		return err
	}
	if p.Sharing != nil && before != nil {
		if ch := SharingChangeOf(before, acl); ch.Changed() {
			ch.Ref, ch.Version, ch.Owner, ch.Spaces = ref, doc.Version, strings.ToLower(strings.TrimSpace(doc.Owner)), holders
			ch.Sensitive = sensitiveChunks(chunks)
			p.Sharing(ctx, ch)
		}
	}
	return nil
}

// reviewBy is the document's own review-by date: the source's, or its front matter's.
func (p Pipeline) reviewBy(kind string, doc ports.Document) time.Time {
	raw := doc.ReviewBy
	if raw == "" {
		raw = FrontMatterDate(doc.Text)
	}
	if raw == "" {
		return time.Time{}
	}
	t, ok := ParseReviewBy(raw)
	if !ok && p.BadReviewBy != nil {
		p.BadReviewBy(kind)
	}
	return t
}

// MaxTitle is the longest indexed title, in characters.
const MaxTitle = 200

// Title is a document's title as indexed: the first line of its sanitised, normalised text
// when the source gave a title (ingest puts the title first), cut to MaxTitle; "" otherwise.
// It is taken from the sanitised text, so a stripped secret or injection never reaches it.
func Title(source, text string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	line, _, _ := strings.Cut(text, "\n")
	if rs := []rune(line); len(rs) > MaxTitle {
		line = string(rs[:MaxTitle])
	}
	return strings.TrimSpace(line)
}

// Sync handles every change after cursor (the last change per document wins) and
// returns the next cursor. A failure returns the old cursor, so a retry loses nothing.
func (p Pipeline) Sync(ctx context.Context, cursor ports.Cursor) (ports.Cursor, error) {
	return p.sync(ctx, cursor, 0, nil)
}

func (p Pipeline) sync(ctx context.Context, cursor ports.Cursor, skip int, progress Progress) (ports.Cursor, error) {
	if err := p.Reroute(ctx); err != nil {
		return cursor, err
	}
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
	report := func(done int) error {
		if progress == nil {
			return nil
		}
		return progress(ctx, Checkpoint{Done: done, Total: len(order)})
	}
	if err := report(min(skip, len(order))); err != nil {
		return cursor, err
	}
	for i, k := range order {
		if i < skip {
			continue
		}
		if i > skip && i%CheckpointEvery == 0 {
			if err := report(i); err != nil {
				return cursor, err
			}
		}
		l := latest[k]
		if l.ch.Deleted {
			if err := p.Index.Delete(ctx, p.Tenant, l.ch.Ref); err != nil {
				return cursor, err
			}
			continue
		}
		if err := p.patient(ctx, l.ch.Ref, l.force); err != nil {
			return cursor, err
		}
	}
	if err := report(len(order)); err != nil {
		return cursor, err
	}
	return next, nil
}

// Reroute re-ingests every document whose chunks were routed under another space
// configuration than the cell's now, so a changed scope or label takes effect on the next
// sync (sb-ingest(7), Spaces and labels). A document of another connector is left alone.
func (p Pipeline) Reroute(ctx context.Context) error {
	if p.Spaces == nil {
		return nil // a cell without spaces routes nothing
	}
	spaces, err := p.spaces(ctx)
	if err != nil {
		return err
	}
	now := route.Fingerprint(spaces)
	all, err := p.Index.All(ctx, p.Tenant)
	if err != nil {
		return err
	}
	stale := map[ports.SourceRef]bool{}
	var refs []ports.SourceRef
	for _, c := range all {
		ref := ports.SourceRef{Kind: c.Kind, ID: c.SourceID}
		if c.Routing != now && !stale[ref] && (p.Kind == "" || p.Kind == c.Kind) {
			stale[ref] = true
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	for _, ref := range refs {
		if err := p.Ingest(ctx, ref, true); err != nil {
			// A document that can't be fetched now (deleted, moved, out of reach) is taken
			// out of the index rather than left routed by an old configuration, and the
			// sync goes on: one bad document never wedges a connector (SB-1401). It comes
			// back when it next changes at the source, or on a rebuild.
			if derr := p.Index.Delete(ctx, p.Tenant, ref); derr != nil {
				return derr
			}
			if p.RerouteDropped != nil {
				p.RerouteDropped(ref.Kind)
			}
		}
	}
	return nil
}

// Rebuild drops the tenant's index and replays every source document from the start.
func (p Pipeline) Rebuild(ctx context.Context) error {
	// First-seen times are carried across, so a rebuild never looks like a week of changes;
	// an index that can't be read loses them, and every version is marked rebuilt.
	p.rebuilding, p.carry = true, map[string]time.Time{}
	if old, err := p.Index.All(ctx, p.Tenant); err == nil {
		for _, c := range old {
			if !c.Seen.IsZero() && !c.Rebuilt {
				if t, ok := p.carry[c.Cite]; !ok || c.Seen.Before(t) {
					p.carry[c.Cite] = c.Seen
				}
			}
		}
	}
	if err := p.Index.Drop(ctx, p.Tenant); err != nil {
		return err
	}
	_, err := p.Sync(ctx, "")
	return err
}

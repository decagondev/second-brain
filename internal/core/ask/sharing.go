package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Shared too widely (sb-permissions(7), Shared too widely; SB-5401 to SB-5403): a document
// whose ACL copy opens it to anyone with the link, the whole domain or a group above the
// cell's size, and that holds something sensitive, is listed for its owner and stewards and
// held back from everyone else's answers until the owner narrows it at the source or says it
// is shared on purpose. The brain never changes a source's sharing; holding only narrows
// (LAW-19).

// DefaultWideGroup is the group size above which a group grant is wide, unless the cell sets
// another (sb-cell(5), wide_group_size).
const DefaultWideGroup = 500

// The wide_sensitive modes of a space (sb-space(5)).
const (
	WideHold = "hold"
	WideWarn = "warn"
)

// The reasons a document is sensitive, as the list shows them.
const (
	ReasonPersonal   = "personal_data"
	ReasonSecret     = "secret_removed"
	ReasonLabel      = "label"
	ReasonRestricted = "restricted_space"
)

// MaxWideReason is the longest Shared on purpose reason, in characters.
const MaxWideReason = 200

// Wide is a cell's sharing checks. A nil Agent.Wide checks nothing.
type Wide struct {
	// Limit is the group size above which a group grant is wide; 0 is DefaultWideGroup.
	Limit int
	// GroupSize is a directory group's member count by the name in its grant (group:NAME);
	// false is a group the directory doesn't know, which is never counted wide.
	GroupSize func(ctx context.Context, name string) (int, bool)
	// Mode is a space's wide_sensitive mode by key: "" or hold, or warn.
	Mode func(space string) string
	// Stewards are a space's stewards by key, as user:ID and group:NAME.
	Stewards func(space string) []string
	// Restricted says a space, by key, is labelled restricted.
	Restricted func(space string) bool
	// Confirms keeps the owners' Shared on purpose; nil keeps none, so nothing is released.
	Confirms WideConfirmStore
	// Changes keeps each document's sharing changes (SB-5403); nil keeps none.
	Changes SharingStore
}

func (w *Wide) limit() int {
	if w.Limit > 0 {
		return w.Limit
	}
	return DefaultWideGroup
}

// WideConfirm is an owner's Shared on purpose: it releases the hold on one version shared
// one way, and lapses when either changes.
type WideConfirm struct {
	Tenant  string    `json:"tenant"`
	Doc     string    `json:"document"` // KIND:ID
	Version int       `json:"version"`
	ACL     string    `json:"acl"` // SHA-256 of the ACL copy it confirmed
	Reason  string    `json:"reason"`
	By      string    `json:"by"`
	At      time.Time `json:"at"`
}

// WideConfirmStore keeps one confirmation per document.
type WideConfirmStore interface {
	PutWideConfirm(ctx context.Context, c WideConfirm) error
	WideConfirm(ctx context.Context, tenant, doc string) (WideConfirm, bool, error)
}

// SharingEvent is one change to a document's sharing seen at sync (SB-5403): grants, never
// content. Widened is a group, domain or anyone added.
type SharingEvent struct {
	Tenant    string    `json:"tenant"`
	ID        string    `json:"id"`
	Doc       string    `json:"document"` // KIND:ID
	Version   int       `json:"version"`
	Owner     string    `json:"owner,omitempty"`
	Spaces    []string  `json:"spaces,omitempty"`
	Added     []string  `json:"added,omitempty"`
	Removed   []string  `json:"removed,omitempty"`
	Widened   bool      `json:"widened"`
	Sensitive bool      `json:"sensitive"`
	At        time.Time `json:"at"`
}

// SharingStore keeps sharing events.
type SharingStore interface {
	PutSharingEvent(ctx context.Context, e SharingEvent) error
	SharingEvents(ctx context.Context, tenant string, since time.Time) ([]SharingEvent, error)
}

// ACLHash is the SHA-256 of an ACL copy, order ignored.
func ACLHash(acl []string) string {
	xs := append([]string(nil), acl...)
	sort.Strings(xs)
	h := sha256.Sum256([]byte(strings.Join(xs, "\n")))
	return hex.EncodeToString(h[:])
}

// widest is an ACL copy's widest grant: anyone, then a domain, then the largest group above
// the limit, with that group's size; "" for none.
func (w *Wide) widest(ctx context.Context, acl []string) (string, int) {
	best, size := "", 0
	for _, g := range acl {
		switch {
		case g == "anyone":
			return g, 0
		case strings.HasPrefix(g, "domain:"):
			if !strings.HasPrefix(best, "domain:") {
				best, size = g, 0
			}
		case strings.HasPrefix(g, "group:") && !strings.HasPrefix(best, "domain:") && w.GroupSize != nil:
			if n, ok := w.GroupSize(ctx, strings.TrimPrefix(g, "group:")); ok && n > w.limit() && n > size {
				best, size = g, n
			}
		}
	}
	return best, size
}

// broad says an ACL copy has any grant that could be wide, before asking for sizes.
func broad(acl []string) bool {
	for _, g := range acl {
		if ingest.IsBroadGrant(g) {
			return true
		}
	}
	return false
}

// sensitivity is why a document's newest chunks are sensitive, sorted; nil for nothing.
func (w *Wide) sensitivity(cs []ports.Chunk) []string {
	set := map[string]bool{}
	for _, c := range cs {
		for _, f := range c.Flags {
			switch f {
			case ingest.FlagPersonal:
				set[ReasonPersonal] = true
			case ingest.FlagSecretRemoved:
				set[ReasonSecret] = true
			}
		}
		if ingest.SensitiveLabel(c.Label) {
			set[ReasonLabel+":"+c.Label] = true
		}
		for _, sp := range c.Spaces {
			if w.Restricted != nil && w.Restricted(sp) {
				set[ReasonRestricted] = true
			}
		}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// WideDoc is one document shared too widely: its widest grant and why it is sensitive.
type WideDoc struct {
	Doc     string
	Chunk   ports.Chunk // its first chunk, newest version
	Grant   string
	People  int // the group's size, for a group grant
	Reasons []string
}

// wideDocs are the documents among chunks (all of each document's chunks, any versions)
// whose newest version is shared too widely, by KIND:ID.
func (w *Wide) wideDocs(ctx context.Context, chunks []ports.Chunk) map[string]WideDoc {
	byDoc := map[string][]ports.Chunk{}
	for _, c := range latest(chunks) {
		k := c.Kind + ":" + c.SourceID
		byDoc[k] = append(byDoc[k], c)
	}
	out := map[string]WideDoc{}
	for k, cs := range byDoc {
		first := firsts(cs)[k]
		if !broad(first.ACLPrefilter) {
			continue
		}
		reasons := w.sensitivity(cs)
		if len(reasons) == 0 {
			continue
		}
		grant, n := w.widest(ctx, first.ACLPrefilter)
		if grant == "" {
			continue
		}
		out[k] = WideDoc{Doc: k, Chunk: first, Grant: grant, People: n, Reasons: reasons}
	}
	return out
}

// confirmed says an owner's Shared on purpose covers d as it is now. A store that can't be
// read confirms nothing: unknown is held.
func (w *Wide) confirmed(ctx context.Context, tenant string, d WideDoc) (WideConfirm, bool) {
	if w.Confirms == nil {
		return WideConfirm{}, false
	}
	c, ok, err := w.Confirms.WideConfirm(ctx, tenant, d.Doc)
	if err != nil || !ok || c.Version != d.Chunk.Version || c.ACL != ACLHash(d.Chunk.ACLPrefilter) {
		return WideConfirm{}, false
	}
	return c, true
}

// mode is the wide_sensitive mode for an ask in space: the space's own, or company-wide,
// warn only when every holding space warns.
func (w *Wide) mode(space string, holders []string) string {
	if w.Mode == nil {
		return WideHold
	}
	if space != "company" && space != "" && !IsPerson(space) {
		if w.Mode(space) == WideWarn {
			return WideWarn
		}
		return WideHold
	}
	if len(holders) == 0 {
		return WideHold
	}
	for _, h := range holders {
		if w.Mode(h) != WideWarn {
			return WideHold
		}
	}
	return WideWarn
}

// exempt says s's person is the document's owner or a steward of a space that holds it.
func (a Agent) exempt(s tenancy.Scope, c ports.Chunk, mine map[string]bool) bool {
	if c.Owner != "" && mine[strings.ToLower(c.Owner)] {
		return true
	}
	if a.Wide.Stewards == nil {
		return false
	}
	groups := map[string]bool{}
	for _, g := range s.Groups() {
		groups["group:"+g] = true
	}
	for _, sp := range c.Spaces {
		for _, st := range a.Wide.Stewards(sp) {
			if st == "user:"+s.User() || groups[st] {
				return true
			}
		}
	}
	return false
}

// held collects the documents one answer left out, and those it warned about.
type wideHeld struct {
	mu           sync.Mutex
	held, warned map[string]bool
}

type heldKey struct{}

func withWideHeld(ctx context.Context) (context.Context, *wideHeld) {
	if h, ok := ctx.Value(heldKey{}).(*wideHeld); ok {
		return ctx, h
	}
	h := &wideHeld{held: map[string]bool{}, warned: map[string]bool{}}
	return context.WithValue(ctx, heldKey{}, h), h
}

func (h *wideHeld) counts() (int, int) {
	if h == nil {
		return 0, 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.held), len(h.warned)
}

// heldFor is which of chunks' documents s's person can't be shown now because they are
// shared too widely and not confirmed (SB-5402), and which they are warned about.
func (a Agent) heldFor(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk) (map[string]bool, map[string]bool, error) {
	heldDocs, warned := map[string]bool{}, map[string]bool{}
	if a.Wide == nil || len(chunks) == 0 {
		return heldDocs, warned, nil
	}
	var refs []ports.SourceRef
	seen := map[string]bool{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if seen[k] || !broad(c.ACLPrefilter) {
			continue
		}
		seen[k] = true
		refs = append(refs, ports.SourceRef{Kind: c.Kind, ID: c.SourceID})
	}
	if len(refs) == 0 {
		return heldDocs, warned, nil
	}
	all := chunks
	if lk, ok := a.Index.(ports.DocLookup); ok {
		cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
		if err != nil {
			return nil, nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
		all = append(append([]ports.Chunk{}, cs...), chunks...)
	}
	wide := a.Wide.wideDocs(ctx, all)
	if len(wide) == 0 {
		return heldDocs, warned, nil
	}
	mine := a.addresses(ctx, s)
	for k, d := range wide {
		if _, ok := a.Wide.confirmed(ctx, a.Tenant, d); ok || a.exempt(s, d.Chunk, mine) {
			continue
		}
		if a.Wide.mode(s.Space(), d.Chunk.Spaces) == WideWarn {
			warned[k] = true
			continue
		}
		heldDocs[k] = true
	}
	return heldDocs, warned, nil
}

// dropHeld takes the held documents out of chunks, and records them and the warned ones on
// ctx's collector.
func (a Agent) dropHeld(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk) ([]ports.Chunk, error) {
	heldDocs, warned, err := a.heldFor(ctx, s, chunks)
	if err != nil {
		return nil, err
	}
	if len(heldDocs) == 0 && len(warned) == 0 {
		return chunks, nil
	}
	h, _ := ctx.Value(heldKey{}).(*wideHeld)
	out := chunks[:0:0]
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if h != nil && (heldDocs[k] || warned[k]) {
			h.mu.Lock()
			if heldDocs[k] {
				h.held[k] = true
			} else {
				h.warned[k] = true
			}
			h.mu.Unlock()
		}
		if !heldDocs[k] {
			out = append(out, c)
		}
	}
	return out, nil
}

// WideItem is one document on the review list's wide section: never its text.
type WideItem struct {
	Result
	Grant     string       `json:"grant"`
	People    int          `json:"people,omitempty"`
	Reasons   []string     `json:"reasons"`
	Held      bool         `json:"held"`
	Confirmed *WideConfirm `json:"confirmed,omitempty"`
	// Widened are the grants a sync added since the last version, for a sensitive document
	// whose sharing widened in the last 30 days (SB-5403).
	Widened []string `json:"widened,omitempty"`
}

// WideList is the review list's wide section.
type WideList struct {
	Items  []WideItem   `json:"items"`
	Hidden []HiddenItem `json:"hidden"`
}

// WidenedWindow is how long a sensitive document whose sharing widened stays on the list.
const WidenedWindow = 30 * 24 * time.Hour

// Wide is the space's documents shared too widely, and its sensitive documents whose
// sharing widened lately, checked as the steward (sb-ask(3), Shared too widely).
func (x Stewards) Wide(ctx context.Context, s tenancy.Scope, name string) (WideList, error) {
	out := WideList{Items: []WideItem{}, Hidden: []HiddenItem{}}
	if err := x.reviewer(ctx, s, name); err != nil {
		return out, err
	}
	a := x.Agent
	if a.Wide == nil {
		return out, nil
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	key := SpaceKey(name)
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Space: key})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var mine []ports.Chunk
	for _, c := range cs {
		if contains(c.Spaces, key) {
			mine = append(mine, c)
		}
	}
	wide := a.Wide.wideDocs(ctx, mine)
	widened := map[string][]string{}
	if a.Wide.Changes != nil {
		evs, err := a.Wide.Changes.SharingEvents(ctx, a.Tenant, a.Now().Add(-WidenedWindow))
		if err != nil {
			return out, fmt.Errorf("%w: sharing: %v", ErrUnavailable, err)
		}
		for _, e := range evs {
			if e.Widened && e.Sensitive && contains(e.Spaces, key) {
				widened[e.Doc] = append(widened[e.Doc], e.Added...)
			}
		}
	}
	firstOf := firsts(latest(mine))
	var cand []ports.Chunk
	for k, c := range firstOf {
		if _, ok := wide[k]; ok || widened[k] != nil {
			cand = append(cand, c)
		}
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].Kind+":"+cand[i].SourceID < cand[j].Kind+":"+cand[j].SourceID })
	if len(cand) == 0 {
		return out, nil
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	gone, err := x.ownersGone(ctx, cand)
	if err != nil {
		return out, err
	}
	allowed := map[string]bool{}
	own := a.owners()
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		allowed[k] = true
		r := a.result(ctx, own, c)
		r.Snippet = "" // never the text
		it := WideItem{Result: r, Reasons: []string{}, Widened: widened[k]}
		if d, ok := wide[k]; ok {
			it.Grant, it.People, it.Reasons = d.Grant, d.People, d.Reasons
			if cf, ok := a.Wide.confirmed(ctx, a.Tenant, d); ok {
				it.Confirmed = &cf
			} else {
				it.Held = a.Wide.mode(key, c.Spaces) == WideHold
			}
		}
		out.Items = append(out.Items, it)
	}
	out.Hidden = hiddenOf(cand, allowed, gone)
	return out, nil
}

// OwnWide are the person's own documents shared too widely and not confirmed, as inbox
// items (SB-5401): read from the index, then checked as them like any search.
func (a Agent) OwnWide(ctx context.Context, s tenancy.Scope) ([]InboxItem, error) {
	if a.Wide == nil {
		return nil, nil
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	mine := a.addresses(ctx, s)
	if len(mine) == 0 {
		return nil, nil
	}
	var owners []string
	for e := range mine {
		owners = append(owners, e)
	}
	sort.Strings(owners)
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Owners: owners})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	wide := a.Wide.wideDocs(ctx, cs)
	var cand []ports.Chunk
	for _, d := range wide {
		if _, ok := a.Wide.confirmed(ctx, a.Tenant, d); !ok {
			cand = append(cand, d.Chunk)
		}
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].Cite < cand[j].Cite })
	if len(cand) == 0 {
		return nil, nil
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, len(cand))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	var out []InboxItem
	for _, c := range res.Context {
		d := wide[c.Kind+":"+c.SourceID]
		link := "/?tab=inbox"
		if a.Link != nil {
			if u := a.Link(c); u != "" {
				link = u
			}
		}
		out = append(out, InboxItem{Kind: InboxWide, ID: c.Cite, Title: c.Title, At: c.Seen,
			Why: "Shared with " + GrantName(d.Grant) + "; holds " + strings.Join(d.Reasons, ", "), Link: link})
	}
	return out, nil
}

// GrantName is a grant as a person reads it.
func GrantName(g string) string {
	switch {
	case g == "anyone":
		return "anyone with the link"
	case strings.HasPrefix(g, "domain:"):
		return "everyone at " + strings.TrimPrefix(g, "domain:")
	case strings.HasPrefix(g, "group:"):
		return "the group " + strings.TrimPrefix(g, "group:")
	}
	return g
}

// ConfirmWide is the owner's Shared on purpose for cite, the document's current version
// (SB-5402): audited, and it lasts until the document or its sharing changes.
func (x Stewards) ConfirmWide(ctx context.Context, s tenancy.Scope, cite, reason string) (WideConfirm, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return WideConfirm{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > MaxWideReason || strings.ContainsAny(reason, "\n\r") {
		return WideConfirm{}, fmt.Errorf("%w: a reason is 1 to %d characters on one line", ErrInvalid, MaxWideReason)
	}
	a := x.Agent
	if a.Wide == nil || a.Wide.Confirms == nil {
		return WideConfirm{}, fmt.Errorf("%w: this cell keeps no sharing confirmations", ErrUnavailable)
	}
	doc := docKey(cite)
	kind, id, _ := strings.Cut(doc, ":")
	lk, ok := a.Index.(ports.DocLookup)
	if !ok || id == "" {
		return WideConfirm{}, fmt.Errorf("%w: a cite is KIND:ID@VERSION", ErrInvalid)
	}
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return WideConfirm{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	cur := firsts(latest(cs))[doc]
	if cur.ID == "" || cur.Owner == "" || !a.addresses(ctx, s)[strings.ToLower(cur.Owner)] {
		return WideConfirm{}, ErrNotOwner
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return WideConfirm{}, err
	}
	if res, err := retrieve.Allowed(ctx, s, who, a.Auth, []ports.Chunk{cur}, 1); err != nil {
		return WideConfirm{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	} else if len(res.Context) == 0 {
		return WideConfirm{}, ErrNotOwner
	}
	c := WideConfirm{Tenant: a.Tenant, Doc: doc, Version: cur.Version, ACL: ACLHash(cur.ACLPrefilter), Reason: reason, By: s.User(), At: x.Now().UTC()}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: cur.Cite, State: "shared_on_purpose",
		Reason: "sharing confirmed by its owner"}); err != nil {
		return WideConfirm{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := a.Wide.Confirms.PutWideConfirm(ctx, c); err != nil {
		return WideConfirm{}, fmt.Errorf("%w: sharing: %v", ErrUnavailable, err)
	}
	return c, nil
}

// RecordSharing keeps one sharing change seen at sync (SB-5403).
func (a Agent) RecordSharing(ctx context.Context, ch ingest.SharingChange) error {
	if a.Wide == nil || a.Wide.Changes == nil {
		return nil
	}
	e := SharingEvent{Tenant: a.Tenant, ID: a.NewID("shr_"), Doc: ch.Ref.Kind + ":" + ch.Ref.ID, Version: ch.Version, Owner: ch.Owner,
		Spaces: ch.Spaces, Added: ch.Added, Removed: ch.Removed, Widened: ch.Widened(), Sensitive: ch.Sensitive, At: a.Now().UTC()}
	return a.Wide.Changes.PutSharingEvent(ctx, e)
}

package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Retire from answers (sb-ask(3), Documents to review; SB-2801): a steward takes a
// superseded document out of their own space's answers. It only narrows (LAW-19), and it is
// on the document's id, so a rebuild keeps it (LAW-13).

// MaxRetireReason is the longest reason, in characters.
const MaxRetireReason = 200

// Retirement is one document taken out of a space's answers.
type Retirement struct {
	Tenant     string    `json:"tenant"`
	Space      string    `json:"space"`    // spc_NAME
	Doc        string    `json:"document"` // kind:source id
	Reason     string    `json:"reason"`
	ReplacedBy string    `json:"replaced_by,omitempty"`
	By         string    `json:"by"`
	At         time.Time `json:"at"`
}

// RetireStore keeps retirements, one per space and document.
type RetireStore interface {
	PutRetirement(ctx context.Context, r Retirement) error
	Retirements(ctx context.Context, tenant, space string) ([]Retirement, error)
	DeleteRetirement(ctx context.Context, tenant, space, doc string) (bool, error)
}

// RetiredItem is a retirement as a steward sees it: the title only of a document they can
// open now.
type RetiredItem struct {
	Doc        string    `json:"document"`
	Title      string    `json:"title,omitempty"`
	Reason     string    `json:"reason"`
	ReplacedBy string    `json:"replaced_by,omitempty"`
	By         string    `json:"by"`
	At         time.Time `json:"at"`
}

// RetiredList is a space's retirements: those the steward can open, and a count of the rest.
type RetiredList struct {
	Items  []RetiredItem `json:"items"`
	Hidden int           `json:"hidden"`
}

// held is the current first chunk of each of docs that space name holds and s can open now.
func (x Stewards) held(ctx context.Context, s tenancy.Scope, name string, docs ...string) (map[string]ports.Chunk, error) {
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return nil, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var refs []ports.SourceRef
	for _, d := range docs {
		kind, id, _ := strings.Cut(d, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var cur []ports.Chunk
	for _, c := range firsts(latest(cs)) {
		if contains(c.Spaces, SpaceKey(name)) {
			cur = append(cur, c)
		}
	}
	out := map[string]ports.Chunk{}
	if len(cur) == 0 {
		return out, nil
	}
	sort.Slice(cur, func(i, j int) bool { return cur[i].ID < cur[j].ID })
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cur, len(cur))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	for _, c := range res.Context {
		out[c.Kind+":"+c.SourceID] = c
	}
	return out, nil
}

// Retire takes the cited document out of the space's answers, with a reason and an
// optional replacement the space holds.
func (x Stewards) Retire(ctx context.Context, s tenancy.Scope, name, cite, reason, replacedBy string) (Retirement, error) {
	if err := x.reviewer(ctx, s, name); err != nil {
		return Retirement{}, err
	}
	if x.Agent.Retired == nil {
		return Retirement{}, fmt.Errorf("%w: this cell keeps no retirements", ErrUnavailable)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > MaxRetireReason || strings.ContainsAny(reason, "\n\r") {
		return Retirement{}, fmt.Errorf("%w: a reason is one line of 1 to %d characters", ErrInvalid, MaxRetireReason)
	}
	doc := docKey(cite)
	if _, id, ok := strings.Cut(doc, ":"); !ok || id == "" {
		return Retirement{}, fmt.Errorf("%w: a cite is KIND:ID@VERSION", ErrInvalid)
	}
	replacedBy = strings.TrimSpace(replacedBy)
	if replacedBy == doc {
		return Retirement{}, fmt.Errorf("%w: a document can't replace itself", ErrInvalid)
	}
	want := []string{doc}
	if replacedBy != "" {
		want = append(want, replacedBy)
	}
	open, err := x.held(ctx, s, name, want...)
	if err != nil {
		return Retirement{}, err
	}
	if _, ok := open[doc]; !ok {
		return Retirement{}, ErrNotSteward // another space's document, or none, looks the same as one they can't open
	}
	if _, ok := open[replacedBy]; replacedBy != "" && !ok {
		return Retirement{}, ErrNotSteward
	}
	r := Retirement{Tenant: x.Agent.Tenant, Space: SpaceKey(name), Doc: doc, Reason: reason, ReplacedBy: replacedBy, By: s.User(), At: x.Now().UTC()}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: doc, State: "retired",
		Reason: "retired from answers", Space: auditSpace(r.Space)}); err != nil {
		return Retirement{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Agent.Retired.PutRetirement(ctx, r); err != nil {
		return Retirement{}, fmt.Errorf("%w: retirements: %v", ErrUnavailable, err)
	}
	return r, nil
}

// Restore brings a retired document back into the space's answers.
func (x Stewards) Restore(ctx context.Context, s tenancy.Scope, name, doc string) error {
	if err := x.reviewer(ctx, s, name); err != nil {
		return err
	}
	if x.Agent.Retired == nil {
		return ErrNotSteward
	}
	rs, err := x.Agent.Retired.Retirements(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return fmt.Errorf("%w: retirements: %v", ErrUnavailable, err)
	}
	found := false
	for _, r := range rs {
		found = found || r.Doc == doc
	}
	if !found {
		return ErrNotSteward
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: doc, State: "restored",
		Reason: "brought back into answers", Space: auditSpace(SpaceKey(name))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if _, err := x.Agent.Retired.DeleteRetirement(ctx, x.Agent.Tenant, SpaceKey(name), doc); err != nil {
		return fmt.Errorf("%w: retirements: %v", ErrUnavailable, err)
	}
	return nil
}

// RetiredOf lists the space's retirements for a steward.
func (x Stewards) RetiredOf(ctx context.Context, s tenancy.Scope, name string) (RetiredList, error) {
	out := RetiredList{Items: []RetiredItem{}}
	if err := x.reviewer(ctx, s, name); err != nil {
		return out, err
	}
	if x.Agent.Retired == nil {
		return out, nil
	}
	rs, err := x.Agent.Retired.Retirements(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil || len(rs) == 0 {
		if err != nil {
			err = fmt.Errorf("%w: retirements: %v", ErrUnavailable, err)
		}
		return out, err
	}
	var docs []string
	for _, r := range rs {
		docs = append(docs, r.Doc)
	}
	open, err := x.held(ctx, s, name, docs...)
	if err != nil {
		return out, err
	}
	for _, r := range rs {
		c, ok := open[r.Doc]
		if !ok {
			out.Hidden++
			continue
		}
		out.Items = append(out.Items, RetiredItem{Doc: r.Doc, Title: c.Title, Reason: r.Reason, ReplacedBy: r.ReplacedBy, By: r.By, At: r.At.UTC()})
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].At.After(out.Items[j].At) })
	return out, nil
}

// retiredBy reads each space's retirements once per request.
type retiredBy struct {
	store  RetireStore
	tenant string
	by     map[string]map[string]Retirement
}

func (r *retiredBy) of(ctx context.Context, space string) (map[string]Retirement, error) {
	if m, ok := r.by[space]; ok {
		return m, nil
	}
	list, err := r.store.Retirements(ctx, r.tenant, space)
	if err != nil {
		return nil, err
	}
	m := map[string]Retirement{}
	for _, x := range list {
		m[x.Doc] = x
	}
	r.by[space] = m
	return m, nil
}

// dropRetired takes out of chunks the documents retired where s asks: in a space, those
// that space retired; company-wide, those every holding space retired.
func (a Agent) dropRetired(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk) ([]ports.Chunk, error) {
	if a.Retired == nil || len(chunks) == 0 {
		return chunks, nil
	}
	rb := &retiredBy{store: a.Retired, tenant: a.Tenant, by: map[string]map[string]Retirement{}}
	out := chunks[:0:0]
	for _, c := range chunks {
		doc := c.Kind + ":" + c.SourceID
		spaces := c.Spaces
		if s.Space() != "company" {
			spaces = []string{s.Space()}
		}
		retired := len(spaces) > 0
		for _, sp := range spaces {
			m, err := rb.of(ctx, sp)
			if err != nil {
				return nil, fmt.Errorf("%w: retirements: %v", ErrUnavailable, err)
			}
			if _, ok := m[doc]; !ok {
				retired = false
				break
			}
		}
		if !retired {
			out = append(out, c)
		}
	}
	return out, nil
}

// markReplaces sets Replaces on each cite of a document a retirement names as the
// replacement, with the old document's title, when s can open the old one now.
func (a Agent) markReplaces(ctx context.Context, s tenancy.Scope, claims []answer.Claim, chunks []ports.Chunk) error {
	lk, ok := a.Index.(ports.DocLookup)
	if a.Retired == nil || !ok {
		return nil
	}
	rb := &retiredBy{store: a.Retired, tenant: a.Tenant, by: map[string]map[string]Retirement{}}
	spacesOf := map[string][]string{}
	for _, c := range chunks {
		spacesOf[c.Cite] = c.Spaces
	}
	old := map[string]string{} // replacement doc -> retired doc
	for _, cl := range claims {
		for _, ci := range cl.Cites {
			for _, sp := range spacesOf[ci.Cite] {
				m, err := rb.of(ctx, sp)
				if err != nil {
					return fmt.Errorf("%w: retirements: %v", ErrUnavailable, err)
				}
				for _, r := range m {
					if r.ReplacedBy == docKey(ci.Cite) {
						old[r.ReplacedBy] = r.Doc
					}
				}
			}
		}
	}
	if len(old) == 0 {
		return nil
	}
	var refs []ports.SourceRef
	for _, d := range old {
		kind, id, _ := strings.Cut(d, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var cur []ports.Chunk
	for _, c := range firsts(latest(cs)) {
		cur = append(cur, c)
	}
	sort.Slice(cur, func(i, j int) bool { return cur[i].ID < cur[j].ID })
	who, err := a.who(ctx, s)
	if err != nil {
		return err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cur, len(cur))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	titles := map[string]string{}
	for _, c := range res.Context {
		titles[c.Kind+":"+c.SourceID] = c.Title
	}
	for i := range claims {
		for j := range claims[i].Cites {
			if t := titles[old[docKey(claims[i].Cites[j].Cite)]]; t != "" {
				claims[i].Cites[j].Replaces = t
			}
		}
	}
	return nil
}

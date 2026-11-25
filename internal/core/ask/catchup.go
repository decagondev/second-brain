package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Catch me up (sb-ask(3), Work across departments; sb-agents(7), The legwork agents;
// SB-4506): one cited summary of what changed in a space, a branch or a collection since a
// day, at most MaxCatchUpDays back. It is one ask on the catch-up agent as the person, now,
// narrowed to the documents with a version first seen since then, so a document they have
// lost since is left out by the same live check as any ask (LAW-26).

// AgentCatchUp is the catch-up agent.
const AgentCatchUp = "catch-up"

// The limits of a catch-up.
const (
	MaxCatchUpDays     = 90
	MaxCatchUpDocs     = 40
	DefaultCatchUpDays = 7
)

// CatchUpQuestion is the catch-up agent's question since a day.
func CatchUpQuestion(since string) string {
	return "What changed here since " + since + ": decisions, documents changed, published updates and goals whose status changed?"
}

// CatchUpRequest is POST /v1/catch-up.
type CatchUpRequest struct {
	Since      string `json:"since,omitempty"` // YYYY-MM-DD
	Collection string `json:"collection,omitempty"`
	Language   string `json:"language,omitempty"`
}

// CatchUpFacts are what a catch-up read: since when, how many documents, and how many of
// the person's Needs you items are in those spaces.
type CatchUpFacts struct {
	Since     string `json:"since"`
	Documents int    `json:"documents"`
	Waiting   int    `json:"waiting"`
}

// CatchUpResponse is the ask's response with what it read.
type CatchUpResponse struct {
	Response
	CatchUp CatchUpFacts `json:"catch_up"`
}

// CatchUpOffer is the inbox's offer to catch up once after a notification pause.
type CatchUpOffer struct {
	Since string `json:"since"`
}

type onlyKey struct{}

// withOnly narrows an ask to these documents, past MaxDocuments: a catch-up's changed ones
// (SB-4506), or a redraft's first sources (SB-4602).
func withOnly(ctx context.Context, docs []string) context.Context {
	return context.WithValue(ctx, onlyKey{}, docs)
}

func onlyDocs(ctx context.Context) []string {
	d, _ := ctx.Value(onlyKey{}).([]string)
	return d
}

// catchUpSince is the day a catch-up starts, checked: today at most, MaxCatchUpDays back at
// least, DefaultCatchUpDays back without one.
func catchUpSince(day string, now time.Time, zone *time.Location) (time.Time, string, error) {
	z := zoneOr(zone)
	if strings.TrimSpace(day) == "" {
		day = now.In(z).AddDate(0, 0, -DefaultCatchUpDays).Format("2006-01-02")
	}
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(day), z)
	n := now.In(z)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, z)
	if err != nil || t.After(today) || t.Before(today.AddDate(0, 0, -MaxCatchUpDays)) {
		return time.Time{}, "", fmt.Errorf("%w: since is a date from today back to %d days ago", ErrInvalid, MaxCatchUpDays)
	}
	return t.UTC(), t.Format("2006-01-02"), nil
}

// CatchUp is one cited summary, as s now, of what changed since req.Since in s's space and
// the branch scopes ctx carries, or in one collection s may read.
func (l Legwork) CatchUp(ctx context.Context, s tenancy.Scope, req CatchUpRequest) (CatchUpResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != l.Agent.Tenant {
		return CatchUpResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	since, day, err := catchUpSince(req.Since, l.now(), l.Zone)
	if err != nil {
		return CatchUpResponse{}, err
	}
	lk, ok := l.Agent.Index.(ports.DocLookup)
	if !ok {
		return CatchUpResponse{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var chunks []ports.Chunk
	var spaces []string                  // names
	scopes := map[string]tenancy.Scope{} // a document's space's scope, to check it as s there
	if req.Collection != "" {
		c, err := l.Agent.readCollection(ctx, s, l.Agent.Collections, req.Collection)
		if err != nil {
			return CatchUpResponse{}, err
		}
		var refs []ports.SourceRef
		for _, it := range c.Items {
			if kind, id, ok := strings.Cut(it, ":"); ok && !strings.HasPrefix(it, "sq_") {
				refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
			}
		}
		if len(refs) > 0 {
			if chunks, err = lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Refs: refs, SeenSince: since}); err != nil {
				return CatchUpResponse{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
			}
		}
	} else {
		if s.Space() == "" || s.Space() == "company" || IsPerson(s.Space()) {
			return CatchUpResponse{}, fmt.Errorf("%w: catch me up runs in a space, a branch or a collection", ErrInvalid)
		}
		keys := []string{s.Space()}
		in := map[string]tenancy.Scope{s.Space(): s}
		for _, b := range tenancy.BranchFrom(ctx, s) {
			if !contains(keys, b.Space()) {
				keys = append(keys, b.Space())
				in[b.Space()] = b
			}
		}
		for _, k := range keys {
			cs, err := lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Space: k, SeenSince: since})
			if err != nil {
				return CatchUpResponse{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
			}
			for _, c := range cs {
				if !contains(c.Spaces, k) {
					continue // the space's own documents only
				}
				if _, ok := scopes[c.Kind+":"+c.SourceID]; !ok {
					scopes[c.Kind+":"+c.SourceID] = in[k]
				}
				chunks = append(chunks, c)
			}
			spaces = append(spaces, spaceName(k))
		}
	}
	docs, err := l.openDocs(ctx, s, scopes, changedDocs(chunks, since), chunks)
	if err != nil {
		return CatchUpResponse{}, err
	}
	out := CatchUpResponse{CatchUp: CatchUpFacts{Since: day, Documents: len(docs)}}
	if out.CatchUp.Waiting, err = l.waitingIn(ctx, s, spaces); err != nil {
		return CatchUpResponse{}, err
	}
	if len(docs) == 0 {
		out.Response = Response{Answer: Answer{Claims: []answer.Claim{}}, People: []PersonRef{}, Agent: AgentCatchUp}
		return out, nil
	}
	resp, err := l.Agent.Ask(withOnly(ctx, docs), s, Request{Question: CatchUpQuestion(day), Agent: AgentCatchUp, Language: req.Language})
	if err != nil {
		return CatchUpResponse{}, err
	}
	out.Response = resp
	return out, nil
}

// changedDocs are the documents of chunks with a version first seen at or after since,
// never a rebuilt one, newest first, at most MaxCatchUpDocs, as KIND:ID.
func changedDocs(chunks []ports.Chunk, since time.Time) []string {
	newest := map[string]time.Time{}
	for _, c := range chunks {
		if c.Rebuilt || c.Seen.Before(since) {
			continue
		}
		k := c.Kind + ":" + c.SourceID
		if c.Seen.After(newest[k]) {
			newest[k] = c.Seen
		}
	}
	keys := make([]string, 0, len(newest))
	for k := range newest {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return newest[keys[i]].After(newest[keys[j]]) || newest[keys[i]].Equal(newest[keys[j]]) && keys[i] < keys[j]
	})
	if len(keys) > MaxCatchUpDocs {
		keys = keys[:MaxCatchUpDocs]
	}
	return keys
}

// waitingIn counts s's Needs you items in spaces (by name); none for no spaces.
func (l Legwork) waitingIn(ctx context.Context, s tenancy.Scope, spaces []string) (int, error) {
	if len(spaces) == 0 || l.Research == nil {
		return 0, nil
	}
	in, err := l.inbox(ctx, s, false)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, it := range in.NeedsYou {
		if contains(spaces, spaceName(it.Space)) {
			n++
		}
	}
	return n, nil
}

// openDocs are the documents of keys s can open now, in order: a document they lost since
// is neither asked about nor counted (LAW-01, LAW-26).
func (l Legwork) openDocs(ctx context.Context, s tenancy.Scope, scopes map[string]tenancy.Scope, keys []string, chunks []ports.Chunk) ([]string, error) {
	first := firsts(latest(chunks))
	bySpace := map[string][]ports.Chunk{}
	in := map[string]tenancy.Scope{}
	var order []string
	for _, k := range keys {
		c, ok := first[k]
		if !ok {
			continue
		}
		sc, ok := scopes[k]
		if !ok {
			sc = s
		}
		if _, ok := bySpace[sc.Space()]; !ok {
			order = append(order, sc.Space())
			in[sc.Space()] = sc
		}
		bySpace[sc.Space()] = append(bySpace[sc.Space()], c)
	}
	open := map[string]ports.Chunk{}
	for _, sp := range order {
		got, err := l.Agent.allowedDocs(ctx, in[sp], bySpace[sp])
		if err != nil {
			return nil, err
		}
		for k, c := range got {
			open[k] = c
		}
	}
	var out []string
	for _, k := range keys {
		if _, ok := open[k]; ok {
			out = append(out, k)
		}
	}
	return out, nil
}

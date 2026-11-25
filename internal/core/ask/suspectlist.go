package ask

import (
	"context"
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// MaxSuspect is the most documents the suspect section lists (SB-5101).
const MaxSuspect = 100

// SuspectItem is one document with suspect passages, and how many (SB-5101).
type SuspectItem struct {
	Result
	Passages int `json:"passages"`
}

// SuspectList is the review list's suspect section: documents the steward can open, and
// the owners of the others. Never a passage's text.
type SuspectList struct {
	Items  []SuspectItem `json:"items"`
	Hidden []HiddenItem  `json:"hidden"`
}

// suspectDocs are the documents space key holds whose newest version has a suspect passage:
// one chunk of each (its first) and the counts by KIND:ID, newest first.
func suspectDocs(cs []ports.Chunk, key string) ([]ports.Chunk, map[string]int) {
	count := map[string]int{}
	for _, c := range latest(cs) {
		if contains(c.Spaces, key) && ingest.Suspect(c.Text, c.Flags) {
			count[c.Kind+":"+c.SourceID]++
		}
	}
	var docs []ports.Chunk
	for k, c := range firsts(latest(cs)) {
		if count[k] > 0 {
			docs = append(docs, c)
		}
	}
	sort.Slice(docs, func(i, j int) bool {
		if !docs[i].Seen.Equal(docs[j].Seen) {
			return docs[i].Seen.After(docs[j].Seen)
		}
		return docs[i].Kind+":"+docs[i].SourceID < docs[j].Kind+":"+docs[j].SourceID
	})
	return docs, count
}

// Suspect lists the documents the space holds with a suspect passage, checked as the
// steward (sb-ask(3), Suspect passages).
func (x Stewards) Suspect(ctx context.Context, s tenancy.Scope, name string) (SuspectList, error) {
	out := SuspectList{Items: []SuspectItem{}, Hidden: []HiddenItem{}}
	if err := x.reviewer(ctx, s, name); err != nil {
		return out, err
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Space: SpaceKey(name)})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	cand, count := suspectDocs(cs, SpaceKey(name))
	if len(cand) > MaxSuspect {
		cand = cand[:MaxSuspect]
	}
	if len(cand) == 0 {
		return out, nil
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	gone, err := x.ownersGone(ctx, cand)
	if err != nil {
		return out, err
	}
	allowed := map[string]bool{}
	own := x.Agent.owners()
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		allowed[k] = true
		r := x.Agent.result(ctx, own, c)
		r.Snippet = "" // never the passage
		out.Items = append(out.Items, SuspectItem{Result: r, Passages: count[k]})
	}
	out.Hidden = hiddenOf(cand, allowed, gone)
	return out, nil
}

// SuspectOwners are, by owner email, the documents of space name whose newest version has a
// suspect passage, as KIND:ID@VERSION, for the owners' notice (SB-5101). Read from the index
// only: no one is checked, because the owner opens their own document at its source.
func (a Agent) SuspectOwners(ctx context.Context, name string) (map[string][]string, error) {
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Space: SpaceKey(name)})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	docs, _ := suspectDocs(cs, SpaceKey(name))
	out := map[string][]string{}
	for _, c := range docs {
		if c.Owner == "" {
			continue
		}
		out[c.Owner] = append(out[c.Owner], fmt.Sprintf("%s:%s@%d", c.Kind, c.SourceID, c.Version))
	}
	return out, nil
}

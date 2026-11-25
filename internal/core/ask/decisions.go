package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Decisions that replace decisions (sb-ask(3), Work across departments; sb-citations(7),
// Replaced decisions; SB-4505): a decision names the one it replaces with supersedes: in
// its front matter, as KIND:ID or the older one's title in the same space. Nothing is
// linked by a model: a link is a document's own front matter, which a steward writes when
// they confirm one publishing a decisions draft.

// The limits of replaced decisions.
const (
	MaxSupersedeChain   = 5
	MaxSupersedeSuggest = 3
)

// TrailDecision is one decision of a topic's trail, as its reader sees it.
type TrailDecision struct {
	Cite       string        `json:"cite"`
	Title      string        `json:"title"`
	URL        string        `json:"url,omitempty"`
	Owner      string        `json:"owner,omitempty"`
	At         time.Time     `json:"at"`
	Supersedes *answer.Newer `json:"supersedes,omitempty"`
	ReplacedBy *answer.Newer `json:"replaced_by,omitempty"`
}

// TrailResponse is GET /v1/spaces/NAME/decisions.
type TrailResponse struct {
	Space     string          `json:"space"`
	Topic     string          `json:"topic"`
	Decisions []TrailDecision `json:"decisions"`
	Hidden    int             `json:"hidden"`
}

// supersedeLinks are, for the decisions of one space (their newest first chunks by
// KIND:ID), which older decision each newer one names: older -> newer.
func supersedeLinks(first map[string]ports.Chunk) map[string]ports.Chunk {
	keys := make([]string, 0, len(first))
	byTitle := map[string]string{}
	for k, c := range first {
		keys = append(keys, k)
		if t := strings.ToLower(strings.TrimSpace(c.Title)); t != "" && isDecision(c) {
			byTitle[t] = k
		}
	}
	sort.Strings(keys)
	out := map[string]ports.Chunk{}
	for _, k := range keys {
		c := first[k]
		if !isDecision(c) {
			continue
		}
		v := strings.TrimSpace(frontMatter(c.Text)["supersedes"])
		if v == "" {
			continue
		}
		old := byTitle[strings.ToLower(v)]
		if d, _, _ := strings.Cut(v, "@"); first[d].SourceID != "" {
			old = d
		}
		if old == "" || old == k {
			continue
		}
		if have, ok := out[old]; !ok || c.Seen.After(have.Seen) {
			out[old] = c
		}
	}
	return out
}

// spaceFirsts are the newest first chunks of the documents a space holds, by KIND:ID.
func (a Agent) spaceFirsts(ctx context.Context, space string) (map[string]ports.Chunk, error) {
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Space: space})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	return firsts(latest(chunks)), nil
}

// allowedDocs are the documents of cand s can open now, by KIND:ID, through every check a
// search makes.
func (a Agent) allowedDocs(ctx context.Context, s tenancy.Scope, cand []ports.Chunk) (map[string]ports.Chunk, error) {
	out := map[string]ports.Chunk{}
	if len(cand) == 0 {
		return out, nil
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := a.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, len(cand))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	for _, c := range res.Context {
		out[c.Kind+":"+c.SourceID] = c
	}
	return out, nil
}

// markReplacedBy sets ReplacedBy on each cite of a decision a newer decision of a space
// holding it replaces, following the chain up to MaxSupersedeChain while s can open each
// newer one now (LAW-01). A newer decision s can't open is never named.
func (a Agent) markReplacedBy(ctx context.Context, s tenancy.Scope, claims []answer.Claim, chunks []ports.Chunk) error {
	if _, ok := a.Index.(ports.DocLookup); !ok {
		return nil
	}
	spacesOf := map[string][]string{}
	for _, c := range chunks {
		if isDecision(c) {
			spacesOf[docKey(c.Cite)] = c.Spaces
		}
	}
	cited := map[string]bool{}
	for _, cl := range claims {
		for _, ci := range cl.Cites {
			if _, ok := spacesOf[docKey(ci.Cite)]; ok {
				cited[docKey(ci.Cite)] = true
			}
		}
	}
	if len(cited) == 0 {
		return nil
	}
	links := map[string]ports.Chunk{}
	seen := map[string]bool{}
	for d := range cited {
		for _, sp := range spacesOf[d] {
			if seen[sp] {
				continue
			}
			seen[sp] = true
			first, err := a.spaceFirsts(ctx, sp)
			if err != nil {
				return err
			}
			for old, c := range supersedeLinks(first) {
				if have, ok := links[old]; !ok || c.Seen.After(have.Seen) {
					links[old] = c
				}
			}
		}
	}
	newest, err := a.chainNewest(ctx, s, cited, links)
	if err != nil {
		return err
	}
	for i := range claims {
		for j := range claims[i].Cites {
			if n, ok := newest[docKey(claims[i].Cites[j].Cite)]; ok {
				claims[i].Cites[j].ReplacedBy = &answer.Newer{Cite: n.Cite, Title: n.Title}
			}
		}
	}
	return nil
}

// chainNewest is, for each document of from, the newest decision of its chain of links
// that s can open without passing one they can't.
func (a Agent) chainNewest(ctx context.Context, s tenancy.Scope, from map[string]bool, links map[string]ports.Chunk) (map[string]ports.Chunk, error) {
	var cand []ports.Chunk
	added := map[string]bool{}
	for d := range from {
		cur := d
		for i := 0; i < MaxSupersedeChain; i++ {
			n, ok := links[cur]
			if !ok {
				break
			}
			cur = n.Kind + ":" + n.SourceID
			if added[cur] {
				break
			}
			added[cur] = true
			cand = append(cand, n)
		}
	}
	open, err := a.allowedDocs(ctx, s, cand)
	if err != nil {
		return nil, err
	}
	out := map[string]ports.Chunk{}
	for d := range from {
		cur, visited := d, map[string]bool{d: true}
		for i := 0; i < MaxSupersedeChain; i++ {
			n, ok := links[cur]
			if !ok {
				break
			}
			k := n.Kind + ":" + n.SourceID
			c, can := open[k]
			if !can || visited[k] {
				break
			}
			visited[k] = true
			out[d], cur = c, k
		}
	}
	return out, nil
}

// Trail is a topic's decisions in space name, for a member: those whose title or text holds
// every word of topic (all of them, with none), oldest first, at most MaxDecisions, each
// checked as them, with the rest counted.
func (x Stewards) Trail(ctx context.Context, s tenancy.Scope, name, topic string) (TrailResponse, error) {
	if err := x.member(ctx, s, name); err != nil {
		return TrailResponse{}, err
	}
	topic = strings.Join(strings.Fields(topic), " ")
	if len([]rune(topic)) > MaxQuestion {
		return TrailResponse{}, fmt.Errorf("%w: a topic is at most %d characters", ErrInvalid, MaxQuestion)
	}
	out := TrailResponse{Space: name, Topic: topic, Decisions: []TrailDecision{}}
	first, err := x.Agent.spaceFirsts(ctx, SpaceKey(name))
	if err != nil {
		return out, err
	}
	words := strings.Fields(strings.ToLower(topic))
	var cand []ports.Chunk
	for _, c := range first {
		if !isDecision(c) {
			continue
		}
		hay := strings.ToLower(c.Title + "\n" + c.Text)
		all := true
		for _, w := range words {
			all = all && strings.Contains(hay, w)
		}
		if all {
			cand = append(cand, c)
		}
	}
	links := supersedeLinks(first)
	cand = inOrder(cand, links)
	if len(cand) > MaxDecisions {
		cand = cand[:MaxDecisions]
	}
	open, err := x.Agent.allowedDocs(ctx, s, append([]ports.Chunk(nil), cand...))
	if err != nil {
		return out, err
	}
	older := map[string]string{} // newer -> older
	for o, n := range links {
		older[n.Kind+":"+n.SourceID] = o
	}
	for _, c := range cand {
		k := c.Kind + ":" + c.SourceID
		v, ok := open[k]
		if !ok {
			out.Hidden++
			continue
		}
		r := x.Agent.result(ctx, nil, v)
		d := TrailDecision{Cite: v.Cite, Title: r.Title, URL: r.URL, At: decidedAt(v)}
		if r.Owner != nil {
			d.Owner = r.Owner.Email
		}
		if o, ok := open[older[k]]; ok {
			d.Supersedes = &answer.Newer{Cite: o.Cite, Title: o.Title}
		}
		if n, ok := links[k]; ok {
			if nv, ok := open[n.Kind+":"+n.SourceID]; ok {
				d.ReplacedBy = &answer.Newer{Cite: nv.Cite, Title: nv.Title}
			}
		}
		out.Decisions = append(out.Decisions, d)
	}
	return out, nil
}

// supersedeSuggestions are at most MaxSupersedeSuggest documents, other than the meeting's
// transcript, that claims marked as disagreeing cite (SB-3605's mark), for the steward a
// decisions draft ran as to confirm one (SB-4505).
func supersedeSuggestions(claims []answer.Claim, transcript string) []Result {
	var out []Result
	seen := map[string]bool{transcript: true}
	for _, c := range claims {
		if !c.Disagree {
			continue
		}
		for _, ci := range c.Cites {
			d := docKey(ci.Cite)
			if seen[d] || len(out) == MaxSupersedeSuggest {
				continue
			}
			seen[d] = true
			out = append(out, Result{Cite: d, Title: ci.Title, Flags: []string{}})
		}
	}
	return out
}

// checkSupersedes is a decisions draft's chosen older decision (KIND:ID), checked: only on
// a decisions draft, and only a document s can open now.
func (x Stewards) checkSupersedes(ctx context.Context, s tenancy.Scope, q StewardQuestion, doc string) (string, error) {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return "", nil
	}
	if q.Thread == nil || q.Thread.Kind != DraftDecisions {
		return "", fmt.Errorf("%w: only a decisions draft can replace a decision", ErrInvalid)
	}
	keys, err := documentKeys([]string{doc})
	if err != nil {
		return "", err
	}
	open, err := x.Agent.opensCites(ctx, s, keys)
	if err != nil {
		return "", err
	}
	if !open[keys[0]] {
		return "", fmt.Errorf("%w: the decision it replaces must be one you can open", ErrInvalid)
	}
	return keys[0], nil
}

// decidedAt is a decision's date: its front matter's date, or when the index first saw it.
func decidedAt(c ports.Chunk) time.Time {
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(frontMatter(c.Text)["date"])); err == nil {
		return t
	}
	return c.Seen
}

// inOrder sorts decisions by date, oldest first, then puts each one after the decision it
// replaces, so a trail reads in the order things were decided.
func inOrder(cs []ports.Chunk, links map[string]ports.Chunk) []ports.Chunk {
	sort.Slice(cs, func(i, j int) bool {
		a, b := decidedAt(cs[i]), decidedAt(cs[j])
		return a.Before(b) || a.Equal(b) && cs[i].ID < cs[j].ID
	})
	replaces := map[string]string{} // newer -> older
	for o, n := range links {
		replaces[n.Kind+":"+n.SourceID] = o
	}
	for pass := 0; pass < len(cs); pass++ {
		moved := false
		at := map[string]int{}
		for i, c := range cs {
			at[c.Kind+":"+c.SourceID] = i
		}
		for i, c := range cs {
			if j, ok := at[replaces[c.Kind+":"+c.SourceID]]; ok && j > i {
				cs = append(cs[:i], append(append([]ports.Chunk{}, cs[i+1:j+1]...), append([]ports.Chunk{c}, cs[j+1:]...)...)...)
				moved = true
				break
			}
		}
		if !moved {
			break
		}
	}
	return cs
}

// markLinks marks the cites a retirement replaced (SB-2801) and the decisions a newer one
// replaces (SB-4505), each only as far as s can open.
func (a Agent) markLinks(ctx context.Context, s tenancy.Scope, claims []answer.Claim, chunks []ports.Chunk) error {
	if err := a.markReplaces(ctx, s, claims, chunks); err != nil {
		return err
	}
	return a.markReplacedBy(ctx, s, claims, chunks)
}

// noteFront is a note's extra front matter: a decisions draft's note is a decision, and
// names the decision the steward confirmed it replaces.
func (x Stewards) noteFront(ctx context.Context, s tenancy.Scope, q StewardQuestion, supersedes string) ([]string, error) {
	doc, err := x.checkSupersedes(ctx, s, q, supersedes)
	if err != nil || q.Thread == nil || q.Thread.Kind != DraftDecisions {
		return nil, err
	}
	front := []string{"type: decision"}
	if doc != "" {
		front = append(front, "supersedes: "+doc)
	}
	return front, nil
}

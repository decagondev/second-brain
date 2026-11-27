// Package retrieve finds candidate chunks and lets only live-authorised ones into the
// context (sb-permissions(7), Retrieval and the live check; SB-301 to SB-303).
package retrieve

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

const (
	// Candidates is how many each ranking returns.
	Candidates = 50
	// RRFK is the reciprocal-rank-fusion constant.
	RRFK = 60
	// Budget is how many authorised chunks enter the context.
	Budget = 12
	// ACLDeadline bounds each connector's batch.
	ACLDeadline = 800 * time.Millisecond
	// MaxBatch is the most documents in one ACL call.
	MaxBatch = 200
)

// BroadTokens match prefilter entries meaning "anyone in the workspace" or "anyone".
var BroadTokens = []string{"anyone", "slack:members", "linear:members", "hubspot:users", "github:members", "bamboohr:employees", "quickbooks:readers"}

// Tokens are the asker's prefilter tokens (sb-permissions(7)).
func Tokens(s tenancy.Scope, emails []string) []string {
	out := []string{s.User()}
	for _, e := range emails {
		e = strings.ToLower(e)
		out = append(out, e)
		if at := strings.LastIndex(e, "@"); at >= 0 {
			out = append(out, "domain:"+e[at+1:])
		}
	}
	for _, g := range s.Groups() {
		out = append(out, g, "group:"+g)
	}
	return append(out, BroadTokens...)
}

// Fuse ranks candidates by reciprocal-rank fusion of their vector and keyword ranks; of two
// with the same score, the version the index saw last comes first.
func Fuse(cs []ports.Candidate) []ports.Candidate {
	score := Score
	out := append([]ports.Candidate(nil), cs...)
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := score(out[i]), score(out[j])
		if si != sj {
			return si > sj
		}
		if a, b := out[i].Chunk.Seen, out[j].Chunk.Seen; !a.Equal(b) {
			return a.After(b) // the same score: the newer version first (SB-2706)
		}
		return out[i].Chunk.ID < out[j].Chunk.ID
	})
	return out
}

// Score is a candidate's reciprocal-rank-fusion score (SB-5507 shows it).
func Score(c ports.Candidate) float64 {
	var s float64
	if c.VectorRank > 0 {
		s += 1 / float64(RRFK+c.VectorRank)
	}
	if c.KeywordRank > 0 {
		s += 1 / float64(RRFK+c.KeywordRank)
	}
	return s
}

// Authoriser runs the live check: one batched ACL call per connector per request.
type Authoriser struct {
	Connectors map[string]ports.Connector // by kind
	Policy     policy.Kernel
	// Killed reports a killed connector kind; its documents stay Unknown (sb-kill-switches(7)).
	Killed func(kind string) bool
	// Spaces returns the cell's spaces and barriers now (SB-1006); nil is a cell without
	// spaces. An error fails the request: space checks are never skipped.
	Spaces func(ctx context.Context) ([]ports.Space, [][2]string, error)
	// SpaceKilled reports a killed space by name (SB-1008): every chunk it holds is refused.
	SpaceKilled func(name string) bool
	// Owners resolves a document owner (an email) to their directory groups now, for
	// barriers (SB-1203; barrier.Directory). nil, or an error, refuses every owned chunk.
	Owners func(ctx context.Context, owner string) ([]string, error)
	// Metrics counts live verdicts per connector (sb_acl_verdicts_total), ids only.
	Metrics interface {
		Add(name, help string, labels map[string]string, v float64)
	}
}

// Verdicts is the live answer per document key (kind:id).
type Verdicts map[string]ports.Verdict

func docKey(c ports.Chunk) string { return c.Kind + ":" + c.SourceID }

// Check asks each connector about its documents, in parallel, each within ACLDeadline.
// Every document starts Unknown; only an answered Allow changes that.
func (a Authoriser) Check(ctx context.Context, user ports.UserRef, chunks []ports.Chunk) Verdicts {
	byKind := map[string][]ports.SourceRef{}
	seen := map[string]bool{}
	out := Verdicts{}
	for _, c := range chunks {
		k := docKey(c)
		if seen[k] {
			continue
		}
		seen[k] = true
		out[k] = ports.VerdictUnknown
		byKind[c.Kind] = append(byKind[c.Kind], ports.SourceRef{Kind: c.Kind, ID: c.SourceID})
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for kind, refs := range byKind {
		conn, ok := a.Connectors[kind]
		if !ok || (a.Killed != nil && a.Killed(kind)) {
			continue // no connector, or a killed one: stays Unknown
		}
		for start := 0; start < len(refs); start += MaxBatch {
			end := start + MaxBatch
			if end > len(refs) {
				end = len(refs)
			}
			batch := refs[start:end]
			wg.Add(1)
			go func() {
				defer wg.Done()
				bctx, cancel := context.WithTimeout(ctx, ACLDeadline)
				defer cancel()
				vs, err := conn.ACL(bctx, user, batch)
				if err != nil || len(vs) != len(batch) || bctx.Err() != nil {
					return // error, short answer or late: all Unknown
				}
				mu.Lock()
				for i, r := range batch {
					out[r.Kind+":"+r.ID] = vs[i]
				}
				mu.Unlock()
			}()
		}
	}
	wg.Wait()
	return out
}

// Result is what enters the context, and how many documents were refused.
type Result struct {
	Context   []ports.Chunk
	Refused   int
	Decisions []Decision // one per candidate document, in rank order
}

// Decision is the live outcome for one candidate document (sb-ask(3), traces).
type Decision struct {
	Cite    string `json:"cite"`
	Verdict string `json:"verdict"` // "allow" or "refuse"
	Reason  string `json:"reason"`  // allowed, source_denied, source_unknown, policy_denied
}

// ErrNoScope is a request without a valid scope.
var ErrNoScope = errors.New("retrieve: no valid scope")

// Context runs retrieve, fuse, live check and budget for one request.
func Context(ctx context.Context, s tenancy.Scope, user ports.UserRef, emails []string, r ports.Retriever, a Authoriser, q ports.Query) (Result, error) {
	return ContextUpTo(ctx, s, user, emails, r, a, q, Budget)
}

// ContextUpTo is Context keeping up to budget allowed chunks: search (SB-1701) keeps every
// allowed candidate, since it shows documents rather than filling a prompt. The checks are
// the same ones, in the same order.
func ContextUpTo(ctx context.Context, s tenancy.Scope, user ports.UserRef, emails []string, r ports.Retriever, a Authoriser, q ports.Query, budget int) (Result, error) {
	if err := s.Check(); err != nil {
		return Result{}, ErrNoScope
	}
	if teams := tenancy.BranchFrom(ctx, s); len(teams) > 0 {
		return branchContext(ctx, append([]tenancy.Scope{s}, teams...), user, emails, r, a, q, budget)
	}
	return contextUpTo(ctx, s, user, emails, r, a, q, budget)
}

// branchContext is a branch ask (SB-3903, LAW-24): retrieval and every check run once per
// space of the branch, each as the asker in that space, and the chunks each run allowed
// are joined, taking each run's best in turn. Nothing one space refused is let in by
// another's run unless that run allowed it on its own checks.
func branchContext(ctx context.Context, scopes []tenancy.Scope, user ports.UserRef, emails []string, r ports.Retriever, a Authoriser, q ports.Query, budget int) (Result, error) {
	var out Result
	runs := make([][]ports.Chunk, 0, len(scopes))
	decided := map[string]int{}
	for _, sc := range scopes {
		res, err := contextUpTo(ctx, sc, user, emails, r, a, q, budget)
		if err != nil {
			return Result{}, err
		}
		runs = append(runs, res.Context)
		for _, d := range res.Decisions {
			if i, ok := decided[d.Cite]; !ok {
				decided[d.Cite] = len(out.Decisions)
				out.Decisions = append(out.Decisions, d)
			} else if d.Verdict == "allow" {
				out.Decisions[i] = d
			}
		}
	}
	seen := map[string]bool{}
	for i := 0; len(out.Context) < budget; i++ {
		more := false
		for _, run := range runs {
			if i >= len(run) {
				continue
			}
			more = true
			if c := run[i]; !seen[c.ID] && len(out.Context) < budget {
				seen[c.ID] = true
				out.Context = append(out.Context, c)
			}
		}
		if !more {
			break
		}
	}
	for _, d := range out.Decisions {
		if d.Verdict != "allow" {
			out.Refused++
		}
	}
	return out, nil
}

func contextUpTo(ctx context.Context, s tenancy.Scope, user ports.UserRef, emails []string, r ports.Retriever, a Authoriser, q ports.Query, budget int) (Result, error) {
	if q.Limit == 0 {
		q.Limit = Candidates
	}
	cands, err := r.Search(ctx, s, Tokens(s, emails), q)
	if err != nil {
		return Result{}, err
	}
	cands = Fuse(cands)
	keep := map[string]bool{}
	for _, k := range q.Kinds {
		keep[k] = true
	}
	docs := map[string]bool{}
	for _, d := range q.Docs {
		docs[d] = true
	}
	var chunks []ports.Chunk
	for _, c := range cands {
		if (len(keep) == 0 || keep[c.Chunk.Kind]) && (len(docs) == 0 || docs[docKey(c.Chunk)]) {
			chunks = append(chunks, c.Chunk)
		}
	}
	if len(docs) > 0 {
		if chunks, err = withRest(ctx, s.Tenant(), r, q.Docs, chunks); err != nil {
			return Result{}, err
		}
	}
	res, err := Allowed(ctx, s, user, a, chunks, len(chunks))
	if err != nil {
		return Result{}, err
	}
	res.Context = OneCopy(res.Context)
	if len(res.Context) > budget {
		res.Context = res.Context[:budget]
	}
	return res, nil
}

// withRest adds, after the ranked chunks of the chosen documents, the rest of their chunks
// in document order (SB-3004), so a chosen document is found however the question is put.
// They still take every check.
func withRest(ctx context.Context, tenant string, r ports.Retriever, docs []string, ranked []ports.Chunk) ([]ports.Chunk, error) {
	lk, ok := r.(ports.DocLookup)
	if !ok {
		return ranked, nil
	}
	var q ports.DocQuery
	for _, d := range docs {
		kind, id, ok := strings.Cut(d, ":")
		if ok {
			q.Refs = append(q.Refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	all, err := lk.Docs(ctx, tenant, q)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		if docKey(all[i]) != docKey(all[j]) {
			return docKey(all[i]) < docKey(all[j])
		}
		return all[i].Start < all[j].Start
	})
	seen := map[string]bool{}
	for _, c := range ranked {
		seen[c.ID] = true
	}
	for _, c := range all {
		if !seen[c.ID] {
			ranked = append(ranked, c)
		}
	}
	return ranked, nil
}

// OneCopy keeps, of documents with the same text (ports.Chunk.TextSHA256), only the newest
// one's chunks, in order (sb-ask(3), Copies; SB-2202). Run it on allowed chunks only, so the
// copy kept is the newest the person can open. Newest is the version the index saw last,
// then the higher version, then the greater cite, so the choice never depends on rank.
func OneCopy(chunks []ports.Chunk) []ports.Chunk {
	keep := map[string]ports.Chunk{} // text hash -> a chunk of the newest copy
	for _, c := range chunks {
		if c.TextSHA256 == "" {
			continue
		}
		if k, ok := keep[c.TextSHA256]; !ok || newer(c, k) {
			keep[c.TextSHA256] = c
		}
	}
	out := chunks[:0:0]
	for _, c := range chunks {
		if k, ok := keep[c.TextSHA256]; ok && docKey(k) != docKey(c) {
			continue // a copy of a newer document
		}
		out = append(out, c)
	}
	return out
}

// PerDocument keeps at most n chunks of each document, the first ones, in order
// (sb-ask(3), One document, three chunks; SB-2902). Run it after OneCopy.
func PerDocument(chunks []ports.Chunk, n int) []ports.Chunk {
	seen := map[string]int{}
	out := chunks[:0:0]
	for _, c := range chunks {
		k := docKey(c)
		if seen[k] < n {
			seen[k]++
			out = append(out, c)
		}
	}
	return out
}

func newer(a, b ports.Chunk) bool {
	if !a.Seen.Equal(b.Seen) {
		return a.Seen.After(b.Seen)
	}
	if a.Version != b.Version {
		return a.Version > b.Version
	}
	return docKey(a) > docKey(b)
}

// Allowed runs the space, label and barrier checks and the live ACL batch over chunks, in
// order, keeping up to budget allowed ones. Retrieval ends with it; so does every other
// path that shows a chunk to someone, such as a steward's cite shown to an asker (SB-1704).
func Allowed(ctx context.Context, s tenancy.Scope, user ports.UserRef, a Authoriser, chunks []ports.Chunk, budget int) (Result, error) {
	if err := s.Check(); err != nil {
		return Result{}, ErrNoScope
	}
	var res Result
	refused := map[string]bool{}
	decided := map[string]int{}
	decide := func(c ports.Chunk, reason string) {
		k := docKey(c)
		if i, ok := decided[k]; !ok {
			decided[k] = len(res.Decisions)
			res.Decisions = append(res.Decisions, Decision{Cite: c.Cite, Verdict: verdictOf(reason), Reason: reason})
		} else if reason != "allowed" && res.Decisions[i].Reason == "allowed" {
			res.Decisions[i] = Decision{Cite: c.Cite, Verdict: "refuse", Reason: reason}
		}
		if reason != "allowed" {
			refused[k] = true
		}
	}
	// Spaces, labels and barriers run first, so the live ACL batch asks only about chunks
	// they allow; the ACL still decides every one of those (sb-permissions(7); SB-1006).
	if a.Spaces != nil {
		all, barriers, err := a.Spaces(ctx)
		if err != nil {
			return Result{}, fmt.Errorf("retrieve: spaces: %w", err)
		}
		if len(all) > 0 {
			rctx := spaces.ReadContext(all, barriers, s.Groups(), s.Space())
			owners := ownerCache{}
			kept := chunks[:0:0]
			for _, c := range chunks {
				if reason := a.spaceStep(ctx, s, c, rctx, owners); reason != "" {
					decide(c, reason)
					continue
				}
				kept = append(kept, c)
			}
			chunks = kept
		}
	}
	verdicts := a.Check(ctx, user, chunks)
	if a.Metrics != nil {
		for k, v := range verdicts {
			kind := strings.SplitN(k, ":", 2)[0]
			a.Metrics.Add("sb_acl_verdicts_total", "Live ACL verdicts by tenant, connector and verdict.",
				map[string]string{"tenant": s.Tenant(), "connector": kind, "verdict": strings.ToLower(v.String())}, 1)
		}
	}
	for _, c := range chunks {
		k := docKey(c)
		if refused[k] {
			continue // another chunk of this document was refused by a space check
		}
		reason := "allowed"
		switch {
		case verdicts[k] == ports.VerdictDeny:
			reason = "source_denied"
		case verdicts[k] != ports.VerdictAllow:
			reason = "source_unknown"
		case a.Policy.Authorise(ctx, s, "read", policy.Entity{Type: "Chunk", ID: c.ID, Tenant: c.Tenant, Attrs: map[string]any{"cite": c.Cite}}, nil).Decision != policy.Allow:
			reason = "policy_denied"
		}
		decide(c, reason)
		if reason != "allowed" {
			continue
		}
		if len(res.Context) < budget {
			res.Context = append(res.Context, c)
		}
	}
	res.Refused = len(refused)
	return res, nil
}

type ownerCache map[string]ownerGroups

type ownerGroups struct {
	groups []string
	ok     bool
}

// spaceStep runs steps 1 to 3 of sb-permissions(7) for one chunk through the kernel: "" if
// it passes, else the reason it was refused.
func (a Authoriser) spaceStep(ctx context.Context, s tenancy.Scope, c ports.Chunk, rctx map[string]any, cache ownerCache) string {
	if a.heldByKilled(c) {
		return "space_killed"
	}
	ent := policy.Entity{Type: "Chunk", ID: c.ID, Tenant: c.Tenant, Attrs: map[string]any{"cite": c.Cite}}
	for k, v := range spaces.ChunkAttrs(c.Label, c.Spaces) {
		ent.Attrs[k] = v
	}
	if c.Owner != "" {
		og, seen := cache[c.Owner]
		if !seen {
			if a.Owners != nil {
				if gs, err := a.Owners(ctx, c.Owner); err == nil {
					og = ownerGroups{spaces.GroupSet(gs), true}
				}
			}
			cache[c.Owner] = og
		}
		if og.ok {
			ent.Attrs["owners"] = og.groups
		} else {
			delete(ent.Attrs, "owners") // unknown owner: the first space rule refuses (LAW-03)
		}
	}
	if a.Policy.Authorise(ctx, s, "read", ent, rctx).Decision != policy.Allow {
		return spaceReason(c, ent.Attrs, rctx)
	}
	return ""
}

// ItemAllowed decides one item used outside retrieval, such as the ticket or thread a draft
// replies to (sb-permissions(7), Barriers beyond reads; SB-1203): in a cell with spaces it
// is routed and labelled as ingest would, then takes steps 1 to 3, and the live ACL always.
func (a Authoriser) ItemAllowed(ctx context.Context, s tenancy.Scope, user ports.UserRef, item ports.Chunk, doc ports.Document) (bool, error) {
	if a.Spaces != nil {
		all, barriers, err := a.Spaces(ctx)
		if err != nil {
			return false, fmt.Errorf("retrieve: spaces: %w", err)
		}
		if len(all) > 0 {
			item.Spaces, item.Label = route.Route(all, item.Kind, doc.Location, doc.Label)
			item.Owner = strings.ToLower(strings.TrimSpace(doc.Owner))
			if a.spaceStep(ctx, s, item, spaces.ReadContext(all, barriers, s.Groups(), s.Space()), ownerCache{}) != "" {
				return false, nil
			}
		}
	}
	return a.Check(ctx, user, []ports.Chunk{item})[docKey(item)] == ports.VerdictAllow, nil
}

// heldByKilled is a chunk any of whose holding spaces is killed: a kill never leaves part
// of a space readable, even through another space that holds the same document.
func (a Authoriser) heldByKilled(c ports.Chunk) bool {
	if a.SpaceKilled == nil {
		return false
	}
	for _, h := range c.Spaces {
		if a.SpaceKilled(strings.TrimPrefix(h, "spc_")) {
			return true
		}
	}
	return false
}

// spaceReason names which space check refused a chunk, for the trace (sb-ask(3)): its
// decision always comes from the policy kernel; this only explains it.
func spaceReason(c ports.Chunk, attrs, rctx map[string]any) string {
	barred, _ := rctx["barred_spaces"].([]string)
	barredGroups, _ := rctx["barred_groups"].([]string)
	in := func(xs []string, x string) bool {
		for _, y := range xs {
			if y == x {
				return true
			}
		}
		return false
	}
	for _, h := range c.Spaces {
		if in(barred, h) {
			return "barrier_denied"
		}
	}
	owners, known := attrs["owners"].([]string)
	if !known {
		return "owner_unknown"
	}
	for _, g := range owners {
		if in(barredGroups, g) {
			return "barrier_denied"
		}
	}
	if sp, _ := rctx["space"].(string); sp != spaces.Company && !in(c.Spaces, sp) {
		return "space_denied"
	}
	return "label_denied"
}

func verdictOf(reason string) string {
	if reason == "allowed" {
		return "allow"
	}
	return "refuse"
}

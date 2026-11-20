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
var BroadTokens = []string{"anyone", "slack:members", "linear:members", "hubspot:users"}

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

// Fuse ranks candidates by reciprocal-rank fusion of their vector and keyword ranks.
func Fuse(cs []ports.Candidate) []ports.Candidate {
	score := func(c ports.Candidate) float64 {
		var s float64
		if c.VectorRank > 0 {
			s += 1 / float64(RRFK+c.VectorRank)
		}
		if c.KeywordRank > 0 {
			s += 1 / float64(RRFK+c.KeywordRank)
		}
		return s
	}
	out := append([]ports.Candidate(nil), cs...)
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := score(out[i]), score(out[j])
		if si != sj {
			return si > sj
		}
		return out[i].Chunk.ID < out[j].Chunk.ID
	})
	return out
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
	if err := s.Check(); err != nil {
		return Result{}, ErrNoScope
	}
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
	var chunks []ports.Chunk
	for _, c := range cands {
		if len(keep) == 0 || keep[c.Chunk.Kind] {
			chunks = append(chunks, c.Chunk)
		}
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
			kept := chunks[:0:0]
			for _, c := range chunks {
				ent := policy.Entity{Type: "Chunk", ID: c.ID, Tenant: c.Tenant, Attrs: map[string]any{"cite": c.Cite}}
				for k, v := range spaces.ChunkAttrs(c.Label, c.Spaces) {
					ent.Attrs[k] = v
				}
				if a.Policy.Authorise(ctx, s, "read", ent, rctx).Decision != policy.Allow {
					decide(c, spaceReason(c, rctx))
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
		if len(res.Context) < Budget {
			res.Context = append(res.Context, c)
		}
	}
	res.Refused = len(refused)
	return res, nil
}

// spaceReason names which space check refused a chunk, for the trace (sb-ask(3)): its
// decision always comes from the policy kernel; this only explains it.
func spaceReason(c ports.Chunk, rctx map[string]any) string {
	barred, _ := rctx["barred_spaces"].([]string)
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

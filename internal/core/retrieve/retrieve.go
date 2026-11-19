// Package retrieve finds candidate chunks and lets only live-authorised ones into the
// context (sb-permissions(7), Retrieval and the live check; SB-301 to SB-303).
package retrieve

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
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
var BroadTokens = []string{"anyone", "slack:members", "linear:members"}

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
		if !ok {
			continue // no connector: stays Unknown
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
	verdicts := a.Check(ctx, user, chunks)
	var res Result
	refused := map[string]bool{}
	decided := map[string]int{}
	for _, c := range chunks {
		k := docKey(c)
		reason := "allowed"
		switch {
		case verdicts[k] == ports.VerdictDeny:
			reason = "source_denied"
		case verdicts[k] != ports.VerdictAllow:
			reason = "source_unknown"
		case a.Policy.Authorise(ctx, s, "read", policy.Entity{Type: "Chunk", ID: c.ID, Tenant: c.Tenant, Attrs: map[string]any{"cite": c.Cite}}, nil).Decision != policy.Allow:
			reason = "policy_denied"
		}
		if i, ok := decided[k]; !ok {
			decided[k] = len(res.Decisions)
			res.Decisions = append(res.Decisions, Decision{Cite: c.Cite, Verdict: verdictOf(reason), Reason: reason})
		} else if reason != "allowed" && res.Decisions[i].Reason == "allowed" {
			res.Decisions[i] = Decision{Cite: c.Cite, Verdict: "refuse", Reason: reason}
		}
		if reason != "allowed" {
			refused[k] = true
			continue
		}
		if len(res.Context) < Budget {
			res.Context = append(res.Context, c)
		}
	}
	res.Refused = len(refused)
	return res, nil
}

func verdictOf(reason string) string {
	if reason == "allowed" {
		return "allow"
	}
	return "refuse"
}

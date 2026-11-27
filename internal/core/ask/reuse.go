package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Answered before, checked now (sb-ask(3), Answered before; SB-5603): the same question in
// the same space, scope and language within ReuseFor reuses the earlier answer when every
// document it read is unchanged and the asker can open all of them now (LAW-26). It costs no
// model call, says so, and offers Ask fresh.

// ReuseFor is how long an answer can be reused.
const ReuseFor = 7 * 24 * time.Hour

// ReuseEntry is one answer that can be reused: its key and its trace.
type ReuseEntry struct {
	Tenant string    `json:"tenant"`
	Key    string    `json:"key"`
	Trace  string    `json:"trace"`
	At     time.Time `json:"at"`
}

// ReuseStore keeps one entry per key, the newest.
type ReuseStore interface {
	PutReuse(ctx context.Context, e ReuseEntry) error
	Reuse(ctx context.Context, tenant, key string) (ReuseEntry, bool, error)
}

// normalise is a question as reuse compares it: lower case, words only, one space apart.
func normalise(q string) string {
	f := strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r <= 127
	})
	return strings.Join(f, " ")
}

// reuseKey is the hash of what makes two asks the same: the normalised question, the space,
// the source kinds, the language, the agent, the length and verified.
func reuseKey(q, space string, req Request) string {
	kinds := append([]string(nil), req.Sources...)
	sort.Strings(kinds)
	agent := req.Agent
	if agent == "" {
		agent = "ask"
	}
	h := sha256.Sum256([]byte(strings.Join([]string{normalise(q), space, strings.Join(kinds, ","), req.Language, agent, req.Length, fmt.Sprint(req.Verified)}, "\x00")))
	return hex.EncodeToString(h[:])
}

// mayReuse says an ask is one reuse may answer or keep: a first question, with nothing that
// makes it the asker's own (memory, exclusions, kept or dropped documents, a file, a
// collection or chosen documents) and nothing extra (a consult, a skill, a choice).
func (a Agent) mayReuse(ctx context.Context, s tenancy.Scope, req Request, turn int, used []MemoryLine, skill string) bool {
	if a.Reuse == nil || req.Fresh || turn > 0 || len(used) > 0 || req.File != nil || req.Collection != "" || len(req.Documents) > 0 ||
		len(req.Keep) > 0 || len(req.Drop) > 0 || req.Consult || req.Choice != "" || skill != "" || isConsulted(ctx) || shadowPrompt(ctx) != "" {
		return false
	}
	ex, err := a.excluded(ctx, s)
	return err == nil && len(ex) == 0
}

// reuseKeyIf is the ask's reuse key, or "" when it may not reuse or be reused.
func (a Agent) reuseKeyIf(ctx context.Context, s tenancy.Scope, req Request, q string, turn int, used []MemoryLine, skill string, never bool) string {
	if never || !a.mayReuse(ctx, s, req, turn, used, skill) {
		return ""
	}
	return reuseKey(q, s.Space(), req)
}

// readSet is chunks' documents as KIND:ID@VERSION, sorted, each once.
func readSet(chunks []ports.Chunk) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range chunks {
		k := fmt.Sprintf("%s:%s@%d", c.Kind, c.SourceID, c.Version)
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// reusedFrom is the earlier answer for key when the documents this ask's live retrieval
// allowed are exactly the ones it was composed from, the same versions: then nothing it
// read has changed and the asker can open all of it now (LAW-26). Its cites are dressed
// again for now (owners, links, review marks).
func (a Agent) reusedFrom(ctx context.Context, key string, chunks []ports.Chunk) ([]answer.Claim, string, bool) {
	if key == "" || len(chunks) == 0 {
		return nil, "", false
	}
	e, ok, err := a.Reuse.Reuse(ctx, a.Tenant, key)
	if err != nil || !ok || a.Now().Sub(e.At) > ReuseFor {
		return nil, "", false
	}
	t, found, err := a.Store.GetTrace(ctx, a.Tenant, e.Trace)
	if err != nil || !found || t.Response.Notice != nil || t.Response.Table != nil || len(t.Response.Answer.Claims) == 0 ||
		strings.Join(t.Read, ",") != strings.Join(readSet(chunks), ",") {
		return nil, "", false
	}
	byCite := map[string]ports.Chunk{}
	for _, c := range chunks {
		byCite[c.Cite] = c
	}
	claims := make([]answer.Claim, len(t.Response.Answer.Claims))
	for i, c := range t.Response.Answer.Claims {
		c.Cites = append([]answer.Cite(nil), c.Cites...)
		for _, ci := range c.Cites {
			if _, ok := byCite[ci.Cite]; !ok {
				return nil, "", false
			}
		}
		claims[i] = c
	}
	a.decorate(ctx, claims, byCite)
	return claims, t.At.UTC().Format("2006-01-02"), true
}

// keepForReuse makes a fresh answer reusable: one whose claims are all cited, with no notice.
func (a Agent) keepForReuse(ctx context.Context, key string, resp Response) {
	if key == "" || resp.Reused != "" || resp.Notice != nil || resp.Table != nil || len(resp.Answer.Claims) == 0 {
		return
	}
	for _, c := range resp.Answer.Claims {
		if c.Uncited || len(c.Cites) == 0 {
			return
		}
	}
	_ = a.Reuse.PutReuse(ctx, ReuseEntry{Tenant: a.Tenant, Key: key, Trace: resp.Trace, At: a.Now().UTC()}) // a reuse missed costs only a model call
}

// decorate dresses each cite from its chunk as of now: link, owner and freshness.
func (a Agent) decorate(ctx context.Context, claims []answer.Claim, byCite map[string]ports.Chunk) {
	own := a.owners()
	marks := &marksCache{store: a.Marks, by: map[string]map[string]ReviewMark{}}
	for i := range claims {
		for j := range claims[i].Cites {
			ch := byCite[claims[i].Cites[j].Cite]
			if a.Link != nil {
				claims[i].Cites[j].URL = a.Link(ch)
			}
			claims[i].Cites[j].Owner = own.of(ctx, ch) // only allowed chunks are ever cited
			f := a.freshness(ctx, marks, ch)
			claims[i].Cites[j].AgeDays, claims[i].Cites[j].Reviewed, claims[i].Cites[j].Stale = f.age, f.reviewed, f.stale
		}
	}
}

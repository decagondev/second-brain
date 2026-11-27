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
)

// Cache hits you can see (sb-admin(3), Cache hits; SB-5706): the share of input tokens the
// provider read from its prompt cache, per space and per agent, for one month, counted from
// traces. A space or agent whose share stays under LowCacheShare while it had repeated asks
// is named, with the parts of its stable prefix (the prompt, language, length or skill
// line) that changed between its asks. Each trace keeps only a short hash of each part, so
// the report never holds a question, a passage, a memory line or any prompt text.

// LowCacheShare is the cached share under which a space or agent is named, and
// MinCacheAsks the asks it needs for the share to count.
const (
	LowCacheShare = 0.1
	MinCacheAsks  = 2
)

// The parts of the stable prefix.
const (
	PrefixPrompt   = "prompt"
	PrefixLanguage = "language"
	PrefixLength   = "length"
	PrefixSkill    = "skill"
)

// CacheRow is one space's or agent's month.
type CacheRow struct {
	Name   string  `json:"name"`
	Asks   int     `json:"asks"`
	Input  int     `json:"input_tokens"`
	Cached int     `json:"cached_tokens"`
	Share  float64 `json:"cached_share"`
	// Low says the share stayed under LowCacheShare with repeated asks, and Changed which
	// prefix parts changed between them, most often first.
	Low     bool     `json:"low,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

// CacheHits is GET /v1/admin/costs/cache.
type CacheHits struct {
	Month  string     `json:"month"`
	Total  CacheRow   `json:"total"`
	Spaces []CacheRow `json:"spaces"`
	Agents []CacheRow `json:"agents"`
}

// prefixHash is a part's short hash: enough to tell two asks' parts apart, never the text.
func prefixHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:4])
}

// prefixRec collects one ask's prefix part hashes as it composes.
type prefixRec struct {
	mu    sync.Mutex
	parts map[string]string
}

type prefixKey struct{}

func withPrefix(ctx context.Context) (context.Context, *prefixRec) {
	r := &prefixRec{}
	return context.WithValue(ctx, prefixKey{}, r), r
}

// notePrefix records the parts of a system prompt as composed (the first composition wins).
func notePrefix(ctx context.Context, parts map[string]string) {
	r, _ := ctx.Value(prefixKey{}).(*prefixRec)
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.parts != nil {
		return
	}
	r.parts = map[string]string{}
	for k, v := range parts {
		r.parts[k] = prefixHash(v)
	}
}

func (r *prefixRec) get() map[string]string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.parts
}

type cacheAcc struct {
	row   CacheRow
	last  map[string]string
	moved map[string]int
}

func (c *cacheAcc) add(t Trace) {
	c.row.Asks++
	c.row.Input += t.InputTokens
	c.row.Cached += t.CachedTokens
	if t.Prefix == nil {
		return
	}
	if c.last != nil {
		for k, v := range t.Prefix {
			if c.last[k] != v {
				c.moved[k]++
			}
		}
	}
	c.last = t.Prefix
}

func (c *cacheAcc) done() CacheRow {
	r := c.row
	if r.Input > 0 {
		r.Share = float64(r.Cached) / float64(r.Input)
	}
	if r.Asks >= MinCacheAsks && r.Input > 0 && r.Share < LowCacheShare {
		r.Low = true
		for k := range c.moved {
			r.Changed = append(r.Changed, k)
		}
		sort.Slice(r.Changed, func(i, j int) bool {
			a, b := r.Changed[i], r.Changed[j]
			if c.moved[a] != c.moved[b] {
				return c.moved[a] > c.moved[b]
			}
			return a < b
		})
	}
	return r
}

// BuildCacheHits counts the month's traces (YYYY-MM, [from, to)) that called a model.
func BuildCacheHits(traces []Trace, month string, from, to time.Time) CacheHits {
	out := CacheHits{Month: month, Spaces: []CacheRow{}, Agents: []CacheRow{}}
	sorted := append([]Trace(nil), traces...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At.Before(sorted[j].At) })
	spaces, agents := map[string]*cacheAcc{}, map[string]*cacheAcc{}
	total := &cacheAcc{moved: map[string]int{}}
	acc := func(m map[string]*cacheAcc, k string) *cacheAcc {
		if m[k] == nil {
			m[k] = &cacheAcc{row: CacheRow{Name: k}, moved: map[string]int{}}
		}
		return m[k]
	}
	for _, t := range sorted {
		if t.InputTokens == 0 || t.At.Before(from) || !t.At.Before(to) {
			continue
		}
		space := spaceName(t.Space)
		if space == "" {
			space = "company"
		}
		acc(spaces, space).add(t)
		if ag := traceAgent(t); ag != "" {
			acc(agents, ag).add(t)
		}
		total.add(t)
	}
	out.Total = total.done()
	out.Total.Name, out.Total.Low, out.Total.Changed = "", false, nil
	for _, m := range []struct {
		in  map[string]*cacheAcc
		out *[]CacheRow
	}{{spaces, &out.Spaces}, {agents, &out.Agents}} {
		for _, a := range m.in {
			*m.out = append(*m.out, a.done())
		}
		sort.Slice(*m.out, func(i, j int) bool { return (*m.out)[i].Name < (*m.out)[j].Name })
	}
	return out
}

// Lines are the admin's week's lines for the low ones, or none.
func (c CacheHits) Lines() []string {
	var out []string
	for _, g := range []struct {
		kind string
		rows []CacheRow
	}{{"space", c.Spaces}, {"agent", c.Agents}} {
		for _, r := range g.rows {
			if !r.Low {
				continue
			}
			l := fmt.Sprintf("The %s %s read %d%% of its input from the prompt cache over %d asks", r.Name, g.kind, int(r.Share*100), r.Asks)
			if len(r.Changed) > 0 {
				l += "; its " + strings.Join(r.Changed, " and ") + " changed between asks"
			}
			out = append(out, l)
		}
	}
	return out
}

package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// How far each agent reaches (sb-agents(7), Reach; SB-5406), and the sensitive map
// (sb-spaces(7), Sensitive map; SB-5407): counts from a replay as each person an agent runs
// as, and from the index. Counts only, never a title.

// ReachGrowth is the week-on-week growth that alerts an agent's owner: more than a tenth.
const ReachGrowth = 0.10

// MaxReachDocs is the most documents of one space a reach replay checks.
const MaxReachDocs = 5000

// SpaceReach is what an agent can read in one space: through how many people, and how many
// of the documents are labelled confidential or above, or held.
type SpaceReach struct {
	Space     string `json:"space"`
	People    int    `json:"people"`
	Documents int    `json:"documents"`
	Labelled  int    `json:"labelled"`
	Held      int    `json:"held"`
}

// AgentReach is one agent's reach this week, and the change since last week.
type AgentReach struct {
	Agent     string       `json:"agent"`
	Week      string       `json:"week"`
	Spaces    []SpaceReach `json:"spaces"`
	Documents int          `json:"documents"`
	Labelled  int          `json:"labelled"`
	Held      int          `json:"held"`
	LastWeek  *int         `json:"last_week,omitempty"`
	Grew      bool         `json:"grew,omitempty"`
}

// Grown says reach went from before to now by more than ReachGrowth.
func Grown(before, now int) bool {
	return now > before && float64(now-before) > ReachGrowth*float64(before)
}

// spaceDocs are the first chunks of the newest version of every document space key holds.
func (a Agent) spaceDocs(ctx context.Context, key string) ([]ports.Chunk, []ports.Chunk, error) {
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, nil, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Space: key})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var mine []ports.Chunk
	for _, c := range cs {
		if contains(c.Spaces, key) {
			mine = append(mine, c)
		}
	}
	var docs []ports.Chunk
	for _, c := range firsts(latest(mine)) {
		docs = append(docs, c)
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Cite < docs[j].Cite })
	if len(docs) > MaxReachDocs {
		docs = docs[:MaxReachDocs]
	}
	return docs, latest(mine), nil
}

// ReachIn is the documents of space key that the scopes can open now, each document once,
// with how many are labelled confidential or above and how many are held.
func (a Agent) ReachIn(ctx context.Context, key string, scopes []tenancy.Scope) (SpaceReach, error) {
	out := SpaceReach{Space: strings.TrimPrefix(key, "spc_"), People: len(scopes)}
	docs, all, err := a.spaceDocs(ctx, key)
	if err != nil || len(docs) == 0 {
		return out, err
	}
	open := map[string]ports.Chunk{}
	for _, s := range scopes {
		who, err := a.who(ctx, s)
		if err != nil {
			return out, err
		}
		res, err := retrieve.Allowed(ctx, s, who, a.Auth, docs, len(docs))
		if err != nil {
			return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		for _, c := range res.Context {
			open[c.Kind+":"+c.SourceID] = c
		}
	}
	var wide map[string]WideDoc
	if a.Wide != nil {
		wide = a.Wide.wideDocs(ctx, all)
	}
	for k, c := range open {
		out.Documents++
		if ingest.PersonOnly(c.Label) {
			out.Labelled++
		}
		if d, ok := wide[k]; ok {
			if _, cf := a.Wide.confirmed(ctx, a.Tenant, d); !cf {
				out.Held++
			}
		}
	}
	return out, nil
}

// Total adds a reach's spaces up.
func (r *AgentReach) Total() {
	r.Documents, r.Labelled, r.Held = 0, 0, 0
	for _, s := range r.Spaces {
		r.Documents += s.Documents
		r.Labelled += s.Labelled
		r.Held += s.Held
	}
}

// SensitiveCounts are one space's documents by kind (SB-5407): counts only.
type SensitiveCounts struct {
	Space    string         `json:"space"`
	Week     string         `json:"week"`
	Personal int            `json:"personal_data"`
	Secrets  int            `json:"secret_removed"`
	Labels   map[string]int `json:"labels"`
	Wide     int            `json:"wide"`
	Held     int            `json:"held"`
	// LastWeek is the same counts a week before, when the cell kept them.
	LastWeek *SensitiveCounts `json:"last_week,omitempty"`
}

// SensitiveIn counts space key's documents by kind, from the index: no one is checked,
// because nothing but numbers is shown.
func (a Agent) SensitiveIn(ctx context.Context, key, week string) (SensitiveCounts, error) {
	out := SensitiveCounts{Space: strings.TrimPrefix(key, "spc_"), Week: week, Labels: map[string]int{}}
	_, all, err := a.spaceDocs(ctx, key)
	if err != nil {
		return out, err
	}
	byDoc := map[string][]ports.Chunk{}
	for _, c := range all {
		byDoc[c.Kind+":"+c.SourceID] = append(byDoc[c.Kind+":"+c.SourceID], c)
	}
	for _, cs := range byDoc {
		personal, secret, label := false, false, ""
		for _, c := range cs {
			label = ingest.Highest(label, c.Label)
			for _, f := range c.Flags {
				personal = personal || f == ingest.FlagPersonal
				secret = secret || f == ingest.FlagSecretRemoved
			}
		}
		if personal {
			out.Personal++
		}
		if secret {
			out.Secrets++
		}
		if label != "" {
			out.Labels[label]++
		}
	}
	if a.Wide != nil {
		for _, d := range a.Wide.wideDocs(ctx, all) {
			out.Wide++
			if _, ok := a.Wide.confirmed(ctx, a.Tenant, d); !ok && a.Wide.mode(key, d.Chunk.Spaces) == WideHold {
				out.Held++
			}
		}
	}
	return out, nil
}

// Sensitive is the review list's sensitive section: the space's counts, for its stewards.
func (x Stewards) Sensitive(ctx context.Context, s tenancy.Scope, name string, last func(space, week string) (SensitiveCounts, bool)) (SensitiveCounts, error) {
	if err := x.reviewer(ctx, s, name); err != nil {
		return SensitiveCounts{}, err
	}
	week := WeekStart(x.Now()).Format("2006-01-02")
	out, err := x.Agent.SensitiveIn(ctx, SpaceKey(name), week)
	if err != nil {
		return out, err
	}
	if last != nil {
		if lw, ok := last(name, WeekStart(x.Now()).AddDate(0, 0, -7).Format("2006-01-02")); ok {
			out.LastWeek = &lw
		}
	}
	return out, nil
}

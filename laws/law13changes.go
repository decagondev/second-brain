package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// W16 Keep it current (SB-1708, SB-1709) under the laws.
//
// LAW-13: first-seen times are derived and a rebuild marks, never invents, changes: after
// a rebuild (the drill's, or one from a lost index) "what changed" lists exactly the
// documents that changed, never a week of everything. LAW-01: the stewards' review list
// names only documents the steward can open now, and counts the rest.

// changeSystem's mutation forgets first-seen times across a rebuild (a self-test).
type changeSystem struct{ forgetSeen bool }

func law13Changes(r *rand.Rand, cs changeSystem) error {
	ctx := context.Background()
	t0 := time.Unix(1_800_000_000, 0).UTC()
	now := t0
	src := mock.NewSource("docs")
	n := 2 + r.Intn(5)
	for i := 0; i < n; i++ {
		src.PutAt(fmt.Sprintf("d%d", i), "Doc", fmt.Sprintf("text %d", i), "/Ops/d", "", "ann@acme.example")
	}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_law", Connector: mock.New(src, false), Embedder: hashing.Embedder{},
		Index: idx, Now: func() time.Time { return now }}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	now = t0.Add(30 * 24 * time.Hour) // a month later
	changed := map[string]bool{}
	for i := 0; i < n; i++ {
		if r.Intn(3) == 0 {
			id := fmt.Sprintf("d%d", i)
			src.PutAt(id, "Doc", fmt.Sprintf("text %d changed", i), "/Ops/d", "", "ann@acme.example")
			changed["docs:"+id] = true
		}
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	now = now.Add(time.Hour)
	switch r.Intn(2) {
	case 0: // the drill's rebuild
		if cs.forgetSeen {
			if err := idx.Drop(ctx, "acme-logistics"); err != nil {
				return err
			}
			if _, err := p.Sync(ctx, ""); err != nil {
				return err
			}
		} else if err := p.Rebuild(ctx); err != nil {
			return err
		}
	case 1: // a lost index: nothing to carry, so everything is marked rebuilt
		if err := idx.Drop(ctx, "acme-logistics"); err != nil {
			return err
		}
		if err := p.Rebuild(ctx); err != nil {
			return err
		}
		changed = map[string]bool{} // their times are lost; they must not be listed either
	}
	a := ask.Agent{Tenant: "acme-logistics", Release: "rel_law", Index: idx.Bind("acme-logistics"), Store: tracemem.New(),
		Auth:   retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": mock.New(src, false)}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}},
		Emails: func(context.Context, string) ([]string, error) { return []string{"ann@acme.example"}, nil },
		Killed: func(context.Context) bool { return false }, NewID: func(p string) string { return p + "1" }, Now: func() time.Time { return now }}
	s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_ann", nil)
	res, err := a.Changes(ctx, s, 7)
	if err != nil {
		return err
	}
	got := map[string]bool{}
	for _, it := range res.Items {
		got[docOf(it.Cite)] = true
	}
	for d := range got {
		if !changed[d] {
			return fmt.Errorf("%s is listed as changed this week, but only a rebuild touched it (LAW-13)", d)
		}
	}
	for d := range changed {
		if !got[d] {
			return fmt.Errorf("%s changed this week and isn't listed", d)
		}
	}
	return nil
}

func law01Review(r *rand.Rand) error {
	w := Gen(r)
	ctx := context.Background()
	now := time.Unix(1_800_000_000, 0).UTC()
	for _, u := range w.Users {
		if len(u.Groups) == 0 {
			continue
		}
		x, err := questionSystem{}.stewards(ctx, w, u.Tenant, u.Groups[0])
		if err != nil {
			return err
		}
		ix0 := x.Agent.Index.(ports.Index)
		all, _ := ix0.All(ctx, u.Tenant)
		for _, c := range all {
			c.Spaces, c.Seen, c.Cite = []string{"spc_ops"}, now.Add(-400*24*time.Hour), "docs:"+c.Cite // a real cite, kind:id@version
			if err := ix0.Replace(ctx, u.Tenant, ports.SourceRef{Kind: c.Kind, ID: c.SourceID}, c.Version, []ports.Chunk{c}); err != nil {
				return err
			}
		}
		all, _ = ix0.All(ctx, u.Tenant)
		if err != nil {
			return err
		}
		x.Now = func() time.Time { return now }
		// Answers in the last month cited every document of the tenant five times.
		st := tracemem.New()
		for _, c := range all {
			for i := 0; i < ask.ReviewCites; i++ {
				_ = st.PutTrace(ctx, ask.Trace{ID: fmt.Sprintf("tr_%s_%d", c.ID, i), Tenant: u.Tenant, User: "usr_x", At: now.Add(-time.Hour),
					Response: ask.Response{Answer: ask.Answer{Claims: []answer.Claim{{Text: "x", Cites: []answer.Cite{{Cite: c.Cite}}}}}}})
			}
		}
		x.Facts = func(string) ([]string, string, bool) { return []string{"group:" + u.Groups[0]}, "", true }
		scope, _ := scopeOf(u)
		res, err := x.Review(ctx, scope, "ops", st, 0)
		if err != nil {
			return err
		}
		for _, it := range res.Items {
			if d := strings.TrimPrefix(docOf(it.Cite), "docs:"); !w.truthAllow(u.ID, d) {
				return fmt.Errorf("%s's review list names %s, which the source doesn't allow them (LAW-01)", u.ID, d)
			}
		}
		if len(res.Items)+len(res.Hidden) != len(all) {
			return fmt.Errorf("%d shown and %d hidden of %d stale documents", len(res.Items), len(res.Hidden), len(all))
		}
	}
	return nil
}

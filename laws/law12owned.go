package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Documents keep an owner, and copies are cited once (SB-2201, SB-2202), under the laws.
//
// LAW-12: someone deprovisioned is gone at once: their documents are on the stewards'
// review list as "owner gone" from the next time it is read (so within the next sync), and
// they are never shown as someone to ask, on the list or on any result. LAW-01: the list
// names only what the steward can open now and counts the rest. LAW-13: copies are found
// by a hash of the text that a rebuild gives again, so an answer cites the newest copy the
// asker can open, before and after a rebuild, and the next newest once that one is deleted.
// The real ingest pipeline, index, agent and stewards' list, over a mock source.

// ownedSystem's mutations are self-tests: a directory check that never finds anyone gone,
// and an index that loses the text hash (an ingest that never kept one).
type ownedSystem struct{ neverGone, noHash bool }

const ownedTenant = "acme-logistics"

// placed puts every indexed chunk in the ops space, as routing would (sb-ingest(7)).
func placed(ctx context.Context, idx *memindex.Index) error {
	all, err := idx.All(ctx, ownedTenant)
	if err != nil {
		return err
	}
	by := map[ports.SourceRef][]ports.Chunk{}
	ver := map[ports.SourceRef]int{}
	for _, c := range all {
		c.Spaces = []string{"spc_ops"}
		ref := ports.SourceRef{Kind: c.Kind, ID: c.SourceID}
		by[ref] = append(by[ref], c)
		ver[ref] = c.Version
	}
	for ref, cs := range by {
		if err := idx.Replace(ctx, ownedTenant, ref, ver[ref], cs); err != nil {
			return err
		}
	}
	return nil
}

func (o ownedSystem) agent(idx *memindex.Index, src *mock.Source, active map[string]bool, now func() time.Time) ask.Agent {
	ids := 0
	var ix ports.Retriever = idx.Bind(ownedTenant)
	if o.noHash {
		ix = hashless{ix.(boundIndex)}
	}
	return ask.Agent{Tenant: ownedTenant, Release: "rel_law", Index: ix, Store: tracemem.New(),
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": mock.New(src, false)}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}},
		Emails: func(_ context.Context, u string) ([]string, error) {
			return []string{strings.TrimPrefix(u, "usr_") + "@acme.example"}, nil
		},
		Owner: ask.ActiveOwner(func(_ context.Context, email string) ([]ports.User, error) {
			return []ports.User{{ID: "usr_" + strings.TrimSuffix(email, "@acme.example"), Active: active[email]}}, nil
		}),
		Killed: func(context.Context) bool { return false }, NewID: func(p string) string { ids++; return fmt.Sprintf("%s%d", p, ids) }, Now: now}
}

// boundIndex is what memory.Index.Bind returns: a retriever that also looks documents up.
type boundIndex interface {
	ports.Retriever
	ports.DocLookup
}

// hashless is an index whose chunks carry no text hash.
type hashless struct{ boundIndex }

func (h hashless) Search(ctx context.Context, s tenancy.Scope, tokens []string, q ports.Query) ([]ports.Candidate, error) {
	cs, err := h.boundIndex.Search(ctx, s, tokens, q)
	for i := range cs {
		cs[i].Chunk.TextSHA256 = ""
	}
	return cs, err
}

func (h hashless) Docs(ctx context.Context, tenant string, q ports.DocQuery) ([]ports.Chunk, error) {
	cs, err := h.boundIndex.Docs(ctx, tenant, q)
	for i := range cs {
		cs[i].TextSHA256 = ""
	}
	return cs, err
}

func law12Orphans(r *rand.Rand, o ownedSystem) error {
	ctx := context.Background()
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	src := mock.NewSource("docs")
	owners := []string{"ann@acme.example", "bob@acme.example", "cat@acme.example", "dan@acme.example"}
	active := map[string]bool{}
	for _, e := range owners {
		active[e] = true
	}
	n := 3 + r.Intn(6)
	owner := map[string]string{}
	opens := map[string]bool{} // what the steward, sam, can open
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("d%d", i)
		viewers := []string{"ann@acme.example"}
		if r.Intn(3) > 0 {
			viewers = append(viewers, "sam@acme.example")
			opens["docs:"+id] = true
		}
		src.PutAt(id, fmt.Sprintf("Doc %d", i), fmt.Sprintf("freight text %d", i), "/Ops/d", "", viewers...)
		if r.Intn(5) > 0 {
			owner["docs:"+id] = owners[r.Intn(len(owners))]
			src.Own(id, owner["docs:"+id])
		}
	}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: ownedTenant, Release: "rel_law", Connector: mock.New(src, false), Embedder: hashing.Embedder{},
		Index: idx, Now: func() time.Time { return now }}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	if err := placed(ctx, idx); err != nil {
		return err
	}
	// Some owners leave or are deprovisioned; then perhaps a sync runs.
	for _, e := range owners {
		if r.Intn(2) == 0 {
			active[e] = false
		}
	}
	if r.Intn(2) == 0 {
		if _, err := p.Sync(ctx, ""); err != nil {
			return err
		}
	}
	a := o.agent(idx, src, active, func() time.Time { return now })
	x := ask.Stewards{Agent: a, Store: tracemem.New(), Audit: &keepRows{}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}},
		Facts: func(string) ([]string, string, bool) { return []string{"group:leads"}, "", true },
		NewID: func(p string) string { return p + "1" }, Now: func() time.Time { return now },
		Gone: func(_ context.Context, email string) (bool, error) { return !o.neverGone && !active[email], nil }}
	scope, _ := tenancy.NewIssuer(ownedTenant).Issue(ownedTenant, "usr_sam", []string{"leads"})
	res, err := x.Review(ctx, scope, "ops", tracemem.New(), 0)
	if err != nil {
		return err
	}
	shown := map[string]bool{}
	for _, it := range res.Items {
		d := docOf(it.Cite)
		shown[d] = true
		if !opens[d] {
			return fmt.Errorf("the review list names %s, which the steward can't open (LAW-01)", d)
		}
		if gone := owner[d] != "" && !active[owner[d]]; !gone || !it.OwnerGone {
			return fmt.Errorf("%s is on the review list with owner_gone %v; its owner %q is active: %v", d, it.OwnerGone, owner[d], active[owner[d]])
		}
	}
	hidden := 0
	for d, e := range owner {
		if active[e] {
			continue
		}
		if !opens[d] {
			hidden++
		} else if !shown[d] {
			return fmt.Errorf("%s's owner was deprovisioned, and it isn't on the review list (LAW-12)", d)
		}
	}
	if len(res.Hidden) != hidden {
		return fmt.Errorf("the review list hides %d documents; the steward can't open %d whose owner has gone", len(res.Hidden), hidden)
	}
	// Nobody gone is ever someone to ask: not on the list, not on a search result.
	sr, err := a.Search(ctx, scope, ask.SearchRequest{Query: "freight text"})
	if err != nil {
		return err
	}
	raw, _ := json.Marshal([]any{res, sr})
	for _, e := range owners {
		if !active[e] && strings.Contains(string(raw), e) {
			return fmt.Errorf("LAW-12: %s was deprovisioned and is still shown as someone to ask", e)
		}
	}
	return nil
}

func law13Copies(r *rand.Rand, o ownedSystem) error {
	ctx := context.Background()
	t0 := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	now := t0
	src := mock.NewSource("docs")
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: ownedTenant, Release: "rel_law", Connector: mock.New(src, false), Embedder: hashing.Embedder{},
		Index: idx, Now: func() time.Time { return now }}
	texts := 1 + r.Intn(3)
	type copyOf struct {
		id    string
		text  int
		round int
		opens bool
	}
	var copies []copyOf
	for round := 0; round < 3; round++ {
		now = t0.Add(time.Duration(round) * 24 * time.Hour)
		for t := 0; t < texts; t++ {
			if round > 0 && r.Intn(2) == 0 {
				continue
			}
			c := copyOf{id: fmt.Sprintf("t%d-r%d", t, round), text: t, round: round, opens: r.Intn(4) > 0}
			viewers := []string{"sam@acme.example"}
			if c.opens {
				viewers = append(viewers, "ann@acme.example")
			}
			// A copy in another place, under another name: the same text.
			src.PutAt(c.id, fmt.Sprintf("Copy %d of policy %d", round, t), fmt.Sprintf("freight policy number %d\n\nevery pallet is weighed", t), fmt.Sprintf("/Ops/%d", round), "", viewers...)
			copies = append(copies, c)
		}
		if _, err := p.Sync(ctx, ""); err != nil {
			return err
		}
	}
	a := o.agent(idx, src, map[string]bool{}, func() time.Time { return now })
	scope, _ := tenancy.NewIssuer(ownedTenant).Issue(ownedTenant, "usr_ann", nil)
	deleted := map[string]bool{}
	check := func(when string) error {
		resp, err := a.Search(ctx, scope, ask.SearchRequest{Query: "freight policy pallet weighed"})
		if err != nil {
			return err
		}
		want := map[string]bool{}
		for t := 0; t < texts; t++ {
			best := -1
			id := ""
			for _, c := range copies {
				if c.text == t && c.opens && !deleted[c.id] && c.round > best {
					best, id = c.round, c.id
				}
			}
			if id != "" {
				want["docs:"+id] = true
			}
		}
		got := map[string]bool{}
		for _, res := range resp.Results {
			got[docOf(res.Cite)] = true
		}
		for d := range got {
			if !want[d] {
				return fmt.Errorf("%s: search cites %s; of its copies the asker can open, only the newest is cited: %v (LAW-13)", when, d, keys(want))
			}
		}
		for d := range want {
			if !got[d] {
				return fmt.Errorf("%s: search doesn't cite %s, the newest copy the asker can open", when, d)
			}
		}
		return nil
	}
	if r.Intn(2) == 0 {
		now = now.Add(time.Hour)
		if err := p.Rebuild(ctx); err != nil { // the drill's rebuild: copies are found again
			return err
		}
	}
	if err := check("before a delete"); err != nil {
		return err
	}
	// The stewards' list names each text with copies once, as its newest copy, with the
	// others it can open (sam opens every copy).
	if err := placed(ctx, idx); err != nil {
		return err
	}
	x := ask.Stewards{Agent: a, Store: tracemem.New(), Audit: &keepRows{}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}},
		Facts: func(string) ([]string, string, bool) { return []string{"group:leads"}, "", true },
		NewID: func(p string) string { return p + "1" }, Now: func() time.Time { return now },
		Gone: func(context.Context, string) (bool, error) { return false, nil }}
	sam, _ := tenancy.NewIssuer(ownedTenant).Issue(ownedTenant, "usr_sam", []string{"leads"})
	rev, err := x.Review(ctx, sam, "ops", tracemem.New(), 0)
	if err != nil {
		return err
	}
	listed := 0
	for t := 0; t < texts; t++ {
		var ids []string
		for _, c := range copies {
			if c.text == t {
				ids = append(ids, "docs:"+c.id)
			}
		}
		if len(ids) < 2 {
			continue
		}
		listed++
		found := false
		for _, it := range rev.Items {
			if docOf(it.Cite) == ids[len(ids)-1] {
				found = true
				if len(it.Copies) != len(ids)-1 {
					return fmt.Errorf("the review list shows %s with %d copies; it has %d", ids[len(ids)-1], len(it.Copies), len(ids)-1)
				}
			}
		}
		if !found {
			return fmt.Errorf("the review list doesn't show %s, the newest of its copies %v", ids[len(ids)-1], ids)
		}
	}
	if len(rev.Items) != listed || len(rev.Hidden) != 0 {
		return fmt.Errorf("the review list has %d items and %d hidden; %d texts have copies", len(rev.Items), len(rev.Hidden), listed)
	}
	// Delete the newest copy of one text the asker can open: the next newest is cited.
	newest := ""
	for _, c := range copies {
		if c.opens && c.text == 0 && (newest == "" || c.round > roundOf(newest)) {
			newest = c.id
		}
	}
	if newest == "" {
		return nil
	}
	src.Delete(newest)
	deleted[newest] = true
	now = now.Add(time.Hour)
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	return check("after deleting " + newest)
}

func roundOf(id string) int {
	var t, r int
	_, _ = fmt.Sscanf(id, "t%d-r%d", &t, &r)
	return r
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

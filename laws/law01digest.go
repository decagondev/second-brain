package laws

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// The weekly digest and watched questions (SB-2103, SB-2104) under the laws.
//
// LAW-01: the digest names only documents the person can open when it is sent: one whose
// ACL dropped them after it changed isn't named. LAW-13: a rebuild is never a change, so
// after one the digest lists exactly what changed. LAW-07: watching keeps no answer or
// result, only a hash, and the message names no source. The real notify job, ask agent,
// ingest pipeline and index, over a mock source.

// DigestSystem is the digest's view of changes: titles of what changed for user, now.
type DigestSystem interface {
	Changes(a ask.Agent, scope func(user string) tenancy.Scope) func(ctx context.Context, user string, days int) ([]string, int, error)
}

// RealDigest is the cell's: GET /v1/changes as the person, now.
type RealDigest struct{}

// Changes implements DigestSystem.
func (RealDigest) Changes(a ask.Agent, scope func(string) tenancy.Scope) func(context.Context, string, int) ([]string, int, error) {
	return func(ctx context.Context, user string, days int) ([]string, int, error) {
		resp, err := a.Changes(ctx, scope(user), days)
		if err != nil {
			return nil, 0, err
		}
		var out []string
		for _, r := range resp.Items {
			out = append(out, r.Title)
		}
		return out, 0, nil
	}
}

func law01Digest(r *rand.Rand, sys DigestSystem) error {
	ctx := context.Background()
	const tenant = "acme-logistics"
	t0 := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	now := t0
	src := mock.NewSource("docs")
	n := 3 + r.Intn(6)
	title := func(i int) string { return fmt.Sprintf("Doc %d", i) }
	for i := 0; i < n; i++ {
		src.PutAt(fmt.Sprintf("d%d", i), title(i), fmt.Sprintf("text %d", i), "/Ops/d", "", "ann@acme.example", "bob@acme.example")
	}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: tenant, Release: "rel_law", Connector: mock.New(src, false), Embedder: hashing.Embedder{},
		Index: idx, Now: func() time.Time { return now }}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	// During the week some documents change, and then some of them drop ann.
	now = time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC) // a Friday
	want := map[string]bool{}
	for i := 0; i < n; i++ {
		if r.Intn(2) == 0 {
			continue
		}
		id := fmt.Sprintf("d%d", i)
		src.PutAt(id, title(i), fmt.Sprintf("text %d changed", i), "/Ops/d", "", "ann@acme.example", "bob@acme.example")
		want[title(i)] = true
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		if r.Intn(3) == 0 {
			src.SetViewers(fmt.Sprintf("d%d", i), "bob@acme.example")
			delete(want, title(i))
		}
	}
	if r.Intn(2) == 0 {
		if err := p.Rebuild(ctx); err != nil { // the drill's rebuild, after the changes
			return err
		}
	}
	monday := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	now = monday
	store := tracemem.New()
	a := ask.Agent{Tenant: tenant, Release: "rel_law", Index: idx.Bind(tenant), Store: store,
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": mock.New(src, false)}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}},
		Emails: func(_ context.Context, u string) ([]string, error) {
			return []string{strings.TrimPrefix(u, "usr_") + "@acme.example"}, nil
		},
		Killed: func(context.Context) bool { return false }, NewID: func(p string) string { return p + "1" }, Now: func() time.Time { return now }}
	iss := tenancy.NewIssuer(tenant)
	scope := func(u string) tenancy.Scope { s, _ := iss.Issue(tenant, u, nil); return s }
	dir := lawDir{"usr_ann": true}
	got := &recorder{}
	clock := memClock{"last": monday.Add(-5 * time.Minute), "weekly": notify.Monday(monday, nil).AddDate(0, 0, -7)}
	// ann watches a saved question; its sources change when a document she can open does.
	sq := ask.SavedQuestion{ID: "sq_1", Tenant: tenant, User: "usr_ann", Question: "text changed", Space: "company", At: t0}
	sources := func(ctx context.Context, user, _, q string) (string, error) {
		resp, err := a.Search(ctx, scope(user), ask.SearchRequest{Query: q})
		if err != nil {
			return "", err
		}
		var cites []string
		for _, x := range resp.Results {
			cites = append(cites, x.Cite)
		}
		return ask.SourcesHash(cites), nil
	}
	sq.Watch = true
	sq.Sources = "before"
	if err := store.PutSaved(ctx, sq); err != nil {
		return err
	}
	job := notify.Job{Tenant: tenant, Base: "https://brain.acme-logistics.example", Questions: store, Dir: dir, Prefs: store, Send: got,
		Clock: clock, Now: func() time.Time { return now }, Zone: time.UTC, Digest: true,
		Askers:  func(context.Context, time.Time, time.Time) ([]string, error) { return []string{"usr_ann"}, nil },
		Changes: sys.Changes(a, scope), Watches: store, Sources: sources,
		Facts: func(string) ([]string, string, bool) { return nil, "", false }}
	if _, err := job.Run(ctx); err != nil {
		return err
	}
	var named []string
	watched := 0
	for _, m := range got.msgs {
		switch m.Kind {
		case notify.KindDigest:
			_, list, _ := strings.Cut(m.Text, ": ")
			named = append(named, strings.Split(list, "; ")...)
		case notify.KindWatch:
			watched++
			for i := 0; i < n; i++ {
				if strings.Contains(m.Text, title(i)) || strings.Contains(m.Text, "text") {
					return fmt.Errorf("LAW-07: the watch message names a source or the question: %q", m.Text)
				}
			}
		}
	}
	if watched != 1 {
		return fmt.Errorf("the watched question's sources changed and %d watch messages were sent", watched)
	}
	var wanted []string
	for t := range want {
		wanted = append(wanted, t)
	}
	sort.Strings(named)
	sort.Strings(wanted)
	if strings.Join(named, ",") != strings.Join(wanted, ",") {
		return fmt.Errorf("the digest names %v; ann can open these changed documents now: %v (LAW-01, LAW-13)", named, wanted)
	}
	kept, _ := store.SavedOf(ctx, tenant, "usr_ann")
	for _, k := range kept {
		if k.Sources != "" && len(k.Sources) != 64 {
			return fmt.Errorf("LAW-07: a watched question keeps more than a hash: %q", k.Sources)
		}
	}
	return nil
}

// lawDir is a directory of active people with email NAME@acme.example.
type lawDir map[string]bool

func (d lawDir) Members(context.Context, string, string) ([]string, error) { return nil, nil }

func (d lawDir) User(_ context.Context, _, id string) (ports.User, bool, error) {
	return ports.User{ID: id, Active: d[id], Emails: []string{strings.TrimPrefix(id, "usr_") + "@acme.example"}}, true, nil
}

type recorder struct{ msgs []notify.Message }

func (r *recorder) Send(_ context.Context, _ notify.Person, m notify.Message) error {
	r.msgs = append(r.msgs, m)
	return nil
}

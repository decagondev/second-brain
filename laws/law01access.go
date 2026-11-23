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

// Asking for access (SB-2205) under the laws.
//
// LAW-01: the asker learns nothing new: the response is counts, and the requests go only to
// the owners of documents the source refused the asker, each about their own document, never
// one a space or barrier check refused. LAW-07: an owner hears the document and who asked,
// never the question. One request per asker, document and week. The real agent, index and
// traces store over a mock source.

// AccessSystem is where requests are remembered.
type AccessSystem interface {
	Store(real ask.AccessStore) ask.AccessStore
}

// RealAccess is the cell's traces store.
type RealAccess struct{}

// Store implements AccessSystem.
func (RealAccess) Store(real ask.AccessStore) ask.AccessStore { return real }

type sentNote struct {
	Owner, Asker string
	Doc          ask.Result
}

func law01Access(r *rand.Rand, sys AccessSystem) error {
	ctx := context.Background()
	const tenant = "acme-logistics"
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	src := mock.NewSource("docs")
	owners := []string{"bob@acme.example", "cat@acme.example", "dan@acme.example"}
	active := map[string]bool{"bob@acme.example": true, "cat@acme.example": r.Intn(2) == 0, "dan@acme.example": true}
	n := 3 + r.Intn(6)
	opens, owner := map[string]bool{}, map[string]string{}
	canary := fmt.Sprintf("QCANARY%x", r.Int63())
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("d%d", i)
		o := owners[r.Intn(len(owners))]
		src.PutAt(id, fmt.Sprintf("Doc %d", i), "freight rules "+canary, "/Ops/d", "", o, "ann@acme.example")
		opens["docs:"+id] = r.Intn(2) == 0
		if r.Intn(5) > 0 {
			src.Own(id, o)
			owner["docs:"+id] = o
		}
	}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: tenant, Release: "rel_law", Connector: mock.New(src, false), Embedder: hashing.Embedder{},
		Index: idx, Now: func() time.Time { return now }}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	// Then the source stops letting ann open some: the index's prefilter still has her, so
	// the live check is what refuses them.
	for i := 0; i < n; i++ {
		if id := fmt.Sprintf("d%d", i); !opens["docs:"+id] {
			o := owner["docs:"+id]
			if o == "" {
				o = "eve@acme.example"
			}
			src.SetViewers(id, o)
		}
	}
	store := tracemem.New()
	ids := 0
	a := ask.Agent{Tenant: tenant, Release: "rel_law", Index: idx.Bind(tenant), Store: store,
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": mock.New(src, false)}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}},
		Emails: func(_ context.Context, u string) ([]string, error) {
			return []string{strings.TrimPrefix(u, "usr_") + "@acme.example"}, nil
		},
		Owner: ask.ActiveOwner(func(_ context.Context, email string) ([]ports.User, error) {
			return []ports.User{{ID: "usr_x", Active: active[email]}}, nil
		}),
		Killed: func(context.Context) bool { return false }, Now: func() time.Time { return now },
		NewID: func(p string) string { ids++; return fmt.Sprintf("%s%d", p, ids) }}
	s, _ := tenancy.NewIssuer(tenant).Issue(tenant, "usr_ann", nil)
	res, err := a.Search(ctx, s, ask.SearchRequest{Query: "freight rules " + canary})
	if err != nil {
		return err
	}
	// A barrier refused one document too: it is never asked about.
	t, _, _ := store.GetTrace(ctx, tenant, res.Trace)
	barred := ""
	for d := range owner {
		if !opens[d] {
			barred = d
			break
		}
	}
	if barred != "" && r.Intn(2) == 0 {
		for i, dc := range t.Decisions {
			if docOf(dc.Cite) == barred {
				t.Decisions[i].Reason = "barrier"
			}
		}
		t.ID = res.Trace + "b"
		if err := store.PutTrace(ctx, t); err != nil {
			return err
		}
	} else {
		barred = ""
	}
	var notes []sentNote
	note := func(_ context.Context, o, asker string, doc ask.Result) error {
		notes = append(notes, sentNote{o, asker, doc})
		return nil
	}
	st := sys.Store(store)
	got, err := a.RequestAccess(ctx, s, t.ID, st, note)
	if err != nil {
		return err
	}
	want := ask.AccessResponse{}
	for d := range opensNot(opens, n) {
		switch {
		case d == barred:
		case owner[d] == "" || !active[owner[d]]:
			want.NoOwner++
		default:
			want.Sent++
		}
	}
	if got != want {
		return fmt.Errorf("access requests %+v, want %+v (LAW-01)", got, want)
	}
	for _, nt := range notes {
		d := docOf(nt.Doc.Cite)
		if opens[d] || d == barred || owner[d] != nt.Owner || nt.Asker != "ann@acme.example" {
			return fmt.Errorf("LAW-01: %s was asked about %s (owner %q, ann opens it: %v, barred: %v)", nt.Owner, d, owner[d], opens[d], d == barred)
		}
	}
	raw, _ := json.Marshal([]any{got, notes})
	if strings.Contains(raw2(got), "docs:") || strings.Contains(string(raw), canary) {
		return fmt.Errorf("LAW-07: the asker's view names a document, or a request carries the question: %s", raw)
	}
	again, err := a.RequestAccess(ctx, s, t.ID, st, note)
	if err != nil {
		return err
	}
	if again.Sent != 0 || again.Already != want.Sent || len(notes) != want.Sent {
		return fmt.Errorf("asking again the same week sent %d more (%+v)", len(notes)-want.Sent, again)
	}
	return nil
}

// opensNot is the documents ann can't open, which the source refused her.
func opensNot(opens map[string]bool, n int) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < n; i++ {
		if d := fmt.Sprintf("docs:d%d", i); !opens[d] {
			out[d] = true
		}
	}
	return out
}

func raw2(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

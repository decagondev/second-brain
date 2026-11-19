package laws

import (
	"context"
	"fmt"
	"math/rand"
	"sort"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// LAW-01 in the ingest gap (SB-209): when the source revokes access or deletes a document,
// the live check refuses at once, even though the prefilter is stale; and after the next
// sync the prefilter matches the source again and deleted documents are gone.

// PrefilterWorld is a source, its connector and the pipeline's index.
type PrefilterWorld interface {
	Share(id string, viewers ...string)
	Delete(id string)
	Sync() error
	Live(user, id string) (string, error)    // the connector's live verdict now
	Prefilter() (map[string][]string, error) // indexed document id -> prefilter (sorted)
	Source() map[string][]string             // the truth: live document id -> viewers (sorted)
}

// NewPrefilterWorld builds the system under test: the real pipeline (SB-206, SB-209)
// over the mock source.
var NewPrefilterWorld = func() PrefilterWorld {
	src := mock.NewSource("docs")
	idx := memindex.New()
	c := mock.New(src, false)
	return &prefilterWorld{src: src, c: c, idx: idx, truth: map[string][]string{},
		p: ingest.Pipeline{Tenant: "acme-logistics", Connector: c, Embedder: hashing.Embedder{}, Index: idx}}
}

type prefilterWorld struct {
	src    *mock.Source
	c      *mock.Connector
	idx    *memindex.Index
	p      ingest.Pipeline
	cursor ports.Cursor
	truth  map[string][]string
}

func (w *prefilterWorld) Share(id string, viewers ...string) {
	v := append([]string{}, viewers...)
	sort.Strings(v)
	if _, live := w.truth[id]; live {
		w.src.SetViewers(id, viewers...)
	} else {
		w.src.Put(id, "Title "+id, "text of "+id, v...)
		if len(v) == 0 {
			w.src.SetViewers(id) // a document nobody may open
		}
	}
	w.truth[id] = v
}

func (w *prefilterWorld) Delete(id string) {
	w.src.Delete(id)
	delete(w.truth, id)
}

func (w *prefilterWorld) Sync() error {
	next, err := w.p.Sync(context.Background(), w.cursor)
	w.cursor = next
	return err
}

func (w *prefilterWorld) Live(user, id string) (string, error) {
	vs, err := w.c.ACL(context.Background(), ports.UserRef{Email: user}, []ports.SourceRef{{Kind: "docs", ID: id}})
	if err != nil || len(vs) != 1 {
		return "unknown", err
	}
	return vs[0].String(), nil
}

func (w *prefilterWorld) Prefilter() (map[string][]string, error) {
	cs, err := w.idx.All(context.Background(), "acme-logistics")
	out := map[string][]string{}
	for _, c := range cs {
		out[c.SourceID] = append([]string{}, c.ACLPrefilter...)
	}
	return out, err
}

func (w *prefilterWorld) Source() map[string][]string {
	out := map[string][]string{}
	for k, v := range w.truth {
		out[k] = append([]string{}, v...)
	}
	return out
}

func law01Prefilter(r *rand.Rand, w PrefilterWorld) error {
	users := []string{"ann@acme.example", "bob@acme.example", "cy@acme.example"}
	pick := func() []string {
		var out []string
		for _, u := range users {
			if r.Intn(2) == 0 {
				out = append(out, u)
			}
		}
		return out
	}
	n := 2 + r.Intn(4)
	for i := 0; i < n; i++ {
		w.Share(fmt.Sprintf("d%d", i), pick()...)
	}
	if err := w.Sync(); err != nil {
		return err
	}
	for round := 0; round < 3; round++ {
		id := fmt.Sprintf("d%d", r.Intn(n))
		if r.Intn(3) == 0 {
			w.Delete(id)
		} else {
			w.Share(id, pick()...)
		}
		// In the gap: the live check follows the source, whatever the prefilter says.
		truth := w.Source()
		for _, u := range users {
			v, err := w.Live(u, id)
			if err != nil {
				return err
			}
			viewers, exists := truth[id]
			if (v == "allow") != (exists && contains(viewers, u)) {
				return fmt.Errorf("in the gap, %s on %s is %s but the source says viewers %v (exists %v)", u, id, v, viewers, exists)
			}
		}
		if err := w.Sync(); err != nil {
			return err
		}
		got, err := w.Prefilter()
		if err != nil {
			return err
		}
		if fmt.Sprint(got) != fmt.Sprint(truth) {
			return fmt.Errorf("after sync the prefilter is %v but the source is %v", got, truth)
		}
	}
	return nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

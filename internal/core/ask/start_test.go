package ask_test

import (
	"context"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

func (ix index) Docs(_ context.Context, _ string, q ports.DocQuery) ([]ports.Chunk, error) {
	var out []ports.Chunk
	for _, c := range ix {
		hit := false
		for _, r := range q.Refs {
			hit = hit || (r.Kind == c.Kind && r.ID == c.SourceID)
		}
		for _, u := range q.Under {
			hit = hit || (u.Kind == c.Kind && strings.HasPrefix(c.Location+"/", u.ID))
		}
		for _, o := range q.Owners {
			hit = hit || c.Owner == o
		}
		for _, sp := range c.Spaces {
			hit = hit || (q.Space != "" && sp == q.Space)
		}
		if hit {
			out = append(out, c)
		}
	}
	return out, nil
}

// SB-1706: documents in the list's order, a folder's by location, refused ones counted.
func TestStartHere(t *testing.T) {
	a, _ := setup(t, &model{})
	ix := a.Index.(index)
	ix[0].Location, ix[1].Location = "/Support/Start/refunds", "/Support/Start/salaries"
	ix = append(ix, ports.Chunk{ID: "chk_faq", Tenant: tenant, Kind: "docs", SourceID: "faq", Version: 1, Cite: "docs:faq@1", Text: "faq", Location: "/Support/faq"},
		ports.Chunk{ID: "chk_faq0", Tenant: tenant, Kind: "docs", SourceID: "faq", Version: 0, Cite: "docs:faq@0", Text: "old faq", Location: "/Support/faq"})
	a.Index = ix
	a.Auth.Connectors["docs"].(source)["faq|alice"] = ports.VerdictAllow
	res, err := a.StartHere(context.Background(), scope(t, "alice"), []string{"docs:faq", "docs:/Support/Start/", "docs:missing"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range res.Items {
		got = append(got, r.Cite)
	}
	if strings.Join(got, " ") != "docs:faq@1 docs:refunds@2" || res.Hidden != 2 {
		t.Fatalf("items %v hidden %d (salaries refused, missing not indexed)", got, res.Hidden)
	}
}

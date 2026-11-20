package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
)

// LAW-07 and LAW-20 for structured sources (SB-1106): over random records, declarations
// and allow-lists, a field's value is indexed only if the field is declared and either
// not sensitive or named by the allow-list of a space that holds the record; a record
// with an allowed sensitive field is restricted; dropped fields are reported by name,
// never value; and a source that can't declare its fields feeds nothing.

// Declarer declares a source's fields (the connector's own declaration in the real system).
type Declarer func(src *mock.Source, truth []ports.Field)

func declareTruth(src *mock.Source, truth []ports.Field) { src.Declare(truth...) }

var hrFields = []ports.Field{{Name: "name"}, {Name: "department"}, {Name: "manager"},
	{Name: "salary", Sensitive: true}, {Name: "review", Sensitive: true}, {Name: "national_id", Sensitive: true}}

func law07Fields(r *rand.Rand, declare Declarer) error {
	ctx := context.Background()
	src := mock.NewSource("crm")
	idx := memindex.New()
	var reports []string
	allowed := []string{}
	for _, f := range hrFields {
		if f.Sensitive && r.Intn(3) == 0 {
			allowed = append(allowed, f.Name)
		}
	}
	all := []ports.Space{{ID: spaces.ID("people"), Name: "people", Label: "restricted",
		Scope: map[string][]string{"crm": {"/People/**"}}, Fields: map[string][]string{"crm": allowed}}}
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test", Connector: mock.New(src, false),
		Embedder: hashing.Embedder{}, Index: idx,
		Spaces:        func(context.Context) ([]ports.Space, error) { return all, nil },
		FieldsDropped: func(_ string, names []string) { reports = append(reports, strings.Join(names, ",")) }}
	declares := r.Intn(5) != 0
	if declares {
		declare(src, hrFields)
	}
	value := func(rec int, field string) string { return fmt.Sprintf("VAL-%d-%s-%x", rec, field, r.Int63()) }
	type rec struct {
		id     string
		fields map[string]string
		held   bool
	}
	var recs []rec
	for i := 0; i < 2+r.Intn(5); i++ {
		x := rec{id: fmt.Sprintf("emp%d", i), fields: map[string]string{}, held: r.Intn(4) != 0}
		for _, f := range hrFields {
			if r.Intn(4) != 0 {
				x.fields[f.Name] = value(i, f.Name)
			}
		}
		x.fields["private_notes"] = value(i, "private_notes") // never declared
		src.PutFields(x.id, "Employee "+x.id, x.fields, "ann@acme.example")
		loc := "/Other/" + x.id
		if x.held {
			loc = "/People/" + x.id
		}
		src.Place(x.id, loc, "")
		recs = append(recs, x)
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	chunks, err := idx.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	text, label := map[string]string{}, map[string]string{}
	for _, c := range chunks {
		text[c.SourceID] += c.Text
		label[c.SourceID] = c.Label
	}
	sensitive := map[string]bool{}
	for _, f := range hrFields {
		sensitive[f.Name] = f.Sensitive
	}
	for _, x := range recs {
		if !declares {
			if text[x.id] != "" {
				return fmt.Errorf("%s was indexed from a source that never declared its fields", x.id)
			}
			continue
		}
		anySensitive := false
		for name, v := range x.fields {
			declared := name != "private_notes"
			ok := declared && (!sensitive[name] || (x.held && contains(allowed, name)))
			got := strings.Contains(text[x.id], v)
			switch {
			case got && !ok:
				return fmt.Errorf("%s's %s was indexed (sensitive %v, allowed %v, held %v)", x.id, name, sensitive[name], allowed, x.held)
			case !got && ok:
				return fmt.Errorf("%s's allowed field %s wasn't indexed", x.id, name)
			}
			anySensitive = anySensitive || (ok && sensitive[name])
		}
		if anySensitive && label[x.id] != "restricted" {
			return fmt.Errorf("%s holds an allowed sensitive field but is labelled %q", x.id, label[x.id])
		}
	}
	for _, rep := range reports {
		if strings.Contains(rep, "VAL-") {
			return fmt.Errorf("a dropped-field report holds a value: %q", rep)
		}
	}
	return nil
}

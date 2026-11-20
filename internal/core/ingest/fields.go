package ingest

import (
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// fieldText builds a structured document's text from its allowed fields (sb-ingest(7),
// Fields; SB-1106): declared fields that aren't sensitive, and sensitive ones a holding
// space's allow-list names. Every other field is dropped and reported by name only.
// sensitive is true when an allowed field is sensitive; ok is false when the connector
// can't declare its fields, and then nothing of the document is ingested.
func (p Pipeline) fieldText(kind string, doc ports.Document, spaces []ports.Space, holders []string) (text string, sensitive, ok bool) {
	decl, isDecl := p.Connector.(ports.FieldDeclarer)
	if !isDecl {
		p.dropped(kind, []string{"(undeclared)"})
		return "", false, false
	}
	declared, can := decl.Fields()
	if !can {
		p.dropped(kind, []string{"(undeclared)"})
		return "", false, false
	}
	allowed := map[string]bool{}
	for _, sp := range spaces {
		for _, h := range holders {
			if sp.ID == h {
				for _, f := range sp.Fields[kind] {
					allowed[f] = true
				}
			}
		}
	}
	var lines []string
	used := map[string]bool{}
	for _, f := range declared {
		v, present := doc.Fields[f.Name]
		if !present {
			continue
		}
		used[f.Name] = true
		if f.Sensitive && !allowed[f.Name] {
			continue
		}
		if f.Sensitive {
			sensitive = true
		}
		lines = append(lines, f.Name+": "+v)
	}
	var drop []string
	for name := range doc.Fields {
		if !used[name] {
			drop = append(drop, name) // never declared: never allowed
		}
	}
	for _, f := range declared {
		if _, present := doc.Fields[f.Name]; present && f.Sensitive && !allowed[f.Name] {
			drop = append(drop, f.Name)
		}
	}
	sort.Strings(drop)
	if len(drop) > 0 {
		p.dropped(kind, drop)
	}
	return strings.Join(lines, "\n"), sensitive, true
}

func (p Pipeline) dropped(kind string, names []string) {
	if p.FieldsDropped != nil {
		p.FieldsDropped(kind, names)
	}
}

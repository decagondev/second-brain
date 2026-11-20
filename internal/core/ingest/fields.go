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
func (p Pipeline) fieldText(kind string, doc ports.Document, spaces []ports.Space, holders []string) (text string, sensitive bool, narrowed []string, ok bool) {
	decl, isDecl := p.Connector.(ports.FieldDeclarer)
	if !isDecl {
		p.dropped(kind, []string{"(undeclared)"})
		return "", false, holders, false
	}
	declared, can := decl.Fields()
	if !can {
		p.dropped(kind, []string{"(undeclared)"})
		return "", false, holders, false
	}
	allows := map[string]map[string]bool{} // holder -> fields it allow-lists
	for _, sp := range spaces {
		for _, h := range holders {
			if sp.ID == h {
				allows[h] = map[string]bool{}
				for _, f := range sp.Fields[kind] {
					allows[h][f] = true
				}
			}
		}
	}
	// A sensitive field goes in only with the holders that allow it: the chunk's holders
	// narrow to the spaces that allow every sensitive field it carries, so another space
	// holding the same record never reads them (SB-1401). If no holder allows them all,
	// only the sensitive fields every holder allows go in.
	var sens []string
	for _, f := range declared {
		if _, present := doc.Fields[f.Name]; present && f.Sensitive {
			for _, h := range holders {
				if allows[h][f.Name] {
					sens = append(sens, f.Name)
					break
				}
			}
		}
	}
	narrowed = holders
	if len(sens) > 0 {
		var keep []string
		for _, h := range holders {
			all := true
			for _, f := range sens {
				all = all && allows[h][f]
			}
			if all {
				keep = append(keep, h)
			}
		}
		if len(keep) == 0 {
			var common []string
			for _, f := range sens {
				every := true
				for _, h := range holders {
					every = every && allows[h][f]
				}
				if every {
					common = append(common, f)
				}
			}
			sens = common
		} else {
			narrowed = keep
		}
	}
	inSens := map[string]bool{}
	for _, f := range sens {
		inSens[f] = true
	}
	var lines []string
	used := map[string]bool{}
	var drop []string
	for _, f := range declared {
		v, present := doc.Fields[f.Name]
		if !present {
			continue
		}
		used[f.Name] = true
		if f.Sensitive && !inSens[f.Name] {
			drop = append(drop, f.Name)
			continue
		}
		if f.Sensitive {
			sensitive = true
		}
		lines = append(lines, f.Name+": "+v)
	}
	for name := range doc.Fields {
		if !used[name] {
			drop = append(drop, name) // never declared: never allowed
		}
	}
	sort.Strings(drop)
	if len(drop) > 0 {
		p.dropped(kind, drop)
	}
	return strings.Join(lines, "\n"), sensitive, narrowed, true
}

func (p Pipeline) dropped(kind string, names []string) {
	if p.FieldsDropped != nil {
		p.FieldsDropped(kind, names)
	}
}

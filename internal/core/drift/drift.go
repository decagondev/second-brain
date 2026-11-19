// Package drift compares a cell's live configuration with what was applied from its
// manifest (sb-tenant(5), DRIFT; SB-510). Each difference names the resource and the
// field; nothing is changed automatically (LAW-16).
package drift

import (
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/ports"
)

// Finding is one difference: a field that differs, or a resource on one side only.
type Finding struct {
	Kind, Name string
	Field      string // "" when the whole resource is on one side only
	Applied    string
	Live       string
	Present    string // "applied", "live" or "" (both)
}

// Key is kind/name/field.
func (f Finding) Key() string { return f.Kind + "/" + f.Name + "/" + f.Field }

// Message is the alert text (manifest_drift).
func (f Finding) Message() string {
	switch f.Present {
	case "applied":
		return fmt.Sprintf("%s %s: applied, not present live", f.Kind, f.Name)
	case "live":
		return fmt.Sprintf("%s %s: present live, not applied", f.Kind, f.Name)
	}
	return fmt.Sprintf("%s %s: attribute %s: applied %q, live %q", f.Kind, f.Name, f.Field, f.Applied, f.Live)
}

// Detect lists every difference, sorted by key.
func Detect(applied, live []ports.Resource) []Finding {
	key := func(r ports.Resource) string { return r.Kind + "/" + r.Name }
	am, lm := map[string]ports.Resource{}, map[string]ports.Resource{}
	for _, r := range applied {
		am[key(r)] = r
	}
	for _, r := range live {
		lm[key(r)] = r
	}
	var out []Finding
	for k, a := range am {
		l, ok := lm[k]
		if !ok {
			out = append(out, Finding{Kind: a.Kind, Name: a.Name, Present: "applied"})
			continue
		}
		fields := map[string]bool{}
		for f := range a.Attrs {
			fields[f] = true
		}
		for f := range l.Attrs {
			fields[f] = true
		}
		for f := range fields {
			av, aok := a.Attrs[f]
			lv, lok := l.Attrs[f]
			if av != lv || aok != lok {
				out = append(out, Finding{Kind: a.Kind, Name: a.Name, Field: f, Applied: av, Live: lv})
			}
		}
	}
	for k, l := range lm {
		if _, ok := am[k]; !ok {
			out = append(out, Finding{Kind: l.Kind, Name: l.Name, Present: "live"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

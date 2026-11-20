package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// StewardChange replaces one space's definition with a steward's proposal and checks the
// result (sb-spaces(7), Delegated changes; SB-1201). It refuses, with an error, a change to
// any other part of the manifest, to the space's template, a lower label, or no stewards;
// and returns the diagnostics of sb tenant check for anything else wrong.
func StewardChange(t Tenant, space string, proposed json.RawMessage) (Tenant, []Diag, error) {
	old, ok := t.Spaces[space]
	if !ok {
		return Tenant{}, nil, fmt.Errorf("manifest: no space %q", space)
	}
	var sp Space
	dec := json.NewDecoder(bytes.NewReader(proposed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sp); err != nil {
		return Tenant{}, nil, fmt.Errorf("manifest: the proposal isn't a space definition: %v", err)
	}
	if sp.Template == "" {
		sp.Template = old.Template
	}
	switch {
	case sp.Label != "" && rankLabel(sp.Label) == len(Labels):
		return Tenant{}, nil, fmt.Errorf("manifest: %q is not a label; use one of %s", sp.Label, strings.Join(Labels, ", "))
	case sp.Template != old.Template:
		return Tenant{}, nil, fmt.Errorf("manifest: a steward can't change the space's template (%s)", old.Template)
	case sp.Label != "" && rankLabel(sp.Label) < rankLabel(old.Label):
		return Tenant{}, nil, fmt.Errorf("manifest: a steward can't lower the space's label from %s to %s", old.Label, sp.Label)
	case len(sp.Stewards) == 0:
		return Tenant{}, nil, fmt.Errorf("manifest: a space changed by its stewards keeps at least one steward")
	}
	if sp.Label == "" {
		sp.Label = old.Label
	}
	checked, ds := WithSpace(t, space, sp)
	if len(ds) > 0 {
		return Tenant{}, ds, nil
	}
	// What was checked is what was written: a value that reads back differently (a YAML
	// comment or quote inside it) is refused, and the label is judged as written (SB-1401).
	got := checked.Spaces[space]
	if rankLabel(got.Label) < rankLabel(old.Label) || got.Label != sp.Label ||
		!sameStrings(got.Members, sp.Members) || !sameStrings(got.Stewards, sp.Stewards) ||
		(sp.Agents != nil && !sameStrings(got.Agents, sp.Agents)) || (sp.Channels != nil && !sameStrings(got.Channels, sp.Channels)) ||
		!reflect.DeepEqual(nonNilScope(got.Scope), nonNilScope(sp.Scope)) {
		return Tenant{}, nil, fmt.Errorf("manifest: the proposal doesn't read back as written; remove comments and quotes from its values")
	}
	// Belt and braces: everything but this space is exactly as it was.
	a, b := checked, t
	a.Spaces, b.Spaces = nil, nil
	if !reflect.DeepEqual(a, b) {
		return Tenant{}, nil, fmt.Errorf("manifest: the proposal changes more than space %s", space)
	}
	for k, v := range t.Spaces {
		if k != space && !reflect.DeepEqual(checked.Spaces[k], v) {
			return Tenant{}, nil, fmt.Errorf("manifest: the proposal changes space %s too", k)
		}
	}
	return checked, nil, nil
}

// WithSpace returns t with one space set to sp, its unset fields filled in as the checker
// fills them (the template's agents and retention), checked as sb tenant check does.
func WithSpace(t Tenant, space string, sp Space) (Tenant, []Diag) {
	tpl := Templates[sp.Template]
	if sp.Template == "" {
		sp.Template, tpl = "custom", Templates["custom"]
	}
	if sp.Label == "" {
		sp.Label = tpl.Label
	}
	if sp.Agents == nil {
		sp.Agents = []string{}
		for _, a := range tpl.Agents {
			if contains(t.Agents, a) {
				sp.Agents = append(sp.Agents, a)
			}
		}
	}
	if sp.RetentionDays == 0 {
		sp.RetentionDays = t.RetentionDays
		if tpl.RetentionDays > 0 && tpl.RetentionDays < sp.RetentionDays {
			sp.RetentionDays = tpl.RetentionDays
		}
	}
	if sp.Stewards == nil {
		sp.Stewards = []string{}
	}
	next := t
	next.Spaces = map[string]Space{}
	for k, v := range t.Spaces {
		next.Spaces[k] = v
	}
	next.Spaces[space] = sp
	return Check(Format(next))
}

func rankLabel(l string) int {
	for i, x := range Labels {
		if x == l {
			return i
		}
	}
	return len(Labels) // unknown counts as above restricted, so nothing can be lowered past it
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func nonNilScope(m map[string][]string) map[string][]string {
	out := map[string][]string{}
	for k, v := range m {
		out[k] = append([]string{}, v...)
	}
	return out
}

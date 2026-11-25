package manifest

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Op is one JSON Patch operation on the manifest (sb-admin(3), The manifest diff).
type Op struct {
	Op    string          `json:"op"` // add, remove, replace
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

// Patchable are the paths the console may change; tenant, region and idp never.
var Patchable = []string{"/agents", "/workflows", "/connectors/", "/approval/", "/exclude/", "/retention_days", "/budget_usd_month", "/pilot_group", "/channel"}

func patchable(p string) bool {
	for _, a := range Patchable {
		if p == a || p == a+"/-" || (strings.HasSuffix(a, "/") && strings.HasPrefix(p, a) && !strings.Contains(p[len(a):], "/")) {
			return true
		}
	}
	return strings.HasPrefix(p, "/agents/") || strings.HasPrefix(p, "/workflows/")
}

// Patch applies ops to t and checks the result exactly as sb tenant check does. It
// returns the new manifest, or the check's diagnostics, or an error for a bad patch.
func Patch(t Tenant, ops []Op) (Tenant, []Diag, error) {
	raw, _ := json.Marshal(t)
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	for _, op := range ops {
		if !patchable(op.Path) {
			return Tenant{}, nil, fmt.Errorf("manifest: %s can't be changed from the console", op.Path)
		}
		if err := apply(doc, op); err != nil {
			return Tenant{}, nil, err
		}
	}
	b, _ := json.Marshal(doc)
	var next Tenant
	if err := json.Unmarshal(b, &next); err != nil {
		return Tenant{}, nil, fmt.Errorf("manifest: the patch gives a value of the wrong type: %v", err)
	}
	for a, usd := range t.AgentCaps { // the caps aren't in the JSON form: keep those still named (SB-4703)
		if contains(CappedOnly, a) || contains(next.Agents, a) {
			if next.AgentCaps == nil {
				next.AgentCaps = map[string]float64{}
			}
			next.AgentCaps[a] = usd
		}
	}
	checked, ds := Check(Format(next))
	if len(ds) > 0 {
		return Tenant{}, ds, nil
	}
	return checked, nil, nil
}

func apply(doc map[string]any, op Op) error {
	parts := strings.Split(strings.TrimPrefix(op.Path, "/"), "/")
	var val any
	if op.Op != "remove" {
		if err := json.Unmarshal(op.Value, &val); err != nil {
			return fmt.Errorf("manifest: %s needs a JSON value", op.Path)
		}
	}
	key := parts[0]
	switch len(parts) {
	case 1:
		switch op.Op {
		case "add", "replace":
			doc[key] = val
		case "remove":
			delete(doc, key)
		default:
			return fmt.Errorf("manifest: unknown op %q", op.Op)
		}
		return nil
	case 2:
		switch cur := doc[key].(type) {
		case []any, nil:
			list, _ := cur.([]any)
			if parts[1] == "-" && op.Op == "add" {
				doc[key] = append(list, val)
				return nil
			}
			if op.Op == "remove" {
				for i, x := range list {
					if fmt.Sprint(x) == parts[1] {
						doc[key] = append(list[:i:i], list[i+1:]...)
						return nil
					}
				}
				return fmt.Errorf("manifest: %s isn't in %s", parts[1], key)
			}
			return fmt.Errorf("manifest: lists take add at /-, or remove by value")
		case map[string]any:
			switch op.Op {
			case "add", "replace":
				cur[parts[1]] = val
			case "remove":
				if _, ok := cur[parts[1]]; !ok {
					return fmt.Errorf("manifest: no %s in %s", parts[1], key)
				}
				delete(cur, parts[1])
			default:
				return fmt.Errorf("manifest: unknown op %q", op.Op)
			}
			return nil
		}
	}
	return fmt.Errorf("manifest: bad path %s", op.Path)
}

// Diff is a line diff of two formatted manifests: "- old" and "+ new" lines. A nested
// line is written with its full path (spaces.finance.label: restricted), and lines are
// compared as a multiset, so a change in one space can't hide behind the same text in
// another (SB-1401).
func Diff(a, b Tenant) []string {
	al, bl := pathLines(Format(a)), pathLines(Format(b))
	count := func(xs []string) map[string]int {
		m := map[string]int{}
		for _, x := range xs {
			m[x]++
		}
		return m
	}
	ac, bc := count(al), count(bl)
	var out []string
	seen := map[string]int{}
	for _, l := range al {
		if seen["-"+l]++; seen["-"+l] > bc[l] {
			out = append(out, "- "+l)
		}
	}
	for _, l := range bl {
		if seen["+"+l]++; seen["+"+l] > ac[l] {
			out = append(out, "+ "+l)
		}
	}
	return out
}

// pathLines turns a formatted manifest into one line per value, each nested line prefixed
// with the keys above it.
func pathLines(src []byte) []string {
	type level struct {
		indent int
		key    string
	}
	var stack []level
	var out []string
	for _, raw := range strings.Split(strings.TrimRight(string(src), "\n"), "\n") {
		line := strings.TrimLeft(raw, " ")
		indent := len(raw) - len(line)
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		path := make([]string, len(stack))
		for i, l := range stack {
			path[i] = l.key
		}
		text := line
		if len(path) > 0 {
			text = strings.Join(path, ".") + "." + line
		}
		out = append(out, text)
		if strings.HasSuffix(line, ":") {
			stack = append(stack, level{indent, strings.Trim(strings.TrimSuffix(line, ":"), `"`)})
		}
	}
	return out
}

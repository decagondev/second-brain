// Package plan turns a manifest into desired resources and diffs them against a tenant's
// applied state (sb-tenant(1), `apply --plan`). It is pure: no I/O, no clock.
package plan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// Resource kinds, in plan order (sb-tenant(1)).
const (
	KindCell      = "cell"
	KindSetting   = "setting"
	KindConnector = "connector"
	KindExclude   = "exclude"
	KindAgent     = "agent"
	KindWorkflow  = "workflow"
	KindSpace     = "space"
	KindBarrier   = "barrier"
)

var kindOrder = map[string]int{KindCell: 0, KindSetting: 1, KindConnector: 2, KindExclude: 3, KindAgent: 4, KindWorkflow: 5, KindSpace: 6, KindBarrier: 7}

// Ops.
const (
	OpAdd    = "add"
	OpChange = "change"
	OpRemove = "remove"
)

// Change is one line of a plan. Was holds the old attributes of a change.
type Change struct {
	Op    string            `json:"op"`
	Kind  string            `json:"kind"`
	Name  string            `json:"name"`
	Attrs map[string]string `json:"attrs,omitempty"`
	Was   map[string]string `json:"was,omitempty"`
}

// Plan is the difference between a manifest and the applied state.
type Plan struct {
	Tenant  string   `json:"tenant"`
	Changes []Change `json:"changes"`
	Add     int      `json:"add"`
	Change  int      `json:"change"`
	Remove  int      `json:"remove"`
}

// ImmutableError reports an attempt to change a field that would need a new tenant.
type ImmutableError struct{ Field, Applied, Wanted string }

func (e *ImmutableError) Error() string {
	return fmt.Sprintf("%s: is immutable; the cell is in %s, not %s, and changing it means a new tenant", e.Field, e.Applied, e.Wanted)
}

// Desired lists the resources a valid manifest asks for, in plan order.
func Desired(t manifest.Tenant) []ports.Resource {
	rs := []ports.Resource{{Kind: KindCell, Name: t.Tenant, Attrs: map[string]string{"region": t.Region}}}
	set := func(name, v string) {
		rs = append(rs, ports.Resource{Kind: KindSetting, Name: name, Attrs: map[string]string{"value": v}})
	}
	set("channel", t.Channel)
	set("idp", t.IdP)
	set("retention_days", strconv.Itoa(t.RetentionDays))
	set("budget_usd_month", strconv.Itoa(t.BudgetUSDMonth))
	if t.PilotGroup != "" {
		set("pilot_group", t.PilotGroup)
	}
	writes := map[string]bool{}
	for _, w := range t.Workflows {
		writes[manifest.Workflows[w]] = true
		rs = append(rs, ports.Resource{Kind: KindWorkflow, Name: w, Attrs: map[string]string{"approval": t.Approval[w]}})
	}
	for kind, vendor := range t.Connectors {
		mode := "read-only"
		if writes[kind] {
			mode = "read-write"
		}
		rs = append(rs, ports.Resource{Kind: KindConnector, Name: kind, Attrs: map[string]string{"vendor": vendor, "mode": mode}})
	}
	for kind, ps := range t.Exclude {
		rs = append(rs, ports.Resource{Kind: KindExclude, Name: kind, Attrs: map[string]string{"patterns": strings.Join(ps, ", ")}})
	}
	for _, a := range t.Agents {
		rs = append(rs, ports.Resource{Kind: KindAgent, Name: a})
	}
	for name, sp := range t.Spaces {
		rs = append(rs, ports.Resource{Kind: KindSpace, Name: name, Attrs: manifest.SpaceAttrs(sp)})
	}
	for _, b := range t.Barriers {
		rs = append(rs, ports.Resource{Kind: KindBarrier, Name: b[0] + " / " + b[1]})
	}
	sortResources(rs)
	return rs
}

func sortResources(rs []ports.Resource) {
	sort.SliceStable(rs, func(i, j int) bool {
		a, b := rs[i], rs[j]
		if kindOrder[a.Kind] != kindOrder[b.Kind] {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		return a.Name < b.Name
	})
}

func key(r ports.Resource) string { return r.Kind + "\x00" + r.Name }

// Make diffs the manifest against the applied state. found is the registry's answer to
// "does this tenant have a cell"; the caller must never default it to false on an error.
func Make(t manifest.Tenant, applied []ports.Resource, found bool) (Plan, error) {
	if !found {
		applied = nil
	}
	for _, r := range applied {
		if r.Kind == KindCell && r.Attrs["region"] != t.Region {
			return Plan{}, &ImmutableError{Field: "region", Applied: r.Attrs["region"], Wanted: t.Region}
		}
	}
	p := Plan{Tenant: t.Tenant, Changes: []Change{}}
	have := map[string]ports.Resource{}
	for _, r := range applied {
		have[key(r)] = r
	}
	want := Desired(t)
	wanted := map[string]bool{}
	for _, r := range want {
		wanted[key(r)] = true
		old, ok := have[key(r)]
		switch {
		case !ok:
			p.Changes = append(p.Changes, Change{Op: OpAdd, Kind: r.Kind, Name: r.Name, Attrs: r.Attrs})
		case !sameAttrs(old.Attrs, r.Attrs):
			p.Changes = append(p.Changes, Change{Op: OpChange, Kind: r.Kind, Name: r.Name, Attrs: r.Attrs, Was: old.Attrs})
		}
	}
	var gone []ports.Resource
	for _, r := range applied {
		if !wanted[key(r)] {
			gone = append(gone, r)
		}
	}
	sortResources(gone)
	for _, r := range gone {
		p.Changes = append(p.Changes, Change{Op: OpRemove, Kind: r.Kind, Name: r.Name, Attrs: r.Attrs})
	}
	sort.SliceStable(p.Changes, func(i, j int) bool {
		a, b := p.Changes[i], p.Changes[j]
		if kindOrder[a.Kind] != kindOrder[b.Kind] {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		return a.Name < b.Name
	})
	for _, c := range p.Changes {
		switch c.Op {
		case OpAdd:
			p.Add++
		case OpChange:
			p.Change++
		case OpRemove:
			p.Remove++
		}
	}
	return p, nil
}

func sameAttrs(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// Apply returns the state after p is applied to applied, in plan order. SB-104's apply
// engine enacts the same plan; the law test uses this to show the plan is complete.
func Apply(applied []ports.Resource, p Plan) []ports.Resource {
	state := map[string]ports.Resource{}
	for _, r := range applied {
		state[key(r)] = r
	}
	for _, c := range p.Changes {
		r := ports.Resource{Kind: c.Kind, Name: c.Name, Attrs: c.Attrs}
		switch c.Op {
		case OpAdd, OpChange:
			state[key(r)] = r
		case OpRemove:
			delete(state, key(r))
		}
	}
	out := make([]ports.Resource, 0, len(state))
	for _, r := range state {
		out = append(out, r)
	}
	sortResources(out)
	return out
}

// detail renders a resource's attributes the way sb-tenant(1) shows them.
func detail(kind string, a map[string]string) string {
	switch kind {
	case KindCell:
		return "region " + a["region"]
	case KindSetting:
		return a["value"]
	case KindConnector:
		return a["vendor"] + ", " + a["mode"]
	case KindExclude:
		return a["patterns"]
	case KindWorkflow:
		return "approval " + a["approval"]
	case KindSpace:
		parts := []string{"template " + a["template"], "label " + a["label"], "members " + a["members"]}
		if a["stewards"] != "" {
			parts = append(parts, "stewards "+a["stewards"])
		}
		parts = append(parts, "agents "+a["agents"], "retention "+a["retention_days"], "scope "+a["scope"])
		if a["channels"] != "" {
			parts = append(parts, "channels "+a["channels"])
		}
		return strings.Join(parts, ", ")
	}
	return ""
}

// Line renders one change, e.g. "~ setting budget_usd_month = 1500 (was 1000)".
func (c Change) Line() string {
	sign := map[string]string{OpAdd: "+", OpChange: "~", OpRemove: "-"}[c.Op]
	head := sign + " " + c.Kind + " " + c.Name
	now := detail(c.Kind, c.Attrs)
	switch {
	case c.Kind == KindAgent || c.Kind == KindBarrier:
		return head
	case c.Kind == KindSetting && c.Op == OpChange:
		return head + " = " + now + " (was " + detail(c.Kind, c.Was) + ")"
	case c.Kind == KindSetting:
		return head + " = " + now
	case c.Op == OpChange:
		return head + " (" + now + "; was " + detail(c.Kind, c.Was) + ")"
	}
	return head + " (" + now + ")"
}

// Text renders the whole plan for people, ending with the summary line.
func (p Plan) Text() string {
	var b strings.Builder
	for _, c := range p.Changes {
		b.WriteString(c.Line() + "\n")
	}
	if len(p.Changes) == 0 {
		b.WriteString("plan: no changes\n")
	} else {
		fmt.Fprintf(&b, "plan: %d to add, %d to change, %d to remove\n", p.Add, p.Change, p.Remove)
	}
	return b.String()
}

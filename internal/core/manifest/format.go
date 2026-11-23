package manifest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Format writes a valid Tenant back as manifest YAML in the examples' style (flow maps
// and lists, one field per line), so that Check(Format(t)) gives t back (SB-602).
func Format(t Tenant) []byte {
	var b strings.Builder
	line := func(k, v string) { fmt.Fprintf(&b, "%s: %s\n", k, v) }
	line("tenant", t.Tenant)
	line("channel", t.Channel)
	line("idp", t.IdP)
	line("region", t.Region)
	if len(t.Connectors) > 0 {
		line("connectors", flowMap(t.Connectors))
	}
	if len(t.Agents) > 0 {
		line("agents", flowList(t.Agents, false))
	}
	if len(t.Workflows) > 0 {
		line("workflows", flowList(t.Workflows, false))
	}
	if len(t.Approval) > 0 {
		line("approval", flowMap(t.Approval))
	}
	if len(t.Duties) > 0 {
		b.WriteString("duties:\n")
		for _, k := range sorted(t.Duties) {
			fmt.Fprintf(&b, "  %s: { second: %s, above: %d }\n", k, t.Duties[k].Second, t.Duties[k].Above)
		}
	}
	line("retention_days", strconv.Itoa(t.RetentionDays))
	line("budget_usd_month", strconv.Itoa(t.BudgetUSDMonth))
	if t.PilotGroup != "" {
		line("pilot_group", t.PilotGroup)
	}
	if t.QuestionsTo != "" {
		line("questions_to", t.QuestionsTo)
	}
	if t.Notes {
		line("notes", "true")
	}
	if t.Digest != "" {
		line("digest", t.Digest)
	}
	if t.Language != "" {
		line("language", t.Language)
	}
	if len(t.Exclude) > 0 {
		keys := sorted(t.Exclude)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + ": " + flowList(t.Exclude[k], true)
		}
		line("exclude", "{ "+strings.Join(parts, ", ")+" }")
	}
	if len(t.Groups) > 0 {
		b.WriteString("groups:\n")
		for _, k := range sorted(t.Groups) {
			fmt.Fprintf(&b, "  %s: %s\n", strconv.Quote(k), flowList(t.Groups[k], false))
		}
	}
	if len(t.Spaces) > 0 {
		b.WriteString("spaces:" + nl)
		names := make([]string, 0, len(t.Spaces))
		for n := range t.Spaces {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			sp := t.Spaces[n]
			fmt.Fprintf(&b, "  %s:%s", n, nl)
			fmt.Fprintf(&b, "    template: %s%s", sp.Template, nl)
			fmt.Fprintf(&b, "    members: %s%s", flowList(sp.Members, false), nl)
			if len(sp.Stewards) > 0 {
				fmt.Fprintf(&b, "    stewards: %s%s", flowList(sp.Stewards, false), nl)
			}
			fmt.Fprintf(&b, "    label: %s%s", sp.Label, nl)
			fmt.Fprintf(&b, "    agents: %s%s", flowList(sp.Agents, false), nl)
			fmt.Fprintf(&b, "    retention_days: %d%s", sp.RetentionDays, nl)
			if len(sp.Fields) > 0 {
				fp := make([]string, 0, len(sp.Fields))
				for _, k := range sorted(sp.Fields) {
					fp = append(fp, k+": "+flowList(sp.Fields[k], false))
				}
				fmt.Fprintf(&b, "    fields: { %s }%s", strings.Join(fp, ", "), nl)
			}
			if len(sp.Channels) > 0 {
				fmt.Fprintf(&b, "    channels: %s%s", flowList(sp.Channels, true), nl)
			}
			if sp.ReviewAfterDays != 0 && sp.ReviewAfterDays != DefaultReviewAfterDays {
				fmt.Fprintf(&b, "    review_after_days: %d%s", sp.ReviewAfterDays, nl)
			}
			if len(sp.StartHere) > 0 {
				fmt.Fprintf(&b, "    start_here: %s%s", flowList(sp.StartHere, true), nl)
			}
			if len(sp.Glossary) > 0 {
				gp := make([]string, 0, len(sp.Glossary))
				for _, k := range sorted(sp.Glossary) {
					gp = append(gp, strconv.Quote(k)+": "+strconv.Quote(sp.Glossary[k]))
				}
				fmt.Fprintf(&b, "    glossary: { %s }%s", strings.Join(gp, ", "), nl)
			}
			parts := make([]string, 0, len(sp.Scope))
			for _, k := range sorted(sp.Scope) {
				parts = append(parts, k+": "+flowList(sp.Scope[k], true))
			}
			fmt.Fprintf(&b, "    scope: { %s }%s", strings.Join(parts, ", "), nl)
		}
	}
	if len(t.Barriers) > 0 {
		b.WriteString("barriers:" + nl)
		for _, p := range t.Barriers {
			fmt.Fprintf(&b, "  - [%s, %s]%s", p[0], p[1], nl)
		}
	}
	return []byte(b.String())
}

const nl = "\n"

func flowMap(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + m[k]
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func flowList(xs []string, quote bool) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		if quote {
			parts[i] = strconv.Quote(x)
		} else {
			parts[i] = x
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func sorted[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

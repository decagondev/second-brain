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
	if v := agentsLine(t); v != "" {
		line("agents", v)
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
	if t.AsksPerMinute != 0 && t.AsksPerMinute != DefaultAsksPerMinute {
		line("asks_per_minute", strconv.Itoa(t.AsksPerMinute))
	}
	if t.ResearchUSD != 0 {
		line("research_usd", strconv.FormatFloat(t.ResearchUSD, 'f', -1, 64))
	}
	formatSettings(t, line)
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
			if sp.Parent != "" {
				fmt.Fprintf(&b, "    parent: %s%s", sp.Parent, nl)
			}
			fmt.Fprintf(&b, "    members: %s%s", flowList(sp.Members, false), nl)
			if len(sp.Stewards) > 0 {
				fmt.Fprintf(&b, "    stewards: %s%s", flowList(sp.Stewards, false), nl)
			}
			fmt.Fprintf(&b, "    label: %s%s", sp.Label, nl)
			fmt.Fprintf(&b, "    agents: %s%s", flowList(sp.Agents, false), nl)
			fmt.Fprintf(&b, "    retention_days: %d%s", sp.RetentionDays, nl)
			if sp.Updates {
				fmt.Fprintf(&b, "    updates: true%s", nl)
			}
			b.WriteString(formatWays(sp, nl)) // SB-4801, SB-4802
			if sp.LeaderWeek != nil {
				fmt.Fprintf(&b, "    leader_week: %t%s", *sp.LeaderWeek, nl)
			}
			if sp.Requests != nil {
				fmt.Fprintf(&b, "    requests: { categories: %s, due_days: %d }%s", flowList(sp.Requests.Categories, true), sp.Requests.DueDays, nl)
			}
			if sp.Model != "" {
				fmt.Fprintf(&b, "    model: %s%s", sp.Model, nl)
			}
			if sp.BudgetUSDMonth > 0 {
				fmt.Fprintf(&b, "    budget_usd_month: %d%s", sp.BudgetUSDMonth, nl)
			}
			if sp.IncidentLabels != nil {
				fmt.Fprintf(&b, "    incident_labels: %s%s", flowList(sp.IncidentLabels, true), nl)
			}
			if sp.Brief != nil {
				fmt.Fprintf(&b, "    brief: { agent: %s, weekly: %s }%s", sp.Brief.Agent, sp.Brief.Weekly, nl)
			}
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
			b.WriteString(glossaryInLine(sp.GlossaryIn, nl)) // SB-6104
			if len(sp.Skills) > 0 {                          // SB-4601
				sk := make([]string, 0, len(sp.Skills))
				for _, k := range sorted(sp.Skills) {
					cs := make([]string, 0, len(sp.Skills[k].Cases))
					for _, q := range sorted(sp.Skills[k].Cases) {
						cs = append(cs, strconv.Quote(q)+": "+strconv.Quote(sp.Skills[k].Cases[q]))
					}
					sk = append(sk, k+": { instruction: "+strconv.Quote(sp.Skills[k].Instruction)+", cases: { "+strings.Join(cs, ", ")+" } }")
				}
				fmt.Fprintf(&b, "    skills: { %s }%s", strings.Join(sk, ", "), nl)
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

// formatSettings writes the optional one-line settings, in sb-tenant(5) order.
func formatSettings(t Tenant, line func(k, v string)) {
	if l := limitsLine(t); l != "" { // SB-5003
		line("limits", l)
	}
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
	if t.Memory == "off" {
		line("memory", "off")
	}
	if t.Events != nil {
		line("events", "{ url: "+t.Events.URL+", kinds: "+flowList(t.Events.Kinds, false)+" }")
	}
}

// agentsLine is the agents row: a list, or the map form when any agent has a cap (SB-4703).
func agentsLine(t Tenant) string {
	if len(t.AgentCaps) == 0 && len(t.AgentGoals) == 0 {
		if len(t.Agents) == 0 {
			return ""
		}
		return flowList(t.Agents, false)
	}
	var parts []string
	var extra []string // capped keys not in the list, in name order so Format is stable
	for a := range t.AgentCaps {
		if !contains(t.Agents, a) {
			extra = append(extra, a)
		}
	}
	for a := range t.AgentGoals { // SB-5004
		if !contains(t.Agents, a) && !contains(extra, a) {
			extra = append(extra, a)
		}
	}
	sort.Strings(extra)
	names := append(append([]string{}, t.Agents...), extra...)
	for _, a := range names {
		var in []string
		if usd, ok := t.AgentCaps[a]; ok {
			in = append(in, "monthly_usd: "+strconv.FormatFloat(usd, 'f', -1, 64))
		}
		if g := goalPart(t.AgentGoals[a]); g != "" {
			in = append(in, g)
		}
		if len(in) == 0 {
			parts = append(parts, a+": {}")
		} else {
			parts = append(parts, a+": { "+strings.Join(in, ", ")+" }")
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// glossaryInLine is a space's glossary_in line, "" for none (SB-6104).
func glossaryInLine(gi map[string]map[string]string, nl string) string {
	if len(gi) == 0 {
		return ""
	}
	gp := make([]string, 0, len(gi))
	for _, k := range sorted(gi) {
		ws := make([]string, 0, len(gi[k]))
		for _, l := range sorted(gi[k]) {
			ws = append(ws, l+": "+strconv.Quote(gi[k][l]))
		}
		gp = append(gp, strconv.Quote(k)+": { "+strings.Join(ws, ", ")+" }")
	}
	return "    glossary_in: { " + strings.Join(gp, ", ") + " }" + nl
}

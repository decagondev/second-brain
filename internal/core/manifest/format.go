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
	line("retention_days", strconv.Itoa(t.RetentionDays))
	line("budget_usd_month", strconv.Itoa(t.BudgetUSDMonth))
	if t.PilotGroup != "" {
		line("pilot_group", t.PilotGroup)
	}
	if len(t.Exclude) > 0 {
		keys := sorted(t.Exclude)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + ": " + flowList(t.Exclude[k], true)
		}
		line("exclude", "{ "+strings.Join(parts, ", ")+" }")
	}
	return []byte(b.String())
}

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

func sorted(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

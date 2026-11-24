package ask

import (
	"context"
	"sort"

	"example.com/second-brain/internal/core/tenancy"
)

// The right department (sb-ask(3), The right department; SB-4007): a company-wide ask whose
// question uses a department space's glossary terms suggests that space's agent, once. The
// suggestion only names a space the asker is a member of, and changes nothing about the
// answer: the person chooses to ask there.

// DeptSpace is a space the asker is a member of that runs a department agent: its name,
// its agent, and its glossary (with its department's, SB-3902).
type DeptSpace struct {
	Name, Agent string
	Glossary    map[string]string
}

// Suggestion is the one space whose agent the answer suggests.
type Suggestion struct {
	Space string   `json:"space"`
	Agent string   `json:"agent"`
	Terms []string `json:"terms"`
}

// suggest is the member space whose usable glossary the question uses most, by the number
// of terms, then by name; nil for none, for an ask in a space, or with an agent.
func (a Agent) suggest(ctx context.Context, s tenancy.Scope, q string, req Request) *Suggestion {
	if a.Departmental == nil || s.Space() != "company" || (req.Agent != "" && req.Agent != "ask") {
		return nil
	}
	cands := a.Departmental(ctx, s)
	sort.Slice(cands, func(i, j int) bool { return cands[i].Name < cands[j].Name })
	var best *Suggestion
	for _, c := range cands {
		if a.AgentKilled != nil && a.AgentKilled(c.Agent) {
			continue
		}
		ok, _ := usable(c.Glossary)
		_, used := expand(q, ok)
		if len(used) == 0 || (best != nil && len(used) <= len(best.Terms)) {
			continue
		}
		best = &Suggestion{Space: c.Name, Agent: c.Agent}
		for _, t := range used {
			best.Terms = append(best.Terms, t.Term)
		}
	}
	return best
}

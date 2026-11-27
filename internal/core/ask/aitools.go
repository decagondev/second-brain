package ask

import (
	"sort"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// The AI tools that reach the brain (sb-admin(3), AI tools; SB-6006): every token made for
// an AI tool names it, and an admin sees per tool how many live tokens, how many people
// and how many asks this month, never who.

// AIClient is one AI tool's counts.
type AIClient struct {
	Client string `json:"client"`
	Tokens int    `json:"tokens"`
	People int    `json:"people"`
	Calls  int    `json:"calls"`
}

// BuildAIClients counts the live tokens of tenant at now by their tool, and this month's
// asks from each tool's tokens. A token with no tool is left out.
func BuildAIClients(tokens []ports.Session, traces []Trace, tenant string, now time.Time) []AIClient {
	by := map[string]*AIClient{}
	people := map[string]map[string]bool{}
	get := func(c string) *AIClient {
		if by[c] == nil {
			by[c] = &AIClient{Client: c}
			people[c] = map[string]bool{}
		}
		return by[c]
	}
	for _, x := range tokens {
		if x.Token == "" || x.Client == "" || x.Tenant != tenant || !now.Before(x.Expires) {
			continue
		}
		get(x.Client).Tokens++
		people[x.Client][x.User] = true
	}
	month := now.UTC().Format("2006-01")
	for _, t := range traces {
		if t.Tool == "" || t.At.UTC().Format("2006-01") != month {
			continue
		}
		get(t.Tool).Calls++
	}
	out := []AIClient{}
	for c, v := range by {
		v.People = len(people[c])
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Client < out[j].Client })
	return out
}

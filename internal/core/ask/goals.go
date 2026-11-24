package ask

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Goals (sb-ask(3), Goals; SB-4101): a document whose front matter says type: goal, with
// owner, quarter, status and optionally parent_goal, is a goal. Goals stay in the source;
// the brain reads them like any document and lists a branch's by quarter, nested, each
// checked as the asker.

// GoalStatuses are the statuses a goal can have.
var GoalStatuses = []string{"on track", "at risk", "off track", "done"}

var quarterRe = regexp.MustCompile(`^\d{4}-Q[1-4]$`)

// GoalMeta is what a goal's front matter says.
type GoalMeta struct {
	Owner, Quarter, Status, Parent string
}

// frontMatter is the key: value lines of a document's front matter, keys lower case: the
// text may open with its title line and a blank line, as an indexed first chunk does.
func frontMatter(text string) map[string]string {
	if !strings.HasPrefix(text, "---\n") {
		if _, rest, ok := strings.Cut(text, "\n\n"); ok && strings.HasPrefix(rest, "---\n") && !strings.Contains(text[:len(text)-len(rest)-2], "\n") {
			text = rest
		} else {
			return nil
		}
	}
	head := text[4:]
	if len(head) > 4096 {
		head = head[:4096]
	}
	end := strings.Index(head, "\n---")
	if end < 0 {
		return nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(head[:end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok {
			out[strings.ToLower(strings.TrimSpace(k))] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return out
}

// ParseGoal reads a goal's front matter; ok is false for a document that isn't a goal. A
// quarter that isn't YYYY-Qn, or a status that isn't one of GoalStatuses, is left empty.
func ParseGoal(text string) (GoalMeta, bool) {
	fm := frontMatter(text)
	if !strings.EqualFold(fm["type"], "goal") {
		return GoalMeta{}, false
	}
	g := GoalMeta{Owner: strings.ToLower(fm["owner"]), Quarter: strings.ToUpper(fm["quarter"]), Status: strings.ToLower(fm["status"]), Parent: fm["parent_goal"]}
	if !quarterRe.MatchString(g.Quarter) {
		g.Quarter = ""
	}
	if !contains(GoalStatuses, g.Status) {
		g.Status = ""
	}
	return g, true
}

// isGoal reports whether chunk c opens a goal document.
func isGoal(c ports.Chunk) bool {
	_, ok := ParseGoal(c.Text)
	return c.Start == 0 && ok
}

// Goal is one goal as its asker sees it, with the goals under it.
type Goal struct {
	Cite    string `json:"cite"`
	Title   string `json:"title"`
	URL     string `json:"url,omitempty"`
	Space   string `json:"space"`
	Owner   string `json:"owner,omitempty"`
	Quarter string `json:"quarter,omitempty"`
	Status  string `json:"status,omitempty"`
	Goals   []Goal `json:"goals,omitempty"`
	parent  string
	key     string
}

// GoalQuarter is one quarter's goals, the top of each tree.
type GoalQuarter struct {
	Quarter string `json:"quarter"` // "" for goals with no quarter
	Goals   []Goal `json:"goals"`
}

// GoalsResponse is GET /v1/spaces/NAME/goals.
type GoalsResponse struct {
	Space    string        `json:"space"`
	Quarters []GoalQuarter `json:"quarters"`
	// Hidden is how many goals of the branch the asker can't open: counted, never named.
	Hidden int `json:"hidden"`
}

// Goals are the goals of space name and the teams under it the asker is a member of, for a
// member of the space, each checked as them now, by quarter (newest first), nested by
// parent_goal. A goal whose parent they can't open is listed without it.
func (x Stewards) Goals(ctx context.Context, s tenancy.Scope, name string) (GoalsResponse, error) {
	if err := x.member(ctx, s, name); err != nil {
		return GoalsResponse{}, err
	}
	out := GoalsResponse{Space: name, Quarters: []GoalQuarter{}}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	branch := []string{name}
	if x.Children != nil {
		for _, c := range x.Children(name) {
			if x.member(ctx, s, c) == nil {
				branch = append(branch, c)
			}
		}
	}
	var cand []ports.Chunk
	spaceOf := map[string]string{}
	for _, sp := range branch {
		chunks, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Space: SpaceKey(sp)})
		if err != nil {
			return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
		for _, c := range firsts(latest(chunks)) {
			if k := c.Kind + ":" + c.SourceID; isGoal(c) && spaceOf[k] == "" {
				spaceOf[k] = sp
				cand = append(cand, c)
			}
		}
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	var goals []*Goal
	seen := map[string]bool{}
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		if seen[k] {
			continue
		}
		seen[k] = true
		m, _ := ParseGoal(c.Text)
		r := x.Agent.result(ctx, nil, c)
		goals = append(goals, &Goal{Cite: c.Cite, Title: r.Title, URL: r.URL, Space: spaceOf[k], Owner: m.Owner, Quarter: m.Quarter, Status: m.Status, parent: m.Parent, key: k})
	}
	out.Hidden = len(cand) - len(goals)
	out.Quarters = nestGoals(goals)
	return out, nil
}

// nestGoals puts each goal under the visible goal its parent_goal names (by title, or the
// end of its source id), and groups the rest by quarter, newest first.
func nestGoals(goals []*Goal) []GoalQuarter {
	sort.Slice(goals, func(i, j int) bool {
		return goals[i].Title < goals[j].Title || goals[i].Title == goals[j].Title && goals[i].key < goals[j].key
	})
	parentOf := func(g *Goal) *Goal {
		p := strings.ToLower(strings.TrimSpace(g.parent))
		if p == "" {
			return nil
		}
		for _, o := range goals {
			if o != g && (strings.ToLower(o.Title) == p || strings.HasSuffix(strings.ToLower(o.key), p)) {
				return o
			}
		}
		return nil
	}
	children := map[*Goal][]*Goal{}
	var tops []*Goal
	for _, g := range goals {
		if p := parentOf(g); p != nil && parentOf(p) != g {
			children[p] = append(children[p], g)
		} else {
			tops = append(tops, g)
		}
	}
	var build func(g *Goal, depth int) Goal
	build = func(g *Goal, depth int) Goal {
		v := *g
		v.Goals = nil
		if depth < 3 {
			for _, c := range children[g] {
				v.Goals = append(v.Goals, build(c, depth+1))
			}
		}
		return v
	}
	byQ := map[string][]Goal{}
	for _, g := range tops {
		byQ[g.Quarter] = append(byQ[g.Quarter], build(g, 0))
	}
	qs := make([]string, 0, len(byQ))
	for q := range byQ {
		qs = append(qs, q)
	}
	sort.Slice(qs, func(i, j int) bool { return qs[i] != "" && (qs[j] == "" || qs[i] > qs[j]) })
	out := []GoalQuarter{}
	for _, q := range qs {
		out = append(out, GoalQuarter{Quarter: q, Goals: byQ[q]})
	}
	return out
}

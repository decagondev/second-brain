package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// Starter questions (sb-spaces(7), Starter questions; SB-4906): a space's suggested questions
// (SB-3307), now up to MaxSuggested, with candidates the cell offers its stewards from the
// space's verified answers and from the asks its members found helpful. A candidate is only
// ever a question: no asker is named or counted by name, and a question just one person
// found helpful is never offered. A steward publishes one by saving the list; whoever asks
// it is answered fresh, as themselves.
const (
	// StarterPeople is how many people must have found a question's answer helpful.
	StarterPeople = 2
	// StarterRate is the share of its ratings that must be helpful, in quarters (3 of 4).
	StarterRate = 3
	// StarterWindow is how far back helpful asks count.
	StarterWindow = 90 * 24 * time.Hour
)

// StarterCandidate is one question the cell suggests a space's stewards publish: From is
// "verified" (a verified answer's question) or "helpful" (asks people found helpful), and
// People how many people found it helpful, never who.
type StarterCandidate struct {
	Question string `json:"question"`
	From     string `json:"from"`
	People   int    `json:"people,omitempty"`
}

// starterKey is a question as candidates compare it: lower case, one space between words.
func starterKey(q string) string { return strings.ToLower(strings.Join(strings.Fields(q), " ")) }

// StarterCandidates are space name's candidates for one of its stewards: questions not on
// the list already, verified first, then by how many found them helpful, at most
// MaxSuggested.
func (x Stewards) StarterCandidates(ctx context.Context, s tenancy.Scope, name string) ([]StarterCandidate, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	have := map[string]bool{}
	if x.Suggested != nil {
		got, _, err := x.Suggested.Suggested(ctx, x.Agent.Tenant, SpaceKey(name))
		if err != nil {
			return nil, fmt.Errorf("%w: suggested: %v", ErrUnavailable, err)
		}
		for _, q := range got.Questions {
			have[starterKey(q)] = true
		}
	}
	out := []StarterCandidate{}
	add := func(c StarterCandidate) {
		k := starterKey(c.Question)
		if _, err := cleanSuggested([]string{c.Question}); err != nil || have[k] || strings.HasPrefix(k, "/") {
			return // too long for a starter, already listed, or a skill's call
		}
		have[k] = true
		out = append(out, c)
	}
	if x.Agent.Pins != nil {
		p, err := x.pins(ctx, name)
		if err != nil {
			return nil, err
		}
		for _, pin := range p.Items {
			add(StarterCandidate{Question: strings.TrimSpace(pin.Question), From: "verified"})
		}
	}
	helpful, err := x.helpfulAsks(ctx, SpaceKey(name))
	if err != nil {
		return nil, err
	}
	for _, c := range helpful {
		add(c)
	}
	if len(out) > MaxSuggested {
		out = out[:MaxSuggested]
	}
	return out, nil
}

// helpfulAsks are the questions asked in space key in the last StarterWindow whose answers
// StarterPeople or more people rated, at least StarterRate in 4 of the ratings helpful, by
// how many found them helpful.
func (x Stewards) helpfulAsks(ctx context.Context, key string) ([]StarterCandidate, error) {
	fl, ok := x.Feedback.(FeedbackLister)
	if !ok {
		return nil, nil
	}
	now := x.Now()
	fbs, err := fl.FeedbackBetween(ctx, x.Agent.Tenant, now.Add(-StarterWindow), now)
	if err != nil {
		return nil, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	type tally struct {
		text     string
		up, down map[string]bool
	}
	byQ := map[string]*tally{}
	var order []string
	for _, f := range fbs {
		if f.Tenant != x.Agent.Tenant || (f.Rating != "up" && f.Rating != "down") {
			continue
		}
		q, ok, err := x.askedIn(ctx, f.Trace, key)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		k := starterKey(q)
		t := byQ[k]
		if t == nil {
			t = &tally{text: strings.Join(strings.Fields(q), " "), up: map[string]bool{}, down: map[string]bool{}}
			byQ[k] = t
			order = append(order, k)
		}
		if f.Rating == "up" {
			t.up[f.User] = true
		} else {
			t.down[f.User] = true
		}
	}
	out := []StarterCandidate{}
	for _, k := range order {
		t := byQ[k]
		up, all := len(t.up), len(t.up)+len(t.down)
		if up >= StarterPeople && up*4 >= all*StarterRate {
			out = append(out, StarterCandidate{Question: t.text, From: "helpful", People: up})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].People > out[j].People })
	return out, nil
}

// askedIn is the question trace id answered, when it was asked in space key.
func (x Stewards) askedIn(ctx context.Context, id, key string) (string, bool, error) {
	t, found, err := x.Agent.Store.GetTrace(ctx, x.Agent.Tenant, id)
	if err != nil {
		return "", false, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	if !found || t.Tenant != x.Agent.Tenant || t.Space != key || t.Response.Conversation == "" {
		return "", false, nil
	}
	qs, found, err := x.Agent.Store.Questions(ctx, x.Agent.Tenant, t.User, t.Response.Conversation)
	if err != nil {
		return "", false, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if !found || t.Turn >= len(qs) {
		return "", false, nil
	}
	return qs[t.Turn], true, nil
}

// SpaceStarter is one starter question of a space the person is a member of, for the MCP
// server: N is its place in the list, from 1.
type SpaceStarter struct {
	Space    string `json:"space"`
	N        int    `json:"n"`
	Question string `json:"question"`
}

// UsableStarters are the starter questions of every space s is a member of now, by space
// then place.
func (x Stewards) UsableStarters(ctx context.Context, s tenancy.Scope) ([]SpaceStarter, error) {
	out := []SpaceStarter{}
	if x.Agent.Auth.Spaces == nil || x.Suggested == nil {
		return out, nil
	}
	all, _, err := x.Agent.Auth.Spaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	names := []string{}
	for _, sp := range all {
		if sp.Tenant == x.Agent.Tenant && spaces.Member(sp, s.Groups()) {
			names = append(names, sp.Name)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		got, _, err := x.Suggested.Suggested(ctx, x.Agent.Tenant, SpaceKey(n))
		if err != nil {
			return nil, fmt.Errorf("%w: suggested: %v", ErrUnavailable, err)
		}
		for i, q := range got.Questions {
			out = append(out, SpaceStarter{Space: n, N: i + 1, Question: q})
		}
	}
	return out, nil
}

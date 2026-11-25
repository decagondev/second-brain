package ask

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/tenancy"
)

// Ask before guessing (sb-ask(3), Clarifying questions; SB-4901): before composing, an ask
// looks for three kinds of ambiguity, in order: a glossary term with two meanings in the
// spaces in scope, a company-wide question that names terms of several departments, and a
// question about an amount or a trend that names no period. A caller that can show a
// question (Request.Clarify) gets one, with up to MaxChoices choices and JustAnswer, and
// nothing is composed; the asker answers by asking again with Request.Choice. Any other
// caller (the chat apps, MCP, routines, scheduled and triggered agents, rhythm steps) never
// asks: it takes the first choice and says so at the top (Response.Assumed). Only the first
// question of a conversation is ever asked about, and a choice goes into the question the
// conversation keeps, never into memory (LAW-25).

// Clarifying question kinds, and the choice that answers as asked.
const (
	ClarifyTerm   = "term"
	ClarifySpace  = "space"
	ClarifyPeriod = "period"
	JustAnswer    = "just answer"
	// MaxChoices is the most choices a clarifying question offers, besides JustAnswer.
	MaxChoices = 4
	// ChoiceLead begins the line a choice adds to the question.
	ChoiceLead = "The asker means: "
	// AssumedLead begins the line an answer that couldn't ask starts with.
	AssumedLead = "Assumed: "
)

// Clarify is the one question an ambiguous ask gets instead of an answer.
type Clarify struct {
	Kind     string   `json:"kind"`
	Question string   `json:"question"`
	Choices  []string `json:"choices"` // up to MaxChoices, then JustAnswer
}

// PeriodChoices are the choices of a question with no period.
var PeriodChoices = []string{"This quarter", "This month", "This year", "All time"}

var (
	periodCue = regexp.MustCompile(`(?i)\b(how many|how much|total|totals|revenue|spend|spent|spending|costs?|sales|growth|grew|trend|trends|churn|headcount|hires|hired)\b`)
	periodSet = regexp.MustCompile(`(?i)\b((19|20)\d\d|q[1-4]|fy\d\d|today|yesterday|tonight|now|current|currently|latest|week|weeks|month|months|quarter|quarters|year|years|daily|weekly|monthly|quarterly|annual|annually|since|between|before|after|during|ytd|mtd|january|february|march|april|may|june|july|august|september|october|november|december|jan|feb|mar|apr|jun|jul|aug|sep|sept|oct|nov|dec|all time|ever)\b`)
)

// ambiguity is the first ambiguity of q as s asks it, nil for none.
func (a Agent) ambiguity(ctx context.Context, s tenancy.Scope, q string) *Clarify {
	if c := a.twoMeanings(ctx, s, q); c != nil {
		return c
	}
	if c := a.severalSpaces(ctx, s, q); c != nil {
		return c
	}
	if periodCue.MatchString(q) && !periodSet.MatchString(q) {
		return &Clarify{Kind: ClarifyPeriod, Question: "For which period?", Choices: append(append([]string{}, PeriodChoices...), JustAnswer)}
	}
	return nil
}

// glossariesInScope are the glossaries of the spaces s asks across, by space name: its own
// space and the teams of a branch ask, or for a company-wide ask the department spaces the
// asker is a member of.
func (a Agent) glossariesInScope(ctx context.Context, s tenancy.Scope) map[string]map[string]string {
	out := map[string]map[string]string{}
	if a.Glossary != nil && strings.HasPrefix(s.Space(), "spc_") {
		out[strings.TrimPrefix(s.Space(), "spc_")] = a.Glossary(strings.TrimPrefix(s.Space(), "spc_"))
		for _, b := range tenancy.BranchFrom(ctx, s) {
			if n := strings.TrimPrefix(b.Space(), "spc_"); n != b.Space() {
				out[n] = a.Glossary(n)
			}
		}
	}
	if s.Space() == "company" && a.Departmental != nil {
		for _, d := range a.Departmental(ctx, s) {
			out[d.Name] = d.Glossary
		}
	}
	return out
}

// twoMeanings is a term q uses that two spaces in scope define differently.
func (a Agent) twoMeanings(ctx context.Context, s tenancy.Scope, q string) *Clarify {
	gs := a.glossariesInScope(ctx, s)
	if len(gs) < 2 {
		return nil
	}
	names := keysOf(gs)
	meanings := map[string][]string{} // term, lower case -> "TERM as SPACE uses it: MEANS"
	seen := map[string]map[string]bool{}
	var order []string
	for _, n := range names {
		ok, _ := usable(gs[n])
		_, used := expand(q, ok)
		for _, t := range used {
			k := strings.ToLower(t.Term)
			if seen[k] == nil {
				seen[k] = map[string]bool{}
				order = append(order, k)
			}
			m := strings.ToLower(strings.Join(strings.Fields(t.Means), " "))
			if !seen[k][m] {
				seen[k][m] = true
				meanings[k] = append(meanings[k], fmt.Sprintf("%s as %s uses it: %s", t.Term, n, t.Means))
			}
		}
	}
	for _, k := range order {
		if ms := meanings[k]; len(ms) >= 2 {
			if len(ms) > MaxChoices {
				ms = ms[:MaxChoices]
			}
			return &Clarify{Kind: ClarifyTerm, Question: fmt.Sprintf("Which %q do you mean?", k), Choices: append(ms, JustAnswer)}
		}
	}
	return nil
}

// severalSpaces is a company-wide question that uses terms of two or more departments.
func (a Agent) severalSpaces(ctx context.Context, s tenancy.Scope, q string) *Clarify {
	if s.Space() != "company" || a.Departmental == nil {
		return nil
	}
	var hit []string
	for _, d := range a.Departmental(ctx, s) {
		ok, _ := usable(d.Glossary)
		if _, used := expand(q, ok); len(used) > 0 && !contains(hit, d.Name) {
			hit = append(hit, d.Name)
		}
	}
	if len(hit) < 2 {
		return nil
	}
	sort.Strings(hit)
	if len(hit) > MaxChoices {
		hit = hit[:MaxChoices]
	}
	var cs []string
	for _, h := range hit {
		cs = append(cs, "About "+h)
	}
	return &Clarify{Kind: ClarifySpace, Question: "Which department is this about?", Choices: append(cs, JustAnswer)}
}

// clarify decides an ask's ambiguity: ask is the question to put to the asker instead of
// answering; otherwise line is the asker's choice, or the assumption made for them, as the
// line the model reads after the question, and assumed is the line to start the answer
// with. A choice that answers
// no question this ask would ask is ErrInvalid: an ask is asked about once.
func (a Agent) clarify(ctx context.Context, s tenancy.Scope, q string, req Request, earlier int, probe bool) (line, assumed string, ask *Clarify, err error) {
	if probe || keepsNothing(ctx) {
		return "", "", nil, nil // an eval case, a consultation or a shadow run is answered as asked
	}
	choice := strings.TrimSpace(req.Choice)
	c := (*Clarify)(nil)
	if earlier == 0 {
		c = a.ambiguity(ctx, s, q)
	}
	switch {
	case choice != "" && (c == nil || !contains(c.Choices, choice)):
		return "", "", nil, fmt.Errorf("%w: that isn't a choice this question offers; ask again without one", ErrInvalid)
	case choice == JustAnswer || c == nil:
		return "", "", nil, nil
	case choice != "":
		return "\n\n" + ChoiceLead + choice, "", nil, nil
	case req.Clarify:
		return "", "", c, nil
	}
	return "\n\n" + ChoiceLead + c.Choices[0], AssumedLead + c.Choices[0], nil, nil
}

type choiceKey struct{}

// withChoice carries the asker's choice to the model, after the question (SB-4901).
func withChoice(ctx context.Context, line string) context.Context {
	return context.WithValue(ctx, choiceKey{}, line)
}

func choiceLine(ctx context.Context) string {
	l, _ := ctx.Value(choiceKey{}).(string)
	return l
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// clarified is clarify for ask: early is the response to give now, the clarifying question
// or the error, and nil when the ask goes on with q.
func (a Agent) clarified(ctx context.Context, s tenancy.Scope, q string, req Request, earlier int, probe bool, dept *Department) (string, string, *Response, error) {
	line, assumed, question, err := a.clarify(ctx, s, q, req, earlier, probe)
	if err != nil {
		return "", "", &Response{}, err
	}
	if question == nil {
		return line, assumed, nil, nil
	}
	r := Response{Release: a.Release, Conversation: req.Conversation, Agent: "ask", People: []PersonRef{}, Answer: Answer{Claims: []answer.Claim{}}, Clarify: question}
	if dept != nil {
		r.Agent = dept.Name
	}
	return "", "", &r, nil
}

package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Agents on a schedule and when something happens (sb-agents(7), Scheduled and triggered
// agents; SB-4005, SB-4006): an agent runs as one person, on the ask agent's path, and its
// answer waits in a steward queue as a draft. It never sends or publishes anything, and has
// no tools: a steward replies and publishes it as a note, or discards it.

// The kinds of agent draft.
const (
	DraftBrief     = "brief"
	DraftDecisions = "decisions"
	DraftUpdate    = "update"   // a weekly status update, published under notes/SPACE/updates/ (SB-4102)
	DraftIncident  = "incident" // an incident brief for the on-call stewards (SB-4204)
	DraftCheckin   = "checkin"  // a goal's weekly check-in, published under notes/SPACE/goals/ (SB-4207)
	// DraftLeaderWeek (leaderweek.go) is published under notes/SPACE/updates/ too (SB-4503).
)

// UpdateQuestion is the fixed question of a weekly status update (SB-4102).
const UpdateQuestion = "What changed in this space last week, what did it decide, and how do its goals stand?"

// BriefQuestion is the fixed question of a scheduled brief.
const BriefQuestion = "What happened in this space this week, and what is still open?"

// DraftText is an answer as a draft's text: one line a claim, with the titles it cites,
// at most MaxReply characters in all. An uncited claim is left out.
func DraftText(a Answer) string {
	var b strings.Builder
	for _, c := range a.Claims {
		if c.Uncited || len(c.Cites) == 0 {
			continue
		}
		var titles []string
		for _, ct := range c.Cites {
			t := ct.Title
			if t == "" {
				t = ct.Cite
			}
			titles = append(titles, t)
		}
		line := "- " + strings.Join(strings.Fields(c.Text), " ") + " [" + strings.Join(titles, "; ") + "]\n"
		if utf8.RuneCountInString(b.String()+line) > MaxReply {
			break
		}
		b.WriteString(line)
	}
	return strings.TrimRight(b.String(), "\n")
}

// AgentDraft runs req (its agent and question, and the documents it is narrowed to) as s,
// in s's space, and puts its answer in that space's steward queue as a draft of kind,
// audited first. An answer with nothing cited puts nothing, and is ErrInvalid.
func (x Stewards) AgentDraft(ctx context.Context, s tenancy.Scope, kind, title string, req Request) (StewardQuestion, error) {
	return x.AgentDraftFor(ctx, s, kind, title, req, "")
}

// AgentDraftFor is AgentDraft for one document (KIND:ID): a check-in's goal (SB-4207), or a
// review's incident ticket, whose action items it carries (SB-4604).
func (x Stewards) AgentDraftFor(ctx context.Context, s tenancy.Scope, kind, title string, req Request, goal string) (StewardQuestion, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant || s.Space() == "" || s.Space() == "company" {
		return StewardQuestion{}, fmt.Errorf("%w: an agent draft runs in a space", ErrInvalid)
	}
	if kind != DraftBrief && kind != DraftDecisions && kind != DraftUpdate && kind != DraftIncident && kind != DraftCheckin && kind != DraftLeaderWeek && kind != DraftReview {
		return StewardQuestion{}, fmt.Errorf("%w: no draft kind %q", ErrInvalid, kind)
	}
	agent := req.Agent
	if agent == "" {
		agent = "ask"
	}
	resp, text, err := x.draftAnswer(ctx, s, kind, req)
	if err != nil {
		return StewardQuestion{}, err
	}
	title = firstLine(title)
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: title, Space: s.Space(),
		Trace: resp.Trace, At: x.Now().UTC(), Thread: &SavedThread{Text: text, Agent: agent, Kind: kind,
			Asked: req.Question, AskDocs: req.Documents, Sources: draftSources(resp)}} // SB-4602: what a redraft asks again
	if kind == DraftDecisions && len(req.Documents) == 1 { // SB-4203: the meeting's action items, for this steward
		sq.Thread.Doc = req.Documents[0]
		sq.Thread.Supersede = supersedeSuggestions(resp.Answer.Claims, req.Documents[0]) // SB-4505
	}
	if kind == DraftReview && goal != "" { // SB-4604: the ticket's action items, for this steward
		sq.Thread.Doc = goal
	}
	if sq.Thread.Doc != "" && x.ActionsOf != nil {
		acts, err := x.ActionsOf(ctx, s, sq.Thread.Doc)
		if err != nil {
			return StewardQuestion{}, err
		}
		sq.Thread.Actions = acts
	}
	if kind == DraftCheckin {
		sq.Thread.Goal = goal
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "drafted",
		PayloadSHA256: hashOf(text), Reason: "agent " + agent + " drafted a " + kind, Space: auditSpace(s.Space())}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if rs := []rune(s); len(rs) > MaxQuestion {
		s = string(rs[:MaxQuestion-1]) + "…"
	}
	return s
}

// draftAnswer asks req as s for a draft of kind, and its text: a voc brief's themes counted
// (SB-4205), a leader's week's counts after its lines (SB-4503). Nothing cited is ErrInvalid.
func (x Stewards) draftAnswer(ctx context.Context, s tenancy.Scope, kind string, req Request) (Response, string, error) {
	agent := req.Agent
	if agent == "" {
		agent = "ask"
	}
	resp, err := x.Agent.Ask(ctx, s, req)
	if err != nil {
		return Response{}, "", err
	}
	if agent == AgentVoC { // SB-4205: at most 8 themes, 3 cites each, counted from their dates
		if resp.Answer.Claims, err = x.Agent.vocThemes(ctx, resp.Answer.Claims, x.Now()); err != nil {
			return Response{}, "", err
		}
	}
	text := DraftText(resp.Answer)
	if text == "" {
		return Response{}, "", fmt.Errorf("%w: nothing %s could cite", ErrInvalid, agent)
	}
	if kind == DraftLeaderWeek { // SB-4503: each team's counts after the cited lines
		counts, err := x.teamCounts(ctx, strings.TrimPrefix(s.Space(), "spc_"))
		if err != nil {
			return Response{}, "", err
		}
		text = withCounts(text, counts)
	}
	return resp, text, nil
}

// DecisionsQuestion is the fixed question of a new transcript's draft (SB-4006).
const DecisionsQuestion = "What did this meeting decide, and who owns each action?"

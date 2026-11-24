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
)

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
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant || s.Space() == "" || s.Space() == "company" {
		return StewardQuestion{}, fmt.Errorf("%w: an agent draft runs in a space", ErrInvalid)
	}
	if kind != DraftBrief && kind != DraftDecisions {
		return StewardQuestion{}, fmt.Errorf("%w: no draft kind %q", ErrInvalid, kind)
	}
	agent := req.Agent
	if agent == "" {
		agent = "ask"
	}
	resp, err := x.Agent.Ask(ctx, s, req)
	if err != nil {
		return StewardQuestion{}, err
	}
	text := DraftText(resp.Answer)
	if text == "" {
		return StewardQuestion{}, fmt.Errorf("%w: nothing %s could cite", ErrInvalid, agent)
	}
	title = firstLine(title)
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: title, Space: s.Space(),
		Trace: resp.Trace, At: x.Now().UTC(), Thread: &SavedThread{Text: text, Agent: agent, Kind: kind}}
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

// DecisionsQuestion is the fixed question of a new transcript's draft (SB-4006).
const DecisionsQuestion = "What did this meeting decide, and who owns each action?"

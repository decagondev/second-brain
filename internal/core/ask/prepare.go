package ask

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Meeting prep (sb-ask(3), Meeting prep; SB-4202): each item of an agenda, pasted or from
// the person's own calendar, asked by the meeting-prep agent as them, with the decisions
// behind it and what each attendee owns. The agenda is data, never instructions (LAW-09).

// The meeting prep limits.
const (
	MaxAgenda       = 4000
	MaxAgendaItems  = 8
	MaxAttendees    = 50
	PrepDecisions   = 90 * 24 * time.Hour
	MaxPrepDecision = 3
	EventsAhead     = 7 * 24 * time.Hour
)

// PrepRequest is the body of POST /v1/prepare.
type PrepRequest struct {
	Agenda    string   `json:"agenda,omitempty"`
	Title     string   `json:"title,omitempty"`
	Attendees []string `json:"attendees,omitempty"`
	Event     string   `json:"event,omitempty"`
}

// EventItem is one of a person's next events as GET /v1/me/events lists it.
type EventItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Start     time.Time `json:"start"`
	Attendees int       `json:"attendees"`
	Agenda    bool      `json:"agenda"`
}

var bulletRe = regexp.MustCompile(`^\s*(?:[-*•]+|\d+[.)])\s*`)

// agendaItems are an agenda's non-empty lines, bullets and numbers stripped, at most
// MaxAgendaItems, each cut to MaxStep.
func agendaItems(agenda string) []string {
	var out []string
	for _, line := range strings.Split(agenda, "\n") {
		it := cut(bulletRe.ReplaceAllString(line, ""), MaxStep)
		if it == "" {
			continue
		}
		out = append(out, it)
		if len(out) == MaxAgendaItems {
			break
		}
	}
	return out
}

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]{4,}`)

func words(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range wordRe.FindAllString(strings.ToLower(s), -1) {
		out[w] = true
	}
	return out
}

func shares(a map[string]bool, s string) bool {
	for w := range words(s) {
		if a[w] {
			return true
		}
	}
	return false
}

func (l Legwork) email(ctx context.Context, s tenancy.Scope) (string, error) {
	who, err := l.Agent.who(ctx, s)
	if err != nil {
		return "", err
	}
	if who.Email == "" {
		return "", fmt.Errorf("%w: no email for this person", ErrInvalid)
	}
	return who.Email, nil
}

// Events are the caller's own events of the next EventsAhead.
func (l Legwork) Events(ctx context.Context, s tenancy.Scope) ([]EventItem, error) {
	if err := s.Check(); err != nil || s.Tenant() != l.Agent.Tenant {
		return nil, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if l.Calendar == nil {
		return nil, fmt.Errorf("%w: this cell has no calendar", ErrUnavailable)
	}
	evs, err := l.events(ctx, s, l.now(), l.now().Add(EventsAhead))
	if err != nil {
		return nil, err
	}
	out := []EventItem{}
	for _, e := range evs {
		out = append(out, EventItem{ID: e.ID, Title: e.Title, Start: e.Start, Attendees: len(e.Attendees), Agenda: strings.TrimSpace(e.Agenda) != ""})
	}
	return out, nil
}

func (l Legwork) events(ctx context.Context, s tenancy.Scope, from, to time.Time) ([]ports.CalendarEvent, error) {
	email, err := l.email(ctx, s)
	if err != nil {
		return nil, err
	}
	evs, err := l.Calendar.Events(ctx, l.Agent.Tenant, email, from, to)
	if err != nil {
		return nil, fmt.Errorf("%w: calendar: %v", ErrUnavailable, err)
	}
	return evs, nil
}

// Prepare briefs a meeting for the caller, in their scope's space, and keeps the brief
// for them like a report.
func (l Legwork) Prepare(ctx context.Context, s tenancy.Scope, req PrepRequest) (ResearchView, error) {
	if err := l.check(s); err != nil {
		return ResearchView{}, err
	}
	if req.Event != "" {
		if l.Calendar == nil {
			return ResearchView{}, fmt.Errorf("%w: this cell has no calendar", ErrInvalid)
		}
		evs, err := l.events(ctx, s, l.now().Add(-24*time.Hour), l.now().Add(2*EventsAhead))
		if err != nil {
			return ResearchView{}, err
		}
		found := false
		for _, e := range evs {
			if e.ID == req.Event {
				req.Agenda, req.Title, req.Attendees, found = e.Agenda, e.Title, e.Attendees, true
			}
		}
		if !found {
			return ResearchView{}, fmt.Errorf("%w: no such event in your calendar", ErrInvalid)
		}
		if len(req.Attendees) > MaxAttendees {
			req.Attendees = req.Attendees[:MaxAttendees]
		}
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(req.Agenda)); n == 0 || n > MaxAgenda {
		return ResearchView{}, fmt.Errorf("%w: an agenda is 1 to %d characters", ErrInvalid, MaxAgenda)
	}
	if len(req.Attendees) > MaxAttendees {
		return ResearchView{}, fmt.Errorf("%w: at most %d attendees", ErrInvalid, MaxAttendees)
	}
	items := agendaItems(req.Agenda)
	if len(items) == 0 {
		return ResearchView{}, fmt.Errorf("%w: the agenda has no items", ErrInvalid)
	}
	title := cut(req.Title, MaxStep)
	if title == "" {
		title = items[0]
	}
	decisions, err := l.decisions(ctx, s)
	if err != nil {
		return ResearchView{}, err
	}
	open, steward, err := l.openQuestions(ctx, s)
	if err != nil {
		return ResearchView{}, err
	}
	r := Research{ID: l.NewID("rs_"), Tenant: l.Agent.Tenant, User: s.User(), Kind: KindPrep, Question: title, Space: s.Space(),
		Steps: items, State: StateDone, At: l.now()}
	owners := map[string][]answer.Cite{}
	for _, it := range items {
		resp, err := l.Agent.Ask(ctx, s, Request{Question: it, Agent: AgentPrep})
		if err != nil {
			return ResearchView{}, err
		}
		claims, _ := citedOnly(resp.Answer.Claims)
		sec := Section{Question: it, Trace: resp.Trace, Claims: claims}
		ws := words(it)
		for _, d := range decisions {
			if len(sec.Decisions) < MaxPrepDecision && shares(ws, d.Title) {
				sec.Decisions = append(sec.Decisions, d)
			}
		}
		if steward {
			n := 0
			for _, q := range open {
				if shares(ws, q) {
					n++
				}
			}
			sec.Questions = &n
		}
		for _, c := range claims {
			for _, ci := range c.Cites {
				if ci.Owner != nil && ci.Owner.Email != "" {
					owners[strings.ToLower(ci.Owner.Email)] = appendCite(owners[strings.ToLower(ci.Owner.Email)], ci)
				}
			}
		}
		r.Sections = append(r.Sections, sec)
	}
	seen := map[string]bool{}
	for _, e := range req.Attendees {
		e = strings.TrimSpace(e)
		if e == "" || seen[strings.ToLower(e)] {
			continue
		}
		seen[strings.ToLower(e)] = true
		a := Attendee{Email: cut(e, 254)}
		if l.Known != nil && l.Known(ctx, e) {
			a.Known, a.Owns = true, owners[strings.ToLower(e)]
		}
		r.Attendees = append(r.Attendees, a)
	}
	r.DoneAt = l.now()
	if err := l.audit(ctx, s.User(), r, StateDone, hashOf(req.Agenda)); err != nil {
		return ResearchView{}, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return ResearchView{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return view(r, r.Sections, 0), nil
}

func appendCite(cs []answer.Cite, c answer.Cite) []answer.Cite {
	for _, have := range cs {
		if have.Cite == c.Cite {
			return cs
		}
	}
	return append(cs, answer.Cite{Cite: c.Cite, Title: c.Title, URL: c.URL})
}

// decisions are the space's decisions of the last PrepDecisions that s can open.
func (l Legwork) decisions(ctx context.Context, s tenancy.Scope) ([]answer.Cite, error) {
	if !strings.HasPrefix(s.Space(), "spc_") {
		return nil, nil
	}
	res, err := l.Agent.spaceDecisions(ctx, s)
	if err != nil {
		return nil, err
	}
	var out []answer.Cite
	since := l.now().Add(-PrepDecisions)
	for _, c := range res.Context {
		if c.Seen.Before(since) {
			continue
		}
		t := c.Title
		if t == "" {
			t = c.Cite
		}
		ci := answer.Cite{Cite: c.Cite, Title: t}
		if l.Agent.Link != nil {
			ci.URL = l.Agent.Link(c)
		}
		out = append(out, ci)
	}
	return out, nil
}

// openQuestions are the space's open questions, for a steward of it only.
func (l Legwork) openQuestions(ctx context.Context, s tenancy.Scope) ([]string, bool, error) {
	name := strings.TrimPrefix(s.Space(), "spc_")
	if !strings.HasPrefix(s.Space(), "spc_") || l.Stewards.Facts == nil || l.Stewards.Store == nil {
		return nil, false, nil
	}
	if _, _, err := l.Stewards.steward(ctx, s, name); err != nil {
		return nil, false, nil //nolint:nilerr // not a steward: no counts
	}
	qs, err := l.Stewards.queue(ctx, name)
	if err != nil {
		return nil, false, err
	}
	var out []string
	for _, q := range qs {
		if q.Reply == nil && q.Thread == nil {
			out = append(out, q.Question)
		}
	}
	return out, true, nil
}

// PrepDue prepares, as user, each of their events that has an agenda and starts in
// [from, to); it returns the briefs made with each event's start.
func (l Legwork) PrepDue(ctx context.Context, user string, from, to time.Time, done func(event string) bool) ([]PrepMade, error) {
	if l.Calendar == nil || l.ScopeIn == nil {
		return nil, nil
	}
	s, err := l.ScopeIn(ctx, user, "company")
	if err != nil {
		return nil, nil //nolint:nilerr // not active: nothing to prepare
	}
	evs, err := l.events(ctx, s, from, to)
	if err != nil {
		return nil, err
	}
	var out []PrepMade
	for _, e := range evs {
		if strings.TrimSpace(e.Agenda) == "" || e.Start.Before(from) || !e.Start.Before(to) || done(e.ID) {
			continue
		}
		accounts, err := l.prepAccounts(ctx, user, e.Attendees) // SB-4605: before the meeting brief
		if err != nil {
			return out, err
		}
		v, err := l.Prepare(ctx, s, PrepRequest{Event: e.ID})
		if err != nil {
			return out, err
		}
		out = append(out, PrepMade{Event: e.ID, Brief: v.ID, Start: e.Start, Accounts: accounts})
	}
	return out, nil
}

// PrepMade is one brief the notify job made before a meeting.
type PrepMade struct {
	Event, Brief string
	Start        time.Time
	// Accounts are the account briefs made before it (SB-4605).
	Accounts []string
}

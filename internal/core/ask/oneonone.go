package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/tenancy"
)

// 1:1 prep (sb-agents(7), 1:1 prep; SB-4807): a manager starts the one-on-one rhythm for a
// direct report, and before each week's 1:1 both get the same brief, drawn only from what
// the report published: the goals they own with their status and waits, and the action
// items from meeting transcripts they own, each a line only when both of them can open its
// cite. Never their asks, inbox, memory or routines, and never a rating (LAW-30). The report
// can add a line, which both see.

// MaxBriefLine is the longest line a report adds to their brief.
const MaxBriefLine = 300

// isManager is nil when manager is user's manager in SCIM now.
func (x Rhythms) isManager(ctx context.Context, manager, user string) error {
	if x.Manager == nil {
		return fmt.Errorf("%w: this cell has no managers from SCIM", ErrInvalid)
	}
	m, err := x.Manager(ctx, user)
	if err != nil {
		return fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if m == "" || m != manager {
		return fmt.Errorf("%w: only a person's manager starts their 1:1 prep", ErrNotSteward)
	}
	return nil
}

// oneOnOne is the week's brief, as the manager s, from what the report published, with only
// the lines the report can open too, so both read exactly the same text.
func (x Rhythms) oneOnOne(ctx context.Context, s tenancy.Scope, r RhythmRun) (string, error) {
	if err := x.isManager(ctx, s.User(), r.Subject); err != nil {
		return "", err // the report has moved to another manager: no brief
	}
	rs, err := x.ScopeIn(ctx, r.Subject, r.Space)
	if err != nil {
		return "", fmt.Errorf("%w: the report isn't a member of the space", ErrInvalid)
	}
	emails, err := x.Stewards.Agent.emailsOf(ctx, r.Subject)
	if err != nil {
		return "", err
	}
	type line struct{ text, cite string }
	var lines []line
	gr, err := x.Stewards.Goals(ctx, s, r.Space)
	if err != nil {
		return "", err
	}
	for _, g := range flatGoals(gr.Quarters) {
		if !contains(emails, strings.ToLower(g.Owner)) {
			continue
		}
		doc, _, _ := strings.Cut(g.Cite, "@")
		t := fmt.Sprintf("Goal %s: %s", g.Title, orDash(g.Status))
		if g.Waiting != nil {
			t += fmt.Sprintf(", waiting on %s until %s", g.Waiting.On, g.Waiting.Until)
		}
		lines = append(lines, line{t, doc})
	}
	if x.Legwork.Actions != nil {
		as, err := x.Legwork.Actions.ActionsOf(ctx, r.Tenant, r.Subject)
		if err != nil {
			return "", fmt.Errorf("%w: action items: %v", ErrUnavailable, err)
		}
		for _, a := range as {
			if a.DoneAt != nil || a.Space != r.Space {
				continue
			}
			t := "Open action item: " + a.Text
			if a.Due != "" {
				t += ", due " + a.Due
			}
			lines = append(lines, line{t, a.Doc})
		}
	}
	var cites []string
	for _, l := range lines {
		cites = append(cites, l.cite+"@0")
	}
	byManager, err := x.Stewards.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return "", err
	}
	byReport, err := x.Stewards.Agent.opensCites(ctx, rs, cites)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, l := range lines {
		if byManager[l.cite+"@0"] && byReport[l.cite+"@0"] {
			fmt.Fprintf(&b, "- %s [%s]\n", l.text, l.cite)
		}
	}
	if b.Len() == 0 {
		return "Nothing they published to bring up this week.", nil
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// AddLine adds the report's own line to the latest brief of their one-on-one rhythm, in
// both copies: the manager's and theirs.
func (x Rhythms) AddLine(ctx context.Context, s tenancy.Scope, id, text string) (RhythmView, error) {
	r, err := x.get(ctx, id)
	if err != nil {
		return RhythmView{}, err
	}
	if s.Check() != nil || r.Rhythm != OneOnOne || s.User() != r.Subject {
		return RhythmView{}, fmt.Errorf("%w: only the report adds a line to their 1:1 brief", ErrInvalid)
	}
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > MaxBriefLine || strings.ContainsAny(text, "\n\r") {
		return RhythmView{}, fmt.Errorf("%w: a line is 1 to %d characters on one line", ErrInvalid, MaxBriefLine)
	}
	last := -1
	for i, st := range r.Steps {
		if st.State == StepOffered && st.Copy != "" {
			last = i
		}
	}
	if last < 0 {
		return RhythmView{}, fmt.Errorf("%w: no brief waits for this 1:1", ErrInvalid)
	}
	st := r.Steps[last]
	for _, it := range []struct{ queue, id string }{{st.Queue, st.Item}, {PersonKey(r.Subject), st.Copy}} {
		if q, err := x.Stewards.question(ctx, it.queue, it.id); err != nil || q.Thread == nil {
			continue
		}
		if _, err := x.Stewards.Store.UpdateStewardQuestion(ctx, x.Stewards.Agent.Tenant, it.id, func(q *StewardQuestion) error {
			if q.Thread != nil {
				q.Thread.Text = strings.TrimRight(q.Thread.Text, "\n") + "\n- From them: " + text
			}
			return nil
		}); err != nil {
			return RhythmView{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
	}
	if err := x.audit(ctx, s.User(), r, "rhythm_line_added", fmt.Sprintf("day %d", st.Day)); err != nil {
		return RhythmView{}, err
	}
	return x.view(r, s.User()), nil
}

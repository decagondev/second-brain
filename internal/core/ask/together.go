package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Edit a draft together (sb-agents(7), Edit a draft together; SB-5202): the draft's reviewer
// (the steward it ran as) edits a section in place, and each change Ask for changes makes is
// a suggestion, kept in the text until they accept or reject it. A rejected suggestion puts
// back what was there; a draft with an open suggestion can't be published.

// DraftSuggestion states.
const (
	SuggestionOpen     = "open"
	SuggestionAccepted = "accepted"
	SuggestionRejected = "rejected"
)

// MaxSectionEdit is the longest section a reviewer may write in place.
const MaxSectionEdit = 4000

// DraftSuggestion is one change a redraft made: section Section (from 1, or 0 for the whole
// draft), what was there and what the agent wrote, in round Round.
type DraftSuggestion struct {
	N       int    `json:"n"`
	Section int    `json:"section"`
	Old     string `json:"old"`
	New     string `json:"new"`
	Round   int    `json:"round"`
	State   string `json:"state"`
}

// openSuggestions counts t's open suggestions.
func openSuggestions(t *SavedThread) int {
	n := 0
	if t != nil {
		for _, s := range t.Suggestions {
			if s.State == SuggestionOpen {
				n++
			}
		}
	}
	return n
}

// suggest adds a redraft's changes to t as open suggestions: one per redrafted section, or
// one for the whole draft. An earlier round's open suggestion is accepted by the redraft
// that built on it.
func suggest(t *SavedThread, old, next string, sections map[int][2]string) {
	for i := range t.Suggestions {
		if t.Suggestions[i].State == SuggestionOpen {
			t.Suggestions[i].State = SuggestionAccepted
		}
	}
	add := func(section int, o, n string) {
		if o == n {
			return
		}
		t.Suggestions = append(t.Suggestions, DraftSuggestion{N: len(t.Suggestions) + 1, Section: section, Old: o, New: n, Round: t.Rounds, State: SuggestionOpen})
	}
	if len(sections) == 0 {
		add(0, old, next)
		return
	}
	for n := 1; n <= len(DraftSections(old)); n++ {
		if s, ok := sections[n]; ok {
			add(n, s[0], s[1])
		}
	}
}

// reviewable is q as an agent draft s reviews: open, and theirs.
func reviewable(q StewardQuestion, user string) error {
	t := q.Thread
	switch {
	case t == nil || t.Agent == "" || t.Discarded || q.Reply != nil:
		return fmt.Errorf("%w: only an open agent draft can be changed", ErrInvalid)
	case q.Asker != user:
		return fmt.Errorf("%w: only the steward the draft ran as can change it", ErrInvalid)
	}
	return nil
}

// Decide accepts or rejects suggestion n of space name's draft id. Rejecting puts back what
// the section held; accepting keeps the agent's text.
func (x Stewards) Decide(ctx context.Context, s tenancy.Scope, name, id string, n int, accept bool) (*SavedThread, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	q, err := x.question(ctx, name, id)
	if err != nil {
		return nil, fmt.Errorf("%w: no such draft in this space", ErrInvalid)
	}
	if err := reviewable(q, s.User()); err != nil {
		return nil, err
	}
	state := SuggestionRejected
	if accept {
		state = SuggestionAccepted
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "suggestion_" + state,
		Reason: fmt.Sprintf("suggestion %d of a %s", n, q.Thread.Kind), Space: auditSpace(q.Space)}); err != nil {
		return nil, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	out, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(cur *StewardQuestion) error {
		if err := reviewable(*cur, s.User()); err != nil {
			return err
		}
		t := cur.Thread
		if n < 1 || n > len(t.Suggestions) || t.Suggestions[n-1].State != SuggestionOpen {
			return fmt.Errorf("%w: no open suggestion %d", ErrInvalid, n)
		}
		sg := &t.Suggestions[n-1]
		if !accept {
			switch {
			case sg.Section == 0:
				t.Text = sg.Old
			case strings.Contains(t.Text, sg.New):
				t.Text = strings.Replace(t.Text, sg.New, sg.Old, 1)
			default:
				return fmt.Errorf("%w: that section was edited since; edit it back by hand", ErrInvalid)
			}
			if t.Generated != "" {
				t.Generated = strings.Replace(t.Generated, sg.New, sg.Old, 1)
			}
		}
		sg.State = state
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out.Thread, nil
}

// EditSection writes section (from 1) of space name's draft id as the reviewer wrote it. The
// draft is then marked edited when published (LAW-28), and the reviewer is one of its editors.
func (x Stewards) EditSection(ctx context.Context, s tenancy.Scope, name, id string, section int, text string) (*SavedThread, error) {
	text = strings.TrimRight(text, " \n")
	if strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > MaxSectionEdit || !printableText(strings.ReplaceAll(text, "\n", " ")) {
		return nil, fmt.Errorf("%w: a section is 1 to %d characters", ErrInvalid, MaxSectionEdit)
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	q, err := x.question(ctx, name, id)
	if err != nil {
		return nil, fmt.Errorf("%w: no such draft in this space", ErrInvalid)
	}
	if err := reviewable(q, s.User()); err != nil {
		return nil, err
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "edited",
		PayloadSHA256: hashOf(text), Reason: fmt.Sprintf("section %d of a %s", section, q.Thread.Kind), Space: auditSpace(q.Space)}); err != nil {
		return nil, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	out, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(cur *StewardQuestion) error {
		if err := reviewable(*cur, s.User()); err != nil {
			return err
		}
		t := cur.Thread
		secs := DraftSections(t.Text)
		if section < 1 || section > len(secs) {
			return fmt.Errorf("%w: the draft has sections 1 to %d", ErrInvalid, len(secs))
		}
		if t.Generated == "" {
			t.Generated = t.Text // what the agent wrote, for the edited mark
		}
		old := secs[section-1]
		secs[section-1] = text + old[len(strings.TrimRight(old, " \n")):]
		t.Text = strings.Join(secs, "")
		if !contains(t.Editors, s.User()) {
			t.Editors = append(t.Editors, s.User())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out.Thread, nil
}

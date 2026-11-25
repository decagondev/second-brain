package ask

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Action items from meetings (sb-ask(3), Action items; SB-4203): read from a transcript's
// text without a model when its decisions draft is made, seen only by the steward the draft
// ran as, and told to their owners only once that steward confirms them.

// The action item limits.
const (
	MaxActions    = 20
	MaxActionText = 300
	ActionsDone   = 30 * 24 * time.Hour // done items stay on their owner's list this long
)

// DraftAction is an action item as the transcript says it.
type DraftAction struct {
	Text  string `json:"text"`
	Owner string `json:"owner"`
	Email string `json:"email,omitempty"`
	Due   string `json:"due,omitempty"` // YYYY-MM-DD
}

// Action is a confirmed action item.
type Action struct {
	ID     string     `json:"id"`
	Tenant string     `json:"tenant"`
	Space  string     `json:"space"` // the space's name
	Doc    string     `json:"doc"`   // the transcript, KIND:ID
	Text   string     `json:"text"`
	Owner  string     `json:"owner"`
	User   string     `json:"user,omitempty"`
	Email  string     `json:"email,omitempty"`
	Due    string     `json:"due,omitempty"`
	By     string     `json:"by"`
	At     time.Time  `json:"at"`
	DoneAt *time.Time `json:"done_at,omitempty"`
	// Waiting is an item waiting on another space (SB-4504).
	Waiting *Wait `json:"waiting,omitempty"`
}

// ActionStore keeps confirmed action items.
type ActionStore interface {
	PutAction(ctx context.Context, a Action) error
	GetAction(ctx context.Context, tenant, id string) (Action, bool, error)
	ActionsOf(ctx context.Context, tenant, user string) ([]Action, error)
	ActionsIn(ctx context.Context, tenant, space string) ([]Action, error)
	// ActionsSince are the items confirmed at or after since, for the notify job.
	ActionsSince(ctx context.Context, tenant string, since time.Time) ([]Action, error)
}

var (
	actionLineRe = regexp.MustCompile(`(?i)^\s*(?:[-*•]\s*)?(?:action item|action|todo|ai)\s*[:\-–]\s*(.+)$`)
	actionAtRe   = regexp.MustCompile(`^@([\w.\-]+)`)
	actionWhoRe  = regexp.MustCompile(`^((?:[A-Za-z][\w.'\-]*\s+){0,2}[A-Za-z][\w.'\-]*)\s+(?:to|will|should)\s+`)
	actionDueRe  = regexp.MustCompile(`(?i)\b(?:by|due)\s+(\d{4}-\d{2}-\d{2})\b`)
)

// ExtractActions reads action items from a transcript's text: lines that start Action:,
// Action item:, TODO: or AI:, at most MaxActions, each cut to MaxActionText.
func ExtractActions(text string) []DraftAction {
	var out []DraftAction
	for _, line := range strings.Split(text, "\n") {
		m := actionLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		body := cut(m[1], MaxActionText)
		if body == "" {
			continue
		}
		a := DraftAction{Text: body}
		if w := actionAtRe.FindStringSubmatch(body); w != nil {
			a.Owner = w[1]
		} else if w := actionWhoRe.FindStringSubmatch(body); w != nil {
			a.Owner = w[1]
		}
		if d := actionDueRe.FindStringSubmatch(body); d != nil {
			if _, err := time.Parse("2006-01-02", d[1]); err == nil {
				a.Due = d[1]
			}
		}
		out = append(out, a)
		if len(out) == MaxActions {
			break
		}
	}
	return out
}

// normName is a name or address part as an owner matches it: lower case, no spaces.
func normName(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), "")) }

// match finds the one active user whose email before the @, or userName, is owner.
func match(users []ports.User, owner string) (ports.User, bool) {
	want := normName(owner)
	if want == "" {
		return ports.User{}, false
	}
	var found []ports.User
	for _, u := range users {
		if !u.Active {
			continue
		}
		hit := normName(u.UserName) == want
		for _, e := range u.Emails {
			local, _, _ := strings.Cut(e, "@")
			hit = hit || normName(local) == want || normName(e) == want
		}
		if hit {
			found = append(found, u)
		}
	}
	if len(found) != 1 {
		return ports.User{}, false
	}
	return found[0], true
}

func (l Legwork) users(ctx context.Context) ([]ports.User, error) {
	if l.Users == nil {
		return nil, nil
	}
	us, err := l.Users(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	return us, nil
}

// DraftActions are the action items of a transcript (doc, KIND:ID) as s, the steward its
// decisions draft runs as, can read them: its newest version, every chunk checked as s,
// and a chunk flagged instruction_like never read (LAW-09).
func (l Legwork) DraftActions(ctx context.Context, s tenancy.Scope, doc string) ([]DraftAction, error) {
	lk, ok := l.Agent.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	kind, id, ok := strings.Cut(doc, ":")
	if !ok {
		return nil, nil
	}
	cs, err := lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	cs = latest(cs)
	if len(cs) == 0 {
		return nil, nil
	}
	who, err := l.Agent.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, l.Agent.Auth, cs, len(cs))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	sort.SliceStable(res.Context, func(i, j int) bool { return res.Context[i].Start < res.Context[j].Start })
	var b strings.Builder
	for _, c := range res.Context {
		if contains(c.Flags, "instruction_like") {
			continue
		}
		b.WriteString(c.Text)
		b.WriteString("\n")
	}
	acts := ExtractActions(b.String())
	us, err := l.users(ctx)
	if err != nil {
		return nil, err
	}
	for i := range acts {
		if u, ok := match(us, acts[i].Owner); ok && len(u.Emails) > 0 {
			acts[i].Email = u.Emails[0]
		}
	}
	return acts, nil
}

// Confirm is the steward a space's decisions draft ran as confirming its action items,
// edited as they choose: each becomes one record, its owner matched again, audited.
func (l Legwork) Confirm(ctx context.Context, s tenancy.Scope, name, draft string, items []DraftAction) ([]Action, error) {
	if _, _, err := l.Stewards.steward(ctx, s, name); err != nil {
		return nil, err
	}
	if l.Actions == nil {
		return nil, fmt.Errorf("%w: this cell keeps no action items", ErrUnavailable)
	}
	if len(items) == 0 || len(items) > MaxActions {
		return nil, fmt.Errorf("%w: 1 to %d action items", ErrInvalid, MaxActions)
	}
	q, found, err := l.Stewards.Store.GetStewardQuestion(ctx, l.Agent.Tenant, draft)
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if !found || q.Space != SpaceKey(name) || q.Thread == nil || (q.Thread.Kind != DraftDecisions && q.Thread.Kind != DraftReview) || q.Thread.Discarded || q.Reply != nil || q.Asker != s.User() {
		return nil, ErrNotSteward // only the steward it ran as sees its text
	}
	for _, it := range items {
		t := strings.TrimSpace(it.Text)
		if t == "" || len([]rune(t)) > MaxActionText || len([]rune(it.Owner)) > 120 {
			return nil, fmt.Errorf("%w: an action item is 1 to %d characters", ErrInvalid, MaxActionText)
		}
		if it.Due != "" {
			if _, err := time.Parse("2006-01-02", it.Due); err != nil {
				return nil, fmt.Errorf("%w: due is YYYY-MM-DD", ErrInvalid)
			}
		}
	}
	us, err := l.users(ctx)
	if err != nil {
		return nil, err
	}
	var out []Action
	for _, it := range items {
		a := Action{ID: l.NewID("ac_"), Tenant: l.Agent.Tenant, Space: name, Doc: q.Thread.Doc, Text: cut(it.Text, MaxActionText),
			Owner: cut(it.Owner, 120), Due: it.Due, By: s.User(), At: l.now()}
		if u, ok := match(us, it.Owner); ok && len(u.Emails) > 0 {
			a.User, a.Email = u.ID, u.Emails[0]
		}
		r := Research{ID: a.ID, Kind: "action", Space: SpaceKey(name)}
		if err := l.audit(ctx, s.User(), r, "action_confirmed", hashOf(a.Text)); err != nil {
			return nil, err
		}
		if err := l.Actions.PutAction(ctx, a); err != nil {
			return nil, fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
		}
		out = append(out, a)
	}
	return out, nil
}

// ActionView is an action item as a list shows it.
type ActionView struct {
	Action
	Overdue bool `json:"overdue,omitempty"`
}

func (l Legwork) viewAction(a Action) ActionView {
	v := ActionView{Action: a}
	if a.DoneAt == nil && a.Due != "" {
		if d, err := time.Parse("2006-01-02", a.Due); err == nil && l.now().After(d.Add(24*time.Hour)) {
			v.Overdue = true
		}
	}
	return v
}

// Mine are the caller's own action items: open ones, and those done in ActionsDone.
func (l Legwork) Mine(ctx context.Context, s tenancy.Scope) ([]ActionView, error) {
	if err := s.Check(); err != nil || s.Tenant() != l.Agent.Tenant {
		return nil, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if l.Actions == nil {
		return []ActionView{}, nil
	}
	as, err := l.Actions.ActionsOf(ctx, l.Agent.Tenant, s.User())
	if err != nil {
		return nil, fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
	}
	out := []ActionView{}
	for _, a := range as {
		if a.DoneAt != nil && l.now().Sub(*a.DoneAt) > ActionsDone {
			continue
		}
		out = append(out, l.viewAction(a))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, nil
}

// Done is an item's owner marking it done.
func (l Legwork) Done(ctx context.Context, s tenancy.Scope, id string) (ActionView, error) {
	if err := s.Check(); err != nil || s.Tenant() != l.Agent.Tenant || l.Actions == nil {
		return ActionView{}, ErrNotFound
	}
	a, found, err := l.Actions.GetAction(ctx, l.Agent.Tenant, id)
	if err != nil {
		return ActionView{}, fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
	}
	if !found || a.User == "" || a.User != s.User() {
		return ActionView{}, ErrNotFound
	}
	if a.DoneAt == nil {
		now := l.now()
		a.DoneAt = &now
		if err := l.audit(ctx, s.User(), Research{ID: a.ID, Kind: "action", Space: SpaceKey(a.Space)}, "action_done", hashOf(a.Text)); err != nil {
			return ActionView{}, err
		}
		if err := l.Actions.PutAction(ctx, a); err != nil {
			return ActionView{}, fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
		}
	}
	return l.viewAction(a), nil
}

// SpaceActions are a space's open action items whose transcript the caller, a member, can
// open now (LAW-01), and how many others.
func (l Legwork) SpaceActions(ctx context.Context, s tenancy.Scope, name string) ([]ActionView, int, error) {
	if err := l.Stewards.member(ctx, s, name); err != nil {
		return nil, 0, err
	}
	if l.Actions == nil {
		return []ActionView{}, 0, nil
	}
	as, err := l.Actions.ActionsIn(ctx, l.Agent.Tenant, name)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
	}
	var cites []string
	newest := map[string]string{}
	for _, a := range as {
		if a.DoneAt == nil {
			if _, ok := newest[a.Doc]; !ok {
				newest[a.Doc] = a.Doc + "@0"
				cites = append(cites, newest[a.Doc])
			}
		}
	}
	open, err := l.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return nil, 0, err
	}
	out, hidden := []ActionView{}, 0
	for _, a := range as {
		if a.DoneAt != nil {
			continue
		}
		if !open[newest[a.Doc]] {
			hidden++
			continue
		}
		out = append(out, l.viewAction(a))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, hidden, nil
}

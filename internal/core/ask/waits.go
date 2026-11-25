package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Waiting on another department (sb-ask(3), Work across departments; SB-4504): a request,
// an action item or a goal says which other space it waits on, why and until when. The
// wait is a field on the record (a goal's front matter for a goal), never a store of its
// own. Both spaces' homes list it, each item named only to someone who can see it, and
// one a working day past its date goes once to the nearest common parent's stewards.

// The kinds of item that wait, and the limits on a wait.
const (
	WaitRequest   = "request"
	WaitAction    = "action"
	WaitGoal      = "goal"
	MaxWaitReason = 200
	MaxWaitDays   = 90
)

// Wait is an item waiting on another space: the space by name, the reason, the day it is
// expected (YYYY-MM-DD), and who marked it when.
type Wait struct {
	On     string    `json:"on"`
	Reason string    `json:"reason"`
	Until  string    `json:"until"`
	By     string    `json:"by,omitempty"`
	At     time.Time `json:"at,omitzero"`
}

// WaitMark is PUT /v1/spaces/NAME/waits.
type WaitMark struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	On     string `json:"on"`
	Reason string `json:"reason"`
	Until  string `json:"until"`
}

// WaitItem is one wait as a reader of a space's home sees it.
type WaitItem struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Space    string `json:"space"`
	On       string `json:"on"`
	Reason   string `json:"reason"`
	Until    string `json:"until"`
	Overdue  bool   `json:"overdue,omitempty"`
	CanClear bool   `json:"can_clear,omitempty"`
}

// WaitsResponse is GET /v1/spaces/NAME/waits: what the space waits on, what waits on it,
// and how many of either the reader can't see.
type WaitsResponse struct {
	Space    string     `json:"space"`
	Waiting  []WaitItem `json:"waiting"`
	WaitedOn []WaitItem `json:"waited_on"`
	Hidden   int        `json:"hidden"`
}

// OverdueWait is a wait a working day past its date, for the notify job: the stewards of
// To are told, once per Key.
type OverdueWait struct {
	Key, Kind, From, On, Until string
	To                         []string // space names
}

// waitRec is one waiting item as the cell holds it.
type waitRec struct {
	kind, id, space string // space by name
	wait            Wait
	asker           string // a request's sender
	owner           string // an action item's owner (a user id)
	category        string
	text            string // a request's text or an action item's
	goal            ports.Chunk
}

// parseUntil is a wait's date, at noon in zone.
func parseUntil(day string, zone *time.Location) (time.Time, bool) {
	if zone == nil {
		zone = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(day), zone)
	if err != nil {
		return time.Time{}, false
	}
	return t.Add(12 * time.Hour), true
}

// waitOverdue reports whether a wait until day is past the end of the next working day.
func waitOverdue(day string, now time.Time, zone *time.Location) bool {
	t, ok := parseUntil(day, zone)
	return ok && now.After(AddWorkingDays(t, 1, zone))
}

// goalWait reads a goal's waiting_on, waiting_reason and waiting_until; nil for none.
func goalWait(fm map[string]string) *Wait {
	on := strings.ToLower(strings.TrimSpace(fm["waiting_on"]))
	if on == "" {
		return nil
	}
	w := &Wait{On: on, Reason: cut(firstLine(fm["waiting_reason"]), MaxWaitReason)}
	if _, ok := parseUntil(fm["waiting_until"], nil); ok {
		w.Until = strings.TrimSpace(fm["waiting_until"])
	}
	return w
}

// spaceNames are the tenant's spaces by name.
func (x Stewards) spaceNames(ctx context.Context) ([]string, error) {
	if x.Agent.Auth.Spaces == nil {
		return nil, nil
	}
	all, _, err := x.Agent.Auth.Spaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	var out []string
	for _, sp := range all {
		if sp.Tenant == x.Agent.Tenant {
			out = append(out, sp.Name)
		}
	}
	sort.Strings(out)
	return out, nil
}

// waitsIn are the open items of space name that wait on another space: its requests not
// yet answered, its action items not done, and its goals not done.
func (x Stewards) waitsIn(ctx context.Context, name string) ([]waitRec, error) {
	var out []waitRec
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	for _, q := range qs {
		if q.Request != nil && q.Waiting != nil && q.Reply == nil && q.Space == SpaceKey(name) {
			out = append(out, waitRec{kind: WaitRequest, id: q.ID, space: name, wait: *q.Waiting, asker: q.Asker, category: q.Request.Category, text: q.Question})
		}
	}
	if x.Actions != nil {
		as, err := x.Actions.ActionsIn(ctx, x.Agent.Tenant, name)
		if err != nil {
			return nil, fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
		}
		for _, a := range as {
			if a.Waiting != nil && a.DoneAt == nil {
				out = append(out, waitRec{kind: WaitAction, id: a.ID, space: name, wait: *a.Waiting, owner: a.User, text: a.Text})
			}
		}
	}
	if lk, ok := x.Agent.Index.(ports.DocLookup); ok {
		chunks, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Space: SpaceKey(name)})
		if err != nil {
			return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
		first := firsts(latest(chunks))
		keys := make([]string, 0, len(first))
		for k := range first {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			c := first[k]
			g, ok := ParseGoal(c.Text)
			if !ok || c.Start != 0 || g.Waiting == nil || g.Status == "done" || g.Waiting.On == name {
				continue
			}
			out = append(out, waitRec{kind: WaitGoal, id: k, space: name, wait: *g.Waiting, goal: c})
		}
	}
	return out, nil
}

// SetWait marks a request in space name's queue, or an action item of it, as waiting on
// another space: a request by its sender or a steward of the space, an action item by its
// owner or a steward of the space. A later mark replaces the earlier one.
func (x Stewards) SetWait(ctx context.Context, s tenancy.Scope, name string, m WaitMark) (Wait, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return Wait{}, ErrNotSteward
	}
	on := strings.ToLower(strings.TrimSpace(m.On))
	reason := strings.TrimSpace(m.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > MaxWaitReason || strings.ContainsAny(reason, "\r\n") {
		return Wait{}, fmt.Errorf("%w: a reason is one line of 1 to %d characters", ErrInvalid, MaxWaitReason)
	}
	now := x.Now().UTC()
	day, ok := parseUntil(m.Until, x.Zone)
	today := now.In(zoneOr(x.Zone)).Format("2006-01-02")
	if !ok || strings.TrimSpace(m.Until) < today || day.After(now.Add(MaxWaitDays*24*time.Hour)) {
		return Wait{}, fmt.Errorf("%w: until is a date from today to %d days ahead", ErrInvalid, MaxWaitDays)
	}
	names, err := x.spaceNames(ctx)
	if err != nil {
		return Wait{}, err
	}
	if on == name || on == "company" || !contains(names, on) {
		return Wait{}, fmt.Errorf("%w: a wait is on another space of the tenant", ErrInvalid)
	}
	w := Wait{On: on, Reason: reason, Until: strings.TrimSpace(m.Until), By: s.User(), At: now}
	if err := x.markWait(ctx, s, name, m.Kind, m.ID, &w); err != nil {
		return Wait{}, err
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: m.ID, State: "waiting",
		PayloadSHA256: hashOf(reason), Reason: "waiting on " + on, Space: name}); err != nil {
		return Wait{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return w, nil
}

// ClearWait ends a request's or action item's wait: anyone who may mark it, or a steward of
// the space it waits on.
func (x Stewards) ClearWait(ctx context.Context, s tenancy.Scope, name, kind, id string) error {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return ErrNotSteward
	}
	if err := x.markWait(ctx, s, name, kind, id, nil); err != nil {
		return err
	}
	_, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "wait_cleared", Reason: "wait cleared", Space: name})
	if err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// markWait sets (w) or clears (nil) the wait of one item, after checking who may.
func (x Stewards) markWait(ctx context.Context, s tenancy.Scope, name, kind, id string, w *Wait) error {
	steward := func(sp string) bool { _, _, err := x.steward(ctx, s, sp); return err == nil }
	switch kind {
	case WaitRequest:
		q, found, err := x.Store.GetStewardQuestion(ctx, x.Agent.Tenant, id)
		if err != nil {
			return fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		if !found || q.Space != SpaceKey(name) || q.Request == nil {
			return ErrNotSteward
		}
		may := q.Asker == s.User() || steward(name) || w == nil && q.Waiting != nil && steward(q.Waiting.On)
		if !may {
			return ErrNotSteward
		}
		if q.Reply != nil && w != nil {
			return fmt.Errorf("%w: an answered request waits on nobody", ErrInvalid)
		}
		if _, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(q *StewardQuestion) error { q.Waiting = w; return nil }); err != nil {
			return fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		return nil
	case WaitAction:
		if x.Actions == nil {
			return ErrNotSteward
		}
		a, found, err := x.Actions.GetAction(ctx, x.Agent.Tenant, id)
		if err != nil {
			return fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
		}
		if !found || a.Space != name {
			return ErrNotSteward
		}
		may := a.User != "" && a.User == s.User() || steward(name) || w == nil && a.Waiting != nil && steward(a.Waiting.On)
		if !may {
			return ErrNotSteward
		}
		if a.DoneAt != nil && w != nil {
			return fmt.Errorf("%w: a done action item waits on nobody", ErrInvalid)
		}
		a.Waiting = w
		if err := x.Actions.PutAction(ctx, a); err != nil {
			return fmt.Errorf("%w: actions: %v", ErrUnavailable, err)
		}
		return nil
	case WaitGoal:
		return fmt.Errorf("%w: a goal waits through its front matter; edit it at the source", ErrInvalid)
	default:
		return fmt.Errorf("%w: kind is request or action", ErrInvalid)
	}
}

// Waits are what space name waits on and what waits on it, for a member: each item named
// only to someone who can see it now, a request's category and an action item's reason
// also to a steward of the space it waits on, and the rest counted.
func (x Stewards) Waits(ctx context.Context, s tenancy.Scope, name string) (WaitsResponse, error) {
	if err := x.member(ctx, s, name); err != nil {
		return WaitsResponse{}, err
	}
	out := WaitsResponse{Space: name, Waiting: []WaitItem{}, WaitedOn: []WaitItem{}}
	names, err := x.spaceNames(ctx)
	if err != nil {
		return out, err
	}
	var recs []waitRec
	for _, sp := range names {
		rs, err := x.waitsIn(ctx, sp)
		if err != nil {
			return out, err
		}
		for _, r := range rs {
			if sp == name || r.wait.On == name {
				recs = append(recs, r)
			}
		}
	}
	open, err := x.goalsOpen(ctx, s, recs)
	if err != nil {
		return out, err
	}
	stewardOf := map[string]bool{}
	steward := func(sp string) bool {
		if v, ok := stewardOf[sp]; ok {
			return v
		}
		_, _, err := x.steward(ctx, s, sp)
		stewardOf[sp] = err == nil
		return stewardOf[sp]
	}
	now := x.Now().UTC()
	for _, r := range recs {
		it := WaitItem{Kind: r.kind, ID: r.id, Space: r.space, On: r.wait.On, Reason: r.wait.Reason, Until: r.wait.Until, Overdue: waitOverdue(r.wait.Until, now, x.Zone)}
		switch {
		case r.kind == WaitGoal && open[r.id] != "":
			it.Title = open[r.id]
		case r.kind == WaitRequest && (r.asker == s.User() || steward(r.space)):
			it.Title, it.CanClear = r.text, true
		case r.kind == WaitAction && (r.owner != "" && r.owner == s.User() || steward(r.space)):
			it.Title, it.CanClear = r.text, true
		case r.kind == WaitRequest && steward(r.wait.On):
			it.Title, it.CanClear = "Request: "+r.category, true
		case r.kind == WaitAction && steward(r.wait.On):
			it.Title, it.CanClear = "Action item", true
		default:
			out.Hidden++ // counted, never named (LAW-01)
			continue
		}
		if r.space == name {
			out.Waiting = append(out.Waiting, it)
		} else {
			out.WaitedOn = append(out.WaitedOn, it)
		}
	}
	return out, nil
}

// goalsOpen are the titles of the waiting goals s can open now, by KIND:ID.
func (x Stewards) goalsOpen(ctx context.Context, s tenancy.Scope, recs []waitRec) (map[string]string, error) {
	out := map[string]string{}
	var cand []ports.Chunk
	for _, r := range recs {
		if r.kind == WaitGoal {
			cand = append(cand, r.goal)
		}
	}
	if len(cand) == 0 {
		return out, nil
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	for _, c := range res.Context {
		out[c.Kind+":"+c.SourceID] = x.Agent.result(ctx, nil, c).Title
	}
	return out, nil
}

// waitingCount is how many of a team's items wait on another space, for its row in a
// roll-up: a count only (LAW-24).
func (x Stewards) waitingCount(ctx context.Context, team string) (int, error) {
	rs, err := x.waitsIn(ctx, team)
	return len(rs), err
}

// ancestors are the spaces above name, nearest first.
func (x Stewards) ancestors(name string) []string {
	var out []string
	if x.Parent == nil {
		return out
	}
	for p := x.Parent(name); p != "" && len(out) < 3 && !contains(out, p); p = x.Parent(p) {
		out = append(out, p)
	}
	return out
}

// commonParent is the lowest space above both a and b, "" for none.
func (x Stewards) commonParent(a, b string) string {
	above := x.ancestors(b)
	for _, p := range x.ancestors(a) {
		if contains(above, p) {
			return p
		}
	}
	return ""
}

// OverdueWaits are the tenant's waits a working day past their date, each with the spaces
// whose stewards are told: the nearest common parent, or both spaces with none.
func (x Stewards) OverdueWaits(ctx context.Context) ([]OverdueWait, error) {
	names, err := x.spaceNames(ctx)
	if err != nil {
		return nil, err
	}
	now := x.Now().UTC()
	var out []OverdueWait
	for _, sp := range names {
		rs, err := x.waitsIn(ctx, sp)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			if r.wait.Until == "" || !waitOverdue(r.wait.Until, now, x.Zone) {
				continue
			}
			o := OverdueWait{Key: r.kind + "-" + strings.NewReplacer(":", "-", "/", "-").Replace(r.id) + "-" + r.wait.Until,
				Kind: r.kind, From: r.space, On: r.wait.On, Until: r.wait.Until}
			if p := x.commonParent(r.space, r.wait.On); p != "" {
				o.To = []string{p}
			} else {
				o.To = []string{r.space, r.wait.On}
			}
			out = append(out, o)
		}
	}
	return out, nil
}

func zoneOr(z *time.Location) *time.Location {
	if z == nil {
		return time.UTC
	}
	return z
}

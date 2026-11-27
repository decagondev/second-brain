package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// One inbox (sb-ask(3), One inbox; SB-4301): everything that waits on a person across every
// space, built when it is asked for, as them, from the calls each item's own tab makes, so
// it shows exactly what those show them now (LAW-26). It lists; it never acts.

// The inbox's kinds and window.
const (
	InboxDraft    = "draft"
	InboxQuestion = "question"
	InboxRequest  = "request"
	InboxPolicy   = "policy"
	InboxApproval = "approval"
	InboxAction   = "action"
	InboxFix      = "fix"
	InboxResearch = "research"
	InboxRoutine  = "routine"
	InboxWide     = "wide" // an owner's document shared too widely (SB-5401)
	InboxDone     = 30 * 24 * time.Hour
)

// InboxItem is one thing that waits on a person, and where it is acted on.
type InboxItem struct {
	Kind    string    `json:"kind"`
	ID      string    `json:"id"`
	Space   string    `json:"space,omitempty"`
	Title   string    `json:"title"`
	At      time.Time `json:"at,omitzero"`
	Due     string    `json:"due,omitempty"`
	Overdue bool      `json:"overdue,omitempty"`
	Why     string    `json:"why,omitempty"`
	Link    string    `json:"link"`
}

// InboxCounts are the lengths of the inbox's lists.
type InboxCounts struct {
	NeedsYou int `json:"needs_you"`
	Failed   int `json:"failed"`
	Done     int `json:"done"`
}

// Inbox is a person's inbox now.
type Inbox struct {
	NeedsYou []InboxItem `json:"needs_you"`
	Failed   []InboxItem `json:"failed"`
	Done     []InboxItem `json:"done"`
	Counts   InboxCounts `json:"counts"`
	// CatchUp offers Catch me up once after a notification pause ends (SB-4506).
	CatchUp *CatchUpOffer `json:"catch_up,omitempty"`
}

func queueLink(name string) string {
	if IsPerson(name) {
		return "/?tab=questions"
	}
	return "/?steward=" + name
}

func spaceName(key string) string { return strings.TrimPrefix(key, "spc_") }

// Inbox is what waits on s now, in every space, with the offer to catch up once after a
// notification pause (SB-4506).
func (l Legwork) Inbox(ctx context.Context, s tenancy.Scope) (Inbox, error) {
	return l.inbox(ctx, s, true)
}

func (l Legwork) inbox(ctx context.Context, s tenancy.Scope, offer bool) (Inbox, error) {
	if err := l.check(s); err != nil {
		return Inbox{}, err
	}
	in := Inbox{NeedsYou: []InboxItem{}, Failed: []InboxItem{}, Done: []InboxItem{}}
	now := l.now()
	var names []string
	if l.Agent.Auth.Spaces != nil {
		all, _, err := l.Agent.Auth.Spaces(ctx)
		if err != nil {
			return Inbox{}, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
		}
		for _, sp := range all {
			if sp.Tenant == l.Agent.Tenant {
				names = append(names, sp.Name)
			}
		}
	}
	sort.Strings(names)
	if err := l.inboxQueues(ctx, s, append(append([]string{"company"}, names...), PersonKey(s.User())), now, &in); err != nil {
		return Inbox{}, err
	}
	if err := l.inboxPolicies(ctx, s, names, &in); err != nil {
		return Inbox{}, err
	}
	if l.Approvals != nil {
		as, err := l.Approvals(ctx, s)
		if err != nil {
			return Inbox{}, fmt.Errorf("%w: approvals: %v", ErrUnavailable, err)
		}
		in.NeedsYou = append(in.NeedsYou, as...)
	}
	if err := l.inboxActions(ctx, s, now, &in); err != nil {
		return Inbox{}, err
	}
	if err := l.inboxResults(ctx, s, now, &in); err != nil {
		return Inbox{}, err
	}
	if err := l.inboxCases(ctx, s, &in); err != nil { // SB-4401
		return Inbox{}, err
	}
	wide, err := l.Agent.OwnWide(ctx, s) // SB-5401
	if err != nil {
		return Inbox{}, err
	}
	in.NeedsYou = append(in.NeedsYou, wide...)
	in.NeedsYou = append(in.NeedsYou, l.Agent.ownProposals(ctx, s)...) // SB-5505
	newest := func(xs []InboxItem) {
		sort.SliceStable(xs, func(i, j int) bool {
			if xs[i].Overdue != xs[j].Overdue {
				return xs[i].Overdue
			}
			return xs[i].At.After(xs[j].At)
		})
	}
	newest(in.NeedsYou)
	newest(in.Failed)
	newest(in.Done)
	in.Counts = InboxCounts{NeedsYou: len(in.NeedsYou), Failed: len(in.Failed), Done: len(in.Done)}
	if offer && l.CatchUpOffer != nil {
		since, ok, err := l.CatchUpOffer(ctx, s.User())
		if err != nil {
			return Inbox{}, fmt.Errorf("%w: settings: %v", ErrUnavailable, err)
		}
		if ok {
			in.CatchUp = &CatchUpOffer{Since: since.In(zoneOr(l.Zone)).Format("2006-01-02")}
		}
	}
	return in, nil
}

// inboxQueues are the open items of every queue s answers now, not claimed by anyone else,
// and those they replied to in InboxDone.
func (l Legwork) inboxQueues(ctx context.Context, s tenancy.Scope, names []string, now time.Time, in *Inbox) error {
	for _, name := range names {
		items, err := l.Stewards.Queue(ctx, s, name)
		if errors.Is(err, ErrNotSteward) {
			continue // not theirs now
		}
		if err != nil {
			return err
		}
		space := name
		if IsPerson(name) {
			space = ""
		}
		for _, it := range items {
			if it.Thread != nil && it.Thread.Hidden {
				continue // another steward's agent draft: theirs to publish
			}
			kind := InboxQuestion
			switch {
			case it.Thread != nil || it.Draft != nil:
				kind = InboxDraft
			case it.Request != nil:
				kind = InboxRequest
			}
			x := InboxItem{Kind: kind, ID: it.ID, Space: space, Title: it.Question, At: it.At, Overdue: it.Overdue, Link: queueLink(name)}
			if it.Request != nil {
				x.Due = it.Request.Due.UTC().Format(time.DateOnly)
			}
			switch {
			case it.State == "open" && (it.Claim == nil || it.Claim.Mine):
				in.NeedsYou = append(in.NeedsYou, x)
			case it.Reply != nil && it.Reply.By == s.User() && now.Sub(it.Reply.At) <= InboxDone:
				x.At, x.Overdue = it.Reply.At, false
				in.Done = append(in.Done, x)
			}
		}
	}
	return nil
}

// inboxPolicies are the policies s hasn't read at their current version, in the spaces they
// are a member of, each once.
func (l Legwork) inboxPolicies(ctx context.Context, s tenancy.Scope, names []string, in *Inbox) error {
	if l.Stewards.Policies == nil {
		return nil
	}
	seen := map[string]bool{}
	for _, name := range names {
		ps, err := l.Stewards.PoliciesFor(ctx, s, name)
		if errors.Is(err, ErrNotMember) {
			continue
		}
		if err != nil {
			return err
		}
		for _, p := range ps {
			if p.Read || seen[p.Space+"\x00"+p.Doc] {
				continue
			}
			seen[p.Space+"\x00"+p.Doc] = true
			title := p.Title
			if title == "" {
				title = p.Doc
			}
			in.NeedsYou = append(in.NeedsYou, InboxItem{Kind: InboxPolicy, ID: p.Doc, Space: name, Title: title, Link: "/?space=" + name})
		}
	}
	return nil
}

// inboxActions are s's own action items whose transcript they can open now.
func (l Legwork) inboxActions(ctx context.Context, s tenancy.Scope, now time.Time, in *Inbox) error {
	as, err := l.Mine(ctx, s)
	if err != nil {
		return err
	}
	var cites []string
	for _, a := range as {
		cites = append(cites, a.Doc+"@0")
	}
	open, err := l.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return err
	}
	for _, a := range as {
		if !open[a.Doc+"@0"] {
			continue // a transcript they lost: the item goes with it
		}
		x := InboxItem{Kind: InboxAction, ID: a.ID, Space: a.Space, Title: a.Text, At: a.At, Due: a.Due, Overdue: a.Overdue, Link: "/?tab=legwork"}
		if a.DoneAt != nil {
			if now.Sub(*a.DoneAt) <= InboxDone {
				x.At, x.Overdue = *a.DoneAt, false
				in.Done = append(in.Done, x)
			}
			continue
		}
		in.NeedsYou = append(in.NeedsYou, x)
	}
	return nil
}

// inboxResults are s's reports, briefs, routines' answers and suggested edits: a plan to
// run and a suggestion to read need them, a failed run is listed with why, and what
// finished in InboxDone is done.
func (l Legwork) inboxResults(ctx context.Context, s tenancy.Scope, now time.Time, in *Inbox) error {
	if l.Research == nil {
		return nil
	}
	rs, err := l.Research.ResearchOf(ctx, l.Agent.Tenant, s.User(), now.Add(-InboxDone))
	if err != nil {
		return fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	var cites []string
	for _, r := range rs {
		if r.Kind == KindFix && len(r.Documents) == 1 {
			cites = append(cites, r.Documents[0])
		}
	}
	open, err := l.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return err
	}
	for _, r := range rs {
		x := InboxItem{ID: r.ID, Space: spaceName(r.Space), Title: r.Question, At: r.At, Link: "/?tab=legwork"}
		if r.Space == "company" {
			x.Space = ""
		}
		state := l.fixState(ctx, r)
		switch {
		case r.Kind == KindFix:
			if state != FixSuggested || len(r.Documents) != 1 || !open[r.Documents[0]] {
				continue // applied, or a document they can't open now
			}
			x.Kind = InboxFix
			in.NeedsYou = append(in.NeedsYou, x)
		case state == StateFailed:
			x.Kind, x.Why = InboxResearch, r.Why
			if r.Kind == KindRoutine {
				x.Kind = InboxRoutine
			}
			in.Failed = append(in.Failed, x)
		case r.Kind == KindResearch && state == StatePlanned:
			x.Kind = InboxResearch
			in.NeedsYou = append(in.NeedsYou, x)
		case state == StateDone:
			x.Kind = InboxResearch
			if r.Kind == KindRoutine {
				x.Kind = InboxRoutine
			}
			if !r.DoneAt.IsZero() {
				x.At = r.DoneAt
			}
			in.Done = append(in.Done, x)
		}
	}
	return nil
}

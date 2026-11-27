package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/tenancy"
)

// Where you left off (sb-ask(3), Where you left off; SB-5805): a new conversation on the
// web, and the person's first direct message to the brain in a day, start with up to
// MaxLeftOff of their own open things from the last LeftOffDays days: a question they asked
// that waits on a steward, a research plan waiting for their approval, an action they
// confirmed and haven't done, and an overdue wait they set. Everything comes from the stores
// those already live in, read as them now, so an item they can no longer open isn't shown;
// nothing new is kept, and Not now is the browser's.

// The limits.
const (
	MaxLeftOff  = 3
	LeftOffDays = 14
)

// The kinds of open thing, beside the inbox's question, research and action.
const LeftOffWait = "wait"

// LeftOff is GET /v1/me/left-off.
type LeftOff struct {
	Items []InboxItem `json:"items"`
}

// LeftOff is s's own open things, overdue first, then newest.
func (l Legwork) LeftOff(ctx context.Context, s tenancy.Scope) (LeftOff, error) {
	if err := l.check(s); err != nil {
		return LeftOff{}, err
	}
	now := l.now()
	from := now.AddDate(0, 0, -LeftOffDays)
	var items []InboxItem
	asked, err := l.Stewards.Mine(ctx, s) // questions waiting on a steward
	if err != nil {
		return LeftOff{}, err
	}
	for _, q := range asked {
		if q.State == "open" && !q.At.Before(from) {
			items = append(items, InboxItem{Kind: InboxQuestion, ID: q.ID, Space: spaceName(q.Space), Title: q.Question, At: q.At, Link: "/?tab=questions"})
		}
	}
	if l.Research != nil { // research plans waiting for their approval
		rs, err := l.Research.ResearchOf(ctx, l.Agent.Tenant, s.User(), from)
		if err != nil {
			return LeftOff{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
		}
		for _, r := range rs {
			if r.Kind == KindResearch && (r.State == StatePlanned || r.State == StateCheckpoint) {
				items = append(items, InboxItem{Kind: InboxResearch, ID: r.ID, Space: spaceName(r.Space), Title: r.Question, At: r.At, Link: "/?tab=legwork"})
			}
		}
	}
	var in Inbox // actions they confirmed and haven't done, whose transcript they can open now
	if err := l.inboxActions(ctx, s, now, &in); err != nil {
		return LeftOff{}, err
	}
	for _, x := range in.NeedsYou {
		if !x.At.Before(from) {
			items = append(items, x)
		}
	}
	waits, err := l.Stewards.ownOverdueWaits(ctx, s)
	if err != nil {
		return LeftOff{}, err
	}
	items = append(items, waits...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Overdue != items[j].Overdue {
			return items[i].Overdue
		}
		return items[i].At.After(items[j].At)
	})
	for i := range items {
		if items[i].Space == "company" {
			items[i].Space = ""
		}
	}
	if len(items) > MaxLeftOff {
		items = items[:MaxLeftOff]
	}
	if items == nil {
		items = []InboxItem{}
	}
	return LeftOff{Items: items}, nil
}

// ownOverdueWaits are the overdue waits s set, in the spaces they are a member of now.
func (x Stewards) ownOverdueWaits(ctx context.Context, s tenancy.Scope) ([]InboxItem, error) {
	names, err := x.spaceNames(ctx)
	if err != nil {
		return nil, err
	}
	now := x.Now().UTC()
	var out []InboxItem
	for _, sp := range names {
		if err := x.member(ctx, s, sp); errors.Is(err, ErrNotMember) {
			continue
		} else if err != nil {
			return nil, err
		}
		rs, err := x.waitsIn(ctx, sp)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			if r.wait.By != s.User() || r.wait.Until == "" || !waitOverdue(r.wait.Until, now, x.Zone) {
				continue
			}
			out = append(out, InboxItem{Kind: LeftOffWait, ID: r.id, Space: sp, Title: "Waiting on " + r.wait.On + ": " + r.wait.Reason,
				Due: r.wait.Until, Overdue: true, At: r.wait.At, Link: "/?space=" + sp})
		}
	}
	return out, nil
}

// LeftOffLines are the open things as plain lines for a chat app's first reply of the day.
func LeftOffLines(lo LeftOff, base string) []string {
	var out []string
	for _, it := range lo.Items {
		out = append(out, "Where you left off: "+it.Title+" ("+base+it.Link+")")
	}
	return out
}

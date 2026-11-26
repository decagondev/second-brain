package wire

import (
	"context"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// workEvents are the pull requests and tickets first seen in [from, to) of the spaces whose
// on lists them (SB-5301): one per space and document.
func (c *Cell) workEvents(ctx context.Context, from, to time.Time) ([]notify.NewDoc, error) {
	lk, ok := c.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	m := c.Manifest()
	chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{SeenSince: from})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []notify.NewDoc
	for _, ch := range chunks {
		if ch.Start != 0 || !ch.Seen.Before(to) {
			continue
		}
		ref := ch.Kind + ":" + ch.SourceID
		for _, sp := range ch.Spaces {
			s, ok := m.Spaces[strings.TrimPrefix(sp, "spc_")]
			if !ok {
				continue
			}
			for _, on := range []string{ask.OnPROpened, ask.OnTicketFiled} {
				if !s.Watches(on) || !ask.IsWorkEvent(on, ch) || seen[sp+"|"+ref] {
					continue
				}
				seen[sp+"|"+ref] = true
				title := ch.Title
				if title == "" {
					title = ch.SourceID
				}
				out = append(out, notify.NewDoc{Space: sp, Ref: ref, Title: title, On: on})
			}
		}
	}
	return out, nil
}

// runWork drafts one work event, or a batch of one space and kind, as the space's first
// active steward, the trigger's approver (SB-5301).
func (c *Cell) runWork(ctx context.Context, docs []notify.NewDoc) (string, error) {
	if len(docs) == 0 {
		return "", nil
	}
	d := docs[0]
	if _, ok := c.Manifest().Spaces[strings.TrimPrefix(d.Space, "spc_")]; !ok {
		return "", nil
	}
	var titles []string
	for _, x := range docs {
		titles = append(titles, x.Title)
	}
	req := ask.Request{Question: ask.WorkQuestion(d.On, titles)}
	title := "New: " + d.Title
	doc := d.Ref
	if len(docs) == 1 {
		req.Documents = []string{d.Ref}
	} else {
		title, doc = "New: "+d.On+" ("+strconv.Itoa(len(docs))+")", ""
	}
	return c.runDraft(ctx, d.Space, ask.DraftWork, title, req, false, doc)
}

// tryTrigger replays trigger on of space name over the last TryDays as s (SB-5302).
func (c *Cell) tryTrigger(ctx context.Context, s tenancy.Scope, name, on string) (ask.TryResult, error) {
	lk, ok := c.Index.(ports.DocLookup)
	if !ok {
		return ask.TryResult{Drafts: []ask.TryDraft{}}, nil
	}
	chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{SeenSince: time.Now().AddDate(0, 0, -ask.TryDays)})
	if err != nil {
		return ask.TryResult{}, err
	}
	ctx, tally := budget.WithTally(ctx)
	return c.Parts().Stewards.Try(ctx, s, name, on, chunks, tally.USD)
}

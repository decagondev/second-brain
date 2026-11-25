package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// When a source changes (sb-agents(7), When a source changes; SB-4905): in a space whose
// manifest lists `on: [source-changed]`, a new version of a document an agent draft of the
// last SourceWindow cited, or was narrowed to, redrafts it for review: once a day at most,
// as the person who approved it, from the same sources, with the change shown as a diff.
// Nothing is republished: the redraft is a draft in the queue like any other.
const (
	SourceWindow = 90 * 24 * time.Hour
	// SourceChanged is the trigger's name in a space's `on`.
	SourceChanged = "source-changed"
	// MaxSourceRedrafts is the most redrafts one run makes; the rest wait for the next day.
	MaxSourceRedrafts = 20
)

// watched are the agent drafts of spaces (by key) from the last SourceWindow that watch their
// sources: not discarded, and not handed on to a later redraft.
func (x Stewards) watched(ctx context.Context, spaces []string, now time.Time) ([]StewardQuestion, error) {
	qs, err := x.Store.StewardQuestionsBetween(ctx, x.Agent.Tenant, now.Add(-SourceWindow), now.Add(time.Second))
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	var out []StewardQuestion
	for _, q := range qs {
		t := q.Thread
		if t == nil || t.Agent == "" || t.Discarded || t.Next != "" || !contains(spaces, q.Space) {
			continue
		}
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, nil
}

// draftDocs are the documents a draft drew on: those it was narrowed to and those it cited.
func draftDocs(t *SavedThread) []string {
	out := append([]string{}, t.AskDocs...)
	for _, d := range t.Sources {
		if !contains(out, d) {
			out = append(out, d)
		}
	}
	return out
}

// NoteSourceChanges marks changed, the documents (KIND:ID) each with when its newest version
// was first seen, as pending on every watched draft of spaces (by key) that drew on one and
// was made before that version; it returns how many drafts it marked.
func (x Stewards) NoteSourceChanges(ctx context.Context, spaces []string, changed map[string]time.Time) (int, error) {
	if len(spaces) == 0 || len(changed) == 0 {
		return 0, nil
	}
	qs, err := x.watched(ctx, spaces, x.Now())
	if err != nil {
		return 0, err
	}
	n := 0
	for _, q := range qs {
		var hit []string
		for _, d := range draftDocs(q.Thread) {
			if seen, ok := changed[d]; ok && seen.After(q.At) && !contains(q.Thread.Pending, d) {
				hit = append(hit, d)
			}
		}
		if len(hit) == 0 {
			continue
		}
		if _, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, q.ID, func(sq *StewardQuestion) error {
			for _, d := range hit {
				if !contains(sq.Thread.Pending, d) {
					sq.Thread.Pending = append(sq.Thread.Pending, d)
				}
			}
			return nil
		}); err != nil {
			return n, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		n++
	}
	return n, nil
}

// approver is who a draft answers to now: the steward who published it, or the one it ran as.
func approver(q StewardQuestion) string {
	if q.Reply != nil && q.Reply.By != "" {
		return q.Reply.By
	}
	return q.Asker
}

// SourceRun is what one run of redrafts did, as counts.
type SourceRun struct {
	Redrafted, Dropped, Waiting int
}

// RedraftChanged redrafts every watched draft of spaces with pending changes that wasn't
// redrafted today (UTC), at most MaxSourceRedrafts; the rest wait for a later run. A draft
// whose approver can't ask in its space any more, or can't open a changed source now, is
// dropped for them: its pending changes are cleared and nothing is made.
func (x Stewards) RedraftChanged(ctx context.Context, spaces []string) (SourceRun, error) {
	var run SourceRun
	if len(spaces) == 0 || x.Rescope == nil {
		return run, nil
	}
	now := x.Now().UTC()
	qs, err := x.watched(ctx, spaces, now)
	if err != nil {
		return run, err
	}
	for _, q := range qs {
		t := q.Thread
		if len(t.Pending) == 0 {
			continue
		}
		got, err := x.redraftChanged(ctx, q, now, sameDay(t.Redrafted, now) || run.Redrafted == MaxSourceRedrafts)
		if err != nil {
			return run, err
		}
		switch got {
		case sourceMade:
			run.Redrafted++
		case sourceDropped:
			run.Dropped++
		default:
			run.Waiting++
		}
	}
	return run, nil
}

func sameDay(a, b time.Time) bool {
	return !a.IsZero() && a.UTC().Format(time.DateOnly) == b.UTC().Format(time.DateOnly)
}

// What redraftChanged did with one draft.
const (
	sourceMade = iota
	sourceDropped
	sourceWaits
)

// redraftChanged redrafts q as its approver for its pending changes, unless wait (it was
// redrafted today, or the run made its most). An approver who lost the space or a changed
// source drops it at once, waiting or not.
func (x Stewards) redraftChanged(ctx context.Context, q StewardQuestion, now time.Time, wait bool) (int, error) {
	t := q.Thread
	who := approver(q)
	drop := func(why string) (int, error) {
		if _, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, q.ID, func(sq *StewardQuestion) error {
			sq.Thread.Pending = nil
			return nil
		}); err != nil {
			return 0, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		if _, err := x.Audit.Write(ctx, audit.Event{Actor: who, Kind: "access", Subject: q.ID, State: "source_dropped",
			Reason: "agent " + t.Agent + ": " + why, Space: auditSpace(q.Space)}); err != nil {
			return 0, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
		}
		return sourceDropped, nil
	}
	rctx, s, err := x.Rescope(ctx, who, q.Space, t.Kind == DraftLeaderWeek)
	if err != nil {
		return drop("its approver can't ask in the space now")
	}
	if _, _, err := x.steward(ctx, s, strings.TrimPrefix(q.Space, "spc_")); err != nil {
		return drop("its approver isn't a steward of the space now")
	}
	for _, d := range t.Pending {
		c, found, err := x.current(ctx, d)
		if err != nil {
			return 0, err
		}
		ok := false
		if found {
			if ok, err = x.opens(ctx, s, c); err != nil {
				return 0, err
			}
		}
		if !ok {
			return drop("its approver can't open a changed source now")
		}
	}
	if wait {
		return sourceWaits, nil
	}
	if err := x.Agent.runsStop(ctx, s, t.Agent); err != nil {
		return sourceWaits, nil //nolint:nilerr // paused, killed or capped: the changes wait for a later day
	}
	req := Request{Question: t.Asked, Documents: t.AskDocs}
	if t.Agent != "ask" {
		req.Agent = t.Agent
	}
	if len(req.Documents) == 0 {
		rctx = withOnly(rctx, t.Sources) // the same sources, never more (LAW-14)
	}
	resp, text, err := x.draftAnswer(rctx, s, t.Kind, req)
	switch {
	case errors.Is(err, ErrInvalid): // nothing it could cite now
		return drop("nothing could be cited from its sources now")
	case err != nil:
		return 0, err
	}
	id := x.NewID("qs_")
	sq := StewardQuestion{ID: id, Tenant: x.Agent.Tenant, Asker: s.User(), Question: firstLine("Source changed: " + q.Question), Space: q.Space,
		Trace: resp.Trace, At: now, Thread: &SavedThread{Text: text, Agent: t.Agent, Kind: t.Kind, Release: resp.Release, Asked: t.Asked,
			AskDocs: t.AskDocs, Sources: draftSources(resp), Versions: []string{t.Text}, Doc: t.Doc, Goal: t.Goal, Changed: t.Pending, Redrafted: now}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "source_redrafted",
		PayloadSHA256: hashOf(text), Reason: fmt.Sprintf("agent %s redrafted a %s for %d changed sources", t.Agent, t.Kind, len(t.Pending)),
		Space: auditSpace(q.Space)}); err != nil {
		return 0, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return 0, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if _, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, q.ID, func(o *StewardQuestion) error {
		o.Thread.Pending, o.Thread.Redrafted, o.Thread.Next = nil, now, id
		return nil
	}); err != nil {
		return 0, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sourceMade, nil
}

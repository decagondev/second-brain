package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Comments on a section (sb-agents(7), Comments on a section; SB-4902): the reviewer of an
// agent draft in a steward queue (the steward it ran as) comments on its sections; Ask for
// changes then writes only the commented sections again, from the same sources, and leaves
// every other section as it was, byte for byte. A comment is the
// reviewer's request about the draft, never a source (LAW-09), shown to the reviewer only,
// and kept with the draft; its hash is in the audit log.

// The comment limits.
const (
	MaxSectionComment = 300
	MaxRoundComments  = 20
)

// SectionComment is one comment on section Section (from 1) of a draft, in round Round.
type SectionComment struct {
	Section int    `json:"section"`
	Text    string `json:"text"`
	By      string `json:"by"`
	Round   int    `json:"round"`
}

// SectionDiff is the line diff of one section a redraft wrote again.
type SectionDiff struct {
	Section int      `json:"section"`
	Diff    []string `json:"diff"`
}

// DraftSections splits a draft into its sections, so that joining them gives the draft back:
// a section starts at a heading (#), at a list item (- or *), or after a blank line, and
// holds the lines that follow until the next one starts; blank lines stay with the section
// above them.
func DraftSections(text string) []string {
	if text == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	blank := false
	for _, ln := range strings.SplitAfter(text, "\n") {
		if ln == "" {
			continue
		}
		body := strings.TrimSpace(ln)
		starts := strings.HasPrefix(body, "#") || strings.HasPrefix(body, "- ") || strings.HasPrefix(body, "* ") || (blank && body != "")
		if starts && cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
		cur.WriteString(ln)
		blank = body == ""
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// roundComments are the comments of the draft's current round.
func roundComments(t *SavedThread) []SectionComment {
	if t == nil {
		return nil
	}
	var out []SectionComment
	for _, c := range t.Comments {
		if c.Round == t.Rounds {
			out = append(out, c)
		}
	}
	return out
}

// CommentSection adds s's comment on section (from 1) of space name's agent draft id. Only
// the reviewer the draft ran as comments, on an open draft that can still change.
func (x Stewards) CommentSection(ctx context.Context, s tenancy.Scope, name, id string, section int, text string) ([]SectionComment, error) {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || utf8.RuneCountInString(text) > MaxSectionComment || !printableText(text) {
		return nil, fmt.Errorf("%w: a comment is 1 to %d characters", ErrInvalid, MaxSectionComment)
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	q, err := x.question(ctx, name, id)
	if err != nil {
		return nil, fmt.Errorf("%w: no such draft in this space", ErrInvalid)
	}
	if err := changeable(q, s.User()); err != nil {
		return nil, err
	}
	if n := len(DraftSections(q.Thread.Text)); section < 1 || section > n {
		return nil, fmt.Errorf("%w: the draft has sections 1 to %d", ErrInvalid, n)
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "commented",
		PayloadSHA256: hashOf(text), Reason: fmt.Sprintf("section %d of a %s, round %d", section, q.Thread.Kind, q.Thread.Rounds+1), Space: auditSpace(q.Space)}); err != nil {
		return nil, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	out, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(cur *StewardQuestion) error {
		if err := changeable(*cur, s.User()); err != nil {
			return err
		}
		if len(roundComments(cur.Thread)) >= MaxRoundComments {
			return fmt.Errorf("%w: at most %d comments a round", ErrInvalid, MaxRoundComments)
		}
		cur.Thread.Comments = append(cur.Thread.Comments, SectionComment{Section: section, Text: text, By: s.User(), Round: cur.Thread.Rounds})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return roundComments(out.Thread), nil
}

// redraftSections writes each commented section of t again, as in, from t's sources (rctx
// is narrowed to them), with that section's comments and note; every other section is kept.
func (x Stewards) redraftSections(rctx context.Context, in tenancy.Scope, t *SavedThread, req Request, comments []SectionComment, note string) (Response, string, []SectionDiff, error) {
	secs := DraftSections(t.Text)
	by := map[int][]string{}
	for _, c := range comments {
		by[c.Section] = append(by[c.Section], c.Text)
	}
	var order []int
	for n := range by {
		order = append(order, n)
	}
	sort.Ints(order)
	var last Response
	var diffs []SectionDiff
	for _, n := range order {
		if n < 1 || n > len(secs) {
			continue
		}
		ask := strings.Join(by[n], " / ")
		if note != "" {
			ask += " / " + note
		}
		ctx := context.WithValue(rctx, noteKey{}, redraftNote{draft: strings.TrimRight(secs[n-1], "\n"), note: ask, section: true})
		resp, err := x.Agent.Ask(ctx, in, req)
		if err != nil {
			return Response{}, "", nil, err
		}
		text := strings.TrimRight(DraftText(resp.Answer), "\n")
		if text == "" {
			return Response{}, "", nil, fmt.Errorf("%w: nothing the section could cite", ErrInvalid)
		}
		old := secs[n-1]
		secs[n-1] = text + old[len(strings.TrimRight(old, "\n")):] // the section's own line ends, kept
		diffs = append(diffs, SectionDiff{Section: n, Diff: LineDiff(lines(strings.TrimRight(old, "\n")), lines(text))})
		last = resp
	}
	return last, strings.Join(secs, ""), diffs, nil
}

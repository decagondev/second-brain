package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Ask for changes (sb-agents(7), Ask for changes; SB-4602): the steward an agent draft ran as
// sends it back with a note, and the agent redrafts once from the same sources. The note is
// the reviewer's request about the draft, never a source.
const (
	MaxChangeNote   = 500
	MaxChangeRounds = 3
	MaxDraftSources = 40
)

// RedraftResponse is a redraft: its new text, the earlier texts, oldest first, its rounds
// and the line diff from the last text to the new one.
type RedraftResponse struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Versions []string `json:"versions"`
	Rounds   int      `json:"rounds"`
	Diff     []string `json:"diff"`
	// Sections is the diff of each section redrafted for its comments (SB-4902).
	Sections []SectionDiff `json:"sections,omitempty"`
}

// draftSources are the documents a draft cited or offered (See also), as KIND:ID, in order.
func draftSources(r Response) []string {
	var out []string
	add := func(cite string) {
		if k := docKey(cite); k != "" && !contains(out, k) && len(out) < MaxDraftSources {
			out = append(out, k)
		}
	}
	for _, c := range r.Answer.Claims {
		for _, ci := range c.Cites {
			add(ci.Cite)
		}
	}
	for _, sa := range r.SeeAlso {
		add(sa.Cite)
	}
	return out
}

// threadDiff is the line diff from a draft's last earlier text to its text, nil for none.
func threadDiff(t *SavedThread) []string {
	if t == nil || len(t.Versions) == 0 {
		return nil
	}
	return LineDiff(lines(t.Versions[len(t.Versions)-1]), lines(t.Text))
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// AskForChanges redrafts space name's agent draft id as s, the steward it ran as, with note.
// The redraft is narrowed to the first draft's sources, so it cites nothing the first draft
// couldn't (LAW-14); a fourth round is ErrInvalid.
func (x Stewards) AskForChanges(ctx context.Context, s tenancy.Scope, name, id, note string) (RedraftResponse, error) {
	note = strings.Join(strings.Fields(note), " ")
	if utf8.RuneCountInString(note) > MaxChangeNote || !printableText(note) {
		return RedraftResponse{}, fmt.Errorf("%w: a note is 1 to %d characters", ErrInvalid, MaxChangeNote)
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return RedraftResponse{}, err
	}
	q, err := x.question(ctx, name, id)
	if err != nil {
		return RedraftResponse{}, fmt.Errorf("%w: no such draft in this space", ErrInvalid)
	}
	if err := changeable(q, s.User()); err != nil {
		return RedraftResponse{}, err
	}
	comments := roundComments(q.Thread)
	if note == "" && len(comments) == 0 {
		return RedraftResponse{}, fmt.Errorf("%w: a note is 1 to %d characters, or comment on a section first", ErrInvalid, MaxChangeNote)
	}
	if x.Agent.pausedIn(q.Thread.Agent, s) { // SB-4702
		return RedraftResponse{}, fmt.Errorf("%w: %s is paused in this space", ErrKilled, q.Thread.Agent)
	}
	if x.Rescope == nil {
		return RedraftResponse{}, fmt.Errorf("%w: this cell can't redraft", ErrUnavailable)
	}
	rctx, in, err := x.Rescope(ctx, s.User(), q.Space, q.Thread.Kind == DraftLeaderWeek)
	if err != nil {
		return RedraftResponse{}, fmt.Errorf("%w: you can't ask in this space now", ErrInvalid)
	}
	t := q.Thread
	req := Request{Question: t.Asked, Documents: t.AskDocs}
	if t.Agent != "ask" {
		req.Agent = t.Agent
	}
	if len(req.Documents) == 0 {
		rctx = withOnly(rctx, t.Sources) // the same sources, never more
	}
	var resp Response
	var text string
	var diffs []SectionDiff
	if len(comments) > 0 { // SB-4902: only the commented sections, the rest byte for byte
		resp, text, diffs, err = x.redraftSections(rctx, in, t, req, comments, note)
	} else {
		resp, text, err = x.draftAnswer(withNote(rctx, t.Text, note), in, t.Kind, req)
	}
	if err != nil {
		return RedraftResponse{}, err
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "redrafted",
		PayloadSHA256: hashOf(note), Reason: fmt.Sprintf("agent %s redrafted a %s, round %d", t.Agent, t.Kind, t.Rounds+1), Space: auditSpace(q.Space)}); err != nil {
		return RedraftResponse{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	out, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(cur *StewardQuestion) error {
		if err := changeable(*cur, s.User()); err != nil {
			return err
		}
		th := cur.Thread
		th.Versions = append(th.Versions, th.Text)
		th.Text, th.Rounds, th.Release = text, th.Rounds+1, resp.Release
		if th.Kind == DraftDecisions && th.Doc != "" {
			th.Supersede = supersedeSuggestions(resp.Answer.Claims, th.Doc) // SB-4505, from the new claims
		}
		cur.Trace = resp.Trace
		return nil
	})
	if err != nil {
		return RedraftResponse{}, err
	}
	th := out.Thread
	return RedraftResponse{ID: id, Text: th.Text, Versions: th.Versions, Rounds: th.Rounds, Diff: threadDiff(th), Sections: diffs}, nil
}

// changeable says whether a question is an open agent draft user may ask changes of.
func changeable(q StewardQuestion, user string) error {
	t := q.Thread
	switch {
	case t == nil || t.Agent == "" || t.Discarded || q.Reply != nil:
		return fmt.Errorf("%w: only an open agent draft can be changed", ErrInvalid)
	case q.Asker != user:
		return fmt.Errorf("%w: only the steward the draft ran as can ask for changes", ErrInvalid)
	case t.Asked == "":
		return fmt.Errorf("%w: this draft was made before changes could be asked; discard it or publish it", ErrInvalid)
	case t.Rounds >= MaxChangeRounds:
		return fmt.Errorf("%w: a draft takes at most %d rounds of changes", ErrInvalid, MaxChangeRounds)
	}
	return nil
}

func printableText(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// LineDiff is the lines removed ("- ") and added ("+ ") from a to b, by their longest
// common subsequence, in order.
func LineDiff(a, b []string) []string {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	out := []string{}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			i, j = i+1, j+1
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			out = append(out, "+ "+b[j])
			j++
		default:
			out = append(out, "- "+a[i])
			i++
		}
	}
	return out
}

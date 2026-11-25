package ask

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// The follow-up agents (sb-agents(7), The legwork agents; SB-4204 to SB-4207): like every
// legwork agent they are read-only, of no template (voc only in a product or support
// space), cite every claim and have no tools. They draft; a person sends or publishes.
const (
	AgentIncident = "incident-brief"
	AgentVoC      = "voc"
	AgentFix      = "fix-draft"
	AgentCheckin  = "goal-check-in"
)

var labelsLineRe = regexp.MustCompile(`(?i)^\s*labels?\s*:\s*(.+)$`)

// IncidentLabels are the labels a ticket's text names on a "Labels:" line among its first
// five lines, lower case (sb-connector(3), tickets: linear).
func IncidentLabels(text string) []string {
	lines := strings.SplitN(text, "\n", 6)
	if len(lines) > 5 {
		lines = lines[:5]
	}
	for _, l := range lines {
		if m := labelsLineRe.FindStringSubmatch(l); m != nil {
			var out []string
			for _, x := range strings.Split(m[1], ",") {
				if x = strings.ToLower(strings.TrimSpace(x)); x != "" {
					out = append(out, x)
				}
			}
			return out
		}
	}
	return nil
}

// IsIncident says a ticket's text carries one of labels (any case).
func IsIncident(text string, labels []string) bool {
	for _, l := range IncidentLabels(text) {
		for _, w := range labels {
			if strings.EqualFold(l, w) {
				return true
			}
		}
	}
	return false
}

// IncidentQuestion is an incident brief's question: the ticket's title is a topic only.
func IncidentQuestion(title string) string {
	return "For the incident " + cut(title, 200) + ": which past tickets were like it and how were they resolved, which runbooks match, what was decided or changed in the last 7 days, and who owns the systems it names?"
}

// VoCQuestion is the voc agent's fixed weekly question.
const VoCQuestion = "What are this week's customer themes from tickets and account notes, and how often does each come up?"

// The voc limits.
const (
	MaxThemes     = 8
	MaxThemeCites = 3
)

// vocThemes keeps at most MaxThemes cited claims, each with at most MaxThemeCites cites,
// and ends each with its count: how many of its cited documents were first seen in the
// last 7 days, and how many in the 4 weeks before (counts from the cites, never the model).
func (a Agent) vocThemes(ctx context.Context, claims []answer.Claim, now time.Time) ([]answer.Claim, error) {
	var kept []answer.Claim
	for _, c := range claims {
		if c.Uncited || len(c.Cites) == 0 {
			continue
		}
		if len(c.Cites) > MaxThemeCites {
			c.Cites = c.Cites[:MaxThemeCites]
		}
		kept = append(kept, c)
		if len(kept) == MaxThemes {
			break
		}
	}
	seen, err := a.seenOf(ctx, kept)
	if err != nil {
		return nil, err
	}
	for i, c := range kept {
		week, before := 0, 0
		docs := map[string]bool{}
		for _, ci := range c.Cites {
			doc, _, _ := strings.Cut(ci.Cite, "@")
			if docs[doc] {
				continue
			}
			docs[doc] = true
			at, ok := seen[ci.Cite]
			switch age := now.Sub(at); {
			case !ok:
			case age <= 7*24*time.Hour:
				week++
			case age <= 35*24*time.Hour:
				before++
			}
		}
		kept[i].Text = fmt.Sprintf("%s (%d this week, %d in the 4 weeks before)", strings.TrimSpace(c.Text), week, before)
	}
	return kept, nil
}

// seenOf is when the index first saw each cited version.
func (a Agent) seenOf(ctx context.Context, claims []answer.Claim) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, nil
	}
	var refs []ports.SourceRef
	have := map[string]bool{}
	for _, c := range claims {
		for _, ci := range c.Cites {
			doc, _, _ := strings.Cut(ci.Cite, "@")
			kind, id, ok := strings.Cut(doc, ":")
			if ok && !have[doc] {
				have[doc] = true
				refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
			}
		}
	}
	if len(refs) == 0 {
		return out, nil
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	for _, ch := range chunks {
		if t, ok := out[ch.Cite]; !ch.Seen.IsZero() && (!ok || ch.Seen.Before(t)) {
			out[ch.Cite] = ch.Seen
		}
	}
	return out, nil
}

// CheckinQuestion is a goal check-in's question.
func CheckinQuestion(title string) string {
	return "For the goal " + cut(title, 200) + ": which tickets moved, what was decided, and which updates and documents changed this week, and is it on track, at risk or off track?"
}

// CheckinSilentAfter is how long a goal may go without a published check-in (SB-4207).
const CheckinSilentAfter = 21 * 24 * time.Hour

// LastCheckin is when a check-in for goal (KIND:ID) was last published as a note in space
// key's queue, or zero.
func LastCheckin(qs []StewardQuestion, goal string) time.Time {
	var last time.Time
	for _, q := range qs {
		if q.Thread != nil && q.Thread.Kind == DraftCheckin && q.Thread.Goal == goal && q.Reply != nil && q.Reply.Note != "" && q.Reply.NoteAt.After(last) {
			last = q.Reply.NoteAt
		}
	}
	return last
}

// Silent says a goal first seen at seen has had no check-in published for CheckinSilentAfter.
func Silent(seen, last, now time.Time) bool {
	from := last
	if from.IsZero() {
		from = seen
	}
	return !from.IsZero() && now.Sub(from) >= CheckinSilentAfter
}

// KindFix is a suggested edit kept for a document's owner (SB-4206).
const KindFix = "fix"

// Fix states: suggested until the source changes, then applied.
const (
	FixSuggested = "suggested"
	FixApplied   = "applied"
)

// FixQuestion is the fix-draft agent's question about one document.
func FixQuestion(title string) string {
	return "What do newer sources say that the document " + cut(title, 200) + " has wrong or out of date? For each change, quote the passage to change and give the new text."
}

// FixResult is POST /v1/spaces/NAME/review/fix.
type FixResult struct {
	ID      string `json:"id,omitempty"`
	Owner   string `json:"owner"`
	Changes int    `json:"changes"`
	Told    bool   `json:"told"`
}

// DraftFix is a steward of space name asking the fix-draft agent for a suggested edit of
// doc (KIND:ID), a document the steward can open. It runs as the document's owner, one
// active user, so it uses only what they can open; each change must cite a newer document
// than doc. The suggestion is kept for the owner, who is told; nothing is written to the
// source.
func (l Legwork) DraftFix(ctx context.Context, s tenancy.Scope, name, doc string) (FixResult, error) {
	if _, _, err := l.Stewards.steward(ctx, s, name); err != nil {
		return FixResult{}, err
	}
	if err := l.check(s); err != nil {
		return FixResult{}, err
	}
	keys, err := documentKeys([]string{doc})
	if err != nil || len(keys) != 1 {
		return FixResult{}, fmt.Errorf("%w: doc is KIND:ID", ErrInvalid)
	}
	lk, ok := l.Agent.Index.(ports.DocLookup)
	if !ok {
		return FixResult{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	kind, id, _ := strings.Cut(keys[0], ":")
	chunks, err := lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return FixResult{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	first, ok := firsts(latest(chunks))[keys[0]]
	if !ok {
		return FixResult{}, ErrNotFound
	}
	if open, err := l.Agent.opensCites(ctx, s, []string{first.Cite}); err != nil || !open[first.Cite] {
		if err != nil {
			return FixResult{}, err
		}
		return FixResult{}, ErrNotFound // the steward can't open it: as if it weren't there
	}
	out := FixResult{Owner: first.Owner}
	us, err := l.users(ctx)
	if err != nil {
		return FixResult{}, err
	}
	var owner ports.User
	n := 0
	for _, u := range us {
		for _, e := range u.Emails {
			if first.Owner != "" && strings.EqualFold(e, first.Owner) {
				owner, n = u, n+1
				break
			}
		}
	}
	if n != 1 || l.ScopeIn == nil {
		return out, fmt.Errorf("%w: the document has no one active owner to suggest an edit to", ErrInvalid)
	}
	as, err := l.ScopeIn(ctx, owner.ID, SpaceKey(name))
	if err != nil {
		return out, fmt.Errorf("%w: the owner isn't a member of %s", ErrInvalid, name)
	}
	if _, err := l.Agent.legworkDept(ctx, as, AgentFix); err != nil {
		return out, err
	}
	title := first.Title
	if title == "" {
		title = first.SourceID
	}
	resp, err := l.Agent.Ask(ctx, as, Request{Question: FixQuestion(title), Agent: AgentFix})
	if err != nil {
		return out, err
	}
	var changes []answer.Claim
	for _, c := range resp.Answer.Claims {
		var newer []answer.Cite
		for _, ci := range c.Cites {
			if d, _, _ := strings.Cut(ci.Cite, "@"); d != keys[0] { // a change cites another, newer source
				newer = append(newer, ci)
			}
		}
		if !c.Uncited && len(newer) > 0 {
			c.Cites = newer
			changes = append(changes, c)
		}
	}
	out.Changes = len(changes)
	if len(changes) == 0 {
		return out, fmt.Errorf("%w: nothing newer that the owner can open says otherwise", ErrInvalid)
	}
	r := Research{ID: l.NewID("rs_"), Tenant: l.Agent.Tenant, User: owner.ID, Kind: KindFix, Question: "Suggested edits: " + cut(title, 200),
		Space: SpaceKey(name), Documents: []string{first.Cite}, Steps: []string{}, State: FixSuggested, At: l.now(), DoneAt: l.now(),
		Sections: []Section{{Question: cut(title, MaxStep), Trace: resp.Trace, Claims: changes}}}
	if err := l.audit(ctx, s.User(), r, "fix_suggested", hashOf(first.Cite)); err != nil {
		return out, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return out, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	out.ID = r.ID
	if l.Tell != nil {
		out.Told = l.Tell(ctx, owner.ID, KindFix, "A steward of "+name+" suggested edits to one of your documents") == nil
	}
	return out, nil
}

// fixState is a suggestion's state now: applied once its document has a newer version.
func (l Legwork) fixState(ctx context.Context, r Research) string {
	if r.Kind != KindFix || len(r.Documents) != 1 {
		return r.State
	}
	lk, ok := l.Agent.Index.(ports.DocLookup)
	if !ok {
		return r.State
	}
	doc, ver, _ := strings.Cut(r.Documents[0], "@")
	kind, id, _ := strings.Cut(doc, ":")
	chunks, err := lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return r.State
	}
	for _, c := range latest(chunks) {
		if _, v, _ := strings.Cut(c.Cite, "@"); v != ver {
			return FixApplied
		}
	}
	return r.State
}

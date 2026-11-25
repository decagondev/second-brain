package ask

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Keep it current (sb-ask(3); SB-1708, SB-1709).
const (
	MaxChanges        = 50
	MaxChangeDays     = 90
	DefaultReviewDays = 365
	ReviewCites       = 5
	ReviewWindow      = 30 * 24 * time.Hour
)

// ChangesResponse is what changed in a window, newest first.
type ChangesResponse struct {
	Since time.Time `json:"since"`
	Items []Result  `json:"items"`
}

// memberSpaces are the ids of the spaces the scope's user is a member of now, and whether
// the cell has spaces at all.
func (a Agent) memberSpaces(ctx context.Context, s tenancy.Scope) (map[string]bool, bool, error) {
	if a.Auth.Spaces == nil {
		return nil, false, nil
	}
	all, _, err := a.Auth.Spaces(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	if len(all) == 0 {
		return nil, false, nil
	}
	mine := map[string]bool{}
	for _, g := range s.Groups() {
		mine["group:"+g] = true
		mine[g] = true
	}
	out := map[string]bool{}
	for _, sp := range all {
		for _, m := range sp.Members {
			if mine[m] {
				out[sp.ID] = true
			}
		}
	}
	return out, true, nil
}

// Changes lists documents with a version first seen in the last days days, in the scope's
// space or, company-wide, in the person's own spaces, each checked live as them.
func (a Agent) Changes(ctx context.Context, s tenancy.Scope, days int) (ChangesResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return ChangesResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if days < 1 || days > MaxChangeDays {
		return ChangesResponse{}, fmt.Errorf("%w: since is 1d to %dd", ErrInvalid, MaxChangeDays)
	}
	if a.Killed == nil || a.Killed(ctx) || spaceKilled(a.SpaceKilled, s) {
		return ChangesResponse{}, ErrKilled
	}
	since := a.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	out := ChangesResponse{Since: since, Items: []Result{}}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{SeenSince: since})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	mine, spaced, err := a.memberSpaces(ctx, s)
	if err != nil {
		return out, err
	}
	var cand []ports.Chunk
	for _, c := range firsts(latest(chunks)) {
		if spaced && s.Space() == "company" && !anyOf(c.Spaces, mine) {
			continue // company-wide: only what the person's own spaces hold
		}
		cand = append(cand, c)
	}
	sort.Slice(cand, func(i, j int) bool {
		if !cand[i].Seen.Equal(cand[j].Seen) {
			return cand[i].Seen.After(cand[j].Seen)
		}
		return cand[i].ID < cand[j].ID
	})
	who, err := a.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, MaxChanges)
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	own := a.owners()
	for _, c := range res.Context {
		r := a.result(ctx, own, c)
		r.Snippet = ""
		out.Items = append(out.Items, r)
	}
	return out, nil
}

func anyOf(xs []string, set map[string]bool) bool {
	for _, x := range xs {
		if set[x] {
			return true
		}
	}
	return false
}

// ReviewSource is where answers' cites and wrong-citation marks come from: the traces.
type ReviewSource interface {
	Between(ctx context.Context, tenant string, from, to time.Time) ([]Trace, error)
	FeedbackBetween(ctx context.Context, tenant string, from, to time.Time) ([]Feedback, error)
}

// ReviewItem is a document to review, shown because the steward can open it now.
type ReviewItem struct {
	Result
	Cited   int `json:"cited"`
	Flagged int `json:"flagged"`
	// Outdated counts reports that the document is out of date (SB-2703).
	Outdated int `json:"outdated,omitempty"`
	// Wider is, for a note, how many of its readers are no longer members of the space
	// (SB-2004): a count, never who. Question is the note's question, to republish it.
	Wider    int    `json:"wider,omitempty"`
	Question string `json:"question,omitempty"`
	// OwnerGone marks a document whose owner at the source is no active user (SB-2201);
	// Copies are the other documents with the same text the steward can open (SB-2202).
	OwnerGone bool     `json:"owner_gone,omitempty"`
	Copies    []Result `json:"copies,omitempty"`
	// SourcesMoved is, for a note, how many of its cites have a newer version or are gone
	// (SB-2804).
	SourcesMoved int `json:"sources_moved,omitempty"`
	// Reports are the traces of the answers reported as citing it wrongly, newest first,
	// for an eval case (SB-2204).
	Reports []string `json:"reports,omitempty"`
	// Unhelpful counts the Not helpful ratings of answers citing it since its mark (SB-3604).
	Unhelpful int `json:"unhelpful,omitempty"`
}

// HiddenItem is a document to review the steward can't open: only its owner, if any and
// still an active user, or that its owner has gone.
type HiddenItem struct {
	Owner     string `json:"owner,omitempty"`
	OwnerGone bool   `json:"owner_gone,omitempty"`
}

// MaxOwnerGone is the most documents with no owner the review list names (SB-2201).
const MaxOwnerGone = 50

// ReviewResponse is the stewards' review list.
type ReviewResponse struct {
	Items  []ReviewItem `json:"items"`
	Hidden []HiddenItem `json:"hidden"`
	// Terms are the terms the space's documents define that its glossary lacks (SB-3407).
	Terms []DefinedTerm `json:"terms"`
	// Usage is the space's own row of usage by space, over the last 7 days (SB-3506).
	Usage *SpaceUsage `json:"usage,omitempty"`
	// Personal is how many documents the space holds with personal data, an IBAN or a
	// card number (SB-1904): a count, never which.
	Personal int `json:"personal_data"`
	// GlossaryFlagged are the space's glossary terms that read like an instruction, which
	// are never used (SB-2203).
	GlossaryFlagged []string `json:"glossary_flagged,omitempty"`
	// SameName are the space's documents with the same title and different text (SB-2802).
	SameName []SameName `json:"same_name,omitempty"`
	// Freshness counts the space's answers resting mostly on stale cites (SB-2807).
	Freshness Freshness `json:"freshness"`
	// EvalsToCheck are the space's eval cases whose expected document changed (SB-2904).
	EvalsToCheck []EvalCheck `json:"evals_to_check,omitempty"`
	// StartHere are the space's start_here items gone stale, and StartHidden how many more
	// the steward can't open (SB-2907).
	StartHere   []StartItem `json:"start_here,omitempty"`
	StartHidden int         `json:"start_here_hidden,omitempty"`
	// FoundNothing counts the space's answers that found nothing (SB-3101), and Feedback
	// its ratings and the documents unhelpful answers cited (SB-3104).
	FoundNothing FoundNothing    `json:"found_nothing"`
	Feedback     StewardFeedback `json:"feedback"`
	// Conflicts are the pairs of documents disagreeing claims cited this month, and
	// ConflictsHidden the pairs the steward can't open both of (SB-3605).
	Conflicts       []Conflict `json:"conflicts,omitempty"`
	ConflictsHidden int        `json:"conflicts_hidden,omitempty"`
	// Teams are a department's teams, counts only (SB-3905).
	Teams []TeamRollup `json:"teams,omitempty"`
}

// TeamRollup is one team's counts on its department's review list (SB-3905): never a
// title, a question or a person. A team with fewer than 5 asks in the last 7 days shows
// FewerThan5 instead of its ask counts, as usage by space does.
type TeamRollup struct {
	Space        string `json:"space"`
	ToReview     int    `json:"to_review"`
	Overdue      int    `json:"overdue"`
	Open         int    `json:"open_questions"`
	Waiting      int    `json:"waiting"` // items waiting on another space (SB-4504)
	FoundNothing *int   `json:"found_nothing,omitempty"`
	Asks         *int   `json:"asks,omitempty"`
	FewerThan5   bool   `json:"fewer_than_5,omitempty"`
}

// answerFacts fills what the list reads from the space's answers and their ratings.
func (x Stewards) answerFacts(ctx context.Context, s tenancy.Scope, space string, all []Trace, months []Feedback, now time.Time, out *ReviewResponse) error {
	var err error
	if out.Feedback, err = x.stewardFeedback(ctx, s, space, all, months, now); err != nil {
		return err
	}
	out.Conflicts, out.ConflictsHidden, err = x.conflicts(ctx, s, space, all, now)
	return err
}

// reviewTraces reads the traces once: the review window's, and the space's stale-answer
// counts this calendar month and last (SB-2807).
func (x Stewards) reviewTraces(ctx context.Context, src ReviewSource, name string, now time.Time) ([]Trace, []Trace, Freshness, error) {
	last := MonthStart(MonthStart(now).Add(-time.Hour))
	from := now.Add(-ReviewWindow)
	if last.Before(from) {
		from = last
	}
	all, err := src.Between(ctx, x.Agent.Tenant, from, now)
	if err != nil {
		return nil, nil, Freshness{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	var traces, thisMonth, lastMonth []Trace
	for _, t := range all {
		if !t.At.Before(now.Add(-ReviewWindow)) {
			traces = append(traces, t)
		}
		switch {
		case !t.At.Before(MonthStart(now)):
			thisMonth = append(thisMonth, t)
		case !t.At.Before(last):
			lastMonth = append(lastMonth, t)
		}
	}
	return traces, all, Freshness{ThisMonth: StaleShare(thisMonth, SpaceKey(name)), LastMonth: StaleShare(lastMonth, SpaceKey(name))}, nil
}

// Review lists the documents a space holds that answers lean on but that are older than
// its review_after_days, and any marked as a wrong citation, for its stewards (SB-1709).
func (x Stewards) Review(ctx context.Context, s tenancy.Scope, name string, src ReviewSource, after int) (ReviewResponse, error) {
	if err := x.reviewer(ctx, s, name); err != nil {
		return ReviewResponse{}, err
	}
	out, err := x.review(ctx, s, name, src, after)
	if err != nil || x.Children == nil {
		return out, err
	}
	out.Teams, err = x.teams(ctx, s, name, src, after)
	return out, err
}

// teams rolls each team's list up to its department's stewards (SB-3905, LAW-24): the
// counts its own stewards see, never what they count. A department's stewards are
// stewards of its teams for this.
func (x Stewards) teams(ctx context.Context, s tenancy.Scope, name string, src ReviewSource, after int) ([]TeamRollup, error) {
	var out []TeamRollup
	now := x.Now().UTC()
	for _, team := range x.Children(name) {
		days := after
		if x.ReviewAfter != nil {
			days = x.ReviewAfter(team)
		}
		r, err := x.review(ctx, s, team, src, days)
		if err != nil {
			return nil, err
		}
		row := TeamRollup{Space: team, ToReview: len(r.Items)} // only what the department's steward can open (LAW-24)
		qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(team))
		if err != nil {
			return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		for _, q := range qs {
			if q.Reply == nil && (q.Thread == nil || !q.Thread.Discarded) {
				row.Open++
				if WorkingDays(q.At, now, x.Zone) >= OverdueAfter {
					row.Overdue++
				}
			}
		}
		if row.Waiting, err = x.waitingCount(ctx, team); err != nil {
			return nil, err
		}
		if r.Usage == nil || r.Usage.FewerThan5 || r.Usage.Asks == nil {
			row.FewerThan5 = true
		} else {
			asks, nothing := *r.Usage.Asks, r.FoundNothing.ThisWeek
			row.Asks, row.FoundNothing = &asks, &nothing
		}
		out = append(out, row)
	}
	return out, nil
}

// review is the list for one space, without the steward check, which its callers make.
func (x Stewards) review(ctx context.Context, s tenancy.Scope, name string, src ReviewSource, after int) (ReviewResponse, error) {
	if after <= 0 {
		after = DefaultReviewDays
	}
	now := x.Now().UTC()
	traces, all, fresh, err := x.reviewTraces(ctx, src, name, now)
	if err != nil {
		return ReviewResponse{}, err
	}
	months, err := src.FeedbackBetween(ctx, x.Agent.Tenant, MonthStart(MonthStart(now).Add(-time.Hour)), now)
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	fbs, err := src.FeedbackBetween(ctx, x.Agent.Tenant, now.Add(-ReviewWindow), now)
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	cited, flagged, refs := reviewCounts(traces, fbs)
	outdated := outdatedOf(fbs)
	refs = withOutdated(refs, outdated, cited, flagged)
	reports := reportsOf(fbs)
	marks, err := x.marked(ctx, name)
	if err != nil {
		return ReviewResponse{}, err
	}
	flaggedAt := lastFlags(fbs)
	unhelpful := unhelpfulSince(traces, fbs, marks)
	out := ReviewResponse{Items: []ReviewItem{}, Hidden: []HiddenItem{}, Freshness: fresh, FoundNothing: foundNothing(all, SpaceKey(name), now),
		Usage: ownUsage(all, fbs, name, now)}
	if err := x.answerFacts(ctx, s, SpaceKey(name), all, months, now, &out); err != nil {
		return out, err
	}
	if x.Agent.Glossary != nil {
		_, out.GlossaryFlagged = usable(x.Agent.Glossary(name))
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	if out.Personal, out.Terms, err = x.heldFacts(ctx, s, lk, name); err != nil {
		return out, err
	}
	widerBy, notes, err := x.widerNotes(ctx, lk, name)
	if err != nil {
		return out, err
	}
	held, err := x.heldBy(ctx, lk, name)
	if err != nil {
		return out, err
	}
	movedBy, moved, err := x.movedNotes(ctx, lk, name, marks)
	if err != nil {
		return out, err
	}
	notes = append(notes, moved...)
	if out.SameName, err = x.sameName(ctx, s, held); err != nil {
		return out, err
	}
	if out.EvalsToCheck, err = x.evalsToCheck(ctx, name); err != nil {
		return out, err
	}
	if out.StartHere, out.StartHidden, err = x.startStale(ctx, s, lk, name); err != nil {
		return out, err
	}
	if len(refs) == 0 && len(notes) == 0 && len(held) == 0 {
		return out, nil
	}
	var chunks []ports.Chunk
	if len(refs) > 0 {
		if chunks, err = lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: refs}); err != nil {
			return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
	}
	cand, in := dueFor(SpaceKey(name), chunks, reviewFacts{cited: cited, flagged: flagged, outdated: outdated, flaggedAt: flaggedAt, marks: marks, unhelpful: unhelpful}, after, now)
	for _, c := range notes {
		if k := c.Kind + ":" + c.SourceID; !in[k] {
			cand = append(cand, c)
			in[k] = true
		}
	}
	gone, err := x.ownersGone(ctx, append(append([]ports.Chunk(nil), cand...), held...))
	if err != nil {
		return out, err
	}
	copied := copiesIn(held)
	cand = withHeld(cand, in, held, gone, copied)
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]bool{}
	own := x.Agent.owners()
	newest := newestCopies(res.Context, copied)
	copies := map[string][]Result{}
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		allowed[k] = true
		r := x.Agent.result(ctx, own, c)
		r.Snippet = ""
		if gone[c.Owner] && (r.Owner == nil || !r.Owner.Caretaker) {
			r.Owner = nil // never someone to ask; a caretaker is (SB-3405)
		}
		if n, ok := newest[c.TextSHA256]; ok && copied[k] && n != k {
			copies[n] = append(copies[n], r)
			continue // listed once, under the newest copy
		}
		it := ReviewItem{Result: r, Cited: cited[k], Flagged: flagged[k], Outdated: outdated[k], Wider: widerBy[k], OwnerGone: gone[c.Owner], Reports: reports[k], SourcesMoved: movedBy[k]}
		if m, ok := marks[k]; ok && m.Version == c.Version && unhelpful[k] >= UnhelpfulAfterMark {
			it.Unhelpful = unhelpful[k]
		}
		if it.Wider > 0 {
			it.Question = strings.TrimSuffix(path.Base(c.SourceID), ".md")
		}
		out.Items = append(out.Items, it)
	}
	for i, it := range out.Items {
		out.Items[i].Copies = copies[docKey(it.Cite)]
	}
	out.Hidden = hiddenOf(cand, allowed, gone)
	sortReview(out.Items)
	return out, nil
}

// sortReview puts a note read wider than its space first, then the most flagged and cited.
func sortReview(items []ReviewItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.Wider > 0) != (b.Wider > 0) {
			return a.Wider > 0
		}
		return a.Flagged+a.Outdated+a.Cited > b.Flagged+b.Outdated+b.Cited
	})
}

// hiddenOf is an entry for each candidate the steward can't open: never its title or cite,
// and its owner only while they are an active user.
func hiddenOf(cand []ports.Chunk, allowed, gone map[string]bool) []HiddenItem {
	out := []HiddenItem{}
	for _, c := range cand {
		if !allowed[c.Kind+":"+c.SourceID] {
			h := HiddenItem{Owner: c.Owner, OwnerGone: gone[c.Owner]}
			if h.OwnerGone {
				h.Owner = ""
			}
			out = append(out, h)
		}
	}
	return out
}

// withHeld adds to cand the held documents whose owner has gone (at most MaxOwnerGone) and
// those with copies, each once.
func withHeld(cand []ports.Chunk, in map[string]bool, held []ports.Chunk, gone, copied map[string]bool) []ports.Chunk {
	orphans := 0
	for _, c := range held {
		k := c.Kind + ":" + c.SourceID
		if in[k] {
			continue
		}
		if gone[c.Owner] && orphans < MaxOwnerGone {
			orphans++
		} else if !copied[k] {
			continue
		}
		cand = append(cand, c)
		in[k] = true
	}
	return cand
}

// heldBy is the first chunk of the newest version of every document the space holds.
func (x Stewards) heldBy(ctx context.Context, lk ports.DocLookup, name string) ([]ports.Chunk, error) {
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Space: SpaceKey(name)})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var out []ports.Chunk
	for _, c := range firsts(latest(cs)) {
		if contains(c.Spaces, SpaceKey(name)) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ownersGone is, of the chunks' owners, those who are no active user in the directory now
// (SB-2201). A directory that can't answer is an error: an owner is never guessed gone.
func (x Stewards) ownersGone(ctx context.Context, chunks []ports.Chunk) (map[string]bool, error) {
	out := map[string]bool{}
	if x.Gone == nil {
		return out, nil
	}
	asked := map[string]bool{}
	for _, c := range chunks {
		if c.Owner == "" || asked[c.Owner] {
			continue
		}
		asked[c.Owner] = true
		g, err := x.Gone(ctx, c.Owner)
		if err != nil {
			return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		if g {
			out[c.Owner] = true
		}
	}
	return out, nil
}

// copiesIn marks the documents that share their text with another one (SB-2202).
func copiesIn(held []ports.Chunk) map[string]bool {
	by := map[string][]string{}
	for _, c := range held {
		if c.TextSHA256 != "" {
			by[c.TextSHA256] = append(by[c.TextSHA256], c.Kind+":"+c.SourceID)
		}
	}
	out := map[string]bool{}
	for _, ks := range by {
		if len(ks) > 1 {
			for _, k := range ks {
				out[k] = true
			}
		}
	}
	return out
}

// newestCopies is, for each text hash among the allowed copies, the newest one's document.
func newestCopies(allowed []ports.Chunk, copied map[string]bool) map[string]string {
	var cs []ports.Chunk
	for _, c := range allowed {
		if copied[c.Kind+":"+c.SourceID] {
			cs = append(cs, c)
		}
	}
	out := map[string]string{}
	for _, c := range retrieve.OneCopy(cs) {
		out[c.TextSHA256] = c.Kind + ":" + c.SourceID
	}
	return out
}

// reportsOf is, by document, the traces of its wrong-citation reports, newest first.
func reportsOf(fbs []Feedback) map[string][]string {
	sorted := append([]Feedback(nil), fbs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At.After(sorted[j].At) })
	out := map[string][]string{}
	for _, f := range sorted {
		if k := docKey(f.WrongCitation); f.WrongCitation != "" && len(out[k]) < MaxReports {
			out[k] = append(out[k], f.Trace)
		}
	}
	return out
}

// reviewFacts are what puts a document on the review list, by document.
type reviewFacts struct {
	cited, flagged, outdated map[string]int
	flaggedAt                map[string]time.Time
	marks                    map[string]ReviewMark
	unhelpful                map[string]int // since the mark (SB-3604)
}

// dueFor picks the documents space id holds that are due for review: cited often and old
// or past their own review-by date, or reported, and not covered by a mark.
func dueFor(id string, chunks []ports.Chunk, f reviewFacts, after int, now time.Time) ([]ports.Chunk, map[string]bool) {
	var cand []ports.Chunk
	in := map[string]bool{}
	for k, c := range firsts(latest(chunks)) {
		age, known := ageDays(c.Seen, now)
		due := pastReviewBy(c, now) // the document's own date, which a mark doesn't lift (SB-2705)
		stale := (known && age > after || due) && f.cited[k] >= ReviewCites
		m, marked := f.marks[k]
		rated := marked && m.Version == c.Version && f.unhelpful[k] >= UnhelpfulAfterMark
		if contains(c.Spaces, id) && (rated || (stale || f.flagged[k] > 0 || f.outdated[k] > 0) && (due || !covers(m, marked, c, f.flaggedAt[k], after, now))) {
			cand = append(cand, c)
			in[k] = true
		}
	}
	return cand, in
}

// outdatedOf counts, by document, the reports that it is out of date (SB-2703).
func outdatedOf(fbs []Feedback) map[string]int {
	out := map[string]int{}
	for _, f := range fbs {
		if f.OutdatedCitation != "" {
			out[docKey(f.OutdatedCitation)]++
		}
	}
	return out
}

// reviewCounts counts, by document, the answers that cited it and the wrong-citation
// marks on it, and lists every document either names.
func reviewCounts(traces []Trace, fbs []Feedback) (cited, flagged map[string]int, refs []ports.SourceRef) {
	cited, flagged = map[string]int{}, map[string]int{}
	for _, t := range traces {
		seen := map[string]bool{}
		for _, c := range t.Response.Answer.Claims {
			for _, ci := range c.Cites {
				if k := docKey(ci.Cite); !seen[k] {
					seen[k] = true
					cited[k]++
				}
			}
		}
	}
	for _, f := range fbs {
		if f.WrongCitation != "" {
			flagged[docKey(f.WrongCitation)]++
		}
	}
	for k := range cited {
		kind, id, _ := strings.Cut(k, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	for k := range flagged {
		if cited[k] == 0 {
			kind, id, _ := strings.Cut(k, ":")
			refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	return cited, flagged, refs
}

// personalIn counts the documents the space holds with personal data (SB-1904).
// heldFacts are the space's personal-data count (SB-1904) and the terms its documents
// define (SB-3407).
func (x Stewards) heldFacts(ctx context.Context, s tenancy.Scope, lk ports.DocLookup, name string) (int, []DefinedTerm, error) {
	n, err := x.personalIn(ctx, lk, name)
	if err != nil {
		return 0, nil, err
	}
	terms, err := x.termsDefined(ctx, s, lk, name)
	return n, terms, err
}

func (x Stewards) personalIn(ctx context.Context, lk ports.DocLookup, name string) (int, error) {
	personal, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Flag: ingest.FlagPersonal})
	if err != nil {
		return 0, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var withPII []ports.Chunk
	for _, c := range personal {
		if contains(c.Flags, ingest.FlagPersonal) {
			withPII = append(withPII, c)
		}
	}
	n := 0
	for _, c := range firsts(withPII) {
		if contains(c.Spaces, SpaceKey(name)) {
			n++
		}
	}
	return n, nil
}

// widerNotes are the space's notes whose readers are wider than the space is now
// (SB-2004), with how many readers each has to lose.
func (x Stewards) widerNotes(ctx context.Context, lk ports.DocLookup, name string) (map[string]int, []ports.Chunk, error) {
	by := map[string]int{}
	if x.Readers == nil {
		return by, nil, nil
	}
	ncs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Under: []ports.SourceRef{{Kind: noteKind, ID: "/notes/" + name + "/"}}})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	if len(ncs) == 0 {
		return by, nil, nil
	}
	ms, err := x.Readers(ctx, SpaceKey(name))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	members := map[string]bool{}
	for _, m := range ms {
		members[strings.ToLower(m)] = true
	}
	var notes []ports.Chunk
	for k, c := range firsts(latest(ncs)) {
		if n := wider(c, members); n > 0 {
			by[k] = n
			notes = append(notes, c)
		}
	}
	return by, notes, nil
}

func docKey(cite string) string {
	if i := strings.LastIndex(cite, "@"); i >= 0 {
		return cite[:i]
	}
	return cite
}

func (a Agent) reviewAfter(spaces []string) int {
	if a.ReviewAfter != nil && len(spaces) > 0 {
		if d := a.ReviewAfter(spaces); d > 0 {
			return d
		}
	}
	return DefaultReviewDays
}

// withOutdated adds the documents only an Outdated rating names to refs.
func withOutdated(refs []ports.SourceRef, outdated, cited, flagged map[string]int) []ports.SourceRef {
	for k := range outdated {
		if cited[k] == 0 && flagged[k] == 0 {
			kind, id, _ := strings.Cut(k, ":")
			refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	return refs
}

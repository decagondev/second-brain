package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Agents that act carefully (sb-agents(7), Start on a work event, Borrow access for one run;
// SB-5301 to SB-5303).

// Work events a space's on may list (sb-space(5), on), and the draft they make.
const (
	OnPROpened    = "pr-opened"
	OnTicketFiled = "ticket-filed"
	DraftWork     = "work-event"
	// OnMailReceived starts the space's agent on a new message in its shared mailbox, and
	// DraftMail is the reply note it drafts for a steward to send themselves (SB-5907).
	OnMailReceived = "mail-received"
	DraftMail      = "mail"

	// MaxWorkBatch is how many events of one kind in one space in a run are drafted one
	// by one; more are drafted as one batch.
	MaxWorkBatch = 10
	// TryDays is how far back Try it replays a trigger, and MaxTry how many it drafts.
	TryDays = 14
	MaxTry  = 5
)

// IsWorkEvent says chunk c (a document's first chunk) is the start of event on: a pull
// request on GitHub, or a ticket.
func IsWorkEvent(on string, c ports.Chunk) bool {
	switch on {
	case OnPROpened:
		return c.Kind == "code" && strings.Contains(c.SourceID, "/pulls/")
	case OnTicketFiled:
		return c.Kind == "tickets"
	}
	return false
}

// WorkQuestion is what a work event's draft asks, about one document or a batch of titles.
func WorkQuestion(on string, titles []string) string {
	what := "the pull request"
	if on == OnTicketFiled {
		what = "the ticket"
	}
	if len(titles) > 1 {
		what = fmt.Sprintf("these %d new items (%s)", len(titles), cut(strings.Join(titles, "; "), 1500))
	} else if len(titles) == 1 {
		what += " " + cut(titles[0], 200)
	}
	return "For " + what + ": what do our documents and decisions say that bears on it, who owns the areas it touches, and what should the reviewer check?"
}

// MailQuestion is what a new message's draft asks; the message itself goes with it as a
// file read for that ask alone, data and never instructions (SB-5907, LAW-09).
func MailQuestion(subject string) string {
	return "A new message to the shared mailbox, about " + cut(strings.TrimSpace(subject), 200) +
		", is attached. Draft a reply for a steward to send: what do our documents say that answers it, and what should the steward check before replying?"
}

// TryResult is what Try it shows: how many events in the window, those the steward can open
// drafted as they would be, how many they couldn't open, and the cost of the drafts.
type TryResult struct {
	Events  int        `json:"events"`
	Drafts  []TryDraft `json:"drafts"`
	Hidden  int        `json:"hidden"`
	CostUSD float64    `json:"cost_usd"`
}

// TryDraft is one replayed event.
type TryDraft struct {
	Doc   string `json:"doc"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Try replays trigger on of space name over docs (each a document's first chunk), as s:
// nothing is sent or kept but the probes' traces (SB-5302).
func (x Stewards) Try(ctx context.Context, s tenancy.Scope, name, on string, docs []ports.Chunk, cost func() float64) (TryResult, error) {
	out := TryResult{Drafts: []TryDraft{}}
	if on != OnPROpened && on != OnTicketFiled {
		return out, fmt.Errorf("%w: a trigger is %s or %s", ErrInvalid, OnPROpened, OnTicketFiled)
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return out, err
	}
	var cand []ports.Chunk
	for _, c := range docs {
		if c.Start == 0 && IsWorkEvent(on, c) && contains(c.Spaces, SpaceKey(name)) {
			cand = append(cand, c)
		}
	}
	out.Events = len(cand)
	if len(cand) == 0 {
		return out, nil
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	out.Hidden = len(cand) - len(res.Context)
	for _, c := range res.Context {
		if len(out.Drafts) == MaxTry {
			break
		}
		title := c.Title
		if title == "" {
			title = c.SourceID
		}
		resp, err := x.Agent.ask(ctx, s, Request{Question: WorkQuestion(on, []string{title}), Documents: []string{c.Kind + ":" + c.SourceID}}, true)
		if err != nil {
			return out, err
		}
		out.Drafts = append(out.Drafts, TryDraft{Doc: c.Kind + ":" + c.SourceID, Title: title, Text: strings.TrimRight(DraftText(resp.Answer), "\n")})
	}
	if cost != nil {
		out.CostUSD = cost()
	}
	return out, nil
}

// Borrowed access (LAW-31): a loan is a steward's request, in the lender's own queue, to
// read one space as the lender for one run. The run's draft lands in the steward's queue,
// names the lender, and shows to a reader only when they can open every cite.

// Kinds of a loan request and the draft a loan makes.
const (
	DraftLoan = "loan"
	DraftLent = "lent"

	MaxLoanPurpose = 300
)

// LoanRequest is a steward's request: who to ask, which space to read as them, and why.
type LoanRequest struct {
	Lender  string `json:"lender"`
	Space   string `json:"space"`
	Purpose string `json:"purpose"`
}

// LentVisible says a reader sees a lent run's draft: they can open every one of its cites
// (LAW-31). A draft with no cites is shown.
func LentVisible(cites []string, canOpen func(cite string) bool) bool {
	for _, c := range cites {
		if !canOpen(c) {
			return false
		}
	}
	return true
}

// RequestLoan puts steward s's loan request in the lender's own queue.
func (x Stewards) RequestLoan(ctx context.Context, s tenancy.Scope, name string, req LoanRequest) (StewardQuestion, error) {
	req.Purpose = strings.Join(strings.Fields(req.Purpose), " ")
	if req.Purpose == "" || utf8.RuneCountInString(req.Purpose) > MaxLoanPurpose {
		return StewardQuestion{}, fmt.Errorf("%w: a purpose is 1 to %d characters", ErrInvalid, MaxLoanPurpose)
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return StewardQuestion{}, err
	}
	if x.Owner == nil || x.Facts == nil {
		return StewardQuestion{}, fmt.Errorf("%w: this cell can't lend access", ErrUnavailable)
	}
	if _, _, ok := x.Facts(SpaceKey(req.Space)); !ok || req.Space == name {
		return StewardQuestion{}, fmt.Errorf("%w: lend another space this cell has", ErrInvalid)
	}
	lender, ok, err := x.Owner(ctx, strings.TrimSpace(req.Lender))
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if !ok || lender == s.User() {
		return StewardQuestion{}, fmt.Errorf("%w: the lender is one active user other than you", ErrInvalid) // LAW-12
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Space: PersonKey(lender), At: x.Now().UTC(),
		Question: fmt.Sprintf("Lend %s for one run, for %s: %s", req.Space, name, req.Purpose),
		Thread:   &SavedThread{Agent: "ask", Kind: DraftLoan, Asked: req.Purpose, Doc: SpaceKey(req.Space), Link: SpaceKey(name)}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "loan_requested",
		PayloadSHA256: hashOf(req.Purpose), Reason: "loan of " + req.Space + " for one run in " + name, Space: auditSpace(SpaceKey(name))}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// Lend is the lender's answer to loan request id: on yes, one run reads the lent space as
// them, and its draft goes to the steward's queue naming them; on no, nothing runs. Either
// way the request is closed, so a loan is never used twice. as is the lender's scope in the
// lent space.
func (x Stewards) Lend(ctx context.Context, s tenancy.Scope, id string, yes bool, as func(space string) (tenancy.Scope, error)) (StewardQuestion, error) {
	q, found, err := x.Store.GetStewardQuestion(ctx, x.Agent.Tenant, id)
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if !found || q.Space != PersonKey(s.User()) || q.Thread == nil || q.Thread.Kind != DraftLoan || q.Reply != nil || q.Thread.Discarded {
		return StewardQuestion{}, fmt.Errorf("%w: no open loan request %q for you", ErrInvalid, id)
	}
	state := "loan_declined"
	if yes {
		state = "loan_lent"
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: state,
		Reason: "loan of " + strings.TrimPrefix(q.Thread.Doc, "spc_") + " for one run"}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	closed, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(cur *StewardQuestion) error {
		if cur.Reply != nil {
			return fmt.Errorf("%w: this loan was answered already", ErrInvalid)
		}
		text := "Declined."
		if yes {
			text = "Lent for one run."
		}
		cur.Reply = &StewardReply{Text: text, At: x.Now().UTC(), By: s.User()}
		return nil
	})
	if err != nil || !yes {
		return closed, err
	}
	ls, err := as(q.Thread.Doc)
	if err != nil {
		return closed, fmt.Errorf("%w: you can't read %s now", ErrInvalid, strings.TrimPrefix(q.Thread.Doc, "spc_"))
	}
	resp, err := x.Agent.ask(ctx, ls, Request{Question: q.Thread.Asked}, true) // the one run, as the lender in the lent space
	if err != nil {
		return closed, err
	}
	email := x.emailOf(ctx, s.User())
	draft := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: q.Asker, Space: q.Thread.Link, At: x.Now().UTC(), Trace: resp.Trace,
		Question: "Lent by " + email + ": " + cut(q.Thread.Asked, 200),
		Thread: &SavedThread{Agent: "ask", Kind: DraftLent, Text: strings.TrimRight(DraftText(resp.Answer), "\n"), Sources: citesOf(resp), Asked: q.Thread.Asked,
			Release: resp.Release, Lender: email}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: draft.ID, State: "lent_run",
		PayloadSHA256: hashOf(draft.Thread.Text), Reason: "one run as the lender for loan " + id, Space: auditSpace(q.Thread.Link)}); err != nil {
		return closed, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, draft); err != nil {
		return closed, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return closed, nil
}

// citesOf are an answer's cites, each once.
func citesOf(r Response) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range r.Answer.Claims {
		for _, ci := range c.Cites {
			if !seen[ci.Cite] {
				seen[ci.Cite] = true
				out = append(out, ci.Cite)
			}
		}
	}
	return out
}

// lentHidden says viewer s may not see a lent run's draft: some cite of it they can't open,
// checked live (LAW-31). Anything that can't be checked hides it.
func (x Stewards) lentHidden(ctx context.Context, s tenancy.Scope, t *SavedThread) bool {
	if t == nil || t.Kind != DraftLent || len(t.Sources) == 0 {
		return false
	}
	lk, ok := x.Agent.Index.(ports.CiteLookup)
	if !ok {
		return true
	}
	chunks, err := lk.ChunksOf(ctx, x.Agent.Tenant, t.Sources)
	if err != nil {
		return true
	}
	firsts := map[string]ports.Chunk{}
	for _, c := range chunks {
		if _, ok := firsts[c.Cite]; !ok {
			firsts[c.Cite] = c
		}
	}
	var cand []ports.Chunk
	for _, c := range t.Sources {
		ch, ok := firsts[c]
		if !ok {
			return true // a cite gone from the index can't be checked
		}
		cand = append(cand, ch)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return true
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return true
	}
	open := map[string]bool{}
	for _, c := range res.Context {
		open[c.Cite] = true
	}
	return !LentVisible(t.Sources, func(c string) bool { return open[c] })
}

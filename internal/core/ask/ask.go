// Package ask is the ask agent (sb-ask(3), "How the ask agent runs, exactly"; SB-306):
// validate, retrieve and check live, compose through the model gateway, and keep a trace.
// Every entry point (web UI, chat app, API) calls Agent.Ask.
package ask

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// MaxQuestion is the longest question, in characters.
const MaxQuestion = 4000

// TraceTTL is how long traces are kept.
const TraceTTL = 30 * 24 * time.Hour

// Kinds are the source kinds a request may restrict to.
var Kinds = map[string]bool{"docs": true, "chat": true, "tickets": true, "crm": true, "code": true, "hris": true, "accounting": true}

// Notices.
const (
	NoticeMostUncited = "most_uncited"
	NoticeNoAccess    = "no_access"
	NoticeNoSources   = "no_sources"
)

// NothingToCite is the one claim of an answer with no allowed context.
const NothingToCite = "I couldn't find anything you have access to that answers this."

// Request is the body of POST /v1/ask.
type Request struct {
	Question     string   `json:"question"`
	Conversation string   `json:"conversation,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	Agent        string   `json:"agent,omitempty"` // "" or ask, or a department agent (SB-1102)
}

// Answer is the claims of a response.
type Answer struct {
	Claims []answer.Claim `json:"claims"`
}

// Response is the 200 body of POST /v1/ask.
type Response struct {
	Trace        string  `json:"trace"`
	Release      string  `json:"release"`
	Answer       Answer  `json:"answer"`
	Refused      int     `json:"refused"`
	Notice       *string `json:"notice"`
	Conversation string  `json:"conversation"`
	// Agent is the agent that answered, and Dropped the uncited claims a department agent
	// left out (sb-agents(7), Department agents); 0 for ask.
	Agent   string `json:"agent"`
	Dropped int    `json:"dropped"`
}

// Trace is a stored answer with its decision list (GET /v1/traces/{id}).
type Trace struct {
	ID, Tenant, User string
	At               time.Time
	Response         Response
	Decisions        []retrieve.Decision
}

// Store keeps traces and conversations for one cell.
type Store interface {
	PutTrace(ctx context.Context, t Trace) error
	GetTrace(ctx context.Context, tenant, id string) (Trace, bool, error)
	// Questions returns the earlier questions of a user's conversation; found is false for
	// an id that doesn't exist or belongs to someone else.
	Questions(ctx context.Context, tenant, user, conversation string) (qs []string, found bool, err error)
	AppendQuestion(ctx context.Context, tenant, user, conversation, question string) error
}

// Errors, mapped by the API to sb-errors(7) codes.
var (
	ErrInvalid     = errors.New("ask: EINVAL")
	ErrKilled      = errors.New("ask: EKILLED")
	ErrUnavailable = errors.New("ask: EUNAVAIL")
	ErrNotFound    = errors.New("ask: no such trace for this user") // EPERM
	ErrBudget      = errors.New("ask: EBUDGET")
)

// Agent is the ask agent for one cell.
type Agent struct {
	Tenant   string
	Release  string
	Index    ports.Retriever
	Embedder ports.Embedder // nil: keyword ranking only
	Auth     retrieve.Authoriser
	Model    ports.Model // the gateway, never a bare adapter
	Store    Store
	// Emails returns the asker's addresses for prefilter tokens; nil means none.
	Emails func(ctx context.Context, user string) ([]string, error)
	// Killed reports a kill switch on the ask agent or the tenant (W5 wires the real one).
	Killed func(ctx context.Context) bool
	// SpaceKilled reports a killed space, by name (SB-1008); nil is a cell without spaces.
	SpaceKilled func(name string) bool
	// Spoof logs one stripped citation label (citation_spoof), with no claim text.
	Spoof func(ctx context.Context, trace, label string)
	NewID func(prefix string) string
	Now   func() time.Time
	// Link returns a URL that opens a chunk's source at its version, or "" (sb-ask(3)).
	Link func(c ports.Chunk) string
	// Metrics records ask latency and outcomes (sb_ask_seconds, sb_asks_total), ids only.
	Metrics interface {
		Add(name, help string, labels map[string]string, v float64)
		Observe(name, help string, labels map[string]string, v float64)
	}
	// IsAuditor says whether a scope may read every trace of this tenant; nil means nobody.
	IsAuditor func(s tenancy.Scope) bool
	// Department agents (SB-1102): which agents the manifest enables, the space a scope
	// is in, and the agent kill switches. With any of them nil, no department agent runs.
	Enabled     func(agent string) bool
	SpaceOf     func(ctx context.Context, id string) (ports.Space, bool, error)
	AgentKilled func(agent string) bool
}

// Ask answers one question for the scope's user.
func (a Agent) Ask(ctx context.Context, s tenancy.Scope, req Request) (resp Response, err error) {
	if a.Metrics != nil && a.Now != nil {
		start := a.Now()
		defer func() {
			outcome := "ok"
			switch {
			case err != nil:
				outcome = "error"
			case resp.Notice != nil && (*resp.Notice == NoticeNoAccess || *resp.Notice == NoticeNoSources):
				outcome = "refused"
			}
			l := map[string]string{"tenant": a.Tenant}
			a.Metrics.Observe("sb_ask_seconds", "Ask latency in seconds.", l, a.Now().Sub(start).Seconds())
			a.Metrics.Add("sb_asks_total", "Answers by outcome.", map[string]string{"tenant": a.Tenant, "outcome": outcome}, 1)
		}()
	}
	return a.ask(ctx, s, req)
}

func (a Agent) ask(ctx context.Context, s tenancy.Scope, req Request) (Response, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return Response{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	q := strings.TrimSpace(req.Question)
	if q == "" || utf8.RuneCountInString(q) > MaxQuestion {
		return Response{}, fmt.Errorf("%w: question must be 1 to %d characters", ErrInvalid, MaxQuestion)
	}
	for _, k := range req.Sources {
		if !Kinds[k] {
			return Response{}, fmt.Errorf("%w: unknown source kind %q", ErrInvalid, k)
		}
	}
	if a.Killed == nil || a.Killed(ctx) {
		return Response{}, ErrKilled // no kill-switch reader is treated as killed
	}
	if spaceKilled(a.SpaceKilled, s) {
		return Response{}, ErrKilled
	}
	var dept *Department
	if req.Agent != "" && req.Agent != "ask" {
		d, err := a.department(ctx, s, req.Agent)
		if err != nil {
			return Response{}, err
		}
		dept = &d
	}
	conv := req.Conversation
	var earlier []string
	if conv != "" {
		qs, found, err := a.Store.Questions(ctx, a.Tenant, s.User(), conv)
		if err != nil {
			return Response{}, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
		}
		if !found {
			return Response{}, fmt.Errorf("%w: no such conversation", ErrInvalid)
		}
		earlier = qs
	} else {
		conv = a.NewID("cv_")
	}

	var emails []string
	if a.Emails != nil {
		e, err := a.Emails(ctx, s.User())
		if err != nil {
			return Response{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		emails = e
	}
	query := ports.Query{Text: q, Kinds: req.Sources}
	if a.Embedder != nil {
		vs, err := a.Embedder.Embed(ctx, []string{q})
		if err != nil || len(vs) != 1 {
			return Response{}, fmt.Errorf("%w: embedder", ErrUnavailable)
		}
		query.Vector = vs[0]
	}
	// Connectors ask the source about the person by email (sb-connector(3), ACL), so the
	// user reference carries the directory's primary email (BUG-0007).
	who := ports.UserRef{ID: s.User()}
	if len(emails) > 0 {
		who.Email = emails[0]
	}
	res, err := retrieve.Context(ctx, s, who, emails, a.Index, a.Auth, query)
	if err != nil {
		return Response{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}

	trace := a.NewID("tr_")
	resp := Response{Trace: trace, Release: a.Release, Refused: res.Refused, Conversation: conv, Agent: "ask"}
	if dept != nil {
		resp.Agent = dept.Name
	}
	if len(res.Context) == 0 {
		n := NoticeNoSources
		if res.Refused > 0 {
			n = NoticeNoAccess
		}
		resp.Notice = &n
		resp.Answer.Claims = []answer.Claim{{Text: NothingToCite, Cites: []answer.Cite{}, Uncited: true}}
	} else {
		prompt, lead := "answer", "Question: "
		if dept != nil {
			prompt, lead = dept.Prompt, dept.Lead
		}
		claims, err := a.composeWith(withUser(ctx, s.User()), prompt, lead, trace, q, earlier, res.Context)
		if err != nil {
			return Response{}, err
		}
		if dept != nil {
			claims, resp.Dropped = citedOnly(claims)
		}
		resp.Answer.Claims = claims
		uncited := 0
		for _, c := range claims {
			if c.Uncited {
				uncited++
			}
		}
		if uncited*2 > len(claims) {
			n := NoticeMostUncited
			resp.Notice = &n
		}
	}

	if err := a.Store.AppendQuestion(ctx, a.Tenant, s.User(), conv, q); err != nil {
		return Response{}, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	t := Trace{ID: trace, Tenant: a.Tenant, User: s.User(), At: a.Now(), Response: resp, Decisions: res.Decisions}
	if err := a.Store.PutTrace(ctx, t); err != nil {
		return Response{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err) // no trace, no answer
	}
	return resp, nil
}

func (a Agent) composeWith(ctx context.Context, prompt, lead, trace, q string, earlier []string, chunks []ports.Chunk) ([]answer.Claim, error) {
	p, err := prompts.Get(prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	actx := answer.NewContext(chunks)
	var msgs []ports.Message
	for _, e := range earlier {
		msgs = append(msgs, ports.Message{Role: "user", Content: e})
		msgs = append(msgs, ports.Message{Role: "assistant", Content: "(earlier answer withheld; sources are checked again)"})
	}
	msgs = append(msgs, ports.Message{Role: "user", Content: actx.Data() + "\n" + lead + q})
	out, err := a.Model.Complete(ctx, ports.ModelRequest{System: p.Text, Messages: msgs, JSON: true, User: a.user(ctx)})
	if err != nil {
		if errors.Is(err, budget.ErrBudget) {
			return nil, fmt.Errorf("%w: %v", ErrBudget, err)
		}
		return nil, fmt.Errorf("%w: model: %v", ErrUnavailable, err)
	}
	res := answer.Compose(out.Text, actx, nil)
	if a.Link != nil {
		byCite := map[string]ports.Chunk{}
		for _, ch := range chunks {
			byCite[ch.Cite] = ch
		}
		for i := range res.Claims {
			for j := range res.Claims[i].Cites {
				res.Claims[i].Cites[j].URL = a.Link(byCite[res.Claims[i].Cites[j].Cite])
			}
		}
	}
	for _, l := range res.Spoofs {
		if a.Spoof != nil {
			a.Spoof(ctx, trace, l)
		}
	}
	return res.Claims, nil
}

// Trace returns a stored trace to the user who asked, or to an auditor.
func (a Agent) Trace(ctx context.Context, s tenancy.Scope, id string) (Trace, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return Trace{}, ErrNotFound
	}
	t, found, err := a.Store.GetTrace(ctx, a.Tenant, id)
	if err != nil {
		return Trace{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	mayRead := t.User == s.User() || (a.IsAuditor != nil && a.IsAuditor(s))
	if !found || !mayRead || t.Tenant != s.Tenant() || a.Now().Sub(t.At) > TraceTTL {
		return Trace{}, ErrNotFound
	}
	return t, nil
}

// spaceKilled is a request in a space whose kill switch is set (sb-kill-switches(7)).
func spaceKilled(killed func(string) bool, s tenancy.Scope) bool {
	sp := s.Space()
	return sp != "company" && killed != nil && killed(strings.TrimPrefix(sp, "spc_"))
}

type userKey struct{}

// withUser carries the asker to the model call for the budget's rate limit.
func withUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

func (Agent) user(ctx context.Context) string {
	u, _ := ctx.Value(userKey{}).(string)
	return u
}

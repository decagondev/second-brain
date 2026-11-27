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

	"crypto/sha256"
	"encoding/hex"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/gateway"
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
	Question     string `json:"question"`
	Conversation string `json:"conversation,omitempty"`
	// Thread is a chat thread the ask is a reply in (SB-5801): the chat app's own key for it,
	// never sent by the web. One conversation per thread and person, kept for them alone.
	Thread  string   `json:"-"`
	Sources []string `json:"sources,omitempty"`
	Agent   string   `json:"agent,omitempty"` // "" or ask, or a department agent (SB-1102)
	// Language is the language to write the claims in: en, de or fr (SB-3003). Anything
	// else is ignored.
	Language string `json:"language,omitempty"`
	// Documents narrows the ask to at most MaxDocuments documents, as KIND:ID (SB-3004).
	Documents []string `json:"documents,omitempty"`
	// Length "short" asks for at most MaxShort claims (SB-3303); anything else is ignored.
	Length string `json:"length,omitempty"`
	// Verified answers only from documents with a live review mark, and notes (SB-3602).
	Verified bool `json:"verified,omitempty"`
	// File is one file this ask is about, read for it alone and kept nowhere (SB-3802).
	File *FileUpload `json:"file,omitempty"`
	// Collection asks about the documents of one collection the asker may read (SB-4304).
	Collection string `json:"collection,omitempty"`
	// Consult also asks the department agents of the branch's teams (SB-4501): a branch
	// ask only, never inside a consultation (LAW-27).
	Consult bool `json:"consult,omitempty"`
	// Clarify says the caller can show one clarifying question, and Choice is the asker's
	// answer to it (SB-4901).
	Clarify bool   `json:"clarify,omitempty"`
	Choice  string `json:"choice,omitempty"`
	// Keep and Drop are the conversation's kept and dropped documents, as KIND:ID, sent by
	// the browser with each ask (SB-5502); each kept one is still checked as the asker.
	Keep []string `json:"keep,omitempty"`
	Drop []string `json:"drop,omitempty"`
	// Fresh is Ask fresh: never reuse an earlier answer (SB-5603).
	Fresh bool `json:"fresh,omitempty"`
}

// MaxShort is the most claims a short answer keeps (SB-3303), and MaxSeeAlso the most
// documents See also lists (SB-3301).
const (
	MaxShort   = 3
	MaxSeeAlso = 3
)

// ShortLine is the answer prompt's extra line for a short answer.
const ShortLine = "\n\nGive at most 3 claims, the most important first."

// SeeAlso is a document the live check allowed for this answer that no claim cites
// (SB-3301): never its text.
type SeeAlso struct {
	Cite    string `json:"cite"`
	Title   string `json:"title,omitempty"`
	Section string `json:"section,omitempty"`
	Span    [2]int `json:"span"`
	AgeDays *int   `json:"age_days,omitempty"`
}

// seeAlso is at most MaxSeeAlso documents of chunks, in their order, that no claim cites.
func seeAlso(claims []answer.Claim, chunks []ports.Chunk, now time.Time) []SeeAlso {
	cited := map[string]bool{}
	for _, c := range claims {
		for _, ci := range c.Cites {
			d, _, _ := strings.Cut(ci.Cite, "@")
			cited[d] = true
		}
	}
	var out []SeeAlso
	for _, ch := range chunks {
		d, _, _ := strings.Cut(ch.Cite, "@")
		if cited[d] || len(out) == MaxSeeAlso || ch.Kind == FileKind {
			continue
		}
		cited[d] = true
		sa := SeeAlso{Cite: ch.Cite, Title: ch.Title, Section: ch.Section, Span: [2]int{ch.Start, ch.End}}
		if d, ok := ageDays(ch.Seen, now); ok {
			sa.AgeDays = &d
		}
		out = append(out, sa)
	}
	return out
}

// MaxDocuments is the most documents one ask may be narrowed to (SB-3004).
const MaxDocuments = 5

// validate checks an ask's question, sources and documents.
func validate(req Request) (string, []string, error) {
	q := strings.TrimSpace(req.Question)
	if q == "" || utf8.RuneCountInString(q) > MaxQuestion {
		return "", nil, fmt.Errorf("%w: question must be 1 to %d characters", ErrInvalid, MaxQuestion)
	}
	for _, k := range req.Sources {
		if !Kinds[k] {
			return "", nil, fmt.Errorf("%w: unknown source kind %q", ErrInvalid, k)
		}
	}
	docs, err := documentKeys(req.Documents)
	return q, docs, err
}

// documentKeys checks an ask's documents and returns them as KIND:ID, any version dropped.
func documentKeys(ds []string) ([]string, error) {
	if len(ds) > MaxDocuments {
		return nil, fmt.Errorf("%w: at most %d documents", ErrInvalid, MaxDocuments)
	}
	var out []string
	for _, d := range ds {
		d, _, _ = strings.Cut(strings.TrimSpace(d), "@")
		kind, id, ok := strings.Cut(d, ":")
		if !ok || !Kinds[kind] || id == "" {
			return nil, fmt.Errorf("%w: documents must be KIND:ID", ErrInvalid)
		}
		out = append(out, d)
	}
	return out, nil
}

// Languages are the languages an answer can be asked for, as the prompt names them.
var Languages = map[string]string{"en": "English", "de": "German", "fr": "French"}

// LanguageLine is the answer prompt's extra line for a language, or "" for none.
func LanguageLine(code string) string {
	name, ok := Languages[code]
	if !ok {
		return ""
	}
	return "\n\nWrite each claim's text in " + name + ". Keep every label exactly as given."
}

// Answer is the claims of a response.
type Answer struct {
	Claims []answer.Claim `json:"claims"`
}

// Response is the 200 body of POST /v1/ask.
type Response struct {
	Trace   string `json:"trace"`
	Release string `json:"release"`
	Answer  Answer `json:"answer"`
	Refused int    `json:"refused"`
	// Skipped is how many allowed passages were kept out of the prompt as suspect (SB-5101).
	Skipped int `json:"skipped,omitempty"`
	// Held is how many documents the asker could open were left out as shared too widely
	// and not yet confirmed by their owner, and WideWarned how many were used with a warning
	// in a space that warns (SB-5402). Never their titles.
	Held         int     `json:"held,omitempty"`
	WideWarned   int     `json:"wide_warned,omitempty"`
	Notice       *string `json:"notice"`
	Conversation string  `json:"conversation"`
	// Agent is the agent that answered, and Dropped the uncited claims a department agent
	// left out (sb-agents(7), Department agents); 0 for ask.
	Agent   string `json:"agent"`
	Dropped int    `json:"dropped"`
	// EarlierDropped is how many earlier questions of the conversation the model didn't
	// read: only the last MaxEarlier are sent (SB-2901).
	EarlierDropped int `json:"earlier_dropped"`
	// Suggest is the one department space whose agent could answer this better (SB-4007).
	Suggest *Suggestion `json:"suggest,omitempty"`
	// People is who to ask: the owners of the answer's cited documents (SB-1702).
	People []PersonRef `json:"people"`
	// Terms are the space's glossary entries the question used (SB-2203).
	Terms []Term `json:"terms,omitempty"`
	// NoticeStale and NoticeSync warn that most cited documents are stale, or that a
	// connector of a cite is behind on syncing (SB-2701, SB-2704).
	NoticeStale *StaleNotice `json:"notice_stale,omitempty"`
	NoticeSync  []SyncNotice `json:"notice_sync,omitempty"`
	// NoticeIndexing names the connectors of the asked space still in their first sync
	// (SB-6004): the answer may be missing what they haven't read yet.
	NoticeIndexing []string `json:"notice_indexing,omitempty"`
	// NoticeSingle says every cite of an answer with two or more cited claims is one
	// document (SB-3005).
	NoticeSingle bool `json:"notice_single,omitempty"`
	// SeeAlso are documents the search found and the asker can open that no claim cites
	// (SB-3301).
	SeeAlso []SeeAlso `json:"see_also,omitempty"`
	// Pinned is the stewards' verified answer to a question like this one (SB-3601).
	Pinned *Pinned `json:"pinned,omitempty"`
	// Verified counts the cited documents and those someone has checked (SB-3607).
	Verified *VerifiedCount `json:"verified,omitempty"`
	// Table is the table tool's result, when the ask used it (SB-4307).
	Table *TableResult `json:"table,omitempty"`
	// Departments are the consulted teams' sections, and NotConsulted the teams that
	// weren't and why (SB-4501).
	Departments  []Consultation `json:"departments,omitempty"`
	NotConsulted []NotConsulted `json:"not_consulted,omitempty"`
	// Handoff is the space whose request category the question names (SB-4502).
	Handoff *Handoff `json:"handoff,omitempty"`
	// Skill is the skill the question loaded (SB-4601).
	Skill string `json:"skill,omitempty"`
	// Clarify is the one question an ambiguous ask gets instead of an answer, and Assumed
	// the line an answer that couldn't ask starts with (SB-4901).
	Clarify *Clarify `json:"clarify,omitempty"`
	Assumed string   `json:"assumed,omitempty"`
	// Label is the highest label among the answer's cites (SB-5405, LAW-32): whatever is
	// built from it carries it, and confidential or above goes only to a person.
	Label string `json:"label,omitempty"`
	// Memory are the ids of the asker's memory lines the answer was given (SB-5504).
	Memory []string `json:"memory,omitempty"`
	// Fallback says the fallback model wrote it (SB-5601), and Passages are the best
	// passages the asker can open when no model could (SB-5602): never an answer.
	Fallback bool     `json:"fallback,omitempty"`
	Passages []Result `json:"passages,omitempty"`
	// Reused is the day an earlier answer to the same question was written, reused because
	// every document it read is unchanged and open to the asker now (SB-5603).
	Reused string `json:"reused,omitempty"`
}

// oneDocument says an answer of at least two cited claims cites one document only.
func oneDocument(claims []answer.Claim) bool {
	doc, cited := "", 0
	for _, c := range claims {
		if c.Uncited || len(c.Cites) == 0 {
			continue
		}
		cited++
		for _, ci := range c.Cites {
			d, _, _ := strings.Cut(ci.Cite, "@")
			if doc == "" {
				doc = d
			} else if d != doc {
				return false
			}
		}
	}
	return cited >= 2
}

// MaxEarlier is the most earlier questions of a conversation the model reads (SB-2901), and
// MaxPerDocument the most chunks of one document it reads for one answer (SB-2902).
const (
	MaxEarlier     = 8
	MaxPerDocument = 3
)

// recent is the last MaxEarlier of a conversation's earlier questions, and how many before
// them were left out.
func recent(earlier []string) ([]string, int) {
	if len(earlier) <= MaxEarlier {
		return earlier, 0
	}
	return earlier[len(earlier)-MaxEarlier:], len(earlier) - MaxEarlier
}

// Trace is a stored answer with its decision list (GET /v1/traces/{id}).
type Trace struct {
	ID, Tenant, User string
	Space            string // the space it was asked in, spc_NAME or company (SB-1207)
	At               time.Time
	// QuerySHA256 is a search's query, hashed (SB-1701); a search keeps no text.
	QuerySHA256 string `json:",omitempty"`
	// Turn is the answer's question's place in its conversation, from 0 (SB-2204).
	Turn      int `json:",omitempty"`
	Response  Response
	Decisions []retrieve.Decision
	// Files is how many files the ask was about (SB-3802): never their name or text.
	Files int `json:",omitempty"`
	// Tier is the model tier that answered, small or main, when the cell has a small
	// model (SB-4405), or deep (SB-5704), and TierWhy why the deep tier was used.
	Tier    string `json:",omitempty"`
	TierWhy string `json:",omitempty"`
	// FlaggedFiles is 1 when the file's passages carried an injection flag (SB-4406).
	FlaggedFiles int `json:",omitempty"`
	// CostUSD is this one answer's model spend, its consultations' within it (SB-4606).
	CostUSD float64 `json:",omitempty"`
	// Feature is what asked it, one of Features or eval (SB-5001); "" is an older ask.
	Feature string `json:",omitempty"`
	// Client is where the ask came from: web, chat, mcp or token (SB-5305), and Tool the AI
	// tool its token was made for (SB-6006), "" for none.
	Client string `json:",omitempty"`
	Tool   string `json:",omitempty"`
	// Suspect are the cites of the answer's suspect passages, skipped or monitored
	// (SB-5101): never their text.
	Suspect []string `json:",omitempty"`
	// Collection, Documents, Kept and Dropped are what the ask was narrowed or steered by
	// (SB-5501, SB-5502), as ids.
	Collection               string   `json:",omitempty"`
	Documents, Kept, Dropped []string `json:",omitempty"`
	// Read are the documents the answer was composed from, as KIND:ID@VERSION (SB-5603).
	Read []string `json:",omitempty"`
	// Saved is what reuse, batching, caching and the economy tier took off this answer's
	// cost, in dollars by why (SB-5606).
	Saved map[string]float64 `json:",omitempty"`
	// InputTokens and CachedTokens are the input its model calls sent and the part the
	// provider read from its prompt cache, and Prefix a short hash of each part of its
	// stable prefix, never the text (SB-5706).
	InputTokens  int               `json:",omitempty"`
	CachedTokens int               `json:",omitempty"`
	Prefix       map[string]string `json:",omitempty"`
}

// SpaceTagger is a store that keeps the space a conversation is asked in, so a legal hold
// on one space keeps only that space's conversations (SB-1207, SB-1401's follow-up).
type SpaceTagger interface {
	TagSpace(ctx context.Context, tenant, conversation, space string) error
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
	// ErrPending is a scheduled step whose model call went to a batch this tick: it runs
	// again on a later tick and gets the result then (SB-5701). Never a person's ask.
	ErrPending = errors.New("ask: batch pending")
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
	Enabled func(agent string) bool
	SpaceOf func(ctx context.Context, id string) (ports.Space, bool, error)
	// AgentPaused says an agent is paused in a space, by name (SB-4702); nil pauses none.
	AgentPaused func(agent, space string) bool
	// AgentCapped says an agent is at its monthly cap (SB-4703); an error is spend that
	// can't be read, which stops its runs too. nil caps none.
	AgentCapped func(ctx context.Context, agent string) (bool, error)
	// Small is the small model tier (SB-4405), nil for none; SmallHeld says the evals
	// hold it (an unreadable hold is held), and PinMain that a space, by key, pins main.
	Small     ports.Model
	SmallHeld func() bool
	// Deep is the deep tier (SB-5704), nil for none, used only while DeepGated says the evals
	// job passed it for this release (nil or false is never).
	Deep        ports.Model
	DeepGated   func() bool
	PinMain     func(space string) bool
	AgentKilled func(agent string) bool
	// Routing is a space's routing policy by key (SB-5002), nil for balanced everywhere;
	// PersonOver says a person has reached their monthly limit (SB-5003), nil for none.
	Routing func(space string) string
	// Suspect is a space's suspect mode by key (SB-5101), nil for skip everywhere; Guard
	// records each answer's suspect passages as guardrail events (SB-5103), nil none.
	Suspect    func(space string) string
	Guard      func(ctx context.Context, e GuardEvent)
	PersonOver func(ctx context.Context, user string) (bool, error)
	// Departmental are the spaces the scope's user is a member of that run a department
	// agent, for the right-department suggestion (SB-4007); nil suggests none.
	Departmental func(ctx context.Context, s tenancy.Scope) []DeptSpace
	// Updates says whether a space, by name, sets updates: true (SB-4102); nil is none.
	Updates func(space string) bool
	// Owner finds the one active directory user with an email (SB-1702; ActiveOwner); nil
	// shows no owners.
	Owner func(ctx context.Context, email string) (answer.Person, bool)
	// Caretaker is a document's (KIND:ID) caretaker's email, or "" (SB-3405); nil is none.
	Caretaker func(ctx context.Context, doc string) string
	// Climb is who to ask when no owner of an answer's cites is active: a team's
	// department's stewards (SB-3906), for a space key; nil climbs nowhere.
	Climb func(ctx context.Context, space string) []PersonRef
	// ReviewAfter is the shortest review_after_days of the spaces holding a document, by
	// their ids (SB-1709); nil, or no spaces, is the default 365.
	ReviewAfter func(spaces []string) int
	// Glossary is a space's glossary by name (sb-space(5); SB-2203); nil is none.
	Glossary func(space string) map[string]string
	// Rate counts each person's asks; AsksPerMinute is the manifest's limit, 0 or nil
	// for DefaultAsksPerMinute (SB-2305). A nil Rate limits nothing (tests and evals).
	Rate          *Rate
	AsksPerMinute func() int
	// Marks are the stewards' review marks, shown on a cite (SB-2702); nil is none.
	Marks MarkStore
	// SyncLag names the connectors of an answer's cites that are behind (SB-2704); nil is none.
	SyncLag SyncLag
	// Indexing names the connectors a space (spc_NAME, or "" for the company) reads that
	// are still in their first sync (SB-6004); nil is none.
	Indexing func(ctx context.Context, space string) []string
	// FileText reads an uploaded file's text by its name's extension, ok false for a type
	// the folder connector doesn't read (SB-3802); nil reads no files.
	FileText func(name string, b []byte) (string, bool)
	// Retired keeps the documents stewards took out of their spaces' answers (SB-2801);
	// nil is none.
	Retired RetireStore
	// Pins are the stewards' verified answers (SB-3601); nil is none.
	Pins PinStore
	// Memory keeps what each person wrote about themselves (SB-4303); nil is a cell with
	// memory: off. Audit writes a row for each change to it.
	Memory MemoryStore
	Audit  Auditor
	// Collections are the cell's collections, for an ask about one (SB-4304); nil keeps none.
	Collections CollectionStore
	// Requests are a space's request categories, by name, for Hand off (SB-4502); nil
	// offers none.
	Requests func(space string) []string
	// LeaderWeek says whether a space, by name, gets the leader's week (SB-4503); nil is none.
	LeaderWeek func(space string) bool
	// Skills are a space's skills by name, its department's with them (SB-4601); nil is none.
	Skills func(space string) []Skill
	// Reuse keeps the answers an ask may reuse (SB-5603); nil reuses none.
	Reuse ReuseStore
	// TracesOf are a person's traces from a time on, for their conversations (SB-5802);
	// nil lists none.
	TracesOf func(ctx context.Context, user string, from time.Time) ([]Trace, error)
	// Exclusions keeps each person's Don't use this for me with their memory record
	// (SB-5503), whatever the manifest's memory setting; nil excludes nothing.
	Exclusions MemoryStore
	// Wide is the cell's sharing checks (SB-5401 to SB-5403); nil checks nothing.
	Wide *Wide
	// Everyone are the tenant's active people, for who can see a document (SB-5404); nil
	// counts no one.
	Everyone func(ctx context.Context) ([]ports.UserRef, error)
}

// Ask answers one question for the scope's user.
func (a Agent) Ask(ctx context.Context, s tenancy.Scope, req Request) (resp Response, err error) {
	if a.Metrics != nil && a.Now != nil {
		start := a.Now()
		defer func() {
			outcome := "ok"
			switch {
			case errors.Is(err, ErrRate):
				outcome = "limited"
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
	return a.ask(ctx, s, req, false)
}

// ask answers; probe is an eval case's run (SB-2204), which keeps no conversation.
// take spends one of the asker's asks this minute (SB-2305), or says how long to wait.
func (a Agent) take(ctx context.Context, s tenancy.Scope) error {
	if a.Rate == nil || isConsulted(ctx) || shadowPrompt(ctx) != "" { // a consultation is part of the asker's one ask; a shadow run is no ask
		return nil
	}
	limit := DefaultAsksPerMinute
	if a.AsksPerMinute != nil && a.AsksPerMinute() > 0 {
		limit = a.AsksPerMinute()
	}
	if wait, ok := a.Rate.Take(s.User(), limit, a.Now()); !ok {
		return a.limited(ctx, s.User(), s.Space(), wait)
	}
	return nil
}

// conversation is the ask's conversation and its earlier questions: a new one for "".
func (a Agent) conversation(ctx context.Context, s tenancy.Scope, conv, thread string) (string, []string, error) {
	if conv == "" && thread != "" { // SB-5801: the thread's conversation for this person, new or not
		conv = ThreadConversation(a.Tenant, s.User(), thread)
		qs, _, err := a.Store.Questions(ctx, a.Tenant, s.User(), conv)
		if err != nil {
			return "", nil, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
		}
		return conv, qs, nil
	}
	if conv == "" {
		return a.NewID("cv_"), nil, nil
	}
	qs, found, err := a.Store.Questions(ctx, a.Tenant, s.User(), conv)
	if err != nil {
		return "", nil, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if !found {
		return "", nil, fmt.Errorf("%w: no such conversation", ErrInvalid)
	}
	return conv, qs, nil
}

// ThreadConversation is the conversation of one person in one chat thread (SB-5801): the
// same id for their every reply in it, another for anyone else's, and nothing in it says
// which thread or person.
func ThreadConversation(tenant, user, thread string) string {
	sum := sha256.Sum256([]byte(tenant + "\x00" + user + "\x00" + thread))
	return "cv_t" + hex.EncodeToString(sum[:12])
}

func (a Agent) ask(ctx context.Context, s tenancy.Scope, req Request, probe bool) (Response, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return Response{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	ctx, tally := budget.WithTally(ctx) // SB-4606: this answer's own cost
	ctx, prefix := withPrefix(ctx)      // SB-5706: its stable prefix's parts, hashed
	ctx, sk := withSkips(ctx)           // SB-5101: its suspect passages
	ctx, hd := withWideHeld(ctx)        // SB-5402: documents shared too widely
	ctx, mu := gateway.WithUsed(ctx)    // SB-5601: whether the fallback model answered
	q, docs, err := validate(req)
	if err != nil {
		return Response{}, err
	}
	ctx, q, skill, err := a.checkExtras(ctx, s, req, q)
	if err != nil {
		return Response{}, err
	}
	if a.Killed == nil || a.Killed(ctx) {
		return Response{}, ErrKilled // no kill-switch reader is treated as killed
	}
	if spaceKilled(a.SpaceKilled, s) {
		return Response{}, ErrKilled
	}
	if !probe {
		if err := a.take(ctx, s); err != nil {
			return Response{}, err
		}
	}
	var dept *Department
	if req.Agent != "" && req.Agent != "ask" {
		d, err := a.department(ctx, s, req.Agent)
		if err != nil {
			return Response{}, err
		}
		dept = &d
	}
	conv, earlier, err := a.conversation(ctx, s, req.Conversation, req.Thread)
	if err != nil {
		return Response{}, err
	}
	choice, assumed, early, err := a.clarified(ctx, s, q, req, len(earlier), probe, dept) // SB-4901
	if early != nil {
		return *early, err
	}
	ctx = withChoice(ctx, choice)
	ctx = withBatch(ctx, probe)            // SB-5604: nobody waits for it
	feature := featureFor(ctx, req, probe) // SB-5001
	lctx := WithFeature(a.atLimit(ctx, s.User()), feature)
	tier := a.tierFor(lctx, s.Space(), req, q, len(docs), len(earlier))      // SB-4405, SB-5002, SB-5003
	tc := tierChoice{tier: tier, feature: feature, economy: economyOf(lctx)} // SB-5704: answerFrom may go deeper
	ctx, used, keep, drop, err := a.steerAndRemember(ctx, s, req, probe)     // SB-4303, SB-5502
	if err != nil {
		return Response{}, err
	}
	rk := a.reuseKeyIf(ctx, s, req, q, len(earlier), used, skill, probe || dept != nil) // SB-5603: "" never reuses
	capped := len(docs) == 0
	if docs, err = a.collectionOr(ctx, s, req.Collection, docs); err != nil { // SB-4304
		return Response{}, err
	}

	res, terms, err := a.retrieve(withVerified(withDocs(ctx, docs), req.Verified), s, q, req.Sources, retrieve.Budget)
	if err != nil {
		return Response{}, err
	}
	files, flaggedFiles := 0, 0
	if req.File != nil { // SB-3802: the file's passages first, never indexed or kept
		fc, err := a.fileChunks(ctx, q, req.File)
		if err != nil {
			return Response{}, err
		}
		res.Context, files = append(fc, res.Context...), 1
		flaggedFiles = fileFlagged(fc)
	}

	trace := a.NewID("tr_")
	turn := len(earlier)
	sent, left := recent(earlier)
	resp := Response{Trace: trace, Release: a.Release, Refused: res.Refused, Conversation: conv, Agent: "ask", People: []PersonRef{}, Terms: terms, EarlierDropped: left, Skill: skill, Assumed: assumed}
	if capped { // chosen documents aren't capped (SB-3004); a collection's are
		res.Context = retrieve.PerDocument(res.Context, MaxPerDocument)
	}
	a.whoAnswers(ctx, s, q, req, dept, probe, &resp)
	res.Context = a.screenIn(ctx, s.Space(), res.Context) // SB-5101: before the model and the table
	if len(res.Context) == 0 {
		n := NoticeNoSources
		if res.Refused > 0 {
			n = NoticeNoAccess
		}
		resp.Notice = &n
		resp.Answer.Claims = []answer.Claim{{Text: NothingToCite, Cites: []answer.Cite{}, Uncited: true}}
	} else if err := a.answerFrom(ctx, s, req, q, dept, &tc, terms, trace, sent, rk, &res, &resp); err != nil {
		return Response{}, err
	}
	if a.Indexing != nil {
		resp.NoticeIndexing = a.Indexing(ctx, s.Space())
	}
	tier, tierWhy := tc.tier, tc.why

	if keepsNothing(ctx) {
		return resp, nil
	}
	if err := a.alongside(ctx, s, q, req, probe || dept != nil, &resp); err != nil {
		return Response{}, err
	}
	resp.Skipped = sk.Skipped() // SB-5101, its consultations' too
	resp.Held, resp.WideWarned = hd.counts()
	resp.Fallback = mu.Fallback()
	for _, l := range used { // SB-5504: which lines, never their text, in the trace
		resp.Memory = append(resp.Memory, l.ID)
	}
	inTok, cachedTok := tally.Tokens() // SB-5706
	if probe {
		t := Trace{ID: trace, Tenant: a.Tenant, User: s.User(), Space: s.Space(), At: a.Now(), Response: resp, Decisions: res.Decisions, Tier: tier, TierWhy: tierWhy, CostUSD: tally.USD(), InputTokens: inTok, CachedTokens: cachedTok, Prefix: prefix.get(), Feature: feature, Suspect: sk.Cites(), Client: tenancy.ClientOf(ctx), Tool: tenancy.ToolOf(ctx)}
		if err := a.Store.PutTrace(ctx, t); err != nil {
			return Response{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
		}
		return resp, nil
	}
	t := Trace{ID: trace, Tenant: a.Tenant, User: s.User(), Space: s.Space(), At: a.Now(), Response: resp, Decisions: res.Decisions, Turn: turn, Files: files, FlaggedFiles: flaggedFiles, Tier: tier, TierWhy: tierWhy, CostUSD: tally.USD(), InputTokens: inTok, CachedTokens: cachedTok, Prefix: prefix.get(), Feature: feature, Suspect: sk.Cites(), Client: tenancy.ClientOf(ctx), Tool: tenancy.ToolOf(ctx),
		Collection: req.Collection, Documents: docs, Kept: keep, Dropped: drop, Read: readSet(res.Context), Saved: tally.Saved()} // SB-5501, SB-5603, SB-5606
	if err := a.keep(ctx, s, conv, q+choice, t); err != nil { // the conversation keeps the choice with the question
		return Response{}, err
	}
	a.noteSynced(ctx, s, conv, q+choice, trace, keep, drop) // SB-5803: on every device, if the person chose so
	a.guard(ctx, s.Space(), resp.Agent, t.Suspect)
	a.keepForReuse(ctx, rk, resp) // SB-5603
	return resp, nil
}

// keepsNothing is an ask whose answer is kept by its caller or not at all: a consultation is
// kept in the asker's trace, as a section of their answer; a shadow run keeps no trace or
// conversation (SB-4704).
func keepsNothing(ctx context.Context) bool {
	return isConsulted(ctx) || shadowPrompt(ctx) != ""
}

// promptFor is the prompt an ask answers with: the department agent's, or answer, unless a
// shadow run names another (SB-4704).
func promptFor(ctx context.Context, dept *Department) (prompt, lead string) {
	prompt, lead = "answer", "Question: "
	if dept != nil {
		prompt, lead = dept.Prompt, dept.Lead
	}
	if sp := shadowPrompt(ctx); sp != "" {
		prompt = sp
	}
	return prompt, lead
}

// keep stores an answer: its question in the conversation, the conversation's space, and
// its trace. No trace, no answer.
func (a Agent) keep(ctx context.Context, s tenancy.Scope, conv, q string, t Trace) error {
	if err := a.Store.AppendQuestion(ctx, a.Tenant, s.User(), conv, q); err != nil {
		return fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if st, ok := a.Store.(SpaceTagger); ok {
		if err := st.TagSpace(ctx, a.Tenant, conv, s.Space()); err != nil {
			return fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
		}
	}
	if err := a.Store.PutTrace(ctx, t); err != nil {
		return fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	return nil
}

// checkExtras checks an ask's file (SB-3802) and its consult (SB-4501, LAW-27), and takes
// its skill off the question (SB-4601): the rest is what is searched.
func (a Agent) checkExtras(ctx context.Context, s tenancy.Scope, req Request, q string) (context.Context, string, string, error) {
	if err := a.checkFile(req.File); err != nil {
		return ctx, q, "", err
	}
	if err := checkConsult(ctx, s, req); err != nil {
		return ctx, q, "", err
	}
	return a.loadSkill(ctx, s, q)
}

// alongside is what an answer carries beside its claims: the consulted teams' sections
// (SB-4501) and, for the ask agent's own answer, the hand off it offers (SB-4502).
func (a Agent) alongside(ctx context.Context, s tenancy.Scope, q string, req Request, noOffer bool, resp *Response) error {
	if req.Consult {
		var err error
		if resp.Departments, resp.NotConsulted, err = a.consult(ctx, s, q, req); err != nil {
			return err
		}
	}
	if !noOffer {
		resp.Handoff = a.handoff(ctx, s, q) // offered, never sent
	}
	return nil
}

func (a Agent) composeWith(ctx context.Context, prompt, lead, trace, q string, earlier []string, chunks []ports.Chunk) ([]answer.Claim, error) {
	p, err := prompts.Get(prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if chunks = a.screen(ctx, chunks); len(chunks) == 0 { // SB-5101: suspect passages never reach the model
		return []answer.Claim{{Text: NothingToCite, Cites: []answer.Cite{}, Uncited: true}}, nil
	}
	actx := answer.NewContext(chunks)
	var msgs []ports.Message
	for _, e := range earlier {
		msgs = append(msgs, ports.Message{Role: "user", Content: e})
		msgs = append(msgs, ports.Message{Role: "assistant", Content: "(earlier answer withheld; sources are checked again)"})
	}
	msgs = append(msgs, ports.Message{Role: "user", Content: memoryData(memoryOf(ctx)) + glossaryData(termsOf(ctx)) + actx.Data() + noteData(ctx) + "\n" + lead + q + choiceLine(ctx)})
	// SB-5605: the system prompt is the stable part (the prompt, language, length and skill),
	// cached where the provider can; the asker's material is only ever in the messages.
	notePrefix(ctx, map[string]string{PrefixPrompt: p.Text, PrefixLanguage: LanguageLine(languageOf(ctx)), PrefixLength: shortLine(ctx), PrefixSkill: skillLine(ctx)}) // SB-5706
	out, err := a.Model.Complete(ctx, ports.ModelRequest{System: p.Text + LanguageLine(languageOf(ctx)) + shortLine(ctx) + skillLine(ctx), Messages: msgs, JSON: true, User: a.user(ctx), Space: spaceOf(ctx),
		CacheSystem: true, Batch: batchOf(ctx)})
	if err != nil {
		if errors.Is(err, ports.ErrBatchPending) { // SB-5701: collected on a later tick
			return nil, fmt.Errorf("%w: %v", ErrPending, err)
		}
		if errors.Is(err, budget.ErrBudget) {
			return nil, fmt.Errorf("%w: %v", ErrBudget, err)
		}
		if errors.Is(err, gateway.ErrBusy) { // SB-5703: kept, so the ask says the model is busy
			return nil, fmt.Errorf("%w: model: %w", ErrUnavailable, gateway.ErrBusy)
		}
		return nil, fmt.Errorf("%w: model: %v", ErrUnavailable, err)
	}
	res := answer.Compose(out.Text, actx, nil)
	byCite := map[string]ports.Chunk{}
	for _, ch := range chunks {
		byCite[ch.Cite] = ch
	}
	a.decorate(ctx, res.Claims, byCite)
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

type languageKey struct{}

type docsKey struct{}

// withDocs carries an ask's chosen documents to retrieval (SB-3004).
func withDocs(ctx context.Context, docs []string) context.Context {
	return context.WithValue(ctx, docsKey{}, docs)
}

func docsOf(ctx context.Context) []string {
	d, _ := ctx.Value(docsKey{}).([]string)
	return d
}

// withLanguage carries the ask's language to the model call (SB-3003).
func withLanguage(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, languageKey{}, code)
}

type shortKey struct{}

// withShort carries a short answer's ask to the prompt (SB-3303).
func withShort(ctx context.Context, short bool) context.Context {
	return context.WithValue(ctx, shortKey{}, short)
}

func shortLine(ctx context.Context) string {
	if short, _ := ctx.Value(shortKey{}).(bool); short {
		return ShortLine
	}
	return ""
}

func languageOf(ctx context.Context) string {
	c, _ := ctx.Value(languageKey{}).(string)
	return c
}

// withUser carries the asker to the model call for the budget's rate limit.
func withUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

type spaceKey struct{}

// withSpace carries the space an ask runs in to the model call, so its spend is metered
// by space (SB-4105); company-wide and a person's own queue meter none.
func withSpace(ctx context.Context, space string) context.Context {
	if !strings.HasPrefix(space, "spc_") {
		return ctx
	}
	return context.WithValue(ctx, spaceKey{}, space)
}

func spaceOf(ctx context.Context) string {
	s, _ := ctx.Value(spaceKey{}).(string)
	return s
}

func (Agent) user(ctx context.Context) string {
	u, _ := ctx.Value(userKey{}).(string)
	return u
}

// whoToAsk is the owners on an answer's cites, or, with none active, who it climbs to
// (SB-3906).
func (a Agent) whoToAsk(ctx context.Context, s tenancy.Scope, claims []answer.Claim) []PersonRef {
	out := people(claims)
	if len(out) == 0 && a.Climb != nil && anyCited(claims) {
		out = a.Climb(ctx, s.Space())
	}
	return out
}

// anyCited reports whether any claim has a cite.
func anyCited(claims []answer.Claim) bool {
	for _, c := range claims {
		if !c.Uncited && len(c.Cites) > 0 {
			return true
		}
	}
	return false
}

// mostUncited is whether more than half the claims are uncited.
func mostUncited(claims []answer.Claim) bool {
	uncited := 0
	for _, c := range claims {
		if c.Uncited {
			uncited++
		}
	}
	return uncited*2 > len(claims)
}

// whoAnswers names the agent of an answer, and for the ask agent's own answer finds its pin
// and the right department.
func (a Agent) whoAnswers(ctx context.Context, s tenancy.Scope, q string, req Request, dept *Department, probe bool, resp *Response) {
	if dept != nil {
		resp.Agent = dept.Name
	}
	if probe {
		resp.Agent, resp.Conversation = AgentEval, ""
	} else if dept == nil {
		resp.Pinned = a.pinned(ctx, s, q)
		resp.Suggest = a.suggest(ctx, s, q, req)
	}
}

// answerFrom composes the answer from res's allowed context into resp: the table tool,
// the model (or passages when it is down), and every check and note on the claims.
func (a Agent) answerFrom(ctx context.Context, s tenancy.Scope, req Request, q string, dept *Department, tc *tierChoice, terms []Term, trace string, sent []string, rk string, res *retrieve.Result, resp *Response) error {
	if claims, at, ok := a.reusedFrom(ctx, rk, res.Context); ok { // SB-5603: the same documents, checked now: no model call
		resp.Answer.Claims, resp.Reused = claims, at
		resp.People = a.whoToAsk(ctx, s, claims)
		resp.Verified = a.verifiedCount(ctx, claims, res.Context)
		resp.Label = CitedLabel(claims, res.Context)
		resp.SeeAlso = seeAlso(claims, res.Context, a.Now())
		return a.markLinks(ctx, s, claims, res.Context)
	}
	prompt, lead := promptFor(ctx, dept)
	if dept == nil { // a department agent uses no table
		tr, ch, err := a.table(ctx, s, q, res.Context) // SB-4307
		if err != nil {
			return err
		}
		if ch != nil {
			res.Context, resp.Table = append([]ports.Chunk{*ch}, res.Context...), tr
		}
	}
	a.goDeeper(ctx, s.Space(), tc, resp.Table != nil) // SB-5704: research, tables, consultations
	ca := a
	switch tc.tier {
	case TierSmall:
		ca.Model = a.Small
	case TierDeep:
		ca.Model = a.Deep
	}
	cctx := withShort(withLanguage(withTerms(withSpace(withUser(ctx, s.User()), s.Space()), terms), req.Language), req.Length == "short")
	claims, err := a.composeOrPassages(ctx, func() ([]answer.Claim, error) { // SB-5602: passages when no model can write
		return ca.composeWith(cctx, prompt, lead, trace, q, sent, res.Context)
	}, res.Context, resp)
	if err != nil {
		return err
	}
	if mostUncited(claims) && resp.Notice == nil && tc.tier != TierDeep && a.deepAllowed(ctx, s.Space(), tc) { // SB-5704: one retry, deeper
		ca.Model = a.Deep
		if again, err := ca.composeWith(cctx, prompt, lead, trace, q, sent, res.Context); err == nil && uncitedCount(again) < uncitedCount(claims) {
			claims, tc.tier, tc.why = again, TierDeep, DeepCitationRetry
		}
	}
	if req.Length == "short" && len(claims) > MaxShort {
		claims = claims[:MaxShort]
	}
	if dept != nil {
		claims, resp.Dropped = citedOnly(claims)
	}
	var off int // a number no passage or tool returned is dropped (SB-4307, LAW-14)
	claims, off = tableCheck(claims, res.Context, resp.Table)
	resp.Dropped += off
	resp.Answer.Claims = claims
	resp.People = a.whoToAsk(ctx, s, claims)
	if mostUncited(claims) {
		n := NoticeMostUncited
		resp.Notice = &n
	}
	if err := a.markLinks(ctx, s, claims, res.Context); err != nil {
		return err
	}
	resp.NoticeStale = staleNotice(claims)
	resp.NoticeSingle = oneDocument(claims)
	resp.NoticeSync = a.syncNotices(ctx, claims)
	resp.SeeAlso = seeAlso(claims, res.Context, a.Now())
	resp.Verified = a.verifiedCount(ctx, claims, res.Context)
	resp.Label = CitedLabel(claims, res.Context) // SB-5405, LAW-32
	return nil
}

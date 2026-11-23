package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Search (sb-ask(3), Search, exactly; SB-1701): the ask path's validation, retrieval, space
// checks and live ACL batch, stopped before the composer. No model call, so no budget.
const (
	MaxResults = 20
	SnippetLen = 300
	// AgentSearch is the agent name a search's trace carries, so reports count it apart.
	AgentSearch = "search"
	// AgentEval is the agent name an eval case's run carries (SB-2204).
	AgentEval = "eval"
)

// SearchRequest is GET /v1/search.
type SearchRequest struct {
	Query   string
	Sources []string
	// Types and SinceDays narrow the results after every check (SB-2404); none is all.
	Types     []string
	SinceDays int
}

// Types are the document types search filters by (SB-2404), from a file's extension.
var Types = map[string][]string{
	"pdf": {".pdf"}, "doc": {".docx", ".doc", ".odt", ".rtf"}, "sheet": {".xlsx", ".xls", ".ods", ".csv"},
	"slides": {".pptx", ".ppt", ".odp"}, "web": {".html", ".htm"}, "text": {".txt", ".md"}, "other": nil,
}

// MaxSinceDays bounds since=.
const MaxSinceDays = 365

// TypeOf is a document's type: by the extension on its source id, else on its title.
func TypeOf(c ports.Chunk) string {
	for _, name := range []string{c.SourceID, c.Title} {
		ext := strings.ToLower(path.Ext(name))
		if ext == "" {
			continue
		}
		for t, exts := range Types {
			for _, e := range exts {
				if e == ext {
					return t
				}
			}
		}
	}
	return "other"
}

// keep reports whether a chunk that passed every check passes the filters too.
func (r SearchRequest) keep(c ports.Chunk, now time.Time) bool {
	if len(r.Types) > 0 {
		want := TypeOf(c)
		found := false
		for _, t := range r.Types {
			found = found || t == want
		}
		if !found {
			return false
		}
	}
	if r.SinceDays > 0 {
		if c.Seen.IsZero() || now.Sub(c.Seen) > time.Duration(r.SinceDays)*24*time.Hour {
			return false
		}
	}
	return true
}

// Result is one document a search found, which the asker may open now.
type Result struct {
	Cite    string         `json:"cite"`
	Title   string         `json:"title,omitempty"`
	URL     string         `json:"url,omitempty"`
	Owner   *answer.Person `json:"owner,omitempty"`
	AgeDays *int           `json:"age_days,omitempty"`
	Snippet string         `json:"snippet"`
	Flags   []string       `json:"flags"`
	Spaces  []string       `json:"spaces,omitempty"`
	Label   string         `json:"label,omitempty"`
}

// SearchResponse is the 200 body of GET /v1/search.
type SearchResponse struct {
	Trace   string   `json:"trace"`
	Release string   `json:"release"`
	Results []Result `json:"results"`
	Refused int      `json:"refused"`
	// Terms are the glossary entries the query used (SB-2203); never kept in the trace.
	Terms []Term `json:"terms,omitempty"`
}

// Search finds documents for the scope's user.
func (a Agent) Search(ctx context.Context, s tenancy.Scope, req SearchRequest) (SearchResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return SearchResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	q := strings.TrimSpace(req.Query)
	if q == "" || utf8.RuneCountInString(q) > MaxQuestion {
		return SearchResponse{}, fmt.Errorf("%w: q must be 1 to %d characters", ErrInvalid, MaxQuestion)
	}
	for _, k := range req.Sources {
		if !Kinds[k] {
			return SearchResponse{}, fmt.Errorf("%w: unknown source kind %q", ErrInvalid, k)
		}
	}
	for _, t := range req.Types {
		if _, ok := Types[t]; !ok {
			return SearchResponse{}, fmt.Errorf("%w: unknown type %q (pdf, doc, sheet, slides, web, text, other)", ErrInvalid, t)
		}
	}
	if req.SinceDays < 0 || req.SinceDays > MaxSinceDays {
		return SearchResponse{}, fmt.Errorf("%w: since is 1 to %d days", ErrInvalid, MaxSinceDays)
	}
	if a.Killed == nil || a.Killed(ctx) || spaceKilled(a.SpaceKilled, s) {
		return SearchResponse{}, ErrKilled
	}
	res, terms, err := a.retrieve(ctx, s, q, req.Sources, retrieve.Candidates)
	if err != nil {
		return SearchResponse{}, err
	}
	trace := a.NewID("tr_")
	out := SearchResponse{Trace: trace, Release: a.Release, Results: []Result{}, Refused: res.Refused, Terms: terms}
	own := a.owners()
	shown := map[string]bool{}
	now := a.Now()
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		if shown[k] || len(out.Results) == MaxResults || !req.keep(c, now) {
			continue
		}
		shown[k] = true
		out.Results = append(out.Results, a.result(ctx, own, c))
	}
	h := sha256.Sum256([]byte(q))
	t := Trace{ID: trace, Tenant: a.Tenant, User: s.User(), Space: s.Space(), At: a.Now(), QuerySHA256: hex.EncodeToString(h[:]),
		Response: Response{Trace: trace, Release: a.Release, Refused: res.Refused, Agent: AgentSearch, People: []PersonRef{},
			Answer: Answer{Claims: []answer.Claim{}}},
		Decisions: res.Decisions}
	if err := a.Store.PutTrace(ctx, t); err != nil {
		return SearchResponse{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err) // no trace, no results
	}
	if a.Metrics != nil {
		a.Metrics.Add("sb_searches_total", "Searches without the model.", map[string]string{"tenant": a.Tenant}, 1)
	}
	return out, nil
}

func (a Agent) result(ctx context.Context, own *owners, c ports.Chunk) Result {
	r := Result{Cite: c.Cite, Title: c.Title, Owner: own.of(ctx, c), Snippet: snippet(ingest.Shown(c.Text, c.Flags)), Flags: c.Flags, Label: c.Label}
	if r.Flags == nil {
		r.Flags = []string{}
	}
	if a.Link != nil {
		r.URL = a.Link(c)
	}
	for _, id := range c.Spaces {
		r.Spaces = append(r.Spaces, strings.TrimPrefix(id, "spc_"))
	}
	if d, ok := ageDays(c.Seen, a.Now()); ok {
		r.AgeDays = &d
	}
	return r
}

// retrieve runs retrieval and the live check for one query, up to budget allowed chunks.
func (a Agent) retrieve(ctx context.Context, s tenancy.Scope, q string, sources []string, budget int) (retrieve.Result, []Term, error) {
	wide, terms := expand(q, a.glossaryOf(s)) // the space's own terms only (SB-2203)
	var emails []string
	if a.Emails != nil {
		e, err := a.Emails(ctx, s.User())
		if err != nil {
			return retrieve.Result{}, nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		emails = e
	}
	query := ports.Query{Text: q, Kinds: sources}
	for _, t := range terms {
		query.Also = append(query.Also, t.Means)
	}
	if a.Embedder != nil {
		vs, err := a.Embedder.Embed(ctx, []string{wide})
		if err != nil || len(vs) != 1 {
			return retrieve.Result{}, nil, fmt.Errorf("%w: embedder", ErrUnavailable)
		}
		query.Vector = vs[0]
	}
	who := ports.UserRef{ID: s.User()}
	if len(emails) > 0 {
		who.Email = emails[0]
	}
	res, err := retrieve.ContextUpTo(ctx, s, who, emails, a.Index, a.Auth, query, budget)
	if err != nil {
		return retrieve.Result{}, nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	if res.Context, err = a.dropRetired(ctx, s, res.Context); err != nil { // SB-2801
		return retrieve.Result{}, nil, err
	}
	return res, terms, nil
}

// snippet is the first SnippetLen characters of a chunk, cut at a space when it can be.
func snippet(text string) string {
	rs := []rune(strings.TrimSpace(text))
	if len(rs) <= SnippetLen {
		return string(rs)
	}
	cut := SnippetLen
	for i := SnippetLen; i > SnippetLen*2/3; i-- {
		if rs[i] == ' ' {
			cut = i
			break
		}
	}
	return strings.TrimSpace(string(rs[:cut])) + "…"
}

// ageDays is whole days since seen; false when the index doesn't know.
func ageDays(seen, now time.Time) (int, bool) {
	if seen.IsZero() || now.Before(seen) {
		return 0, !seen.IsZero()
	}
	return int(now.Sub(seen) / (24 * time.Hour)), true
}

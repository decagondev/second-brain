package ask

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// MaxInstruction is the longest draft instruction, in characters.
const MaxInstruction = 2000

// DraftRequest is the body of POST /v1/drafts (sb-actions(3)).
type DraftRequest struct {
	Context     string `json:"context"`
	Instruction string `json:"instruction"`
}

// DraftResponse is an answer-shaped draft plus its context item.
type DraftResponse struct {
	Response
	Context string `json:"context"`
}

// Counter is the metrics sink for draft outcomes.
type Counter interface {
	Add(name, help string, labels map[string]string, v float64)
}

// Drafts is the draft-reply agent (SB-409). It has no tools: it never sends.
type Drafts struct {
	Agent   Agent // retrieval, live check, model, store, ids, clock
	Killed  func(ctx context.Context) bool
	Policy  policy.Kernel
	Metrics Counter
}

var contextRe = regexp.MustCompile(`^(chat:[A-Z0-9]{2,20}/[0-9]{1,12}\.[0-9]{1,8}|tickets:[A-Za-z0-9_-]{1,64})$`)
var projectRe = regexp.MustCompile(`^tickets:([A-Z][A-Z0-9]{0,9})-[0-9]+$`)

// Draft writes a cited reply to one ticket or thread (sb-actions(3), How a draft is made).
func (d Drafts) Draft(ctx context.Context, s tenancy.Scope, req DraftRequest) (DraftResponse, error) {
	a := d.Agent
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return DraftResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	instr := strings.TrimSpace(req.Instruction)
	if !contextRe.MatchString(req.Context) {
		return DraftResponse{}, fmt.Errorf("%w: context must be chat:<channel>/<ts> or tickets:<key>", ErrInvalid)
	}
	if instr == "" || utf8.RuneCountInString(instr) > MaxInstruction {
		return DraftResponse{}, fmt.Errorf("%w: instruction must be 1 to %d characters", ErrInvalid, MaxInstruction)
	}
	if d.Killed == nil || d.Killed(ctx) || spaceKilled(d.Agent.SpaceKilled, s) {
		return DraftResponse{}, ErrKilled
	}
	pctx := map[string]any{}
	if m := projectRe.FindStringSubmatch(req.Context); m != nil {
		pctx["target_project"] = m[1]
	}
	use := d.Policy.Authorise(ctx, s, "use_agent", policy.Entity{Type: "Agent", ID: "draft-reply", Tenant: a.Tenant,
		Attrs: map[string]any{"name": "draft-reply", "consequential": false, "enabled": true}}, pctx) // mounted only when the manifest lists it
	if use.Decision != policy.Allow {
		return DraftResponse{}, ErrNotFound // EPERM, saying nothing about why
	}

	kind, id, _ := strings.Cut(req.Context, ":")
	ref := ports.SourceRef{Kind: kind, ID: id}
	var emails []string
	if a.Emails != nil {
		e, err := a.Emails(ctx, s.User())
		if err != nil {
			return DraftResponse{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		emails = e
	}
	who := ports.UserRef{ID: s.User()}
	if len(emails) > 0 {
		who.Email = emails[0]
	}
	item := ports.Chunk{ID: "draft-context", Tenant: a.Tenant, Kind: kind, SourceID: id}
	if a.Auth.Check(ctx, who, []ports.Chunk{item})[kind+":"+id] != ports.VerdictAllow {
		return DraftResponse{}, ErrNotFound // the item itself must be live-allowed
	}
	conn, ok := a.Auth.Connectors[kind]
	if !ok {
		return DraftResponse{}, ErrNotFound
	}
	doc, err := conn.Fetch(ctx, ref)
	if err != nil {
		return DraftResponse{}, fmt.Errorf("%w: fetch: %v", ErrUnavailable, err)
	}
	// In a cell with spaces the item itself takes the space, label and barrier steps, with
	// its owner at the source, before it is used (sb-permissions(7); SB-1203).
	if a.Auth.Spaces != nil {
		ok, err := a.Auth.ItemAllowed(ctx, s, who, ports.Chunk{ID: "draft-context", Tenant: a.Tenant, Kind: kind, SourceID: id,
			Cite: ingest.Cite(ref, doc.Version)}, doc)
		if err != nil {
			return DraftResponse{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		if !ok {
			return DraftResponse{}, ErrNotFound
		}
	}
	clean, flags := ingest.Sanitiser{}.Sanitise(doc.Title + "\n\n" + doc.Text)
	text := ingest.Normalise(clean)
	if r := []rune(text); len(r) > ingest.MaxChunk*4 {
		text = string(r[:ingest.MaxChunk*4])
	}
	item = ports.Chunk{ID: "draft-context", Tenant: a.Tenant, Kind: kind, SourceID: id, Version: doc.Version,
		Cite: ingest.Cite(ref, doc.Version), Text: text, End: len([]rune(text)), Flags: flags}

	query := ports.Query{Text: instr + " " + doc.Title}
	if a.Embedder != nil {
		if vs, err := a.Embedder.Embed(ctx, []string{query.Text}); err == nil && len(vs) == 1 {
			query.Vector = vs[0]
		}
	}
	res, err := retrieve.Context(ctx, s, who, emails, a.Index, a.Auth, query)
	if err != nil {
		return DraftResponse{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	if res.Context, err = a.dropRetired(ctx, s, res.Context); err != nil { // SB-2801
		return DraftResponse{}, err
	}
	if res.Context, err = a.dropHeld(ctx, s, res.Context); err != nil { // SB-5402
		return DraftResponse{}, err
	}
	chunks := []ports.Chunk{item}
	for _, c := range retrieve.PerDocument(res.Context, MaxPerDocument) { // SB-2902
		if c.Kind == kind && c.SourceID == id {
			continue // the item is already C1
		}
		chunks = append(chunks, c)
	}
	trace := a.NewID("tr_")
	claims, err := a.composeWith(withUser(ctx, s.User()), "draft", "Instruction: ", trace, instr, nil, chunks)
	if err != nil {
		return DraftResponse{}, err
	}
	resp := DraftResponse{Response: Response{Trace: trace, Release: a.Release, Refused: res.Refused, Answer: Answer{Claims: claims}}, Context: req.Context}
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
	decisions := append([]retrieve.Decision{{Cite: item.Cite, Verdict: "allow", Reason: "allowed"}}, res.Decisions...)
	if err := a.Store.PutTrace(ctx, Trace{ID: trace, Tenant: a.Tenant, User: s.User(), Space: s.Space(), At: a.Now(), Response: resp.Response, Decisions: decisions}); err != nil {
		return DraftResponse{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	return resp, nil
}

// Outcome records whether the person used or discarded their draft (sb_drafts_total).
func (d Drafts) Outcome(ctx context.Context, s tenancy.Scope, trace, outcome string) error {
	if outcome != "used" && outcome != "discarded" {
		return fmt.Errorf("%w: outcome must be used or discarded", ErrInvalid)
	}
	t, err := d.Agent.Trace(ctx, s, trace)
	if err != nil || t.User != s.User() {
		return ErrNotFound
	}
	if d.Metrics != nil {
		d.Metrics.Add("sb_drafts_total", "Drafts by outcome (used or discarded).", map[string]string{"tenant": d.Agent.Tenant, "outcome": outcome}, 1)
	}
	return nil
}

package ask

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Stewards' tools (sb-ask(3), Documents to review and Questions to the stewards; SB-2501 to
// SB-2503): a document marked reviewed at its version, a soft claim on a question, and a
// space's reply templates.

// Bounds.
const (
	ClaimFor        = time.Hour
	MaxTemplates    = 20
	MaxTemplateName = 60
	MaxTemplateText = 2000
)

// ReviewMark says a steward reviewed a document at a version and found it still right.
type ReviewMark struct {
	Tenant  string    `json:"tenant"`
	Space   string    `json:"space"`    // spc_NAME
	Doc     string    `json:"document"` // kind:source id
	Version int       `json:"version"`
	By      string    `json:"by"`
	At      time.Time `json:"at"`
	// Days is how long the mark lasts instead of review_after_days (SB-3603); 0 is the default.
	Days int `json:"days,omitempty"`
}

// MarkStore keeps review marks, one per space and document.
type MarkStore interface {
	PutReviewMark(ctx context.Context, m ReviewMark) error
	ReviewMarks(ctx context.Context, tenant, space string) ([]ReviewMark, error)
}

// StewardClaim is a steward's "I'll answer this" on an open question.
type StewardClaim struct {
	By    string    `json:"by"`
	Until time.Time `json:"until"`
}

// ClaimView is a live claim as a steward sees it.
type ClaimView struct {
	By    string    `json:"by"` // the holder's email
	Mine  bool      `json:"mine"`
	Until time.Time `json:"until"`
}

// Claimant reports whether user is an active member of one of stewards now, with the email
// to show; ok false is someone whose claim isn't shown.
type Claimant func(ctx context.Context, user string, stewards []string) (email string, ok bool, err error)

// ReplyTemplate is one of a space's saved replies.
type ReplyTemplate struct {
	ID     string    `json:"id"`
	Tenant string    `json:"tenant"`
	Space  string    `json:"space"`
	Name   string    `json:"name"`
	Text   string    `json:"text"`
	By     string    `json:"by"`
	At     time.Time `json:"at"`
}

// TemplateStore keeps a space's reply templates.
type TemplateStore interface {
	PutTemplate(ctx context.Context, t ReplyTemplate) error
	Templates(ctx context.Context, tenant, space string) ([]ReplyTemplate, error)
	DeleteTemplate(ctx context.Context, tenant, space, id string) (bool, error)
}

// marked are the space's marks by document, or none when the cell keeps none.
func (x Stewards) marked(ctx context.Context, name string) (map[string]ReviewMark, error) {
	out := map[string]ReviewMark{}
	if x.Marks == nil {
		return out, nil
	}
	ms, err := x.Marks.ReviewMarks(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return nil, fmt.Errorf("%w: review marks: %v", ErrUnavailable, err)
	}
	for _, m := range ms {
		out[m.Doc] = m
	}
	return out, nil
}

// covers reports whether mark m still takes chunk c off the list: the same version, younger
// than after days, and no wrong-citation report since.
func covers(m ReviewMark, ok bool, c ports.Chunk, lastFlag time.Time, after int, now time.Time) bool {
	return ok && m.Version == c.Version && now.Sub(m.At) <= time.Duration(markAfter(m, after))*24*time.Hour && !lastFlag.After(m.At)
}

// lastFlags is the time of the newest wrong-citation or outdated report of each document.
func lastFlags(fbs []Feedback) map[string]time.Time {
	out := map[string]time.Time{}
	for _, f := range fbs {
		for _, c := range []string{f.WrongCitation, f.OutdatedCitation} { // either brings it back (SB-2703)
			if k := docKey(c); c != "" && f.At.After(out[k]) {
				out[k] = f.At
			}
		}
	}
	return out
}

// MarkReviewed marks a document of the space reviewed at its current version (SB-2501).
func (x Stewards) MarkReviewed(ctx context.Context, s tenancy.Scope, name, cite string, days int) (ReviewMark, error) {
	if err := x.reviewer(ctx, s, name); err != nil {
		return ReviewMark{}, err
	}
	if days != 0 && !MarkDays[days] {
		return ReviewMark{}, fmt.Errorf("%w: review again in 30, 90, 180 or 365 days", ErrInvalid)
	}
	if x.Marks == nil {
		return ReviewMark{}, fmt.Errorf("%w: this cell keeps no review marks", ErrUnavailable)
	}
	doc := docKey(cite)
	ver, err := strconv.Atoi(strings.TrimPrefix(cite, doc+"@"))
	kind, id, _ := strings.Cut(doc, ":")
	if err != nil || id == "" {
		return ReviewMark{}, fmt.Errorf("%w: a cite is KIND:ID@VERSION", ErrInvalid)
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return ReviewMark{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return ReviewMark{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var cur []ports.Chunk
	for _, c := range firsts(latest(cs)) {
		if contains(c.Spaces, SpaceKey(name)) {
			cur = append(cur, c)
		}
	}
	if len(cur) == 0 {
		return ReviewMark{}, ErrNotSteward // another space's document, or none, looks the same as one they can't open
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return ReviewMark{}, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cur, 1)
	if err != nil {
		return ReviewMark{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		return ReviewMark{}, ErrNotSteward
	}
	if res.Context[0].Version != ver {
		return ReviewMark{}, fmt.Errorf("%w: mark the document's current version, %d", ErrInvalid, res.Context[0].Version)
	}
	m := ReviewMark{Tenant: x.Agent.Tenant, Space: SpaceKey(name), Doc: doc, Version: ver, By: s.User(), At: x.Now().UTC(), Days: days}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: cite, State: "reviewed",
		Reason: "reviewed, still right", Space: auditSpace(m.Space)}); err != nil {
		return ReviewMark{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Marks.PutReviewMark(ctx, m); err != nil {
		return ReviewMark{}, fmt.Errorf("%w: review marks: %v", ErrUnavailable, err)
	}
	return m, nil
}

// Claim claims an open question for an hour, or answers with someone else's live claim
// (SB-2502). Release lets the caller's own claim go.
func (x Stewards) Claim(ctx context.Context, s tenancy.Scope, name, id string, release bool) (*ClaimView, error) {
	stewards, _, err := x.steward(ctx, s, name)
	if err != nil {
		return nil, err
	}
	if _, err := x.question(ctx, name, id); err != nil {
		return nil, err
	}
	now := x.Now().UTC()
	q, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(q *StewardQuestion) error {
		if q.Reply != nil {
			return fmt.Errorf("%w: this question already has a reply", ErrInvalid)
		}
		mine := q.Claim != nil && q.Claim.By == s.User()
		switch {
		case release:
			if mine {
				q.Claim = nil
			}
		case mine || q.Claim == nil || !q.Claim.Until.After(now):
			q.Claim = &StewardClaim{By: s.User(), Until: now.Add(ClaimFor)}
		default:
			// someone else's claim: check it still stands below, and leave it as it is
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return x.claimView(ctx, s, q.Claim, stewards, now)
}

// claimView is a claim as s sees it: none when it lapsed or its holder is no steward now.
func (x Stewards) claimView(ctx context.Context, s tenancy.Scope, c *StewardClaim, stewards []string, now time.Time) (*ClaimView, error) {
	if c == nil || !c.Until.After(now) || x.Claimant == nil {
		return nil, nil
	}
	email, ok, err := x.Claimant(ctx, c.By, stewards)
	if err != nil {
		return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if !ok {
		return nil, nil
	}
	return &ClaimView{By: email, Mine: c.By == s.User(), Until: c.Until}, nil
}

// Templates lists the space's reply templates for its stewards.
func (x Stewards) Templates(ctx context.Context, s tenancy.Scope, name string) ([]ReplyTemplate, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	if x.Replies == nil {
		return []ReplyTemplate{}, nil
	}
	ts, err := x.Replies.Templates(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return nil, fmt.Errorf("%w: templates: %v", ErrUnavailable, err)
	}
	return ts, nil
}

// SaveTemplate adds a template (id "") or changes one of the space's.
func (x Stewards) SaveTemplate(ctx context.Context, s tenancy.Scope, name, id, tname, text string) (ReplyTemplate, error) {
	ts, err := x.Templates(ctx, s, name)
	if err != nil {
		return ReplyTemplate{}, err
	}
	if x.Replies == nil {
		return ReplyTemplate{}, fmt.Errorf("%w: this cell keeps no templates", ErrUnavailable)
	}
	tname, text = strings.TrimSpace(tname), strings.TrimSpace(text)
	if tname == "" || utf8.RuneCountInString(tname) > MaxTemplateName || text == "" || utf8.RuneCountInString(text) > MaxTemplateText {
		return ReplyTemplate{}, fmt.Errorf("%w: a template is a name of 1 to %d characters and a text of 1 to %d", ErrInvalid, MaxTemplateName, MaxTemplateText)
	}
	found := false
	for _, t := range ts {
		found = found || t.ID == id
	}
	switch {
	case id == "" && len(ts) >= MaxTemplates:
		return ReplyTemplate{}, fmt.Errorf("%w: a space keeps at most %d templates", ErrInvalid, MaxTemplates)
	case id == "":
		id = x.NewID("rt_")
	case !found:
		return ReplyTemplate{}, ErrNotSteward // another space's template looks like none
	}
	t := ReplyTemplate{ID: id, Tenant: x.Agent.Tenant, Space: SpaceKey(name), Name: tname, Text: text, By: s.User(), At: x.Now().UTC()}
	if err := x.Replies.PutTemplate(ctx, t); err != nil {
		return ReplyTemplate{}, fmt.Errorf("%w: templates: %v", ErrUnavailable, err)
	}
	return t, nil
}

// DeleteTemplate removes one of the space's templates.
func (x Stewards) DeleteTemplate(ctx context.Context, s tenancy.Scope, name, id string) error {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return err
	}
	if x.Replies == nil {
		return ErrNotSteward
	}
	ok, err := x.Replies.DeleteTemplate(ctx, x.Agent.Tenant, SpaceKey(name), id)
	if err != nil {
		return fmt.Errorf("%w: templates: %v", ErrUnavailable, err)
	}
	if !ok {
		return ErrNotSteward
	}
	return nil
}

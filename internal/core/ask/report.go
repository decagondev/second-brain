package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Report this answer (sb-ask(3), Questions to the stewards; SB-3606): the asker sends an
// answer they rated Not helpful to the space's stewards, by their own click, named.

// MaxReportNote is the longest note a report carries.
const MaxReportNote = 500

// AnswerReport marks a steward question as a report of its trace's answer.
type AnswerReport struct {
	Note string `json:"note,omitempty"`
}

// ReportView is a report as a steward sees it: only what they can open now.
type ReportView struct {
	By        string   `json:"by"`
	Note      string   `json:"note,omitempty"`
	Claims    []string `json:"claims"`
	Documents []Result `json:"documents"`
	Hidden    int      `json:"hidden"`
}

// Report sends the asker's answer trace, rated Not helpful by them, to the stewards of the
// space it was asked in, with the question as text and an optional note.
func (x Stewards) Report(ctx context.Context, s tenancy.Scope, text, trace, note string) (StewardQuestion, error) {
	q, err := question(s, x.Agent.Tenant, text)
	if err != nil {
		return StewardQuestion{}, err
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > MaxReportNote || strings.ContainsAny(note, "\r\n") {
		return StewardQuestion{}, fmt.Errorf("%w: a note is one paragraph of at most %d characters", ErrInvalid, MaxReportNote)
	}
	t, err := x.Agent.Trace(ctx, s, trace)
	if err != nil || t.User != s.User() || t.Space != s.Space() || t.Response.Agent == AgentSearch || t.Response.Agent == AgentEval {
		return StewardQuestion{}, fmt.Errorf("%w: report your own answer, in the same space", ErrInvalid)
	}
	if x.Feedback == nil {
		return StewardQuestion{}, fmt.Errorf("%w: this cell keeps no ratings", ErrUnavailable)
	}
	f, ok, err := x.Feedback.GetFeedback(ctx, x.Agent.Tenant, trace)
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	if !ok || f.User != s.User() || f.Rating != "down" {
		return StewardQuestion{}, fmt.Errorf("%w: rate the answer Not helpful before you report it", ErrInvalid)
	}
	if _, _, ok := x.Facts(s.Space()); !ok {
		return StewardQuestion{}, fmt.Errorf("%w: nobody answers questions sent from here", ErrInvalid)
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: q, Space: s.Space(), Trace: trace,
		At: x.Now().UTC(), Report: &AnswerReport{Note: note}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "reported",
		PayloadSHA256: hashOf(q + "\n" + note), Reason: "answer reported to stewards", Space: auditSpace(sq.Space)}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// reportView is q's report as steward s sees it: a claim only when s can open every one of
// its cites now, the documents they can open by title, and the rest counted (LAW-01).
func (x Stewards) reportView(ctx context.Context, s tenancy.Scope, q StewardQuestion) (*ReportView, error) {
	if q.Report == nil {
		return nil, nil
	}
	v := &ReportView{By: q.Asker, Note: q.Report.Note, Claims: []string{}, Documents: []Result{}}
	t, found, err := x.Agent.Store.GetTrace(ctx, x.Agent.Tenant, q.Trace)
	if err != nil {
		return nil, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	if !found || t.Tenant != x.Agent.Tenant || x.Agent.Now().Sub(t.At) > TraceTTL {
		return v, nil
	}
	var cites []string
	seen := map[string]bool{}
	for _, c := range t.Response.Answer.Claims {
		for _, ci := range c.Cites {
			if !seen[ci.Cite] {
				seen[ci.Cite] = true
				cites = append(cites, ci.Cite)
			}
		}
	}
	docs, hidden, err := x.Agent.CheckCites(ctx, s, cites)
	if err != nil {
		return nil, err
	}
	open := map[string]bool{}
	for _, r := range docs {
		open[r.Cite] = true
	}
	v.Documents, v.Hidden = docs, hidden
	for _, c := range t.Response.Answer.Claims {
		if c.Uncited || len(c.Cites) == 0 {
			v.Hidden++ // no source stands behind it: shown to no one but the asker
			continue
		}
		all := true
		for _, ci := range c.Cites {
			all = all && open[ci.Cite]
		}
		if all {
			v.Claims = append(v.Claims, c.Text)
		} else {
			v.Hidden++
		}
	}
	return v, nil
}

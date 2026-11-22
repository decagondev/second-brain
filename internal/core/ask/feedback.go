package ask

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/tenancy"
)

// MaxComment is the longest feedback comment, in characters.
const MaxComment = 2000

// FeedbackRequest is the body of POST /v1/feedback (sb-ask(3)).
type FeedbackRequest struct {
	Trace         string `json:"trace"`
	Rating        string `json:"rating"`
	WrongCitation string `json:"wrong_citation,omitempty"`
	Comment       string `json:"comment,omitempty"`
}

// Feedback is one stored feedback, keyed by tenant and trace.
type Feedback struct {
	Tenant, User, Trace, Release string
	Rating, WrongCitation        string
	Comment                      string
	At                           time.Time
	Claims, Uncited, Refused     int
	// ReviewedBy and ReviewedAt: the operator who reviewed it in the queue (SB-1609). A
	// later feedback on the trace replaces the record, and so puts it back in the queue.
	ReviewedBy string
	ReviewedAt time.Time
}

// FeedbackReader finds one trace's feedback (the operators' queue, SB-1609).
type FeedbackReader interface {
	GetFeedback(ctx context.Context, tenant, trace string) (Feedback, bool, error)
}

// FeedbackStore keeps feedback for one cell. Put replaces any earlier feedback on a trace.
type FeedbackStore interface {
	PutFeedback(ctx context.Context, f Feedback) error
	FeedbackBetween(ctx context.Context, tenant string, from, to time.Time) ([]Feedback, error)
}

// Feedbacks records feedback for traces this cell answered.
type Feedbacks struct {
	Agent Agent
	Store FeedbackStore
}

// Give stores feedback from the user who asked.
func (f Feedbacks) Give(ctx context.Context, s tenancy.Scope, req FeedbackRequest) error {
	if req.Rating != "up" && req.Rating != "down" {
		return fmt.Errorf("%w: rating must be up or down", ErrInvalid)
	}
	if utf8.RuneCountInString(req.Comment) > MaxComment {
		return fmt.Errorf("%w: comment must be at most %d characters", ErrInvalid, MaxComment)
	}
	t, err := f.Agent.Trace(ctx, s, req.Trace)
	if err != nil {
		return err // ErrNotFound for anyone but a reader of the trace
	}
	if t.User != s.User() {
		return ErrNotFound // auditors may read a trace, but only the asker gives feedback
	}
	fb := Feedback{Tenant: t.Tenant, User: t.User, Trace: t.ID, Release: t.Response.Release, Rating: req.Rating,
		Comment: req.Comment, At: f.Agent.Now(), Refused: t.Response.Refused}
	found := req.WrongCitation == ""
	for _, c := range t.Response.Answer.Claims {
		fb.Claims++
		if c.Uncited {
			fb.Uncited++
		}
		for _, ci := range c.Cites {
			if ci.Cite == req.WrongCitation {
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf("%w: wrong_citation is not a citation in that answer", ErrInvalid)
	}
	fb.WrongCitation = req.WrongCitation
	if err := f.Store.PutFeedback(ctx, fb); err != nil {
		return fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	return nil
}

// Week is the export window that ends at the most recent Monday 00:00 UTC at or before now.
func Week(now time.Time) (from, to time.Time) {
	d := now.UTC()
	to = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	for to.Weekday() != time.Monday {
		to = to.AddDate(0, 0, -1)
	}
	return to.AddDate(0, 0, -7), to
}

// exportLine is one exported feedback: ids, counts and the user's own comment, never the
// question or the answer.
type exportLine struct {
	Trace         string    `json:"trace"`
	Release       string    `json:"release"`
	Rating        string    `json:"rating"`
	WrongCitation string    `json:"wrong_citation,omitempty"`
	Comment       string    `json:"comment,omitempty"`
	At            time.Time `json:"at"`
	Claims        int       `json:"claims"`
	Uncited       int       `json:"uncited"`
	Refused       int       `json:"refused"`
}

// Export writes the feedback in [from, to) as JSON lines and returns how many it wrote.
func (f Feedbacks) Export(ctx context.Context, tenant string, from, to time.Time, w io.Writer) (int, error) {
	fs, err := f.Store.FeedbackBetween(ctx, tenant, from, to)
	if err != nil {
		return 0, err
	}
	enc := json.NewEncoder(w)
	for _, x := range fs {
		if x.Tenant != tenant {
			return 0, fmt.Errorf("feedback: store returned another tenant's feedback")
		}
		if err := enc.Encode(exportLine{Trace: x.Trace, Release: x.Release, Rating: x.Rating, WrongCitation: x.WrongCitation,
			Comment: x.Comment, At: x.At.UTC(), Claims: x.Claims, Uncited: x.Uncited, Refused: x.Refused}); err != nil {
			return 0, err
		}
	}
	return len(fs), nil
}

// QueueWindow is how far back the operators' queue looks (sb-admin(3)).
const QueueWindow = 90 * 24 * time.Hour

// QueueItem is one answer in the operators' feedback queue: ids, counts, the trace's policy
// decisions counted by verdict and reason, and the user's own comment. Never the question,
// the answer, or the cites the answer was refused (SB-1609).
type QueueItem struct {
	Trace         string          `json:"trace"`
	At            time.Time       `json:"at"`
	User          string          `json:"user"`
	Release       string          `json:"release"`
	Rating        string          `json:"rating"`
	WrongCitation string          `json:"wrong_citation,omitempty"`
	Comment       string          `json:"comment,omitempty"`
	Claims        int             `json:"claims"`
	Uncited       int             `json:"uncited"`
	Refused       int             `json:"refused"`
	Decisions     []DecisionCount `json:"decisions"`
	Reviewed      *Review         `json:"reviewed,omitempty"`
}

// DecisionCount is how many of a trace's retrieval decisions had one verdict and reason.
type DecisionCount struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
	Count   int    `json:"count"`
}

// Review is who marked an item reviewed, and when.
type Review struct {
	By string    `json:"by"`
	At time.Time `json:"at"`
}

// Queue lists one tenant's feedback of a rating from the last QueueWindow, newest first,
// leaving out reviewed items unless withReviewed. A trace that has expired keeps its
// counts and has no decisions.
func Queue(ctx context.Context, fs FeedbackStore, traces Store, tenant, rating string, withReviewed bool, now time.Time) ([]QueueItem, error) {
	if rating != "up" && rating != "down" {
		return nil, fmt.Errorf("%w: rating must be up or down", ErrInvalid)
	}
	all, err := fs.FeedbackBetween(ctx, tenant, now.Add(-QueueWindow), now.Add(time.Second))
	if err != nil {
		return nil, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	out := []QueueItem{}
	for i := len(all) - 1; i >= 0; i-- {
		f := all[i]
		if f.Tenant != tenant {
			return nil, fmt.Errorf("feedback: store returned another tenant's feedback")
		}
		if f.Rating != rating || (f.ReviewedBy != "" && !withReviewed) {
			continue
		}
		it := QueueItem{Trace: f.Trace, At: f.At.UTC(), User: f.User, Release: f.Release, Rating: f.Rating, WrongCitation: f.WrongCitation,
			Comment: f.Comment, Claims: f.Claims, Uncited: f.Uncited, Refused: f.Refused, Decisions: []DecisionCount{}}
		if f.ReviewedBy != "" {
			it.Reviewed = &Review{By: f.ReviewedBy, At: f.ReviewedAt.UTC()}
		}
		t, found, err := traces.GetTrace(ctx, tenant, f.Trace)
		if err != nil {
			return nil, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
		}
		if found {
			counts := map[[2]string]int{}
			for _, d := range t.Decisions {
				counts[[2]string{d.Verdict, d.Reason}]++
			}
			for k, n := range counts {
				it.Decisions = append(it.Decisions, DecisionCount{Verdict: k[0], Reason: k[1], Count: n})
			}
			sort.Slice(it.Decisions, func(a, b int) bool {
				if it.Decisions[a].Verdict != it.Decisions[b].Verdict {
					return it.Decisions[a].Verdict < it.Decisions[b].Verdict
				}
				return it.Decisions[a].Reason < it.Decisions[b].Reason
			})
		}
		out = append(out, it)
	}
	return out, nil
}

// MarkReviewed marks a trace's feedback reviewed by an operator. first is false when it
// was already reviewed, which keeps the first review.
func MarkReviewed(ctx context.Context, fs FeedbackStore, fr FeedbackReader, tenant, trace, by string, now time.Time) (Feedback, bool, error) {
	f, found, err := fr.GetFeedback(ctx, tenant, trace)
	if err != nil {
		return Feedback{}, false, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	if !found || f.Tenant != tenant {
		return Feedback{}, false, ErrNotFound
	}
	if f.ReviewedBy != "" {
		return f, false, nil
	}
	f.ReviewedBy, f.ReviewedAt = by, now.UTC()
	if err := fs.PutFeedback(ctx, f); err != nil {
		return Feedback{}, false, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	return f, true, nil
}

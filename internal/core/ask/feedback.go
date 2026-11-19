package ask

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

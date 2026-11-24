package ask

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// MaxPassage is the most of a cited chunk's text a passage shows, in characters (SB-3001).
const MaxPassage = 600

// ErrSourceDenied is a passage of a source the person can't open now (EPERM).
var ErrSourceDenied = errors.New("ask: the person can't open this source now")

// Passage is one cite's text, as the person may read it now (sb-ask(3), The cited passage).
type Passage struct {
	Cite string `json:"cite"`
	Text string `json:"text,omitempty"`
	Cut  bool   `json:"cut,omitempty"`
	Gone bool   `json:"gone,omitempty"`
}

// Passage returns the text of one cite of a trace's answer, checked live as s: only a cite
// in the answer, only for whoever may read the trace, and never once access is gone. start
// picks the span when the answer cites the document more than once; a negative start is the
// first.
func (a Agent) Passage(ctx context.Context, s tenancy.Scope, trace, cite string, start int) (Passage, error) {
	t, err := a.Trace(ctx, s, trace)
	if err != nil {
		return Passage{}, err
	}
	span, ok := spanOf(t.Response, cite, start)
	if !ok {
		return Passage{}, fmt.Errorf("%w: that cite isn't in this answer", ErrInvalid)
	}
	lk, ok := a.Index.(ports.CiteLookup)
	if !ok {
		return Passage{}, fmt.Errorf("%w: the index can't look up cites", ErrUnavailable)
	}
	chunks, err := lk.ChunksOf(ctx, a.Tenant, []string{cite})
	if err != nil {
		return Passage{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var found []ports.Chunk
	for _, c := range chunks {
		if c.Cite == cite && c.Start == span[0] && c.End == span[1] {
			found = append(found, c)
			break
		}
	}
	if len(found) == 0 {
		return Passage{Cite: cite, Gone: true}, nil
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return Passage{}, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, found, 1)
	if err != nil {
		return Passage{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		return Passage{}, ErrSourceDenied // denied or unknown: never shown (LAW-01, LAW-03)
	}
	text, cut := Excerpt(res.Context[0].Text, MaxPassage)
	return Passage{Cite: cite, Text: text, Cut: cut}, nil
}

func spanOf(r Response, cite string, start int) ([2]int, bool) {
	for _, cl := range r.Answer.Claims {
		for _, c := range cl.Cites {
			if c.Cite == cite && (start < 0 || c.Span[0] == start) {
				return c.Span, true
			}
		}
	}
	for _, c := range r.SeeAlso { // a See also document opens the same way (SB-3301)
		if c.Cite == cite && (start < 0 || c.Span[0] == start) {
			return c.Span, true
		}
	}
	return [2]int{}, false
}

// Excerpt is at most max characters of text, cut after the last full sentence that fits,
// or at the last space with an ellipsis when no sentence ends in it; cut says it was cut.
func Excerpt(text string, max int) (string, bool) {
	text = strings.TrimSpace(text)
	rs := []rune(text)
	if len(rs) <= max {
		return text, false
	}
	head := rs[:max]
	end := -1
	for i := len(head) - 1; i > 0; i-- {
		if (head[i-1] == '.' || head[i-1] == '!' || head[i-1] == '?') && (head[i] == ' ' || head[i] == '\n') {
			end = i
			break
		}
	}
	if end > max/3 {
		return strings.TrimSpace(string(head[:end])), true
	}
	if sp := strings.LastIndexAny(string(head), " \n"); sp > 0 {
		return strings.TrimSpace(string(head)[:sp]) + "…", true
	}
	return string(head) + "…", true
}

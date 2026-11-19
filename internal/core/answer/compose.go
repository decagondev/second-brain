// Package answer builds cited answers (sb-citations(7), SB-304): the prompt from allowed
// chunks, and claims from the model's output. Whether a claim is cited is decided here,
// in code, never by the model (LAW-14).
package answer

import (
	"encoding/json"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// Cite is one citation on a claim.
type Cite struct {
	Cite  string   `json:"cite"`
	Title string   `json:"title,omitempty"`
	Span  [2]int   `json:"span"`
	URL   string   `json:"url,omitempty"` // opens the source at this version, when the connector can link
	Flags []string `json:"flags"`
}

// Claim is one sentence or short passage of an answer.
type Claim struct {
	Text    string `json:"text"`
	Cites   []Cite `json:"cites"`
	Uncited bool   `json:"uncited,omitempty"`
}

// Context is this answer's allowed chunks, labelled C1, C2, … in order.
type Context struct {
	Labels map[string]ports.Chunk
	Order  []string
}

// NewContext labels allowed chunks for the prompt.
func NewContext(chunks []ports.Chunk) Context {
	c := Context{Labels: map[string]ports.Chunk{}}
	for i, ch := range chunks {
		l := fmt.Sprintf("C%d", i+1)
		c.Labels[l] = ch
		c.Order = append(c.Order, l)
	}
	return c
}

// Data renders the context as data blocks for the prompt (each label with its cite).
func (c Context) Data() string {
	var b strings.Builder
	for _, l := range c.Order {
		ch := c.Labels[l]
		b.WriteString(ingest.Wrap(ch.Text, l+" "+ch.Cite, ch.Flags))
		b.WriteString("\n")
	}
	return b.String()
}

// Result is the composed answer and what was stripped.
type Result struct {
	Claims []Claim
	Spoofs []string // labels the model used that aren't in this context (ids only)
}

// Compose turns raw model output into claims. Labels that aren't in the context are
// stripped (and returned as spoofs); a claim left without citations is uncited; output
// that isn't the expected JSON becomes one uncited claim.
func Compose(raw string, ctx Context, titles map[string]string) Result {
	var parsed struct {
		Claims []struct {
			Text  string   `json:"text"`
			Cites []string `json:"cites"`
		} `json:"claims"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil || len(parsed.Claims) == 0 {
		return Result{Claims: []Claim{{Text: strings.TrimSpace(raw), Cites: []Cite{}, Uncited: true}}}
	}
	var res Result
	for _, pc := range parsed.Claims {
		c := Claim{Text: pc.Text, Cites: []Cite{}}
		seen := map[string]bool{}
		for _, label := range pc.Cites {
			ch, ok := ctx.Labels[label]
			if !ok {
				res.Spoofs = append(res.Spoofs, label)
				continue
			}
			if seen[label] {
				continue
			}
			seen[label] = true
			flags := ch.Flags
			if flags == nil {
				flags = []string{}
			}
			c.Cites = append(c.Cites, Cite{Cite: ch.Cite, Title: titles[ch.Cite], Span: [2]int{ch.Start, ch.End}, Flags: flags})
		}
		c.Uncited = len(c.Cites) == 0
		res.Claims = append(res.Claims, c)
	}
	return res
}

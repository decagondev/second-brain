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
	Cite  string `json:"cite"`
	Title string `json:"title,omitempty"`
	// Section is the nearest heading above the chunk (SB-3002), or "".
	Section string   `json:"section,omitempty"`
	Span    [2]int   `json:"span"`
	URL     string   `json:"url,omitempty"` // opens the source at this version, when the connector can link
	Flags   []string `json:"flags"`
	// In a cell with spaces (SB-1007): the spaces that hold the cited document, by name,
	// and its label.
	Spaces []string `json:"spaces,omitempty"`
	Label  string   `json:"label,omitempty"`
	// Owner is who to ask about the cited document (SB-1702): set only for an owner who is
	// one active directory user, and only ever on a cite the asker was allowed.
	Owner *Person `json:"owner,omitempty"`
	// AgeDays is whole days since the index first saw this version, and Stale is an age
	// past the review_after_days of a space that holds it (SB-1709).
	AgeDays *int `json:"age_days,omitempty"`
	Stale   bool `json:"stale,omitempty"`
	// Reviewed is the date of a steward's review mark at this version (SB-2702), YYYY-MM-DD.
	Reviewed string `json:"reviewed,omitempty"`
	// Replaces is the title of a document a steward retired in favour of this one, shown
	// only when the asker can open it too (SB-2801).
	Replaces string `json:"replaces,omitempty"`
	// ReplacedBy is the newer decision that replaces a cited one, shown only when the
	// asker can open it too (SB-4505).
	ReplacedBy *Newer `json:"replaced_by,omitempty"`
	// GeneratedBy, ApprovedBy and Edited mark a cited note an agent drafted (SB-4706,
	// LAW-28): AGENT@VERSION, the person who published it, and whether they changed it.
	GeneratedBy string `json:"generated_by,omitempty"`
	ApprovedBy  string `json:"approved_by,omitempty"`
	Edited      bool   `json:"edited,omitempty"`
	// AskedBy and EditedBy are who asked for a cited agent note and who edited it (SB-5204).
	AskedBy  string `json:"asked_by,omitempty"`
	EditedBy string `json:"edited_by,omitempty"`
}

// Newer is a document that replaces a cited one: its cite and title.
type Newer struct {
	Cite  string `json:"cite"`
	Title string `json:"title,omitempty"`
}

// Person is a document owner as an asker sees them: their email and job title, nothing else.
type Person struct {
	Email string `json:"email"`
	Title string `json:"title,omitempty"`
	// Caretaker marks someone a steward named for a document whose owner is gone (SB-3405).
	Caretaker bool `json:"caretaker,omitempty"`
}

// Claim is one sentence or short passage of an answer.
type Claim struct {
	Text    string `json:"text"`
	Cites   []Cite `json:"cites"`
	Uncited bool   `json:"uncited,omitempty"`
	// Disagree is the model's word that the claim's sources say different things, kept only
	// when its citations name at least two documents (SB-3304).
	Disagree bool `json:"disagree,omitempty"`
}

// TwoDocuments says the cites name at least two different documents (KIND:ID).
func TwoDocuments(cs []Cite) bool {
	first := ""
	for _, c := range cs {
		d, _, _ := strings.Cut(c.Cite, "@")
		if first == "" {
			first = d
		} else if d != first {
			return true
		}
	}
	return false
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
		b.WriteString(ingest.Wrap(ingest.Shown(ch.Text, ch.Flags), l+" "+ch.Cite, ch.Flags)) // personal data masked (SB-1904)
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
			Text     string   `json:"text"`
			Cites    []string `json:"cites"`
			Disagree bool     `json:"disagree"`
		} `json:"claims"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil || len(parsed.Claims) == 0 {
		return Result{Claims: []Claim{{Text: ingest.MaskPII(strings.TrimSpace(raw), false), Cites: []Cite{}, Uncited: true}}}
	}
	var res Result
	for _, pc := range parsed.Claims {
		c := Claim{Text: ingest.MaskPII(pc.Text, false), Cites: []Cite{}} // never a whole IBAN or card (SB-1904)
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
			var held []string
			for _, id := range ch.Spaces {
				held = append(held, strings.TrimPrefix(id, "spc_"))
			}
			title := titles[ch.Cite]
			if title == "" {
				title = ch.Title
			}
			c.Cites = append(c.Cites, Cite{Cite: ch.Cite, Title: title, Span: [2]int{ch.Start, ch.End}, Flags: flags,
				Spaces: held, Label: ch.Label, Section: ch.Section, GeneratedBy: ch.GeneratedBy, ApprovedBy: ch.ApprovedBy, Edited: ch.Edited, AskedBy: ch.AskedBy, EditedBy: ch.EditedBy})
		}
		c.Uncited = len(c.Cites) == 0
		c.Disagree = pc.Disagree && TwoDocuments(c.Cites) // the model's word, kept only with two documents
		res.Claims = append(res.Claims, c)
	}
	return res
}

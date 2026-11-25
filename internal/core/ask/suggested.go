package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// A space's suggested questions (sb-ask(3), Questions to the stewards; SB-3307): the
// stewards' own starter questions, shown to the space's members when the ask box is empty.

// Bounds.
const (
	MaxSuggested     = 12 // SB-4906: starter questions; 8 before
	MaxSuggestedText = 120
)

// Suggested is one space's list.
type Suggested struct {
	Tenant    string    `json:"tenant"`
	Space     string    `json:"space"` // spc_NAME
	Questions []string  `json:"questions"`
	By        string    `json:"by"`
	At        time.Time `json:"at"`
}

// SuggestedStore keeps one list per space.
type SuggestedStore interface {
	PutSuggested(ctx context.Context, s Suggested) error
	Suggested(ctx context.Context, tenant, space string) (Suggested, bool, error)
}

// cleanSuggested is the list trimmed, or an error for one that isn't at most MaxSuggested
// lines of plain text of 1 to MaxSuggestedText characters.
func cleanSuggested(qs []string) ([]string, error) {
	if len(qs) > MaxSuggested {
		return nil, fmt.Errorf("%w: at most %d suggested questions", ErrInvalid, MaxSuggested)
	}
	out := []string{}
	for _, q := range qs {
		q = strings.TrimSpace(q)
		bad := q == "" || utf8.RuneCountInString(q) > MaxSuggestedText || !utf8.ValidString(q)
		for _, r := range q {
			bad = bad || unicode.IsControl(r)
		}
		if bad {
			return nil, fmt.Errorf("%w: a suggested question is one line of 1 to %d characters", ErrInvalid, MaxSuggestedText)
		}
		out = append(out, q)
	}
	return out, nil
}

// SetSuggested replaces a space's suggested questions: its stewards only, audited with the
// list's digest, never its text.
func (x Stewards) SetSuggested(ctx context.Context, s tenancy.Scope, name string, qs []string) ([]string, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	if x.Suggested == nil {
		return nil, fmt.Errorf("%w: this cell keeps no suggested questions", ErrUnavailable)
	}
	clean, err := cleanSuggested(qs)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(strings.Join(clean, "\n")))
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: "suggested:" + name, State: "set",
		PayloadSHA256: hex.EncodeToString(sum[:]), Reason: fmt.Sprintf("%d suggested questions", len(clean))}); err != nil {
		return nil, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Suggested.PutSuggested(ctx, Suggested{Tenant: x.Agent.Tenant, Space: SpaceKey(name), Questions: clean, By: s.User(), At: x.Now().UTC()}); err != nil {
		return nil, fmt.Errorf("%w: suggested: %v", ErrUnavailable, err)
	}
	return clean, nil
}

// ErrNotMember is someone outside a space, or a space that doesn't exist.
var ErrNotMember = errors.New("ask: not a member of this space")

// SuggestedFor is a space's suggested questions for one of its members now; anyone else,
// or a space that doesn't exist, gets ErrNotMember, which the API answers as EPERM.
func (x Stewards) SuggestedFor(ctx context.Context, s tenancy.Scope, name string) ([]string, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant || x.Agent.Auth.Spaces == nil {
		return nil, ErrNotMember
	}
	all, _, err := x.Agent.Auth.Spaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	member := false
	for _, sp := range all {
		member = member || (sp.Name == name && sp.Tenant == x.Agent.Tenant && spaces.Member(sp, s.Groups()))
	}
	if !member {
		return nil, ErrNotMember
	}
	if x.Suggested == nil {
		return []string{}, nil
	}
	got, ok, err := x.Suggested.Suggested(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return nil, fmt.Errorf("%w: suggested: %v", ErrUnavailable, err)
	}
	if !ok || got.Questions == nil {
		return []string{}, nil
	}
	return got.Questions, nil
}

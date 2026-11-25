package ask

import (
	"context"
	"fmt"

	"example.com/second-brain/internal/core/tenancy"
)

// Branch a conversation (sb-ask(3), Conversations; SB-5206): a new conversation of s's own,
// starting with the first at questions of conversation conv, so the next ask goes on from
// there in another direction. The first conversation is left as it was. Only the questions
// are copied, never an answer: everything is retrieved and checked again, as in any turn.
func (a Agent) Branch(ctx context.Context, s tenancy.Scope, conv string, at int) (string, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return "", fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	qs, found, err := a.Store.Questions(ctx, a.Tenant, s.User(), conv)
	if err != nil {
		return "", fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if !found {
		return "", fmt.Errorf("%w: no such conversation", ErrInvalid)
	}
	if at < 1 || at > len(qs) {
		return "", fmt.Errorf("%w: branch from question 1 to %d", ErrInvalid, len(qs))
	}
	next := a.NewID("cv_")
	for _, q := range qs[:at] {
		if err := a.Store.AppendQuestion(ctx, a.Tenant, s.User(), next, q); err != nil {
			return "", fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
		}
	}
	if st, ok := a.Store.(SpaceTagger); ok {
		if err := st.TagSpace(ctx, a.Tenant, next, s.Space()); err != nil {
			return "", fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
		}
	}
	return next, nil
}

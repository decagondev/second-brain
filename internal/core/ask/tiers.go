package ask

import (
	"context"
	"unicode/utf8"
)

// Model tiers (sb-evals(7), Model tiers; SB-4405).

// The tiers, and the longest question the small one takes.
const (
	TierMain             = "main"
	TierSmall            = "small"
	SmallMaxRunes        = 120
	tierKey       ctxKey = "tier"
)

type ctxKey string

// WithTier runs an eval probe on one tier, whatever the rule would choose.
func WithTier(ctx context.Context, tier string) context.Context {
	return context.WithValue(ctx, tierKey, tier)
}

// tierFor is the tier an ask goes to: small only for a plain, short, first question in a
// space that doesn't pin main, while the small tier isn't held; "" with no small model.
func (a Agent) tierFor(ctx context.Context, space string, req Request, q string, docs, earlier int) string {
	if a.Small == nil {
		return ""
	}
	if t, ok := ctx.Value(tierKey).(string); ok && (t == TierMain || t == TierSmall) {
		return t
	}
	plain := (req.Agent == "" || req.Agent == "ask") && req.File == nil && docs == 0 && req.Collection == ""
	short := utf8.RuneCountInString(q) <= SmallMaxRunes && earlier == 0
	pinned := a.PinMain != nil && a.PinMain(space)
	held := a.SmallHeld == nil || a.SmallHeld()
	if plain && short && !pinned && !held {
		return TierSmall
	}
	return TierMain
}

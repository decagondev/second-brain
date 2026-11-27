package ask

import (
	"context"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
)

// Model tiers (sb-evals(7), Model tiers; SB-4405).

// The tiers, and the longest question the small one takes.
const (
	TierMain  = "main"
	TierSmall = "small"
	// TierVision reads scans and images at ingest (SB-6101); it answers no question.
	TierVision           = "vision"
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
	pinned := a.PinMain != nil && a.PinMain(space)
	held := a.SmallHeld == nil || a.SmallHeld()
	feature, _ := ctx.Value(featureKey{}).(string)
	if pinned {
		return TierMain
	}
	if t := a.policyTier(ctx, space, feature, held); t != "" { // SB-5002, SB-5003
		return t
	}
	plain := (req.Agent == "" || req.Agent == "ask") && req.File == nil && docs == 0 && req.Collection == ""
	short := utf8.RuneCountInString(q) <= SmallMaxRunes && earlier == 0
	if plain && short && !held {
		return TierSmall
	}
	return TierMain
}

// Think harder only when it helps (sb-runtime(7), The deep tier; SB-5704): the deep tier
// writes a research synthesis, an answer with a table, a consultation's synthesis, and one
// retry of an answer whose claims are mostly uncited, and nothing else. Only while the
// evals job gated it for this release (LAW-06), never in a space whose routing policy is
// economy, and never for a person at their monthly limit.

// TierDeep is the deep tier, and the reasons the trace gives for using it.
const (
	TierDeep          = "deep"
	DeepResearch      = "research"
	DeepTable         = "table"
	DeepConsultation  = "consultation"
	DeepCitationRetry = "citation_retry"
)

// tierChoice is one ask's tier as it goes: the rule's pick, and why it went deeper.
type tierChoice struct {
	tier, why, feature string
	economy            bool
}

func economyOf(ctx context.Context) bool {
	over, _ := ctx.Value(economyKey{}).(bool)
	return over
}

// deepAllowed says the deep tier may answer in space now.
func (a Agent) deepAllowed(ctx context.Context, space string, tc *tierChoice) bool {
	if a.Deep == nil || a.DeepGated == nil || !a.DeepGated() || tc.economy {
		return false
	}
	if t, ok := ctx.Value(tierKey).(string); ok && t != TierDeep { // an eval probe on one tier
		return false
	}
	return a.Routing == nil || a.Routing(space) != RoutingEconomy
}

// goDeeper moves tc to the deep tier for the work that needs it, saying why.
func (a Agent) goDeeper(ctx context.Context, space string, tc *tierChoice, table bool) {
	if t, ok := ctx.Value(tierKey).(string); ok && t == TierDeep && a.Deep != nil { // the evals job's gate run
		tc.tier, tc.why = TierDeep, "eval"
		return
	}
	why := ""
	switch {
	case tc.feature == FeatureResearch:
		why = DeepResearch
	case tc.feature == FeatureConsultation:
		why = DeepConsultation
	case table:
		why = DeepTable
	}
	if why != "" && a.deepAllowed(ctx, space, tc) {
		tc.tier, tc.why = TierDeep, why
	}
}

func uncitedCount(cs []answer.Claim) int {
	n := 0
	for _, c := range cs {
		if c.Uncited {
			n++
		}
	}
	return n
}

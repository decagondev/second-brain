package ask

import "context"

// Features, routing policies and personal limits (sb-tenant(5), routing and limits;
// sb-admin(3), AI command centre; SB-5001 to SB-5003).

// The features an answer is counted under, by what asked it.
const (
	FeatureAsk          = "ask"
	FeatureResearch     = "research"
	FeatureBrief        = "brief"
	FeatureRoutine      = "routine"
	FeatureRhythm       = "rhythm"
	FeatureConsultation = "consultation"
	FeatureEval         = "eval"
)

// Features are the features the command centre counts, in its order.
var Features = []string{FeatureAsk, FeatureResearch, FeatureBrief, FeatureRoutine, FeatureRhythm, FeatureConsultation}

// The routing policies (SB-5002).
const (
	RoutingEconomy  = "economy"
	RoutingBalanced = "balanced"
	RoutingFrontier = "frontier"
)

type featureKey struct{}

// WithFeature counts the asks under ctx as feature f.
func WithFeature(ctx context.Context, f string) context.Context {
	return context.WithValue(ctx, featureKey{}, f)
}

// featureFor is the feature an ask counts under: its caller's, an eval's probe, research
// for the legwork agents, consultation for a consulting ask, otherwise ask.
func featureFor(ctx context.Context, req Request, probe bool) string {
	if f, ok := ctx.Value(featureKey{}).(string); ok && f != "" {
		return f
	}
	switch {
	case probe:
		return FeatureEval
	case req.Agent == AgentResearch || req.Agent == AgentPrep:
		return FeatureResearch
	case req.Agent == AgentCatchUp:
		return FeatureBrief
	case req.Consult:
		return FeatureConsultation
	}
	return FeatureAsk
}

// personal says a feature is the person's own work, which their limit moves to the economy
// tier: their asks and research. Briefs, rhythm steps and evals are work others review.
func personal(feature string) bool {
	return feature == FeatureAsk || feature == FeatureConsultation || feature == FeatureResearch
}

type economyKey struct{}

// policyTier is the tier space's routing policy, or the person's limit, picks for an ask of
// feature, or "" to leave it to the balanced rule. A held small tier is never chosen.
func (a Agent) policyTier(ctx context.Context, space, feature string, held bool) string {
	if over, _ := ctx.Value(economyKey{}).(bool); over && personal(feature) && !held {
		return TierSmall // SB-5003: at the limit, the person's own work uses the economy tier
	}
	policy := RoutingBalanced
	if a.Routing != nil {
		policy = a.Routing(space)
	}
	switch {
	case policy == RoutingFrontier:
		return TierMain
	case policy == RoutingEconomy && feature != FeatureResearch && !held:
		return TierSmall
	case policy == RoutingEconomy:
		return TierMain
	}
	return ""
}

// atLimit marks ctx when s's user has reached their personal limit this month (SB-5003).
func (a Agent) atLimit(ctx context.Context, user string) context.Context {
	if a.PersonOver == nil {
		return ctx
	}
	over, err := a.PersonOver(ctx, user)
	if err != nil || !over {
		return ctx // an unreadable meter limits nobody; the tenant's budget still holds
	}
	return context.WithValue(ctx, economyKey{}, true)
}

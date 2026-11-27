package ask

import (
	"context"
	"testing"
)

// SB-5704: the deep tier writes research syntheses, tables and consultations only, only
// while gated, never in an economy space or for a person at their limit, and an eval probe
// names it.
func TestThinkHarderOnlyWhenItHelps(t *testing.T) {
	ctx := context.Background()
	gated := true
	a := Agent{Deep: smallStub{}, DeepGated: func() bool { return gated },
		Routing: func(s string) string {
			if s == "spc_cheap" {
				return RoutingEconomy
			}
			return RoutingBalanced
		}}
	for _, c := range []struct {
		space, feature string
		table, economy bool
		want, why      string
	}{
		{"spc_ops", FeatureAsk, false, false, TierMain, ""},
		{"spc_ops", FeatureAsk, true, false, TierDeep, DeepTable},
		{"spc_ops", FeatureResearch, false, false, TierDeep, DeepResearch},
		{"spc_ops", FeatureConsultation, false, false, TierDeep, DeepConsultation},
		{"spc_ops", FeatureBrief, false, false, TierMain, ""},
		{"spc_cheap", FeatureAsk, true, false, TierMain, ""},
		{"spc_ops", FeatureAsk, true, true, TierMain, ""},
	} {
		tc := tierChoice{tier: TierMain, feature: c.feature, economy: c.economy}
		a.goDeeper(ctx, c.space, &tc, c.table)
		if tc.tier != c.want || tc.why != c.why {
			t.Errorf("%+v: %q %q", c, tc.tier, tc.why)
		}
	}
	gated = false
	tc := tierChoice{tier: TierMain, feature: FeatureResearch}
	if a.goDeeper(ctx, "spc_ops", &tc, true); tc.tier != TierMain {
		t.Fatal("an ungated deep tier was used")
	}
	tc = tierChoice{tier: TierMain}
	if a.goDeeper(WithTier(ctx, TierDeep), "spc_ops", &tc, false); tc.tier != TierDeep {
		t.Fatal("the gate run didn't use the deep tier")
	}
	tc = tierChoice{tier: TierMain, feature: FeatureResearch}
	gated = true
	if a.goDeeper(WithTier(ctx, TierMain), "spc_ops", &tc, false); tc.tier != TierMain {
		t.Fatal("a probe on main went deep")
	}
	if (Agent{}).deepAllowed(ctx, "spc_ops", &tierChoice{}) {
		t.Fatal("no deep model, allowed")
	}
}

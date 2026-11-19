package laws

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
)

// Law tests run against the system under test (NewSystem). Each is expected red
// until its ticket wires the real system in; see laws/expected_red.txt.

const runs = 300

func TestLAW01_CiteOnlyWhatTheAskerCanOpen(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { w := Gen(r); return law01(w, NewSystem(w)) })
}

func TestLAW02_NoTenantCrossing(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { w := Gen(r); return law02(w, NewSystem(w)) })
}

func TestLAW03_UnknownMeansRefuse(t *testing.T) {
	Check(t, "LAW-03", runs, func(r *rand.Rand) error { w := Gen(r); return law03(w, NewSystem(w)) })
}

func TestLAW04_AuditBeforeAct(t *testing.T) {
	Check(t, "LAW-04", runs, func(r *rand.Rand) error { w := Gen(r); return law04(w, NewSystem(w), r) })
}

func TestLAW05_RanMeansReceipt(t *testing.T) {
	Check(t, "LAW-05", runs, func(r *rand.Rand) error { w := Gen(r); return law05(w, NewSystem(w), r) })
}

func TestLAW16_TheManifestIsTheTruth(t *testing.T) {
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16(r, manifest.Check) })
}

func TestLAW16_ThePlanIsTheManifestDiff(t *testing.T) {
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16Plan(r, plan.Make) })
}

func TestLAW02_CellsShareNothing(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02Cells(r, NewCellStamper(t.TempDir())) })
}

func TestLAW16_ApplyMakesTheCellMatch(t *testing.T) {
	Check(t, "LAW-16", 100, func(r *rand.Rand) error { return law16Apply(r, t.TempDir(), NewApplier()) })
}

func TestLAW02_RequestsNeverCrossTenants(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02Requests(r, NewRouter()) })
}

func TestLAW02_TokensStayInTheirTenant(t *testing.T) {
	check := NewTokenCheck()
	Check(t, "LAW-02", 100, func(r *rand.Rand) error { return law02Tokens(r, check) })
}

func TestLAW12_DeprovisionRevokesFast(t *testing.T) {
	Check(t, "LAW-12", runs, func(r *rand.Rand) error { return law12(r, NewDeprovSystem) })
}

func TestLAW02_ScopeIsTheOnlyWayIn(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02Scope(r, NewScopedCell) })
}

func TestLAW03_PolicyUnknownRefuses(t *testing.T) {
	k := NewPolicyKernel()
	Check(t, "LAW-03", 100, func(r *rand.Rand) error { return law03Policy(r, k) })
}

func TestLAW02_PoliciesNeverCrossTenants(t *testing.T) {
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02Policies(r, ev) })
}

func TestLAW04_PolicyDecisionsAreRecordedFirst(t *testing.T) {
	k := NewDecisionKernel()
	Check(t, "LAW-04", runs, func(r *rand.Rand) error { return law04Decisions(r, k) })
}

func TestLAW18_RootKeyAlwaysHasACustodian(t *testing.T) {
	check := NewCustodyCheck()
	Check(t, "LAW-18", 60, func(r *rand.Rand) error {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "tenants"), 0o755); err != nil {
			return err
		}
		return law18(r, root, check)
	})
}

func TestLAW03_ConnectorUnknownIsNeverAllow(t *testing.T) {
	Check(t, "LAW-03", 60, func(r *rand.Rand) error { return law03Connector(r, NewConnectorWorld()) })
}

func TestLAW05_ConnectorActNeedsReceipt(t *testing.T) {
	Check(t, "LAW-05", runs, func(r *rand.Rand) error { return law05Connector(r, NewConnectorWorld()) })
}

func TestLAW07_NoSecretInLogs(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07(r, NewSecretWorld(t.TempDir())) })
}

func TestLAW13_TheIndexIsDerived(t *testing.T) {
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return law13(r, NewIngestWorld()) })
}

func TestLAW03_DriveUnknownIsNeverAllow(t *testing.T) {
	w := NewDriveWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW13_DriveIndexIsDerived(t *testing.T) {
	Check(t, "LAW-13", 40, func(r *rand.Rand) error { return law13(r, NewDriveIngestWorld()) })
}

func TestLAW09_DataIsNotInstructions(t *testing.T) {
	Check(t, "LAW-09", runs, func(r *rand.Rand) error { return law09(r, NewSanitiserWorld()) })
}

func TestLAW01_PrefilterCatchesUp(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01Prefilter(r, NewPrefilterWorld()) })
}

func TestLAW13_RebuildDrill(t *testing.T) {
	Check(t, "LAW-13", runs, law13Drill)
}

func TestLAW03_SlackUnknownIsNeverAllow(t *testing.T) {
	w := NewSlackWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW03_LinearUnknownIsNeverAllow(t *testing.T) {
	w := NewLinearWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW14_EveryClaimCitedOrLabelled(t *testing.T) {
	Check(t, "LAW-14", runs, func(r *rand.Rand) error { return law14(r, NewComposer()) })
}

func TestLAW07_NoSecretInPrompts(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Prompt(r, NewPromptWorld(t.TempDir())) })
}

func TestLAW14_AskAnswersCitedOrLabelled(t *testing.T) {
	Check(t, "LAW-14", runs, func(r *rand.Rand) error { w := Gen(r); return law14Ask(w, NewSystem(w)) })
}

func TestLAW01_ChatAnswersCiteOnlyWhatTheAskerCanOpen(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { w := Gen(r); return law01Chat(w, NewChatAsk()) })
}

func TestLAW11_AuditChainUnbroken(t *testing.T) {
	Check(t, "LAW-11", runs, func(r *rand.Rand) error { return law11(r, NewAuditChain()) })
}

func TestLAW10_KillWins(t *testing.T) {
	Check(t, "LAW-10", runs, func(r *rand.Rand) error { return law10(r, NewKillWorld()) })
}

func TestLAW15_ApprovalBindsOnePayload(t *testing.T) {
	Check(t, "LAW-15", runs, func(r *rand.Rand) error { return law15(r, NewApprovalWorld()) })
}

func TestLAW05_LinearActNeedsReceipt(t *testing.T) {
	Check(t, "LAW-05", 30, func(r *rand.Rand) error { return law05Connector(r, NewLinearWorld()) })
}

func TestLAW07_NoSecretInAudit(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Audit(r, NewAuditSecretWorld()) })
}

func TestLAW06_OnlySignedGatedBuildsRun(t *testing.T) {
	Check(t, "LAW-06", runs, func(r *rand.Rand) error { return law06(r, NewReleaseWorld()) })
}

func TestLAW08_AtMostOneBehindStable(t *testing.T) {
	Check(t, "LAW-08", runs, func(r *rand.Rand) error { return law08(r, NewRolloutWorld(stableTenants, testingTenants)) })
}

func TestLAW16_DriftIsReported(t *testing.T) {
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16Drift(r, NewDriftDetector()) })
}

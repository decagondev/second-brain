package laws

import (
	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/verify"
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

func TestLAW03_HealthNeverShowsAnUnreadableRecordAsHealthy(t *testing.T) {
	Check(t, "LAW-03", runs, law03Health)
}

func TestLAW03_BoxNeverShowsAGapAsHealthy(t *testing.T) {
	Check(t, "LAW-03", runs, func(r *rand.Rand) error { return law03Box(r, box.Check) })
}

func TestLAW06_OfflineBundlesLoadOnlyTheReleasesImages(t *testing.T) {
	Check(t, "LAW-06", runs, func(r *rand.Rand) error { return law06Bundle(r, realImageCheck) })
}

func TestLAW03_FolderUnknownIsNeverAllow(t *testing.T) {
	w := NewFolderWorld(t.TempDir())
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW13_FolderIndexIsDerived(t *testing.T) {
	Check(t, "LAW-13", 20, func(r *rand.Rand) error { return law13(r, NewFolderIngestWorld(t.TempDir())) })
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

func TestLAW01_CRMRecordPermissionsAreLive(t *testing.T) {
	Check(t, "LAW-01", 40, func(r *rand.Rand) error { return law01CRM(r, NewCRMWorld()) })
}

func TestLAW03_HubSpotUnknownIsNeverAllow(t *testing.T) {
	w := NewHubSpotWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW13_RetentionLeavesTheIndexDerived(t *testing.T) {
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return law13Retention(r, NewRetentionWorld()) })
}

func TestLAW11_ChainSurvivesRestore(t *testing.T) {
	Check(t, "LAW-11", 60, func(r *rand.Rand) error { return law11Restore(r, NewDRWorld(t.TempDir())) })
}

func TestLAW07_NoSecretInLogsEncoded(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Encoded(r, NewSecretWorld(t.TempDir())) })
}

func TestLAW09_PenTestCorpusIsFlagged(t *testing.T) {
	corpus, err := InjectionCorpus("../security/attacks/injection.txt")
	if err != nil {
		t.Fatal(err)
	}
	Check(t, "LAW-09", 30, func(r *rand.Rand) error { return law09Corpus(r, NewSanitiserWorld(), corpus) })
}

func TestLAW02_HostileHostsAreRefused(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02Attacks(r, NewRouter()) })
}

func TestLAW17_HandoffLeavesOnlyRetainedGrants(t *testing.T) {
	Check(t, "LAW-17", runs, func(r *rand.Rand) error { return law17Handoff(r, NewHandoffWorld(t.TempDir())) })
}

func TestLAW18_HandoffNeverOrphansTheKey(t *testing.T) {
	Check(t, "LAW-18", runs, func(r *rand.Rand) error { return law18Handoff(r, NewHandoffWorld(t.TempDir())) })
}

func TestLAW02_GCPCellsShareNothing(t *testing.T) {
	Check(t, "LAW-02", 40, func(r *rand.Rand) error {
		stamp, done := NewGCPCellStamper()
		defer done()
		return law02Cells(r, stamp)
	})
}

func TestLAW18_DestroyNeedsAWorkingWrap(t *testing.T) {
	Check(t, "LAW-18", runs, func(r *rand.Rand) error { return law18Wraps(r, NewWrapWorld(t.TempDir())) })
}

func TestLAW17_AccessCheckFindsEveryUnlistedPath(t *testing.T) {
	Check(t, "LAW-17", runs, func(r *rand.Rand) error {
		w, done := NewAccessWorld()
		defer done()
		return law17Access(r, w)
	})
}

func TestLAW06_TenantAccountPullsOnlySignedGatedBuilds(t *testing.T) {
	Check(t, "LAW-06", runs, func(r *rand.Rand) error {
		w, done := NewPullReleaseWorld()
		defer done()
		return law06(r, w)
	})
}

func TestLAW08_TenantCustodyCellsStayOneBehind(t *testing.T) {
	Check(t, "LAW-08", runs, func(r *rand.Rand) error { return law08(r, NewTenantCustodyRolloutWorld(stableTenants, testingTenants)) })
}

func TestLAW11_ExportPackageVerifiesOffline(t *testing.T) {
	Check(t, "LAW-11", runs, func(r *rand.Rand) error { return law11Export(r, NewExportWorld(t.TempDir())) })
}

func TestLAW16_SpacesAreExactlyAsDeclared(t *testing.T) {
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16Spaces(r, manifest.Check) })
}

func TestLAW19_SpacesOnlyNarrow(t *testing.T) {
	Check(t, "LAW-19", runs, func(r *rand.Rand) error { return law19(r, NewSpacePolicySystem()) })
}

func TestLAW20_RestrictedStaysHome(t *testing.T) {
	Check(t, "LAW-20", runs, func(r *rand.Rand) error { return law20(r, NewSpaceSystem()) })
}

func TestLAW21_MovesRevokeFast(t *testing.T) {
	Check(t, "LAW-21", runs, func(r *rand.Rand) error { return law21(r, NewMoveSystem()) })
}

func TestLAW22_BarriersHold(t *testing.T) {
	Check(t, "LAW-22", runs, func(r *rand.Rand) error { return law22(r, NewBarrierSystem()) })
}

func TestLAW24_TheTreeNeverWidens(t *testing.T) {
	Check(t, "LAW-24", runs, func(r *rand.Rand) error { return law24(r, NewTreeSystem()) })
}

func TestLAW25_AMemoryIsItsPersonsOwn(t *testing.T) {
	Check(t, "LAW-25", runs, func(r *rand.Rand) error { return law25(r, RealMemory{}) })
}

func TestLAW26_AStoredResultIsCheckedWhenRead(t *testing.T) {
	Check(t, "LAW-26", runs, func(r *rand.Rand) error { return law26(r, NewStoredSystem()) })
}

func TestLAW28_AnAgentsWritingSaysSo(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-28", runs, func(r *rand.Rand) error { return law28(r, dir, RealMarks{}) })
}

func TestLAW29_ARhythmOnlyDraftsAndStopsWithItsSubject(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-29", runs, func(r *rand.Rand) error { return law29(r, dir, RealRhythms{}) })
}

func TestLAW30_ABriefAboutAPersonDrawsOnlyOnWhatTheyPublished(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-30", runs, func(r *rand.Rand) error { return law30(r, dir, RealBriefs{}) })
}

func TestLAW27_AConsultationIsTheAskersOwn(t *testing.T) {
	Check(t, "LAW-27", runs, func(r *rand.Rand) error { return law27(r, NewConsultSystem()) })
}

func TestLAW23_GrantsAreConsentedAndRevocable(t *testing.T) {
	Check(t, "LAW-23", runs, func(r *rand.Rand) error { return law23(r, NewGrantSystem()) })
}

func TestLAW06_CellsRunOnlyAdmittedImages(t *testing.T) {
	Check(t, "LAW-06", 40, func(r *rand.Rand) error {
		w, done := NewRuntimeWorld()
		defer done()
		return law06Runtime(r, w)
	})
}

func TestLAW02_RuntimeStaysInItsProject(t *testing.T) {
	Check(t, "LAW-02", 40, func(r *rand.Rand) error {
		w, done := NewRuntimeWorld()
		defer done()
		return law02Runtime(r, w)
	})
}

func TestLAW02_PerimetersHoldEvenWithAnIAMMistake(t *testing.T) {
	Check(t, "LAW-02", 40, func(r *rand.Rand) error { return law02Perimeter(r, r.Intn(2) == 0) })
}

func TestLAW11_AnchorsAreWriteOnce(t *testing.T) {
	Check(t, "LAW-11", 40, law11WORM)
}

func TestLAW11_AnchorsAreWriteOnceOnTheBox(t *testing.T) {
	Check(t, "LAW-11", 40, law11Box)
}

func TestLAW11_MissedAnchorOrBackupAlerts(t *testing.T) {
	Check(t, "LAW-11", runs, func(r *rand.Rand) error { return lawWatched("LAW-11")(r, NewWatch()) })
}

func TestLAW13_MissedSyncAlerts(t *testing.T) {
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return lawWatched("LAW-13")(r, NewWatch()) })
}

func TestLAW17_MissedAccessCheckAlerts(t *testing.T) {
	Check(t, "LAW-17", runs, func(r *rand.Rand) error { return lawWatched("LAW-17")(r, NewWatch()) })
}

func TestLAW06_AttestationReportsWhatRuns(t *testing.T) {
	Check(t, "LAW-06", 40, func(r *rand.Rand) error {
		w, done := NewRuntimeWorld()
		defer done()
		return law06Attestation(r, w.(*runtimeWorld))
	})
}

func TestLAW17_VerifyReportsAccessPaths(t *testing.T) {
	Check(t, "LAW-17", runs, func(r *rand.Rand) error { return law17Verify(r, verify.Run) })
}

func TestLAW02_SitesReachOnlyTheirOwnCell(t *testing.T) {
	Check(t, "LAW-02", 40, func(r *rand.Rand) error { return law02Hosting(r, NewPublisher()) })
}

func TestLAW01_SearchShowsOnlyWhatTheAskerCanOpen(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01Search(r, searchSystem{}) })
}

func TestLAW12_OwnersAreActiveAndAllowed(t *testing.T) {
	Check(t, "LAW-12", runs, func(r *rand.Rand) error { return law12Owners(r, searchSystem{}) })
}

func TestLAW02_SharesStayInTheirTenant(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02Shares(r, shareSystem{}) })
}

func TestLAW01_StewardRepliesShowOnlyWhatTheAskerCanOpen(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01StewardCites(r, questionSystem{}) })
}

func TestLAW01_TheDigestNamesOnlyWhatThePersonCanOpenNow(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01Digest(r, RealDigest{}) })
}

func TestLAW13_ARebuildIsNeverInTheDigest(t *testing.T) {
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return law01Digest(r, RealDigest{}) })
}

func TestLAW21_NotificationsGoToTheStewardsThereAreNow(t *testing.T) {
	Check(t, "LAW-21", runs, func(r *rand.Rand) error { return law21Notify(r, RealNotify{}) })
}

func TestLAW07_NotificationsSayNothingOfWhatWasAsked(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law21Notify(r, RealNotify{}) })
}

func TestLAW07_EmailNotificationsSayNothingOfWhatWasAsked(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07NotifyEmail(r, RealNotify{}) })
}

func TestLAW21_AMovedStewardLosesTheQueue(t *testing.T) {
	Check(t, "LAW-21", runs, func(r *rand.Rand) error { return law21StewardQueue(r, questionSystem{}) })
}

func TestLAW04_QuestionsAreAuditedFirst(t *testing.T) {
	Check(t, "LAW-04", 100, law04Questions)
}

func TestLAW05_NotesRanMeansTheFileOnTheShare(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-05", 60, func(r *rand.Rand) error { return law05Notes(r, dir) })
}

func TestLAW01_StartHereShowsOnlyWhatAMemberCanOpen(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01StartHere(r, searchSystem{}) })
}

func TestLAW12_HandoverBriefsOnlyWhatTheAskerCanOpen(t *testing.T) {
	Check(t, "LAW-12", runs, func(r *rand.Rand) error { return law12Handover(r, searchSystem{}) })
}

func TestLAW13_ARebuildIsNeverAWeekOfChanges(t *testing.T) {
	Check(t, "LAW-13", 100, func(r *rand.Rand) error { return law13Changes(r, changeSystem{}) })
}

func TestLAW01_TheReviewListNamesOnlyWhatTheStewardCanOpen(t *testing.T) {
	Check(t, "LAW-01", 100, law01Review)
}

func TestLAW07_SupportBundlesCarryNoSecret(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Support(r, realSupportScan) })
}

func TestLAW07_ProxyCredentialsStayInAFile(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Proxy(r, realProxyCheck) })
}

func TestLAW02_ExternalDatabaseIsVerified(t *testing.T) {
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02DSN(r, realDSNCheck) })
}

func TestLAW07_NothingUnsealedReachesTheBucket(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-07", 30, func(r *rand.Rand) error { return law07Offsite(r, RealPush{}, dir) })
}

func TestLAW11_OffBoxCopiesAreSealed(t *testing.T) {
	Check(t, "LAW-11", 40, func(r *rand.Rand) error { return law11Sealed(r, realSeal) })
}

func TestLAW18_AnyThreeSharesAndNoTwo(t *testing.T) {
	Check(t, "LAW-18", runs, func(r *rand.Rand) error { return law18Kit(r, realKit) })
}

func TestLAW16_RetentionPlanIsWhatTheSweepDeletes(t *testing.T) {
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16RetentionPlan(r, RealRetentionStores) })
}

func TestLAW07_SubjectExportIsWhatEraseRemoves(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07SubjectExport(r, RealRetentionStores) })
}

func TestLAW12_SignOutEverywhereEndsEverySession(t *testing.T) {
	Check(t, "LAW-12", runs, func(r *rand.Rand) error { return law12Revoke(r, RealSessionRevoker) })
}

func TestLAW07_PersonalDataIsMasked(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07PII(r, RealPIIMasker) })
}

func TestLAW18_ARetiredBoxLeavesASignedStatement(t *testing.T) {
	Check(t, "LAW-18", runs, func(r *rand.Rand) error { return law18Retire(r, RealRetirer) })
}

func TestLAW22_NotesPublishOnTheirSideOfABarrier(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-22", 60, func(r *rand.Rand) error { return law22Notes(r, dir, RealNotePlacer) })
}

func TestLAW09_OfficeFilesAreData(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-09", 60, func(r *rand.Rand) error { return law09Office(r, dir, EncodeOffice) })
}

func TestLAW13_OfficeIndexIsDerived(t *testing.T) {
	Check(t, "LAW-13", 20, func(r *rand.Rand) error { return law13(r, NewOfficeIngestWorld(t.TempDir())) })
}

func TestLAW19_NotesStayAsNarrowAsTheirSpace(t *testing.T) {
	dir := t.TempDir()
	Check(t, "LAW-19", 60, func(r *rand.Rand) error { return law19Notes(r, dir, RealNoteNarrowing{}) })
}

func TestLAW01_MattermostAnswersCiteOnlyWhatTheAskerCanOpen(t *testing.T) {
	Check(t, "LAW-01", 50, func(r *rand.Rand) error { w := Gen(r); return law01Chat(w, mmSystem{}.ask) })
}

func TestLAW01_MattermostSharesOnlyTheAskersCites(t *testing.T) {
	Check(t, "LAW-01", 30, func(r *rand.Rand) error { return law01MattermostShare(Gen(r), false) })
}

func TestLAW01_HandoverSearchListsOnlyWhomTheAskerMayBrief(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01HandoverPeople(r, searchSystem{}, false) })
}

func TestLAW12_MattermostRefusesTheDeprovisioned(t *testing.T) {
	Check(t, "LAW-12", 30, func(r *rand.Rand) error { return law12Mattermost(Gen(r), false) })
}

// SB-2201, SB-2202
func TestLAW12_ADeprovisionedOwnersDocumentsAreOnTheReviewList(t *testing.T) {
	Check(t, "LAW-12", runs, func(r *rand.Rand) error { return law12Orphans(r, ownedSystem{}) })
}

func TestLAW01_OwnerGoneNamesOnlyWhatTheStewardCanOpen(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law12Orphans(r, ownedSystem{}) })
}

func TestLAW13_AnswersCiteTheNewestCopyBeforeAndAfterARebuild(t *testing.T) {
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return law13Copies(r, ownedSystem{}) })
}

// SB-2203
func TestLAW19_ASpacesGlossaryIsUsedOnlyInThatSpace(t *testing.T) {
	Check(t, "LAW-19", runs, func(r *rand.Rand) error { return law19Glossary(r, RealGlossary{}) })
}

func TestLAW09_GlossaryEntriesAreDataNeverInstructions(t *testing.T) {
	Check(t, "LAW-09", runs, func(r *rand.Rand) error { return law19Glossary(r, RealGlossary{}) })
}

// SB-2204
func TestLAW06_AReleaseThatRegressesATenantCaseIsHeld(t *testing.T) {
	Check(t, "LAW-06", 40, func(r *rand.Rand) error {
		w, done := NewRuntimeWorld()
		defer done()
		return law06Evals(r, w.(*runtimeWorld), RealHold{})
	})
}

// SB-2205
func TestLAW01_AskingForAccessTellsTheAskerOnlyCounts(t *testing.T) {
	Check(t, "LAW-01", runs, func(r *rand.Rand) error { return law01Access(r, RealAccess{}) })
}

func TestLAW07_AnAccessRequestNeverCarriesTheQuestion(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law01Access(r, RealAccess{}) })
}

func TestLAW11_TheSIEMGetsEveryRowInOrder(t *testing.T) {
	Check(t, "LAW-11", runs, func(r *rand.Rand) error { return law11Stream(r, RealStream{}) })
}

func TestLAW14_NumbersFromTablesComeFromTheTool(t *testing.T) {
	Check(t, "LAW-14", runs, func(r *rand.Rand) error { return law14Tables(r, RealTables) })
}

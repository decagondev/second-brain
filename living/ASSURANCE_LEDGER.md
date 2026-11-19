---
title: Assurance Ledger
summary: The status of every law, from Stated to Watched.
---

# Assurance ledger

One row per law. Status only moves up: **Stated → Modelled → Tested → Enforced → Watched**. Moving down is a Sev-1. Statuses are defined in [04-testing-and-assurance.md](../docs/04-testing-and-assurance.md#the-assurance-ledger). A law is *Modelled* only after a clean model-checker run **and** a recorded mutation check.

| Law | Short name | Status | Model | Tests | Production | Tickets |
|---|---|---|---|---|---|---|
| [LAW-01](../manual/sb-laws.7.md#law-01) | Cite only what the asker can open | Tested | `Permissions.als`: 5 checks UNSAT at scope 6; mutation caught | `laws/` check + self-tests; red until SB-302 | — | SB-010, SB-013, SB-302, SB-601 |
| [LAW-02](../manual/sb-laws.7.md#law-02) | No tenant crossing | Tested | `Permissions.als`: 5 checks UNSAT at scope 6; mutation caught | `laws/` check + self-tests; retrieval level red until SB-301; cell level (`TestLAW02_CellsShareNothing`, SB-103) request level (`TestLAW02_RequestsNeverCrossTenants`, SB-105) sign-in (`TestLAW02_TokensStayInTheirTenant`, SB-106) request context (`TestLAW02_ScopeIsTheOnlyWayIn`, SB-108) and policies (`TestLAW02_PoliciesNeverCrossTenants`, SB-110) green | — | SB-010, SB-013, SB-103, SB-108, SB-301 |
| [LAW-03](../manual/sb-laws.7.md#law-03) | Unknown means refuse | Tested | `Permissions.als`: 5 checks UNSAT at scope 6; mutation caught | `laws/` check + self-tests; retrieval level red until SB-302; policy level (`TestLAW03_PolicyUnknownRefuses`, SB-109) green | — | SB-010, SB-013, SB-109, SB-302 |
| [LAW-04](../manual/sb-laws.7.md#law-04) | Audit before act | Tested | `Approvals.tla`: TLC clean (1,068 states); 4 mutations caught | `laws/` check + self-tests through the real tool bus and audit log; ordering tests in `internal/core/toolbus` | — | SB-010, SB-011, SB-111, SB-401, SB-405 |
| [LAW-05](../manual/sb-laws.7.md#law-05) | Ran means receipt | Tested | `Approvals.tla`: TLC clean (1,068 states); 4 mutations caught | `laws/` check + self-tests through the real executor; connector-level check | — | SB-010, SB-011, SB-408 |
| [LAW-06](../manual/sb-laws.7.md#law-06) | Only signed, gated builds run | Tested | `Rollout.tla`: TLC clean (6,290 states); 4 mutations caught | `laws/` check through the real bundle, signer, release store and admission, + self-test; admission refusal tests; reproducible-build check | — | SB-012, SB-506, SB-509 |
| [LAW-07](../manual/sb-laws.7.md#law-07) | No secret in prompt, log or audit | Tested | — | `laws/` canary check: resolved secrets never reach logs by any path, nor the model provider through the gateway, nor audit rows; CI canary scan (`tests/canary`), + self-tests | — | SB-205, SB-411 |
| [LAW-08](../manual/sb-laws.7.md#law-08) | At most one behind stable | Modelled | `Rollout.tla`: TLC clean (6,290 states); 4 mutations caught | — | — | SB-012, SB-507 |
| [LAW-09](../manual/sb-laws.7.md#law-09) | Data is not instructions | Tested | — | `laws/` ingest-level check (flag, strip, unescapable wrap) + self-test; injection corpus in `make check`; agent level with SB-405/SB-501 | — | SB-207, SB-501 |
| [LAW-10](../manual/sb-laws.7.md#law-10) | Kill wins | Tested | `Approvals.tla`: TLC clean (1,068 states); 4 mutations caught | `laws/` timed test on a fake clock (random kills, unkills, store outages) through the real control, store and checker, + self-test; drill test | — | SB-011, SB-403, SB-404 |
| [LAW-11](../manual/sb-laws.7.md#law-11) | Audit chain unbroken | Tested | — | `laws/` tamper test over exports (edit, drop, swap, duplicate, cut-off, forged anchor) through the real log, export and verifier, + self-test; Postgres append-only trigger test | — | SB-401, SB-402, SB-412 |
| [LAW-12](../manual/sb-laws.7.md#law-12) | Deprovision revokes fast | Tested | — | `laws/` timed check on a fake clock, with and without lost fan-out, + self-test | — | SB-107 |
| [LAW-13](../manual/sb-laws.7.md#law-13) | The index is derived | Tested | — | `laws/` ingest, re-ingest, drop and rebuild give the same `(cite, chunk, ACL)` set, + self-test; scheduled drill SB-210 | — | SB-210 |
| [LAW-14](../manual/sb-laws.7.md#law-14) | Every claim cited or labelled | Tested | — | `laws/` property test over random model output (valid, spoofed and garbage labels) through `answer.Compose`, end to end through the ask agent, and in the web UI's rendering (`web/src/ui.test.tsx`), + self-tests | — | SB-304, SB-309 |
| [LAW-15](../manual/sb-laws.7.md#law-15) | Approval binds one payload | Tested | `Approvals.tla`: TLC clean (1,068 states); 4 mutations caught | `laws/` property test (stale, made-up and self approvals, edits) through the real bus, + self-test; canonical-hash tests | — | SB-011, SB-406 |
| [LAW-16](../manual/sb-laws.7.md#law-16) | The manifest is the truth | Tested | — | `laws/` manifest check (round trip and rejection) plan check (plan is the manifest diff) and apply check (recorded state equals the manifest; re-apply changes nothing), + self-tests; drift side with SB-510 | — | SB-101, SB-510, SB-602 |
| [LAW-17](../manual/sb-laws.7.md#law-17) | Hand-off leaves no unlisted access | Modelled | `Handoff.tla`: TLC clean (23 states); 4 mutations caught | — | — | SB-705, SB-706 |
| [LAW-18](../manual/sb-laws.7.md#law-18) | The root key always has a custodian | Tested | `Handoff.tla`: TLC clean (23 states); 4 mutations caught | `laws/` W1 check: recorded custodian and working root key, + self-test; hand-off in W7 | — | SB-114, SB-704, SB-705 |

## History

| Date | Law | From → To | Evidence |
|---|---|---|---|
| 2026-11-18 | LAW-01 to LAW-16 | — → Stated | `sb-laws(7)` first draft |
| 2026-11-18 | LAW-17, LAW-18 | — → Stated | Factory and hand-off design (ADR-0015) |
| 2026-11-18 | LAW-01 to LAW-05 | Stated → Tested (red) | SB-010: law checks proven by self-tests; ratchet in `laws/expected_red.txt` |
| 2026-11-19 | LAW-06, LAW-08, LAW-10, LAW-15, LAW-17, LAW-18 | Stated → Modelled | `make proofs` clean on TLC (tla2tools v1.8.0 release) and Alloy 6.2.0; 13 of 13 mutations caught (`tools/ci/mutations.py`); devlog 2026-11-19 |
| 2026-11-19 | LAW-01 to LAW-05 | stays Tested (red) | Models now checked (same evidence); status doesn't move down to Modelled |
| 2026-11-19 | LAW-16 | Stated → Tested | SB-101: `TestLAW16_TheManifestIsTheTruth` green; three broken checkers caught by `TestSelf_LAW16MutationsAreCaught` |
| 2026-11-19 | LAW-16 | stays Tested | SB-102: `TestLAW16_ThePlanIsTheManifestDiff` green; planners that drop removals or changes are caught |
| 2026-11-19 | LAW-02 | stays Tested (red) | SB-103: cell-level isolation test green over 300 seeds; three broken templates caught; retrieval-level test still red (SB-301) |
| 2026-11-19 | LAW-16 | stays Tested | SB-104: `TestLAW16_ApplyMakesTheCellMatch` green on the local control plane; an applier that skips removals is caught |
| 2026-11-19 | LAW-02 | stays Tested (red) | SB-105: request-level routing test green through the real edge router; a router that routes by token tenant is caught |
| 2026-11-19 | LAW-02 | stays Tested (red) | SB-106: sign-in test green with real OIDC verification and test IdPs; a cell that ignores the tenant claim is caught |
| 2026-11-19 | LAW-12 | Stated → Tested | SB-107: `TestLAW12_DeprovisionRevokesFast` green through the real SCIM endpoint and session checks; a cell that skips re-validation is caught |
| 2026-11-19 | LAW-02 | stays Tested (red) | SB-108: scope test green through the real middleware; a directory that ignores the scope is caught; only the retrieval-level test (SB-301) is still red |
| 2026-11-19 | LAW-03 | stays Tested (red) | SB-109: policy-kernel test green under injected errors, timeouts, panics and invalid values; a "not denied" caller is caught; retrieval level still red (SB-302) |
| 2026-11-19 | LAW-02 | stays Tested (red) | SB-110: policy-level test found and now guards BUG-0006 (tenant-less `use_agent` permit) |
| 2026-11-19 | LAW-04 | stays Tested (red) | SB-111: decisions recorded before return, failed write is Unknown; a kernel that ignores a failed write is caught |
| 2026-11-19 | LAW-18 | Modelled → Tested | SB-114: `TestLAW18_RootKeyAlwaysHasACustodian` green on the local control plane; missing custodian and dead root key caught |
| 2026-11-19 | LAW-03, LAW-05 | stay Tested (red) | SB-201: connector-level tests green on the mock (no Allow under any fault; receipt or error, idempotent Act); the retrieval and tool-call levels are still red |
| 2026-11-19 | LAW-07 | Stated → Tested | SB-205: `TestLAW07_NoSecretInLogs` green through the real Resolver, local vault and redacting logger |
| 2026-11-19 | LAW-13 | Stated → Tested | SB-206: `TestLAW13_TheIndexIsDerived` green on the real pipeline over the mock source |
| 2026-11-19 | LAW-03, LAW-13 | no change | SB-203: connector-level LAW-03 and LAW-13 tests also run on the Google Drive connector and its recorded fake |
| 2026-11-19 | LAW-09 | Stated → Tested | SB-207: `TestLAW09_DataIsNotInstructions` green through the real pipeline; a pipeline without a sanitiser is caught |
| 2026-11-19 | LAW-01 | stays Tested (red) | SB-209: ingest-gap test green (live check follows the source; prefilter catches up after one sync); retrieval level red until SB-302 |
| 2026-11-19 | LAW-01, LAW-02, LAW-03 | Tested (red) → Tested | S3.1 (SB-301 to SB-303): the retrieval-level tests, red since W0, are green on the real pipeline; mutations of the real pipeline are caught |
| 2026-11-19 | LAW-14 | Stated → Tested | SB-304: `TestLAW14_EveryClaimCitedOrLabelled` green on the real composer |
| 2026-11-19 | LAW-07 | Tested (extended) | SB-305: `TestLAW07_NoSecretInPrompts` green through the model gateway and the recorded fake provider |
| 2026-11-19 | LAW-01, LAW-14 | Tested (extended) | SB-306: the ask-level law tests run through the real ask agent and composer; `TestLAW14_AskAnswersCitedOrLabelled` added, with a mutation that cites refused candidates caught |
| 2026-11-19 | LAW-14 | Tested (extended) | SB-307: the web UI renders every claim as cited or labelled Uncited, never plain text; property test over random answers, mutation caught |
| 2026-11-19 | LAW-01 | Tested (extended) | SB-308: `TestLAW01_ChatAnswersCiteOnlyWhatTheAskerCanOpen`, red first, green through the Slack app, the email-to-directory mapping and the ask agent; a wrong-person mutation is caught |
| 2026-11-19 | LAW-14 | Tested (measured) | SB-309: the citation suite measures support per claim on the local cell; a misattributing model must fail it in `make evals` |
| 2026-11-19 | LAW-11 | Stated → Tested | SB-401, SB-402: `TestLAW11_AuditChainUnbroken` red first, then green; verifier mutation caught |
| 2026-11-19 | LAW-10 | → Tested | SB-403, SB-404: `TestLAW10_KillWins` red first, then green; stale-checker mutation caught |
| 2026-11-19 | LAW-04, LAW-05 | Tested (red) → Tested | SB-405, SB-408: green on the real tool bus; the red ratchet is empty for the first time |
| 2026-11-19 | LAW-15 | Stated → Tested | SB-406: `TestLAW15_ApprovalBindsOnePayload` red first, then green; ignore-the-echo mutation caught |
| 2026-11-19 | LAW-15 | Tested (extended) | SB-407: the approver view echoes the hash it showed; a UI test changes the payload after load and checks the stale hash is sent, so the server refuses |
| 2026-11-19 | LAW-04, LAW-05, LAW-15 | Tested (extended) | SB-410: end to end through the manifest's registry, real cedar policies, the bus and the Linear fake; `TestLAW05_LinearActNeedsReceipt` on the Linear write |
| 2026-11-19 | LAW-07 | Tested (extended) | SB-411: `TestLAW07_NoSecretInAudit` red first, then green; CI canary scan over logs, prompts and audit exports with a leak control |
| 2026-11-19 | LAW-11 | Tested (extended) | SB-412: the restore drill verifies the chain on every backup copy; a tampered copy is named and fails the drill |
| 2026-11-19 | LAW-06 | Stated → Tested (red) | SB-503: `TestLAW06_OnlySignedGatedBuildsRun` written, red for the right reason until signing and admission (SB-504, SB-506) |
| 2026-11-19 | LAW-01, LAW-03, LAW-09, LAW-14 | Tested (measured) | SB-501, SB-502: permission, injection, refusal, citation and regression suites on the local cell, gated on every `make check` |
| 2026-11-19 | LAW-06 | Tested (red) → Tested | SB-504, SB-506: green on real bundles, Ed25519 and cosign verifiers, and admission; a cell ignoring the gate is caught |

---
title: Devlog
summary: The running story of the project, newest first.
---

# Devlog

Every PR adds an entry, and so does every significant design step. Newest first. Entries are never rewritten; corrections go in a new entry. Format: [05-living-documents.md](../docs/05-living-documents.md#devlog-entry-format).

## 2026-11-19 S7.1a The hand-off, specified (SB-701)

**Wave/slice:** W7 / S7.1a · **Author:** platform lead + Claude · **Status:** In review (security and tenant review)

**What:** `manual/sb-factory.1.md`, a final sb-factory(7) with "The hand-off, exactly", `docs/runbooks/handoff.md`, `laws/law17.go` (the hand-off world, LAW-17 and LAW-18 checks), and a reference controller with four mutants in the self-tests.

**Why this way:** Manual first, and the manual follows the model. The table of phases is `Handoff.tla` written out for people, and the law tests check its invariants after every random operation, so the controller (SB-704, SB-705) has a precise target before a line of it exists. Writing a reference controller from the model shows the checks pass on something correct, and the mutants show they are not vacuous. Without that, a red test proves only that it is red.

**Learned:** The model's dual approval says nothing about who approves, so a planner approving their own plan would satisfy it. The manual now requires a second factory operator, and the law checks it. The model may follow (W7 follow-up).

**Laws:** LAW-17 and LAW-18 move to Tested (red) through the ratchet, with two lines in `expected_red.txt`, both SB-705. Everything else is green. LAW-17 now has a **Watch** line, so the alert test required a rule: `sb_unlisted_access_paths` above 0 is Sev-1, with `docs/runbooks/LAW-17.md`. It fires once SB-706 emits the metric.

**Next:** S7.1b, the factory roles (SB-702).

## 2026-11-19 S6.4b Pen-test readiness (SB-612)

**Wave/slice:** W6 / S6.4b · **Author:** platform lead + Claude · **Status:** In review (the external test is human)

**What:** `security/` (scope, rules of engagement, `findings.yaml`, `attacks/injection.txt`), `tools/check/findings.py` in `make check`, three law tests (`law02attack.go`, `law07encoded.go`, `law09corpus.go`), and fixes in `internal/obs` (encoded secrets, credential-named attributes) and `internal/core/ingest` (Markdown role labels). The manual changed first: sb-laws(7) gained External review, and sb-observability(7) the redactor's encodings. The threat model was refreshed.

**Why this way:** A pen test is worth most when its findings can't be forgotten, so the register is checked by the build: a critical or high finding that isn't fixed fails `make check`. Before the outside team starts, we attacked the three laws in the ticket ourselves the way a tester would. Two of the three tests went red, which makes them the register's first two entries, recorded and fixed the same way an external finding would be.

**Learned:** The LAW-07 and LAW-09 tests were too polite. One only logged secrets made of letters, digits and dashes, exactly as written. The other drew from five fixed attacks, none of them in Markdown. Both now generate hostile inputs, and the corpus is a file a tester can add to without writing Go.

**Laws:** LAW-02, LAW-07 and LAW-09 extended. LAW-07 and LAW-09 went red first and are green after the fixes, and LAW-02 was green first time. All law tests are green.

**Next:** W7.

## 2026-11-19 S6.4a DR game day tooling (SB-613)

**Wave/slice:** W6 / S6.4a · **Author:** platform lead + Claude · **Status:** In review (the game day itself is human)

**What:** `internal/core/gameday` (Restore, Score, SameIndex), `tests/gameday` (the rehearsal, now in `make check`), `laws/law11restore.go`, and `docs/runbooks/dr-game-day.md`. The manual changed first: sb-observability(7) gained the game day's scenario, targets and scoring.

**Why this way:** A game day is only worth something if it's scored the same way every time, so the score is code and a day passes only when every line passes. The restore reuses the drill's per-copy checks instead of adding new ones. It writes only into empty stores and checks each row's `seq` and `prev` as it goes, so a restore can't splice two chains together. When two good copies disagree there's no safe automatic choice, so it stops and a human chooses.

**Learned:** The architecture page promises an RPO of 15 minutes from continuous log archiving, but the local adapters only back up nightly. The rehearsal now says so plainly by failing the RPO when the backup is an hour old, instead of hiding it. The real RPO comes with W7's cloud adapter.

**Laws:** LAW-11 extended (`TestLAW11_ChainSurvivesRestore`, with its mutation caught). LAW-13 is covered by the rebuild drill and checked again in the rehearsal. All green.

**Next:** S6.4b, the pen test and threat model refresh (SB-612).

## 2026-11-19 S6.3c Retention and deletion (SB-611)

**Wave/slice:** W6 / S6.3c · **Author:** platform lead + Claude · **Status:** In review (nightly schedule, W7)

**What:** `internal/core/retention` (Sweep, Erase), retention views on the trace store, `internal/adapters/logs/localfs` (daily log files that can expire and erase), `POST /v1/admin/subjects/erase`, the `retention` audit kind, and `laws/law13retention.go`. The manual changed first: sb-tenant(5) gained Retention and deletion, sb-admin(3) the endpoint, and sb-audit(5) the kind.

**Why this way:** Retention is a list of stores that the job is given, and the index and audit are deliberately not on it. The index is derived and ages with its sources (LAW-13). Audit has its own seven-year WORM policy. The law test makes that a checked property instead of a convention. Erasure audits first, the same way the tool bus does (LAW-04): an erasure nobody can prove was asked for is worse than one that waits for the audit log to come back.

**Learned:** Conversations had no timestamp, so they couldn't be aged; they now carry the time of their last question. Log erasure matches the user id as a whole JSON value, never as a substring, so erasing `usr_ann` keeps `usr_anna`'s lines.

**Laws:** LAW-13 extended (`TestLAW13_RetentionLeavesTheIndexDerived`, with two mutations caught). LAW-11 accepts the new kind. All green.

**Next:** S6.4, the pen test and threat model (SB-612) and the DR game day (SB-613). Both are human-led, so this slice builds their tooling and runbooks.

## 2026-11-19 S6.3b HubSpot CRM connector (SB-610)

**Wave/slice:** W6 / S6.3b · **Author:** platform lead + Claude · **Status:** In review (real sandbox)

**What:** `internal/adapters/connectors/hubspot` and its recorded fake `hubspottest`, plus `laws/law01crm.go` (the CRM LAW-01 world, the reference model, and the LAW-03 world). The manual changed first: sb-connector(3) gained `crm: hubspot`, and sb-permissions(7) the `hubspot:users` broad token.

**Why this way:** HubSpot's API doesn't expose permission sets, so the tenant states each role's view level (`all`, `team` or `owned`) in the vault next to the token. A role that isn't stated is `Unknown`, not a guess. The prefilter is the owner's email plus a broad token, so role and team edits can't leave a stale prefilter that hides a record from someone who can now see it, and the live check does all the deciding. Contacts are personal data, and we don't index them.

**Learned:** The contract kit wants reassignments reported as permission-only changes. HubSpot's search results carry `hubspot_owner_assigneddate`, and comparing it with `hs_lastmodifieddate` and `createdate` tells a reassignment apart from an edit or a new record without keeping any per-record state in the cursor.

**Laws:** LAW-01 extended (`TestLAW01_CRMRecordPermissionsAreLive`, with its mutation caught), and LAW-03 over the new connector. All green.

**Next:** S6.3c, retention and deletion (SB-611).

## 2026-11-19 S6.3a Alerts, telemetry and budgets (SB-607 to SB-609)

**Wave/slice:** W6 / S6.3a · **Author:** platform lead + Claude · **Status:** In review (running cell, staging)

**What:** `internal/core/alerts` with ten runbooks, `obs.Catalog`, histograms, the label rule, ask and ACL metrics, a tenant dashboard, and `internal/core/budget` in the gateway. The manual changed first: sb-observability(7) gained "Law alerts, exactly", "Telemetry labels" and "Budgets and rate limits", and sb-tenant(5) the budget's meaning.

**Why this way:** Law alerts are driven by the manual. A test reads sb-laws(7), so a law that gains a production check can't ship without an alert and a runbook. The label rule fails closed: a bad value becomes `invalid` rather than being trusted, and it is counted so the bug is visible. The budget check uses the call's worst case before calling and real usage after, so a call can't overshoot because tokens are only known later. A meter that can't be read refuses, because budgets that fail open are not budgets.

**Learned:** `ModelRequest` had no idea who a call was for, so the rate limit needed a `User` field. Adapters never serialise it, and their tests show the request bodies, so it can't leak to a provider.

**Laws:** unchanged and all green; LAW-07 is reinforced by the label rule.

**Next:** S6.3b, the HubSpot CRM connector (SB-610) and retention and deletion (SB-611).

## 2026-11-19 S6.2 Onboarding and the weekly report (SB-604 to SB-606)

**Wave/slice:** W6 / S6.2 · **Author:** platform lead + Claude · **Status:** In review (dry run, friendly tenant, delivery)

**What:** The runbook's steps and gates tightened and checked by a lint, `internal/core/report`, and the usage endpoint. The manual changed first: sb-onboarding(7) now gives an expected result for every step and a checklist for every gate, and sb-observability(7) defines every line of the weekly report.

**Why this way:** A runbook a company's IT team follows unaided can't have a step whose success they have to guess, so the lint makes "every step says what to expect" a build rule rather than a style hope. The report counts gaps by kind, not by question, because a "top unanswered questions" list would be content. It is built from the same records the rest of the platform keeps, so there is no second analytics store to secure.

**Learned:** The first lint run found three steps without expectations, two of them on day 0, the first day a company meets the platform. The tenant-eval step had also pointed at `--model mock`, which needs canned responses a tenant doesn't have; it now uses the local cell.

**Laws:** unchanged, all green.

**Next:** S6.3: per-tenant observability (SB-607), a Sev-1 for every law violation (SB-608), cost controls (SB-609), the HubSpot CRM connector (SB-610), and retention and deletion (SB-611).

## 2026-11-19 S6.1b Admin console and roles (SB-602, SB-603)

**Wave/slice:** W6 / S6.1b · **Author:** platform lead + Claude · **Status:** SB-603 Done, SB-602 In review (manifest repo integration)

**What:** The operator role in base policy, `manifest.Format`, `Patch` and `Diff`, the admin API, and the web console. The manual changed first: sb-admin(3) gained "Roles, exactly" and "The manifest diff, exactly".

**Why this way:** The console proposes manifest changes and never applies config directly. The manifest stays the only way the cell's configuration changes (LAW-16), and every proposal is audited even if it's never applied. The patch goes through the exact checker `sb tenant check` uses, by formatting the patched manifest and checking that text. Validation has one implementation, and `Format`'s round-trip test keeps it honest. Kill switches are the one thing that acts at once, as the manual says, and "make permanent" is just a proposed diff.

**Learned:** Writing `Format` needed no new parser knowledge: defaults the checker applies come back out explicitly, so a formatted manifest is a fully spelled-out one. That's exactly what a reviewer of a console proposal wants to see.

**Laws:** all green. LAW-02 at the policy level holds with the new operator permit, and LAW-16 now covers console changes too.

**Next:** S6.2, onboarding: the runbook (SB-604), a friendly tenant unaided (SB-605, human), and the weekly leadership report (SB-606).

## 2026-11-19 S6.1a Tenant verify (SB-601)

**Wave/slice:** W6 / S6.1a · **Author:** platform lead + Claude · **Status:** In review (a deployed cell's endpoint, W7)

**What:** `internal/core/verify` and a local-cell target in `tools/evalcell --verify`. The manual changed first: sb-tenant(1) gained "How verify decides, exactly".

**Why this way:** The laws suite proves the code over generated worlds. Verify asks the same questions of one tenant's real data, so a misconfigured connector or a source whose permission model we misread shows up before staff do. LAW-01 re-asks the source live instead of trusting the answer's own record, and also requires that record, because "the source said yes later" isn't the same as "the source was asked". The target interface is the seam between local and deployed cells: the runner never changes, only the target does.

**Learned:** The local target needed the eval cell's construction as a reusable function. That refactor (`build`) now serves both the eval harness and verify, so the two can't disagree about what a cell is. The issuer rule kept the target inside `tools/evalcell`, the one harness allowed to mint scopes, rather than in a new production package.

**Laws:** LAW-01, LAW-02, LAW-03 and LAW-14 are checked by verify as well as by the laws suite. All green.

**Next:** S6.1b, the admin console and API, where every change is a manifest diff and an audit row (SB-602), and console roles (SB-603).

## 2026-11-19 S5.3b Attestation and drift (SB-509, SB-510), W5 built

**Wave/slice:** W5 / S5.3b · **Author:** platform lead + Claude · **Status:** In review (cell loops, W7)

**What:** `internal/core/attest` (signed cell reports and the controller's check) and `internal/core/drift` (live config against the applied manifest). The manual changed first: sb-release(7) gained the attestation rules and findings, and sb-tenant(5) gained "DRIFT".

**Why this way:** Cells push and the controller only listens, so attestation works for a cell in any account, including after hand-off (W7). The report is signed with the cell's own key, so a compromised control plane can't fake a healthy cell and a compromised cell can't impersonate another. `seq` makes a replayed good report useless. Silence is a finding in its own right, because a cell that stops reporting is the one to worry about. Drift reports and never repairs: repairing would make the live state the source of truth, which LAW-16 forbids.

**Learned:** The drift law's first draft compared only attributes, and its mutation (missing whole resources) showed why the property must also require resources present on one side only. W5 is built. Releases are bundles with derived ids; they are reproducible, SBOM'd and signed (keyless in CI), admitted only when gated, rolled out by a controller that is the TLA+ model in code, attested while running, and checked for drift.

**Laws:** all green, LAW-06 and LAW-16 extended.

**Next:** W6, starting with `sb tenant verify` (SB-601).

## 2026-11-19 S5.3a The release controller and rollback (SB-507, SB-508)

**Wave/slice:** W5 / S5.3a · **Author:** platform lead + Claude · **Status:** In review (real cells, W7)

**What:** `internal/core/rollout`, a timed rollback drill, and `sb release status|promote|rollback`. The manual changed first: sb-release(1) gained "How the controller decides, exactly".

**Why this way:** The controller is a transcription of `Rollout.tla`, with the same moves, guards and names, and it checks the model's invariants after every move. A bug that slips past a guard can therefore never reach a state the model forbids; the move is refused instead. That's the same defence in depth as the policy kernel: the guard is the rule, the invariant check is the proof. Soak time is measured per ring from the first deploy. Draining comes straight from BUG-0001, and a failed redeploy leaves a cell draining, never back on the revoked build.

**Learned:** The first cut of the LAW-08 mutation (promoting without the guard) was caught twice over: by the law, and by the controller's own invariant check on the next promotion. That is the point of having both.

**Laws:** LAW-08 moves from Stated to Tested. Every law test is green.

**Next:** S5.3b, running-build attestation per cell (SB-509) and manifest drift detection (SB-510).

## 2026-11-19 S5.2b Lockfiles and the CVE path (SB-505)

**Wave/slice:** W5 / S5.2b · **Author:** platform lead + Claude · **Status:** In review (first scheduled run, testing channel)

**What:** `make lockfiles`, a lockfile-currency step in the release workflow, a daily advisory workflow, and the patch-release playbook. The manual changed first: sb-release(7) "CVEs" now says exactly what is regenerated, what the watcher runs, and the 24-hour path.

**Why this way:** "Regenerated every release" is enforced by failing the release when regenerating would change a lockfile, not by regenerating silently in CI. That way every dependency change is a reviewed commit, and the build stays reproducible from the repository alone. `govulncheck` reports only reachable vulnerabilities, so the watcher's failures are ones that matter, and the playbook doesn't train people to ignore it. The playbook skips nothing: a CVE fix goes through the same gate and signature as any release.

**Learned:** Nothing surprising. Lockfiles were current, and `npm audit` was clean.

**Laws:** unchanged, all green. LAW-08 comes with the promote guard (SB-507).

**Next:** S5.3, the release controller with channels and rings and the promote guard (SB-507, LAW-08), one-step rollback (SB-508), running-build attestation (SB-509), and manifest drift detection (SB-510).

## 2026-11-19 S5.2a Reproducible builds, SBOM, signing and admission (SB-504, SB-506)

**Wave/slice:** W5 / S5.2a · **Author:** platform lead + Claude · **Status:** In review (first CI signature, cell pull loop)

**What:** A reproducible-build check, an SPDX SBOM generator, `make release-build`, a release workflow with cosign keyless signing, `release.Admit` with Ed25519 and cosign verifiers, a local release store, and LAW-06 green. The manual changed first: sb-release(7) gained the exact build, SBOM, signing and gate rules, and "What a cell admits".

**Why this way:** The signature covers one digest, the bundle's, and the bundle lists every artefact's digest, the SBOM's included. So one signature check plus digest checks covers everything a cell runs. The gate's verdict is a signed record of its own, bound to the release id, so passing the gate can't be claimed by editing a file. Both verifiers refuse a signature whose key or identity isn't the one the cell trusts. The cosign adapter refuses when cosign is missing: no verifier means no start. Cells pull, so the factory never needs a way in.

**Learned:** The first LAW-06 world rebuilt identical bundles and found them "already signed", which is correct: same contents, same release. The law now asks for a fresh bundle when it wants an unsigned one. A safety check refused a repro script that cleaned up with `rm -rf` on a variable, so the script now cleans its caches with `go clean -cache` and never removes a directory. Vendored builds carry no module hashes in their build info, so the SBOM takes them from `go.sum`.

**Laws:** LAW-06 moves from Tested (red) to Tested. Every law test is green.

**Next:** S5.2b, lockfile regeneration and the CVE patch path (SB-505).

## 2026-11-19 S5.1b Eval suites and the gate (SB-501, SB-502)

**Wave/slice:** W5 / S5.1b · **Author:** platform lead + Claude · **Status:** In review (staging runs, branch protection)

**What:** Four new shared suites with their worlds, fault injection and stale prefilters in the local cell, results stored per release, and `evals gate`. The manual changed first: sb-eval(1) gained `--results`, `--release` and `gate`, and sb-evals(7) "The suites, exactly".

**Why this way:** The gate reads stored results instead of running suites itself, so the same results can be checked again, compared with stable, and archived with the release. A missing result fails: "we didn't run it" must never look like "it passed". Scores mean different things per suite (support for citation, pass rate for the rest), so the gate file stays the same shape as in the manual.

**Learned:** The first injection world hid a canary between zero-width characters, and the scanner-minded test "failed": the sanitiser stripped the invisible characters and left the canary, which was visible text all along. Real hidden text lives in comments and hidden styles, so that's where the world now hides it. The local cell's refusal case also had to change: a stand-in model answers from whatever it is given, so "refuse" on the local cell means "nothing allowed came back". Judging a model's own refusals needs the real model on staging.

**Laws:** all green except LAW-06 (red until SB-506).

**Next:** S5.2, reproducible builds with an SBOM and signing (SB-504), lockfiles and the CVE path (SB-505), and deploy-time verification (SB-506).

## 2026-11-19 S5.1a The release bundle (SB-503)

**Wave/slice:** W5 / S5.1a · **Author:** platform lead + Claude · **Status:** In review (signing and admission next)

**What:** `internal/core/release` (bundle, canonical form, id, check), `internal/core/canonjson` (shared with approvals), `tools/release/bundle`, `release/model.json`, and the LAW-06 law test, red. The manual changed first: sb-release(7) now gives the bundle's fields and the id rule.

**Why this way:** The id is derived from the contents, so "any change is a new release" is arithmetic, not a process rule. Tool schemas are hashed as text, pinned with the release, so a change to what an approver can be shown is a release too. The model lives in a file in the repository, so changing it is a reviewed commit and a new id, never a runtime setting (ADR-0013). Canonical JSON had one home in approvals. A second user meant moving it to a package of its own, rather than having release import approvals.

**Learned:** The bundler's own test runs against the real repository. That way, a new prompt file or policy is included automatically, and a missing input fails the build rather than producing a bundle that quietly omits it.

**Laws:** LAW-06 moves from Stated to Tested (red), for the right reason, until SB-504 and SB-506.

**Next:** S5.1b, the eval suites with results stored per release, and the gate (SB-501, SB-502).

## 2026-11-19 S4.4b Backups and the restore drill (SB-412), W4 built

**Wave/slice:** W4 / S4.4b · **Author:** platform lead + Claude · **Status:** In review (cloud WORM bucket, monthly schedule)

**What:** `internal/core/backup` (the 3-2-1 check, the nightly job, the restore drill and its report), a write-once local target, and an end-to-end test. The manual changed first: sb-audit(7) "Retention and backup" now says exactly what is copied, where, for how long, and what the drill checks.

**Why this way:** The backup reuses the signed export, so a restored copy verifies with the same `audit.Verify` and pinned key as any export, with no second verifier to maintain. The 3-2-1 rule is checked by the job itself, not assumed from configuration, and a write-once target means a compromised job can add a night but never rewrite one. The drill checks every copy, not just one, because the copy you'd restore from after an incident is the one nobody looked at.

**Learned:** The first version of the drill test lived in the local target's package and imported the memory stores, which the dependency rule forbids for sibling adapters. Moving it to `tests/backup` is the honest place anyway: it is a system test of three packages working together.

**Laws:** LAW-11 now also holds over backups. All laws green. W4 is built: every tool call goes through the bus, every approval is bound to its payload, every outcome needs a receipt, and the audit chain is verified, backed up and drilled.

**Next:** W5, evals against real cells and the release gate (SB-501 onward).

## 2026-11-19 S4.4a Canaries and the secret scanner (SB-411)

**Wave/slice:** W4 / S4.4a · **Author:** platform lead + Claude · **Status:** In review (running-cell wiring, source sandboxes)

**What:** Canaries, a scanner, Sev-1 alerts through a webhook, redaction in the audit log, the gateway hook, and a CI scan. The manual changed first: sb-observability(7) gained "Canaries and the secret scanner", sb-audit(5) the redaction rule, sb-tenant(5) the `canary` vault path, and sb-laws(7) the checks.

**Why this way:** A canary is a secret nothing legitimate ever uses, so any sighting is a leak, with no false positives to tune. The scanner learns every resolved secret too, through the same registrar as the redactor, so the two can't drift apart. Findings and alerts carry ids and line numbers, never values: an alert that quoted the leak would be a second leak. The audit log scrubs before hashing, so the chain verifies over what was actually stored.

**Learned:** The first CI run found the model API key "leaking" to the model provider. It was in the `Authorization` header, which is exactly where it should go. The scan was wrong, not the code: a prompt is the request body. That's the scanner doing its job, and it sharpened the manual's definition of where prompts are scanned.

**Laws:** LAW-07 now covers audit rows, with a red-first law test and a mutation. All laws green.

**Next:** S4.4b, backups of the audit chain and approvals (3-2-1, WORM) and the monthly restore drill (SB-412).

## 2026-11-19 S4.3c draft-reply (SB-409)

**Wave/slice:** W4 / S4.3c · **Author:** platform lead + Claude · **Status:** Done

**What:** The draft-reply agent, its API and outcome metric, a draft panel in the web UI, a `draft` prompt, and a base permit for an enabled `draft-reply`. The manual changed first: sb-actions(3) now says exactly how a draft is made, and adds the outcome endpoint.

**Why this way:** A draft reuses the ask pipeline (live check, composer, trace) with one addition: the ticket or thread being answered is live-checked as an item in its own right before its text is used. Drafting on a ticket you can't open would otherwise leak it through the draft. The agent has no tools at all, so "a human always sends" is structural, not a rule someone has to remember. The base permit needs an `enabled` attribute, optional in the schema so older entities still evaluate cleanly, and tenant policies can then narrow by group or by ticket project, as the example policy already does for `FIN`.

**Learned:** Making `enabled` a required attribute would have broken every older fixture with evaluation errors, which cedar turns into Unknown. That would be safe, but every existing test would change meaning. Optional plus `has` keeps old entities at Deny and new ones explicit.

**Laws:** all green. LAW-14 holds through the same composer, and drafts make no tool calls (LAW-04).

**Next:** S4.4, secret canaries and the scanner (SB-411), and audit and approval backups with a restore drill (SB-412).

## 2026-11-19 S4.3b The ticket-update workflow (SB-410)

**Wave/slice:** W4 / S4.3b · **Author:** platform lead + Claude · **Status:** In review (real Linear sandbox)

**What:** The workflow template and registry, Linear writes with `Find`, the eval gate in apply, `--record` in the eval harness, and an end-to-end test from proposal to Linear receipt. The manual changed first: sb-approvals(7) gained "The ticket-update workflow", sb-tenant(1) "Workflows wait for their evals", sb-connector(3) Linear's write scope and `Act` row, and sb-eval(1) `--record`.

**Why this way:** The comment's id comes from the idempotency key, so Linear itself enforces "once per approval". A retry, a lost answer or the reconciler all land on the same comment. The connector never returns success without a receipt: an answer with no id is read back by that id, or it is an error. That keeps LAW-05 true at the connector, not just in the bus. The eval gate refuses the whole plan before any step runs, so an apply never half-enables a workflow. The template's payload is strict: an extra field is refused, because an approver shown `{comment}` must not approve a hidden `assignee` change.

**Learned:** Building the end-to-end test from the example manifest and the real base policies caught nothing new, and that is the point: it proves the pieces from four slices fit, with the audit trail in the order the model says. The Linear law world first shared one fake across runs, and effects leaked from one seed into the next. Connector-level law worlds need a fresh target per run.

**Laws:** all green. LAW-04, LAW-05 and LAW-15 are extended end to end, and LAW-05 also holds at the Linear connector.

**Next:** S4.3c, the draft-reply agent (SB-409).

## 2026-11-19 S4.3a Idempotency store and reconciler (SB-408)

**Wave/slice:** W4 / S4.3a · **Author:** platform lead + Claude · **Status:** In review (Postgres stores, real `Find`)

**What:** `toolbus.Receipts` and its memory store, `unconfirmed` outcomes, and `Bus.Reconcile` with `RunReconciler`. The manual changed first: sb-approvals(7) now states the idempotency store, what "unconfirmed" means, and the reconciler's schedule and limits.

**Why this way:** The receipt store is keyed by the approval id, the same key the target sees. So the bus's own state, the receipt store and the target all agree on "this ran once", and losing one of them can't cause a second run. Only errors and timeouts are unconfirmed: a target that answered without a receipt has told us what happened. The reconciled action stays `DidNotRun`, because that state is terminal in `Approvals.tla`. The receipt rides alongside it, and the audit log gets the `Ran` row that LAW-05 needs. That keeps the model and the code in step without a new transition.

**Learned:** It was tempting to let the reconciler move the action to `Ran`. That would be a transition the model doesn't have, and proving it safe would mean changing the TLA+ spec. Recording the receipt without the state change keeps every proof as it is.

**Laws:** all green. LAW-05 also holds across reconciliation, because only a found receipt makes a `Ran` row.

**Next:** S4.3b, the `ticket-update` workflow end to end: Linear writes, the manifest's eval gate, and draft to receipt (SB-410).

## 2026-11-19 S4.2b The actions API and the approver view (SB-407)

**Wave/slice:** W4 / S4.2b · **Author:** platform lead + Claude · **Status:** In review (notification adapters, human WCAG pass)

**What:** HTTP handlers for actions over the tool bus, `Bus.Get` with access rules and a field diff, policy reasons and id-only notifications from the bus, and the web approver view. The manual changed first: sb-actions(3) gained the propose body, the read rules and fields, "The approver view", and the notification rule.

**Why this way:** The approver view sends back the hash it displayed, not one it fetches again, so the server's hash check (LAW-15) also covers "the page went stale". A test proves it by changing the payload after the page loads. The diff compares canonical values field by field, so a key-order change isn't shown as a change. Reading an action is limited to the requester, the approver group and auditors, and anyone else gets the same answer as for an unknown id, so ids can't be probed.

**Learned:** Go's JSON gives the shallower of two same-named fields precedence. That's handy for one response shape where `diff` must be `null` rather than missing, but it is easy to misread, so the handler builds that shape explicitly instead of relying on `omitempty`.

**Laws:** LAW-15 now also covers the UI. Every law test is green.

**Next:** S4.3, the reconciler for unknown outcomes (SB-408), draft-reply (SB-409) and the first gated workflow template (SB-410).

## 2026-11-19 S4.2a The tool bus, approvals and the executor (SB-405, SB-406, SB-408)

**Wave/slice:** W4 / S4.2a · **Author:** platform lead + Claude · **Status:** In review (bundle registry, reconciler, Postgres store)

**What:** `internal/core/toolbus`, `internal/core/approvals`, a memory action store, a lint that only the bus calls `Act`, and three tenant-scoped base permits. The manual changed first: sb-agents(7) gained the bus's two paths and the lint, and sb-approvals(7) gained "The payload hash, exactly" and "Approving, exactly".

**Why this way:** One package owns the whole order (kill, authorise, audit, approve, execute), so there's no second path to get wrong, and the lint keeps it that way. The hash covers the target and the workflow as well as the payload, so an approved comment can't be replayed on another ticket. The approval checks run in a fixed order, and each failure names its reason. The approval row is written before the state moves, and moves are compare-and-set, so two approvers racing can't both run it. Payload numbers are integers within ±2^53, because RFC 8785 float formatting is where implementations disagree.

**Learned:** With the bus in, `laws/expected_red.txt` is empty for the first time: every law test runs against real code. One trap in the law harness: the bus writes the `Executing` row inside `Invoke`, before it returns the action id, so the world must learn the id when it is minted, not when `Invoke` returns. A second fix along the way: `decision` rows had the action in `reason` twice.

**Laws:** LAW-04 and LAW-05 move from Tested (red) to Tested. LAW-15 moves from Stated to Tested. `make proofs` and `make crosscheck` pass with the new permits.

**Next:** S4.2b, the actions API (`POST /v1/actions`, approve, reject) and the approver view (SB-407).

## 2026-11-19 S4.1b Kill switches and the kill drill (SB-403, SB-404)

**Wave/slice:** W4 / S4.1b · **Author:** platform lead + Claude · **Status:** In review (tool-bus check points, push channel, release pipeline)

**What:** `internal/core/kill`: an audited `Control`, the per-cell `Checker` and the `Drill`. There are memory and local-control-plane stores, and `sb kill`, `sb unkill` and `sb kill list`. The manual changed first: sb-kill-switches(7) gained "How it works, exactly" and the drill's procedure, and sb-kill(1) the local control plane, the actor and the list output.

**Why this way:** A 2-second poll leaves 8 seconds of margin under the 10-second bound even if every push is lost, and push only makes it faster. The checker keeps its last good list when a reload fails, because dropping known switches on an error would be the wrong way to fail. After 30 seconds it fails safe for consequential calls only, so reads keep working through a control-plane outage. Kill and unkill write their audit row first, so a switch never changes untraced (LAW-04 in spirit). The drill goes through the same `Control`, so drills are audited like real kills.

**Learned:** The first version of the timed law test was wrong: it expected a kill set during an outage to take effect within 10 seconds, which no cell can do if it can't see the store. The manual's rule is the fail-safe. The law now counts a kill from when the cell could first see it, and checks the 30-second fail-safe separately. Putting the law in front of the manual sharpened both. The smoke test that expected `sb kill` to be `ENOTIMPL` now expects `EUSAGE` without `--reason`, plus a new end-to-end kill, list and unkill test. Nothing was deleted.

**Laws:** LAW-10 is Tested. LAW-04 and LAW-05 are still red until the tool bus and the executor.

**Next:** S4.2, the tool bus (SB-405, which turns LAW-04 green), approvals bound to content (SB-406, LAW-15) and the approver view (SB-407).

## 2026-11-19 S4.1a The audit chain (SB-401, SB-402)

**Wave/slice:** W4 / S4.1a · **Author:** platform lead + Claude · **Status:** In review (tool-bus hook, cell runtime)

**What:** `internal/core/audit` has rows, canonical form, chain hashing, signed anchors, export and an offline verifier. There is an append-only Postgres store and a memory store, and `sb audit verify --export`. The manual changed first: sb-audit(5) gained "Canonical form and hash, exactly", "Storage" and "Anchors", and the export now starts with an anchor. sb-audit(1) documents the offline verify and `--key`.

**Why this way:** Only the store holds the write lock, and it hands the last seq and hash to the core's `build`. So the hashing rules live in one place, and Postgres can serialise with an advisory lock without knowing them. Rows are append-only in two independent ways: triggers refuse UPDATE, DELETE and TRUNCATE whatever the role, and the writer role holds only INSERT and SELECT. The anchor key is the cell's own: its seed is a data key under the cell's root key, never a factory-wide key (ADR-0015). Verify pins a public key, because a file that carries its own key proves only that someone signed it.

**Learned:** A tamper the law test found by design: an export that starts mid-chain can't tell a dropped first row from a later start. The fix is a signed start anchor on the row before the first exported row. RFC 8785 and Go's JSON disagree on HTML escaping and on `U+2028`, so the canonical encoder turns off HTML escaping, and rows refuse those characters outright. Shell heredocs mangled `\n` in Go string literals twice, so Go code with escapes now goes through the Edit tool only.

**Laws:** LAW-11 moves from Stated to Tested. LAW-04 and LAW-05 are still red until the tool bus (SB-405) and the executor (SB-408).

**Next:** S4.1b, the kill switches (SB-403) and the kill drill (SB-404), with a LAW-10 timed test.

## 2026-11-19 S3.8 Pilot tooling (SB-311), W3 built

**Wave/slice:** W3 / S3.8 · **Author:** platform lead + Claude · **Status:** In review (the pilot itself is human work)

**What:** The pilot review runbook in sb-onboarding(7), a Gate 1 report from the weekly feedback export, and a retro template. This ends W3. Done: SB-301, SB-302, SB-304 and SB-305. In review: SB-303, SB-306 to SB-311, each for a named human or staging item.

**Why this way:** The pilot's numbers come from the export, and the export holds ids and counts only, so the report can be shared with leadership without leaking content. Every wrong-citation report becomes a tenant eval case, so pilot findings turn into regressions that stay tested forever.

**Learned:** "No feedback" has to fail the gate. Dividing by zero would otherwise read as a pass at 0%, or as a crash.

**Laws:** W3 ends with LAW-01, 02, 03, 07 and 14 green on the real pipeline. LAW-04 and LAW-05 (the tool bus) are red until W4.

**Next:** W4, the tool bus (SB-405) and receipts (SB-408) first, to turn LAW-04 and LAW-05 green.

## 2026-11-19 S3.7 Feedback (SB-310)

**Wave/slice:** W3 / S3.7 · **Author:** platform lead + Claude · **Status:** In review (weekly schedule with the cell runtime)

**What:** `POST /v1/feedback`, the feedback store, a weekly export and the web UI controls. The manual changed first: sb-ask(3) now gives the endpoint's rules and the export's contents.

**Why this way:** Feedback goes through `Agent.Trace`, so the trace's ownership check is the same code that guards `GET /v1/traces`, with no second copy to drift. Only the asker rates: an auditor reading a trace isn't the person the answer was for. The export carries ids, counts and the user's own comment, never the question or the answer (AGENTS.md: no content in exports, logs or golden files). A reviewer who needs the text opens the trace with auditor rights, and that access is audited.

**Learned:** It is tempting to offer every citation in the wrong-citation picker. Limiting it to cited claims keeps the UI consistent with the rule that uncited claims have no citations (LAW-14).

**Laws:** none new. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-311, the pilot. It is human-only, so it gets a runbook, and then W4.

## 2026-11-19 S3.6 Citation-faithfulness eval (SB-309)

**Wave/slice:** W3 / S3.6 · **Author:** platform lead + Claude · **Status:** In review (`sb tenant verify`, staging model)

**What:** A strict lexical support scorer. `tools/evalcell`, a local cell that runs the real ask path over a fixture world. The shared `citation` suite, and per-tenant golden questions in `tenants/<name>/evals/`. All of it runs in `make evals`, including a self-test that a misattributing model fails. The manual changed first: sb-eval(5) gained Faithfulness, Worlds for the local cell and Tenant golden questions, sb-eval(1) the new models and `--tenant-dir`, and sb-evals(7) Faithfulness.

**Why this way:** The scorer is lexical, and it zeroes any claim with a number the source doesn't say. It can't be fooled into a high score, which matters more for a gate than catching every good paraphrase. A model judge can replace it once one is pinned in a release. The eval cell is a separate tool, not a CLI command, so the product CLI stays as specified. It reuses every production package and swaps only the model for a deterministic one. A tenant's gate may only be stricter than the shared gate.

**Learned:** Building the local cell found BUG-0007: the agent never gave connectors the asker's email, so real connectors would have refused everything. Our fakes keyed on ids, real connectors key on emails, and no test put the two together. The eval cell now does, on every `make check`. Also, chunk spans index into the normalised "title, blank line, text", not the text alone. That's easy to get wrong when quoting a span back.

**Laws:** LAW-14 is now measured as well as enforced. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-310, feedback capture (`POST /v1/feedback`).

## 2026-11-19 S3.5 Slack `/ask` (SB-308)

**Wave/slice:** W3 / S3.5 · **Author:** platform lead + Claude · **Status:** In review (real Slack workspace)

**What:** The Slack app as a second entry point to the same ask agent: a signed slash command and a Share button. The manual changed first: sb-ask(3) gained "The chat app (Slack)", sb-identity(7) gained "Chat users", and sb-connector(3) notes the one added scope, `commands`.

**Why this way:** Replies go through Slack's `response_url`, so the bot needs no Slack write scope at all, and every reply is ephemeral. The identity mapping needs exactly one active directory user for the Slack email, checked on every request. Two matches, an inactive user or no match means no answer, never a guess. Share posts citation links only, so a reader who can't open a source learns nothing from the share. That keeps LAW-01 true for everyone in the channel, not just the asker. `response_url` must be on `hooks.slack.com`, so a forged request can't make the cell POST anywhere else.

**Learned:** The law-test harness could reuse the ask agent by pulling its construction into `askSystem.agent`, so the chat law runs the same cell as the web law. The red-first step worked as intended here: a pending `NewChatAsk`, red for the right reason, then the real app. The mutation that answers as the tenant's first user is caught because the live ACL is asked about the wrong person.

**Laws:** LAW-01 extended to the chat path. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-309, the citation-faithfulness eval.

## 2026-11-19 S3.4 Web chat UI (SB-307)

**Wave/slice:** W3 / S3.4 · **Author:** platform lead + Claude · **Status:** In review (human WCAG audit, connector links)

**What:** A small React, Vite and TypeScript app in `web/`, served by the cell from the release's built assets with a self-only CSP. It asks, shows cited claims with chips and uncited claims with an "Uncited" label, and shows notices, refused counts and errors with their trace. The manual changed first: sb-ask(3) gained "The web UI", which covers what each element shows, the CSP, and the LAW-14 rendering rule.

**Why this way:** The UI decides "cited" defensively. A claim with `uncited` set, or with no citations, shows as uncited even if the server sent something inconsistent, so the server's label always wins and nothing renders as plain text. A missing `url` falls back to the trace, so every chip leads somewhere honest. Accessibility is tested with axe-core in jsdom. That can't measure colour contrast, so the palette was chosen for 4.5:1 and a human audit stays on the ticket. Five runtime-free dev tools and React itself, all pinned exactly with a lockfile, keep the supply chain small (ADR-0005).

**Learned:** My first LAW-14 rendering property passed against a mutation that ignored `uncited`: it only checked "chips xor label", and the mutated UI still showed chips. The property now also says the server's uncited label must show. Same lesson as S3.1: every property needs a mutation to prove it can fail. Go's mux cleans `..` paths with a redirect before any handler runs, so the static handler never sees one.

**Laws:** LAW-14 extended to the UI. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-308, the Slack app as an entry point to `POST /v1/ask`.

## 2026-11-19 S3.3 The ask agent and `POST /v1/ask` (SB-306)

**Wave/slice:** W3 / S3.3 · **Author:** platform lead + Claude · **Status:** In review (audit-chain record waits for SB-401)

**What:** `internal/core/ask` runs one question end to end: validate, retrieve and check live, compose through the gateway, keep a trace. `internal/api` serves it as `POST /v1/ask` and `GET /v1/traces/{id}`. The manual changed first: sb-ask(3) gained "How the ask agent runs, exactly", the conversation rule, the trace reasons, the `conversation` field and `EINVAL`.

**Why this way:** When no chunk is allowed, the model isn't called at all. That's cheaper, and it means there's nothing for the model to make up. A conversation sends only the user's earlier questions, never earlier answers, so a turn can't carry content the user has lost access to since; every turn is checked live again. A missing kill-switch reader counts as killed, and a trace that can't be stored means no answer: missing infrastructure fails closed. The request and response types are checked against the man page's tables and example by a test, rather than generated; the man page stays the source and CI catches drift.

**Learned:** The laws' ask system now goes through the real agent, so LAW-01 to LAW-03 test the whole path, model included. The stand-in model cites everything it sees and tries a label that was never there. The end-to-end LAW-14 test is an extension of a law that was already Tested, so it didn't go through the red ratchet; its mutation (citing refused candidates) is caught.

**Laws:** LAW-01 and LAW-14 extended end to end. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-307, the web UI (React, Vite, TypeScript), served by the cell.

## 2026-11-19 S3.2 Answer composer and model gateway (SB-304, SB-305)

**Wave/slice:** W3 / S3.2 · **Author:** platform lead + Claude · **Status:** Done

**What:** The composer (`internal/core/answer`) turns the model's JSON claims into cited claims, deciding every citation in code. The gateway (`internal/core/gateway`) sits in front of every model adapter: it refuses secrets, pins the release's model, retries rate limits within the deadline and meters tokens. There are four standard-library adapters (OpenAI-compatible, Anthropic, Groq, OpenRouter) with one contract test against a recorded fake, and an embedded prompt set with SHA-256s. The manual changed first: sb-citations(7) "How the composer decides" and sb-agents(7) "MODEL GATEWAY".

**Why this way:** The model only ever names labels (C1, C2, ...), never citations, so it can't cite anything outside the allowed context, even by accident. The secret check reuses the log redactor's registry: every resolved secret is already there, so nothing new has to be kept in sync. Groq and OpenRouter are thin wrappers over the OpenAI-compatible client; Anthropic gets its own body shape.

**Learned:** The secret scan has to check the joined request too, since a secret split between the system prompt and a message is one string by the time it reaches the provider. A shell heredoc mangled a backslash in a Go string literal, so Go source with escapes now goes through the Write or Edit tool.

**Laws:** LAW-14 moves from Stated to Tested. LAW-07 now also covers prompts. Only LAW-04 and LAW-05 (the tool bus, W4) stay red.

**Next:** S3.3, the ask agent and `POST /v1/ask` (SB-306).

## 2026-11-19 S3.1 Retrieval and live authorisation (SB-301 to SB-303)

**Wave/slice:** W3 / S3.1 · **Author:** platform lead + Claude · **Status:** SB-301 and SB-302 Done, SB-303 In review

**What:** Hybrid retrieval over a tenant-bound pgvector index, with the prefilter inside the query, followed by one batched live ACL call per connector and a policy `read` check, then a 12-chunk budget. The laws harness's `NewSystem` now runs this real pipeline. The manual changed first: sb-permissions(7) gained "Retrieval and the live check, exactly", and sb-ask(3) links to it.

**Why this way:** Three independent barriers: the index can only serve its own tenant, the prefilter narrows inside the SQL, and the live check plus kernel decide. The laws are written against the outcome, so each barrier can fail without a leak. The pgvector adapter ships the vector as text cast to `vector`, which avoids a pgvector Go module.

**Learned:** My first "real pipeline" mutation (a cell indexing other tenants' documents) wasn't caught, and for a good reason: the prefilter alone stopped it. To prove LAW-02 itself would see a leak, the mutation also has to break the prefilter. That's the honest version of "defence in depth": each layer is shown to matter by removing the others. A sibling-adapter import in a test tripped the dependency rule, so the pgvector test now has its own tiny embedder.

**Laws:** LAW-01, LAW-02 and LAW-03 move from Tested (red) to Tested. Only the tool-call tests (LAW-04, LAW-05) are still red, until W4.

**Next:** S3.2, the answer composer (SB-304) and the model gateway (SB-305, ADR-0019).

## 2026-11-19 SB-211 Linear connector (W2 built)

**Wave/slice:** W2 / S2.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** A read-only Linear connector over GraphQL, with team-based ACL and polled changes. The manual changed first: sb-connector(3) gained a `tickets: linear` section. This completes Wave 2's build. Every W2 ticket is Done or In review, and each one's Progress says what a human, a staging cell or a later ticket must add.

**Why this way:** Linear's API can filter by `updatedAt`, so polling gives a complete change feed without a public endpoint. Polling team and membership changes turns permission changes into permission-only changes for that team's issues, which is the same shape the ingest pipeline already handles for Drive and Slack.

**Learned:** The contract kit caught a real bug on the first run: a poll timestamp moved during pagination. My first fix, a one-second overlap, failed the kit's "nothing after the latest cursor" case. That was right, because overlap pushes the duplicate problem onto every consumer. An exact tie-break on (timestamp, ids seen) is the correct cursor.

**Laws:** connector-level LAW-03 (with LAW-01's truth) now runs on four connectors: mock, Drive, Slack and Linear.

**Next:** W3 at SB-301 (tenant-scoped hybrid retrieval with the ACL prefilter, on pgvector).

## 2026-11-19 SB-204 Slack connector

**Wave/slice:** W2 / S2.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** A read-only Slack connector with live channel-membership ACL, an Events API receiver with signature checks, and an inbox-backed change feed. The manual changed first: sb-connector(3) gained a `chat: slack` section, and sb-tenant(5) gained the model secret reference for ADR-0019.

**Why this way:** One document per message makes edits and deletes exact: an edit raises that message's version, and a delete removes it. Per-thread documents would make a deleted reply look like an *older* version. Events are verified before they're stored, so the inbox only ever holds Slack's word.

**Learned:** Law worlds can be wrong too. The Slack world's fault injection didn't match what "the source can't answer" means for a batch spanning channels, and the law caught it. Each time, the right fix was to the world, not the connector, and the devlog says so.

**Laws:** connector-level LAW-03 (with LAW-01's truth check) also runs on Slack.

**Next:** SB-211, the Linear connector.

## 2026-11-19 SB-210 Rebuild drill

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** `ingest.Drill`, a weekly schedule constant, and a law test running the drill. The manual changed first: sb-ingest(7) Rebuild describes the drill.

**Why this way:** The drill compares the set LAW-13 talks about, not row counts, and reports differences by id so a failure can be investigated without exposing content.

**Learned:** I had been gating commits on `tail`'s exit code, not on `make check`'s. The command now exits with make's own status.

**Laws:** LAW-13 stays Tested, with the production drill now under test.

**Next:** SB-204 (Slack) and SB-211 (Linear), the last W2 tickets.

## 2026-11-19 SB-209 Deletes and permission changes reach the prefilter

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** A LAW-01 law test for the ingest gap, plus manual wording. The behaviour itself came from SB-206.

**Why this way:** A separate tombstone table would be a second source of truth about deletions. The source's change feed already is one, and replaying it during a rebuild reproduces the deletes.

**Learned:** The law went green on its first non-pending run, because SB-206 had already handled permission-only changes. The self-test proves the law would notice if that handling went away.

**Laws:** LAW-01 stays Tested (red); the ingest-gap level is now green.

**Next:** SB-210, the rebuild drill.

## 2026-11-19 SB-208 Durable cursor, scheduler and freshness

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** A cursor store port and file adapter, a syncer that advances the cursor only after a successful sync, a 60-second scheduler with jitter, and freshness metrics through a small Prometheus text registry in `internal/obs`. The manual changed first: sb-ingest(7) gained a Sync section.

**Why this way:** "Save the cursor last" plus idempotent ingest means a crash anywhere repeats work rather than losing it, with no transaction across the source, the index and the cursor. Freshness is the age of the last success, not a queue depth, because it is the one number that says whether the 15-minute promise holds.

**Learned:** The metrics registry is small enough to own (about 100 lines), which keeps OpenTelemetry out until there's a reason for it.

**Laws:** none.

**Next:** SB-209 (deletes and ACL changes within the freshness SLA) and SB-210 (rebuild drill), then the Slack and Linear connectors.

## 2026-11-19 SB-207 Sanitiser and injection flags

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** Ingest now strips hidden text and flags text addressed to an AI, per chunk, and `Wrap` puts chunks in data blocks they can't close. The manual changed first: sb-ingest(7) gained the rule table.

**Why this way:** The sanitiser doesn't try to make hostile text safe. It records that text is suspicious and keeps it out of the instruction channel. Flagged chunks are still indexed and citable, because hiding them would make the assistant silently wrong about what a document says. Invisible characters are matched by Unicode category, not a hand list, so new zero-width tricks are covered.

**Learned:** While writing the law, the Write tool turned my `\u200b` escapes into real zero-width characters inside the Go source, which is exactly the attack this ticket defends against. I caught it by reading the file back. The repo now has a regression test that fails on any invisible or control character in source (mutation-checked with a planted file), and code spells such characters as code points. The law itself needed one correction: zero-width obfuscation reveals text when stripped, so the right check is "flagged", not "removed".

**Laws:** LAW-09 moves from Stated to Tested (ingest level). The agent-level check waits for SB-405/SB-501.

**Next:** SB-208 (durable cursor and freshness), then the Slack and Linear connectors.

## 2026-11-19 SB-203 Google Drive connector

**Wave/slice:** W2 / S2.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The user chose Google Drive as the first docs vendor. The connector is standard library only (OAuth JWT bearer and the Drive v3 REST calls), read-only, and passes the contract kit against `drivetest`, a recorded fake. The manual changed first: sb-connector(3) gained a CONNECTORS section, and 03-connectors names the vendor.

**Why this way:** Drive has no "can user X see file Y" endpoint, so the live check asks Drive *as* the user, through domain-wide delegation. A 404 is ambiguous (no access, or gone), so it's a Deny only when the service account still sees the file; otherwise it's Unknown. A Google SDK would have meant several new modules under ADR-0017. Two endpoints and a JWT are small enough to own.

**Learned:** The kit wants permission-only changes flagged, but Drive's change feed doesn't say what changed. Comparing the change time with the file's `modifiedTime` gives a good enough signal, and a wrong guess only costs a re-ingest. Building the fake surfaced a race of my own: a shared fault queue between user and service-account calls. Faults now have separate queues. These law tests had no red phase, because I wrote the connector first; that's noted on the ticket.

**Laws:** connector-level LAW-03 (with LAW-01's truth check) and LAW-13 now also run on Drive. No status change.

**Next:** a real Workspace sandbox run when the user can provide a service account. Then SB-204 (chat, vendor needed), SB-207 or SB-208.

## 2026-11-19 SB-206 Ingest pipeline

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** Done

**What:** Source changes now become provenance-tagged, embedded chunks in an index. The pure parts (normalise, chunk, cite, chunk id) are in `internal/core/ingest`, and Fetch, Embed and the index write go through ports. The manual changed first: sb-ingest(7) gained a Chunks section.

**Why this way:** LAW-13 needs the index to be a function of the sources. So everything that ends up in the index is derived: chunk boundaries from the text, ids from tenant, cite and span, and the prefilter sorted. There's no clock or randomness in a chunk. A rebuild is simply "drop, then replay the change feed from the start", which uses the same code path as normal sync. The embedder follows the cloud decision's pattern: a port with a deterministic local adapter until the provider is chosen.

**Learned:** Sync must collapse multiple changes per document (the last one wins) and carry a "permission changed" flag through, so a permission-only change re-ingests the same version. Otherwise the version check would skip it, and the prefilter would go stale until the next content edit.

**Laws:** LAW-13 moves from Stated to Tested.

**Next:** SB-207 (sanitiser and injection flags), then SB-208 (durable cursor and freshness).

## 2026-11-19 SB-205 Secrets by reference, and a logger that can't leak them

**Wave/slice:** W2 / S2.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** Validated vault references, a resolver that registers every value with the log redactor before returning it, a local vault, and a slog-based logger that scrubs those values everywhere. The manual changed first: sb-tenant(5) gained a Secrets section.

**Why this way:** The usual secret leak isn't a secret field; it's a URL, header or error message that happens to contain the secret. So redaction works by value (what was resolved), on the rendered text of every attribute, as well as by key name. The resolver refuses to hand out a secret if there's no redactor to tell, which makes forgetting to wire redaction an error instead of a leak.

**Learned:** The law test needs a "you must have logged something" clause. Otherwise a logger that drops every line passes trivially, and the self-test proves it.

**Laws:** LAW-07 moves from Stated to Tested.

**Next:** SB-203 once the user picks the docs vendor; otherwise the next W2 tickets that don't depend on it.

## 2026-11-19 SB-202 Scriptable mock connector

**Wave/slice:** W2 / S2.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** The SB-201 mock gained a fault queue and forced verdicts. A fault-matrix test shows each behaviour on demand, and the connector-level LAW-03 test now mixes a scripted Unknown into otherwise truthful batches. The manual changed first: sb-connector(3) gained a MOCK CONNECTOR section.

**Why this way:** Laws and evals need to name the bad thing that happens ("the second call times out"), not hope a random fault lands. A queue is deterministic and reproducible with `SB_SEED`.

**Learned:** A no-receipt target is ambiguous in real life (did it apply?). The mock models the safe reading: no id, no effect, and an error. The real "applied but unconfirmed" case belongs to the tool bus (SB-408), which must report `DidNotRun` and reconcile.

**Laws:** no status change.

**Next:** SB-205 (credentials via vault references), then SB-203 once the docs vendor is chosen.

## 2026-11-19 SB-201 Connector contract, kit and mock

**Wave/slice:** W2 / S2.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** The connector contract gained a version and negotiation. The contract kit turns 03-connectors' case table into `connectorkit.Run`. A controllable mock source and connector passes it in read-only and writable modes. The manual changed first: sb-connector(3) gained version negotiation and how to run the kit, and 03-connectors gained two kit cases.

**Why this way:** A kit is only worth something if a connector can't pass it while being wrong. So the kit drives faults through the sandbox, not through flags on the connector. It checks idempotency by counting effects at the target, not by trusting the receipt. Version negotiation refuses rather than degrades, because a degraded connector is an unknown permission model.

**Learned:** The ratchet earned its keep again. My first LAW-05 connector test swallowed the "not implemented" error and passed against nothing, and the ratchet reported it as green before any code existed. Law tests must surface `ErrPending`, not treat it as an ordinary error.

**Laws:** LAW-03 and LAW-05 stay Tested (red), with the connector level now green.

**Next:** SB-202, the fault-injecting mock for laws and evals, then SB-203, which needs the user to pick the docs vendor.

## 2026-11-19 First push and first CI run

**Wave/slice:** W0/W1 · **Author:** platform lead + Claude · **Status:** done

**What:** The user created the git repository and its remote. The first commit went to `main`. CI's first run failed in `check` and `proofs` with "permission denied", because the scripts `make` runs directly had been committed from Windows without the executable bit. `c0a12be` set the bit, and the second run was green in every job. A `.gitattributes` keeps LF line endings, so WSL never sees CRLF scripts.

**Learned:** A repository created on Windows needs its executable bits set in git explicitly (`git update-index --chmod=+x`). Local WSL runs on `/mnt/c` can't show the problem, because drvfs reports every file as executable.

## 2026-11-19 SB-114 Factory seams (design note) and W1 status

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The last W1 ticket. It adds the custody constants (one file), a custody lint, read-only `sb factory status`, and the first LAW-18 law test. The manual changed first: sb-factory(7) now covers status output and the lint.

**Design note, every hand-off seam and its test:**

| Seam | Where | Test |
|---|---|---|
| Custody record (`custody`, `placement`, `key_custodian`) | registry (`cells` table, `localfs`), `ports.CellRecord` | `registrytest.Contract` (Postgres and local), `TestTenantApply` smoke |
| Custody values in one place | `internal/core/cell/custody.go` | `tools/check/custody.py` with self-test |
| No hard-coded account | `Placement` port; local adapter stamps into `factory-account` | `celltest.Contract` (two mock accounts), custody lint |
| Envelope key hierarchy | `Keys` port, `cell.NewDataKey` | `celltest.Contract` envelope test, `TestLAW18_RootKeyAlwaysHasACustodian` |
| Custody isn't a manifest field | `manifest` forbidden names | golden `invalid.expected`, `TestEveryErrorMessage` |
| Cell can be re-pointed | `cell.LoadConfig` (`SB_CONTROL_PLANE`, `SB_IDENTITY_CERT`) | `TestConfig` |
| Read-only view | `sb factory status` | smoke |

**Why this way:** Hand-off (W7) only works if no code in W1 to W6 assumes the factory holds a cell. With each assumption behind a seam that has a test, W7 becomes a matter of new adapters and a workflow, not a refactor.

**Learned:** This law test had no red phase, because the code it tests landed in earlier tickets. The self-test's mutations stand in for it. That's worth stating rather than hiding.

**Laws:** LAW-18 moves from Modelled to Tested.

**W1 status:** all 13 tickets are In review, `make check`, `make integration` and `make proofs` are green in WSL, and laws stand at 32 tests with 5 expected red (retrieval and tool-call levels, W3 and W4). What's left for every W1 ticket: the user's review and the first CI run (the folder isn't a git repository yet). Real-cloud items (30-minute cell, staging timing, IaC module) wait for a cloud adapter (ADR-0018).

## 2026-11-19 SB-112 Policy analysis in CI

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** One script, `tools/check/policy.py`, runs schema validation, the tenant-condition rule and the tenant-policy narrowing rule on every `make check`. It also produces a permission diff for pull requests. The manual changed first: sb-policy(5) now says how CI enforces its rules.

**Why this way:** A permission diff computed by evaluating a fixed matrix shows what reviewers care about ("IT leads gained approve"), not a text diff of Cedar. The static tenant check complements SB-110's dynamic law test: the static check catches the rule being broken in the source, and the dynamic test catches it being broken in effect.

**Learned:** After BUG-0005, every new checker in this repo has a self-test that runs before it scans. That cost three lines here and would have caught that bug immediately.

**Laws:** LAW-02 is guarded statically and dynamically at the policy level. Status is unchanged (Tested, red, until SB-301).

**Next:** SB-114, factory seams (custody record, placement, key hierarchy), the last W1 ticket.

## 2026-11-19 SB-111 Policy decisions are audited

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** The kernel became a value, `policy.Kernel` (engine plus recorder), and the free `Authorise` function is gone. Every decision that must be audited is written before it is returned, and a failed write turns it into `Unknown`. Read allows are sampled. The manual changed first: sb-audit(7) gained "Policy decisions".

**Why this way:** As long as the free function existed, code could ask without being audited. Removing it makes "every decision is audited" true by construction rather than by review. Treating a failed audit write as `Unknown` applies LAW-04's audit-before-act at the decision itself: if we can't prove we decided, we didn't. Only read allows are sampled, because they're the high-volume, low-risk path, and refusals always matter.

**Learned:** The existing `LAW-04 SB-405` ratchet line was law-wide, so a green decision-level test would have been reported as "LAW-04 is green". The per-test lines from SB-103 made the change one line.

**Laws:** LAW-04 stays Tested (red), with the decision level now green. The tool-call level stays red until SB-405.

**Next:** SB-112, policy analysis in CI: schema validation, the tenant-condition check, and permission diffs.

## 2026-11-19 SB-110 Cedar schema and base policies

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** The schema now covers documents and gives every entity a tenant. Base policies scope every permit. A new law test evaluates the real platform and example tenant policies with the kernel's tenant guard turned off. The manual changed first: the sb-policy(5) entity table and actions.

**Why this way:** The kernel guard (SB-109) and the policies are two independent layers. Testing the policies with the guard off shows each layer holds alone, so removing either one can't silently open a cross-tenant path.

**Learned:** The test was red on its first run for a real reason. The shipped `use_agent ask` permit had no tenant condition (BUG-0006), and the rule against that existed only in prose. That's the second guardrail this session that existed only on paper (after BUG-0005). The pattern holds: write the test that would fail, run it, and only then believe the rule.

**Laws:** LAW-02 stays Tested (red), with the policy level now green and a self-test for a tenant-less permit. Only retrieval (SB-301) is still red.

**Next:** SB-111, policy decisions are audited.

## 2026-11-19 SB-109 Policy kernel: three answers, Unknown refuses

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** `policy.Authorise` is the single entry point and `policy.Require` the caller helper. The Cedar adapter runs on cedar-go v1.8.0, now vendored. The kernel owns everything that can go wrong around the engine: deadline, panic, error, invalid value, missing scope and another tenant's resource. Each of these is `Unknown` or `Deny`, never `Allow`. A new lint, `decisions.py`, plus `exhaustive`, stops callers from writing "not denied" or hiding Unknown behind `default:`. The manual changed first: sb-permissions(7) gained "The policy kernel".

**Why this way:** Cedar itself is default-deny, but LAW-03 is about everything around Cedar: timeouts, crashes and erroring policies, which Cedar skips. Putting those rules in the kernel, not the adapter, means a future engine can't weaken them. The principal comes only from the Scope, so a caller can't authorise as someone else.

**Learned:** cedar-go reports a policy that errors during evaluation in `Diagnostic.Errors` and otherwise treats it as not applying. That can turn a broken `forbid` into an Allow. The adapter maps any evaluation error to `Unknown`, and a test pins it. Mindful of BUG-0005, the new lint runs a self-test before it scans, so it can't silently stop matching. Vendoring cedar-go pulled in 6 MB of its test-corpus archives, because `go mod vendor` copies whole package directories. That's accepted for now and noted for the SBOM work (SB-504).

**Laws:** LAW-03 stays Tested (red), with the policy level now green. `TestLAW03_PolicyUnknownRefuses` allows only when a clean engine said Allow for an own-tenant resource with a valid scope. A caller using `!= Deny` is caught.

**Next:** SB-110, the Cedar schema and base policies validated in CI with cedar-go.

## 2026-11-19 SB-108 Tenant scope as a type

**Wave/slice:** W1 / S1.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** `tenancy.Scope` can only be made by the cell's single `Issuer`, which only the auth middleware holds. The middleware issues the scope after authentication and a directory read of groups, and puts it in the request context. The first cell-data port (`People`) takes a scope on every method and refuses the zero scope. The manual changed first: sb-tenancy(7) "Scope in code" gained the rules, and the architecture dependency table now lets ports import `core/tenancy`.

**Why this way:** Go has no "friend" access, so "only auth can make a Scope" is two things: a type with unexported fields and a single constructor on a capability (`Issuer`), plus a CI rule on who may create the capability. The zero scope is invalid rather than "all tenants", so forgetting a scope fails loudly.

**Learned:** The CI rule was useless at first. A heredoc turned its `\b` into a backspace character, so it matched nothing (BUG-0005). It was found only because I mutation-checked the rule by adding a violating file. A guardrail without a test isn't a guardrail, and the rule now has a regression test. I then scanned the repository for control characters: none are left.

**Laws:** LAW-02 stays Tested (red). It now has green tests at the cell, routing, sign-in and scope levels; only retrieval (SB-301) is red. `TestLAW02_ScopeIsTheOnlyWayIn` seeds users of three tenants into one shared store, as the worst case. It requires that only the cell's own tenant's sessions get a scope, that lookups find exactly own-tenant users, that groups are the user's own, and that a scope-less lookup is an error.

**Next:** SB-109, the policy kernel on cedar-go (ADR-0017).

## 2026-11-19 SB-107 SCIM and fast deprovisioning

**Wave/slice:** W1 / S1.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** Each cell has a SCIM 2.0 endpoint, which is the only way users and groups enter its directory. A deactivate, whether by `PATCH`, `PUT` or `DELETE`, revokes the user's sessions before the response. Groups are read at decision time. The manual changed first: sb-identity(7) gained the SCIM table and the order of deactivation.

**Why this way:** There are two independent mechanisms: immediate revocation, and re-validation every 60 seconds. Each alone meets the 5-minute bound, so losing one doesn't break LAW-12. A revocation failure fails the SCIM request, because a `200` that didn't revoke would tell the IdP the leaver is gone when they aren't. `DELETE` keeps the user, inactive, so the audit trail can still name them.

**Learned:** The law test had to check collateral damage as well. A "revoke everything" implementation passes the leaver check, so the test also requires active users to keep working. Real IdPs differ in small ways, for example `op: "Replace"` and `"value": "False"`. The endpoint accepts those, and a test pins them.

**Laws:** LAW-12 moves from Stated to Tested. `TestLAW12_DeprovisionRevokesFast` provisions 2 to 4 users through SCIM, deactivates one, then replays requests over 20 minutes of fake time. Half the runs drop the revocation fan-out. No request from the leaver may succeed at 5 minutes or later, and every other user must keep being served.

**Next:** SB-108, the tenant-scoped request context (`tenancy.Scope`) and the session-level LAW-02 test.

## 2026-11-19 SB-106 OIDC sign-in per tenant

**Wave/slice:** W1 / S1.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** Each cell signs users in with its own tenant's IdP, using the code flow with PKCE, a state and a nonce. It verifies the ID token itself, with RS256 over the issuer's key set, and checks issuer, audience, time, nonce, tenant and subject, all standard library. Sessions follow sb-identity(7): 12 hours with no extension, re-validated every 60 seconds against the directory. A user the directory doesn't know never gets a session. The manual changed first: sb-identity(7) gained the check table and session wording.

**Why this way:** JWT verification is small enough to own, and owning it lets us refuse everything except RS256 with a known `kid`. That closes the classic `alg: none` and algorithm-confusion holes by construction. The tenant claim is checked twice: by the cell's provider and by the edge, so either alone stops a cross-tenant sign-in. Sessions are per cell, so a stolen session id is worthless in another cell.

**Learned:** Refreshing the key set on every unknown `kid` would let anyone with a forged token make the cell hammer the company's IdP. Refreshes are now limited to one a minute, with a test. The cost is up to a minute's delay after a key rotation, and that's now in the manual. httptest TLS servers share one certificate, which made the two-IdP browser fixture simpler than expected.

**Laws:** LAW-02 stays Tested (red). `TestLAW02_TokensStayInTheirTenant` mints tokens with a random issuer (including an unknown IdP), signer, tenant claim and audience, and presents each at a random cell. It requires exactly the well-formed, own-tenant tokens to sign in. A cell that ignores the tenant claim is caught. The test was red (`not implemented`) before the adapter existed.

**Next:** SB-107, SCIM provisioning and fast deprovisioning, with the LAW-12 law test on a fake clock.

## 2026-11-19 SB-105 Tenant registry in Postgres, and the edge router

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The registry now holds cell records (endpoint, hosts and custody) next to applied resources and the journal. There are two adapters, Postgres (pgx v5.8.0, the first vendored module, 7.9 MB) and the local directory, and one contract test both must pass. The edge router (`internal/edge`) refuses with coded JSON errors and forwards only to the host's cell. Apply writes the cell record when it creates a cell. Cells read their control plane from required configuration. The user installed Postgres 17 in WSL, so the Postgres adapter is tested against a real database, and CI gets a `postgres` job. The manual changed first: sb-tenancy(7) "Registry and routing".

**Why this way:** Routing is data, so no code names a company. The host decides the cell and the token can only confirm it, never redirect it. That's why a router that "helpfully" routes by the token's tenant is the mutation the self-test must catch. A shared host is impossible by schema (primary key), not by a check that could be skipped. The edge sits outside `internal/core`, because it speaks HTTP and the error registry.

**Learned:** The request-level law found a hole on its first green run: a valid token with an empty tenant claim. The router said `EAUTH` and the law said `ETENANT`. `EAUTH` is right (that token isn't sign-in for any cell), and the manual now says so. A contract test pays off at once: the local adapter needed a `tenants/` directory to count as an empty registry, which is the same "unknown is not absent" rule written down for a second adapter.

**Laws:** LAW-02 stays Tested (red) overall. `TestLAW02_RequestsNeverCrossTenants` drives the real router over HTTP with 2 or 3 cells, extra hosts, unknown hosts, bad and foreign tokens. It requires exactly `EPERM`/`EAUTH`/`ETENANT`, or forwarding to the host's own cell. It went red (`not implemented`) before the router existed.

**Next:** SB-106, OIDC sign-in per tenant, which fills the token-verifier port.

## 2026-11-19 ADR-0017 First third-party modules, allow-listed

**Wave/slice:** W1 / S1.2–S1.4 · **Author:** platform lead + Claude · **Status:** Accepted

**What:** The user approved cedar-go and pgx/v5. ADR-0017 pins cedar-go v1.8.0 and pgx v5.8.0, plus six transitive modules. `tools/deps/allowed.txt` holds every `go.sum` line we accept, and `make deps` now fails on any other `go.sum` line or `go.mod` requirement. Mutation-checked: adding `require github.com/evil/mod v1.0.0` fails `make deps`, and removing it passes again. `go.mod` has no `require` yet; each module is vendored when SB-105 or SB-109 first imports it.

**Why this way:** `go.sum` proves the bytes, not the intent. With an allow-list, a new module, or a version bump that pulls in a new indirect module, is a visible line in review.

**Learned:** The newest cedar-go needs Go 1.26 and the newest pgx needs Go 1.25. `go get` silently upgraded the probe module's toolchain to 1.26 until `GOTOOLCHAIN=local` stopped it. Without that pin, CI would build with a different Go than `tools/versions.env` says.

**Laws:** none directly. This is supply-chain groundwork for LAW-06 (only signed, gated builds run).

**Next:** SB-105 (registry and router), then SB-106 to SB-109.

## 2026-11-19 SB-104 Idempotent, resumable `sb tenant apply`

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** `internal/core/apply` enacts the SB-102 plan. It stamps a new cell first, through the SB-103 template, then records each change in the registry, with a write-ahead journal entry before and after each step. The new ports are `RegistryWriter` and `Journal`. The local control plane (`file:///DIR`) implements them in `localfs`, and stamps into `DIR/accounts/factory-account`, the v1 placement from sb-factory(7). The manual changed first: sb-tenant(1) "How apply works", and sb(1) describes the local control plane layout.

**Why this way:** Resumability comes from idempotent steps, not from replaying a journal. Stamping finds what exists, and each registry write is the whole state after one change. A rerun therefore just plans what is left. The journal is for people and for audit (SB-401), not for recovery, so a lost journal can't corrupt a cell. The cell is recorded only after every resource exists, so a recorded cell is always a complete one. A no-change apply returns before taking an apply id, so it really does write nothing.

**Learned:** The crash test counts writes and fails each one in turn. It found no bug, but writing it showed that "finding" a resource has to cost nothing: a no-change re-stamp that counted as a write would have broken "touches nothing". The fakes now count only creations.

**Laws:** LAW-16 stays Tested. `TestLAW16_ApplyMakesTheCellMatch` applies two random manifests in turn on the real local control plane. It requires the recorded state to equal the second manifest's resources, and a repeat apply to change nothing. An applier that skips removals is caught.

**Next:** SB-105, the control-plane tenant registry. Ask about ADR-0017 (cedar-go, pgx) before SB-109.

## 2026-11-19 SB-103 Cell template behind Placement and Keys

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The user decided that cells are cloud-neutral, with a local adapter now and AWS, GCP and Azure later (ADR-0018). The template is Go in `internal/core/cell`: five resources, each named from the tenant and pinned to the region. `Stamp` applies it through two new small ports. `Placement` finds or creates resources in one account, and `Keys` holds the root key and wraps data keys. `celltest.Contract` is the test every adapter must pass. The `local` adapter keeps one directory per mock account and wraps keys with AES-256-GCM. The manual changed first: sb-tenancy(7) now has a Cell anatomy section.

**Why this way:** "Isolation is a property of the template" only holds if the template is code we can test. IaC text per cloud can't be unit-tested the same way, so the template lives in the core and each cloud's IaC sits behind an adapter. The ports are split by job (interface segregation), so a key store and a resource placer can come from different vendors. The core depends only on the ports (dependency inversion). Adding a cloud means adding an adapter folder that passes the contract (open for extension, closed for modification). Envelope keys now mean hand-off (LAW-18) re-wraps one root key instead of re-encrypting data.

**Learned:** The ratchet keyed expected-red by law, so a green cell-level LAW-02 test would have been reported as "LAW-02 is green" while retrieval is still red. `laws/expected_red.txt` now takes an optional test name. The cell test was listed red, failed with `not implemented`, then went green, and its line was removed. One `make check` run hit a transient missing file in `site/` while building on `/mnt/c`, and the rerun was clean. Keep an eye on it; building `site/` inside the WSL filesystem would avoid it.

**Laws:** LAW-02 stays Tested (red) overall. The new `TestLAW02_CellsShareNothing` stamps 2 or 3 tenants into one or two mock accounts, twice each. It requires unique ids, the right tenant and region on every resource, idempotent re-stamps and all five kinds. Three broken templates are caught by the self-test: one shares the database, one drops the region and one forgets the queue.

**Next:** SB-104, idempotent and resumable `sb tenant apply` on the local adapter. Ask about ADR-0017 before SB-109.

## 2026-11-19 SB-102 `sb tenant apply --plan`

**Wave/slice:** W1 / S1.1 · **Author:** platform lead + Claude · **Status:** In review

**What:** The plan is a pure function in `internal/core/plan`: the manifest becomes a sorted list of resources, which is diffed against the applied resources from a new `ports.Registry`. The human output follows sb-tenant(1) (`+`, `~` with `was`, `-`, then a summary), and `--json` is the same plan as data. Until SB-105 there is a local registry adapter, `internal/adapters/registry/localfs`, chosen with `--endpoint file:///DIR`. The manual changed first: sb-tenant(1) gained the resource table, plan format and failure modes, and its example now shows settings. sb(1) gained `file://` endpoints.

**Why this way:** The plan is the manifest diff made concrete, so it lives in core and has no I/O. SB-104's apply engine can then enact exactly what was shown. `Registry.Applied` returns `found` separately from `err`, and the adapter returns "no cell" only when it positively knows. A missing directory or a bad record is `EUNAVAIL`, never an empty state, because an empty state would plan a second cell. That's the same rule as "unknown is refusal". The region is checked against the applied cell, which is the part of sb-tenant(5) rule 1 that `check` can't do offline.

**Learned:** Go map order would have made the plan unstable. Sorting by kind order and then name in one place, plus a test that plans 50 times and compares bytes, keeps it deterministic. Settings shown as resources make the first plan longer than the old man page example, which listed 4 lines. I updated the example rather than hide the settings, because a dry run that hides changes is worse than a long one.

**Laws:** LAW-16 stays Tested, now with a plan-side law test. For random pairs of manifests, enacting the plan on the first cell yields the second, a matching cell plans nothing, and a new cell plans only additions. Self-tests catch planners that drop removals or changes.

**Next:** SB-103 (cell template). It needs the cloud and IaC choice (Q-04, CLAUDE.md decision 1) before it starts.

## 2026-11-19 SB-101 Manifest parser and `sb tenant check`

**Wave/slice:** W1 / S1.1 · **Author:** platform lead + Claude · **Status:** In review

**What:** `internal/core/manifest` adds a strict YAML-subset parser that keeps the line and column of every node, a checker for every rule in sb-tenant(5), and a JSON Schema built from the same field table and vocabulary. `sb tenant check` prints compiler-format errors and exits 1 with `EMANIFEST`. The manual's `yaml run` manifests and `examples/tenants/*.yaml` now go through the real binary in `make examples`, and `invalid.yaml` must print `invalid.expected` exactly. The manual changed first: sb-tenant(5) SYNTAX and exact rules, sb-tenant(1) output format, sb-tenancy(7) regions.

**Why this way:** The parser is our own, not a YAML library, as decided for W1 (CLAUDE.md decision 3). A general YAML parser accepts anchors, tags, merge keys and type coercion (the "Norway problem"), and most lose positions after decoding. The manifest needs few constructs, and every one of them gets a position, so the parser and checker come to about 1,000 lines of standard-library Go and stay inside ADR-0016. Errors about a field point at its key; errors about a value point at the value. The first syntax error stops the check, because positions after it can't be trusted.

**Learned:** The golden file fixed the rule order. With two errors at the same spot (a workflow missing both its approval and its connector), the order is the emit order, so the sort has to be stable. Reporting "approval names an unknown workflow" when `workflows` is itself malformed doubled the noise, so cross-field rules now apply only when the fields they relate are well-formed. Writing the property test found no checker bug, but the generator had to indent nested block lists to be valid YAML at all.

**Laws:** LAW-16 moves from Stated to Tested (manifest side). `TestLAW16_TheManifestIsTheTruth` renders 300 random valid manifests in mixed flow, block, quote and comment styles. It requires each one to parse to exactly the tenant it describes, and requires one injected forbidden or unknown field to be rejected at its key with nothing applied. Three broken checkers (one ignores unknown fields, one drops `exclude`, one invents defaults) are caught by the self-test. The drift side of LAW-16 waits for SB-510.

**Next:** SB-102 (`sb tenant apply --plan`). Then SB-103 and SB-104, which need the cloud/IaC decision (Q-04) first.

## 2026-11-19 SB-011 to SB-013 First model-checker run

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see tickets (In review)

**What:** Ran `make proofs` for the first time, locally on Windows (Temurin 25; `tla2tools.jar` from the pinned v1.8.0 release, which reports TLC2 2026.11.15; Alloy 6.2.0; both checksums match), then again in WSL Debian on OpenJDK 21 with the same results. All four models pass. TLC state counts are identical to `crosscheck.py`: Approvals 1,068, Rollout 6,290, Handoff 23. All five `Permissions.als` checks are UNSAT at scope 6, and the sanity `run` is SAT. Added `tools/ci/mutations.py`, run by `make proofs`. It applies the 13 mutations from `proofs/README.md`, and all 13 give the expected counterexample:

| # | Mutation | Counterexample |
|---|---|---|
| 1 | Approvals: no `u # requester` | `NoSelfApproval` |
| 2 | Approvals: no `payload = approvedHash` | `AuditBeforeExec` (through `Tamper`) |
| 3 | Approvals: no `~killed` | `KillWins` |
| 4 | Approvals: no `Log` in `Execute` | `AuditBeforeExec` |
| 5 | Rollout: no promote guard | `OneBehindMax` |
| 6 | Rollout: `Promote` drops `r \in gated` | `OnlyGatedRuns` |
| 7 | Rollout: testing tenant deploys a revoked release | `RevokedOnlyWhileDraining` |
| 8 | Rollout: rollback without draining | `RevokedOnlyWhileDraining` |
| 9 | Handoff: provision without both approvals | `DualApproval` |
| 10 | Handoff: destroy factory wrap before tenant wrap | `KeyNeverOrphaned` |
| 11 | Handoff: cutover keeps factory paths | `NoUnlistedAccess` |
| 12 | Handoff: cutover before tenant wrap | `ServingCanUnwrap` |
| 13 | Permissions: `context = candidates - live.Deny` | `UnknownNeverUsed`, `MissingNeverUsed`, `CitedImpliesLiveAllow` SAT |

**Why this way:** We automated the mutations instead of only recording them once. That way a later model edit that makes a guard redundant fails `make proofs`, instead of quietly weakening the evidence behind *Modelled*.

**Learned:** The first run found two bugs in `proofs.sh` and none in the models. (1) TLC rejects `proofs/tla/X.tla` on Windows ("File name ... does not match the name ... of the top level module"), so the script now runs TLC from the model's directory, with `-metadir` in a temp dir so no `states/` is left behind. (2) Alloy 6.2 `exec` runs only the default command, so the old `grep -c unsat` check could never pass. The script now runs each `check` and `run` by name and needs `UNSAT` or `SAT` for each. `-c '*'` is not an option: the Windows Java launcher expands it as a file glob.

**Laws:** LAW-06, LAW-08, LAW-10, LAW-15, LAW-17 and LAW-18 move from Stated to Modelled. LAW-01 to LAW-05 now have model evidence but stay Tested (red), because the ladder only moves up.

**Next:** Get two reviewers and one green CI `proofs` job, then mark SB-011 to SB-013 Done. W1 starts at SB-101.

## 2026-11-18 W0 Exit gate status

**Wave/slice:** W0 / all · **Author:** platform lead + Claude · **Status:** see ticket

**What:** 10 of 16 W0 tickets Done, 6 In review. `make check` is green on a clean copy. What's left needs people or CI: two-reviewer sign-off of the first man pages, the ADRs and the threat model (SB-004, SB-005, SB-014), and the first CI run of TLC and Alloy (SB-011 to SB-013).

**Why this way:** W0's exit gate asks for reviews and model-checker runs, and neither can be faked. Marking those tickets *In review* with an explicit Progress section keeps the ledger honest.

**Learned:** The order of work matters. Writing the manual change (`ENOTIMPL`) first, then the error generator, then the CLI meant that every later check had a stable vocabulary to test against.

**Laws:** LAW-01 to LAW-05 are now Tested (red). No law is Modelled yet, because none has had a TLC or Alloy run.

**Next:** Push to a repo with CI, watch the `proofs` job, then collect reviews. W1 can start on SB-101 in parallel.

## 2026-11-18 SB-012 Rollout model: non-atomic rollback

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Fixed BUG-0001 in `Rollout.tla` and the cross-check. Rollback now revokes at once and redeploys cell by cell; cells on a revoked build are *draining*. `NoRevokedRuns` became `RevokedOnlyWhileDraining` and `DrainingOnlyRevoked`. `OneBehindMax` and `StableNotAhead` now apply to serving (non-draining) cells.

**Why this way:** The atomic model was simpler but promised something rollback can't do in 10 minutes across many cells.

**Learned:** The promote guard also protects rollback: a draining stable cell isn't on the current stable, so no promotion can happen until every cell has redeployed. The cross-check grew from 2,974 to 6,290 states. The two new mutations (deploying a revoked build, rolling back without draining) are both caught.

**Laws:** LAW-06 and LAW-08 stay Stated until TLC runs.

## 2026-11-18 SB-011 Proofs wired into CI

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Pinned `tla2tools.jar` 1.8.0 and `alloy.jar` 6.2.0 by SHA-256 in `tools/versions.env`. `tools/ci/fetch_tools.sh` refuses a jar whose checksum doesn't match. `tools/ci/proofs.sh` passes a model only when the checker positively reports success. SB-013 and SB-705 use the same script.

**Why this way:** Unknown means no (P1), applied to our own tooling: output we don't recognise is a failure.

**Learned:** The jars could be downloaded and hashed here but not run, so the checksums are real and the first run will happen in CI. The Alloy CLI flags still need confirming on that run.

**Laws:** No change until the first run.

## 2026-11-18 SB-007 Manual-diff rule tested

**Wave/slice:** W0 / S0.3 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `tools/ci/test_manual_diff.sh` builds a throwaway git repo and proves four cases: code-only fails, code plus manual passes, the override label passes, docs-only passes. It runs in `make check` and in CI before the real rule.

**Why this way:** A one-off test PR proves the rule once. A scripted test proves it on every build.

**Learned:** POSIX `sh` has no `$RANDOM`. The script now uses `date +%s%N`.

**Laws:** No change.

## 2026-11-18 SB-009 The manual can't lie

**Wave/slice:** W0 / S0.3 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Added `tools/check/examples.py` (man-page section order, `run` blocks, example YAML) and moved the diagram renderer into the repo as `tools/check/diagrams.py`, with Mermaid pinned in `tools/check/package.json`. Results: 51 diagrams render, all 32 man pages have their sections in order, and 0 links are broken.

**Why this way:** Checks that report *pending by ticket*, such as manifest validation until SB-101, keep the gap visible without failing the build for work that isn't due yet.

**Learned:** Mutation-checked with a planted page whose sections were out of order and whose `run` block didn't parse. Both were caught.

**Laws:** No change.

## 2026-11-18 SB-016 Smoke and regression conventions

**Wave/slice:** W0 / S0.1 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `tests/smoke/` builds the real `sb` binary and checks the exit-status contract. `tests/regress/` has the BUG-0003 Cedar test and a README mapping every bug to its test.

**Why this way:** Smoke tests are black-box on purpose: they test the binary users run, not the package.

**Learned:** A missing validator in a regression test is a hard failure, not a skip. The BUG-0003 test exits if `cedarpy` isn't installed.

**Laws:** No change.

## 2026-11-18 SB-015 Eval harness scaffold

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `evals/`: loader, the seven checks from `sb-eval(5)`, worst-of-three runs, a mock target and a smoke suite. `sb eval run` delegates to it.

**Why this way:** Evals become the main gate in W5. Building the harness now means suites can accumulate from W2 on.

**Learned:** Two rules got sharper by being written as code: an unknown expectation fails rather than being ignored, and `faithful_min` fails when there are no cited claims at all, because silence can't demonstrate faithfulness.

**Laws:** LAW-09 and LAW-14 have their first harness; their suites arrive with SB-207 and SB-309.

## 2026-11-18 SB-010 Law harness: five laws red, checks proven

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Added `laws/`: a seeded `Check` helper, a world generator, LAW-01 to LAW-05 checks, the red ratchet (`expected_red.txt` plus `tools/check/laws.py`), and self-tests against a reference system and six broken systems.

**Why this way:** A red law that fails for the wrong reason proves nothing, and neither does a check that would never fail. The ratchet handles the first; the self-tests handle the second.

**Learned:** The first self-test run failed: the cross-tenant mutation wasn't caught, because the live permission check masked it. Users in another tenant always got Unknown. Real sources can say Allow to outsiders, for example a doc shared with "anyone with the link". The generator now creates public docs, and LAW-02 is proven independent of the live check. Logged as BUG-0004.

**Laws:** LAW-01 to LAW-05: Stated → Tested (red, ratcheted).

## 2026-11-18 SB-008 Dependency rule enforced

**Wave/slice:** W0 / S0.3 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `depguard` rules in `.golangci.yml`, plus `tools/check/deps.py` reading `go list`. A planted import of `internal/errs` from `internal/core/ports` was caught by both.

**Why this way:** Two independent checks, so a lint config change can't silently drop the rule.

**Learned:** `golangci-lint` v2 needs `version: "2"` and the new `linters.settings` layout.

**Laws:** No change.

## 2026-11-18 SB-006 Error registry generated from the manual

**Wave/slice:** W0 / S0.2 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `tools/gen/errs.py` turns the registry table in `sb-errors(7)` into `internal/errs/codes.go`, formatted with `gofmt`. `make gen-check` fails on drift. `ENOTIMPL` (501, exit 2) was added to the manual first.

**Why this way:** The manual is the spec (P2), so the code is derived from it, never the reverse.

**Learned:** An uncoded error must exit 1, not 0. That's now a unit test.

**Laws:** No change.

## 2026-11-18 SB-001 Repo skeleton and toolchain

**Wave/slice:** W0 / S0.1 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `go.mod`, `cmd/sb`, `internal/cli`, `internal/core/ports`, `internal/version`, the Makefile, `tools/versions.env` and CI. Every command in `sb(1)` exists; unbuilt ones exit `ENOTIMPL` naming their ticket. `make check` passes on a clean copy.

**Why this way:** One module rather than a `go.work` workspace (P3). Standard library only, because the Go module proxy refused the request and every dependency is supply-chain surface (ADR-0016).

**Learned:** A CLI that honestly says *not built yet (SB-101)* is better than a missing command. Scripts and agents get a stable exit code (2) and a ticket to look at.

**Laws:** No change.

## 2026-11-18 SB-002, SB-003, SB-004, SB-005, SB-014 Closing the design-baseline tickets

**Wave/slice:** W0 / S0.1 to S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** The design baseline already delivered the template, the site, the living documents, first drafts of the man pages, the threat model and the ADRs. SB-002 and SB-003 are Done. SB-004, SB-005 and SB-014 are In review, with sign-off tables added for the reviews.

**Why this way:** Reviews are people's work. Leaving them visibly open is better than ticking boxes.

**Learned:** None beyond the sign-off tables.

**Laws:** No change.

## 2026-11-18 SB-000 Factory and hand-off planned, not built

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Named the control plane, cell template and release train together as the **factory**. Added custody records (`factory`, `tenant`, `detached`), a dual-approved hand-off workflow, ADR-0015, architecture doc 16, `sb-factory(7)`, LAW-17 and LAW-18, and `Handoff.tla`. The hand-off itself is epic E08 in a new wave W7, after v1. v1 gains one ticket (SB-114) and extra acceptance criteria on five others; no v1 wave changes length.

**Why this way:** Building hand-off now would derail v1. Ignoring it would let cheap v1 shortcuts (hard-coded account ids, one factory signing key, push deploys) turn into a migration across every cell. The seams cost almost nothing now.

**Learned:**

- **Pull beats push for more than security.** Cells that pull signed bundles and push attestations work identically in the factory's account and in a company's own account. Choosing pull in v1 is what makes LAW-06 and LAW-08 survive hand-off with no new mechanism.
- **Key order is the whole game.** The cross-check's `destroyfirst` mutation orphans the root key in 16 states. Re-wrap first, then destroy, and LAW-18 forbids *zero* custodians, not two.
- **Binding approvals to a plan hash makes a forgotten "void approvals on re-plan" harmless.** The `keepapprovals` probe found no violation, because an approval of an old hash can never satisfy the new one. Same lesson as LAW-15, one level up.
- **Custody must not be a manifest field.** If it were, anyone with manifest write access could start a hand-off. So `sb tenant check` rejects it.

**Laws:** LAW-17 and LAW-18 Stated. `Handoff.tla` written, cross-checked in Python (23 states, 4 of 4 mutations caught), and not yet run in TLC (SB-705).

**Next:** SB-114 lands in W1 alongside the cell template.

## 2026-11-18 SB-000 Design logic cross-checked in Python

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Wrote `proofs/crosscheck.py`, a breadth-first search over every reachable state of hand-written Python mirrors of `Approvals.tla` and `Rollout.tla`. All invariants hold: 1,068 states for approvals and 2,974 for rollout. Each of the seven mutations in `proofs/README.md` produces a counterexample.

**Why this way:** TLC couldn't be run in the design environment. A second, independent encoding of the same rules catches design mistakes now, and gives SB-011 and SB-012 an expected result to compare against.

**Learned:**

- Removing the hash re-check from `Execute` trips `AuditBeforeExec` before `ExecutedMatchesApproved`, because the audit row carries the payload hash. So the audit log is itself evidence of LAW-15: an auditor can see that the executed hash differs from the approved one.
- The promote-guard counterexample needs only two promotions and one slow tenant: 208 states. It's a small, likely mistake, which is exactly why it's worth a law.
- This is not a proof of the TLA+ text. The ledger keeps these laws at *Stated* until TLC runs clean in CI.

**Laws:** no status change.

## 2026-11-18 SB-000 Cedar policies validated against the schema

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Wrote `policy/schema.cedarschema` and `policy/platform/base.cedar`, then validated every Cedar snippet in the repo (platform policy, example tenant policy, `sb-policy(5)`, architecture doc 02) against the schema with the Cedar validator. All pass.

**Why this way:** Policy snippets in docs are code that people copy. If they don't validate, the manual is lying (P2).

**Learned:**

- The first draft named the proposed-action resource `Action`, which collides with Cedar's built-in action type, so `resource is Action` could never match. It's now `ProposedAction`. Logged as [BUG-0003](BUGS_AND_MITIGATIONS.md#bug-0003-policy-examples-used-a-reserved-type-and-an-unguarded-optional-attribute).
- An optional context attribute (`target_project`) must be guarded with `has`; the validator refuses unguarded access. Good: that's the soundness we want from the policy language (ADR-0008).
- Mutation check: three deliberately broken policies (typo'd attribute, wrong resource type, unguarded optional) were all rejected. The validator is a real gate, so SB-112 uses it.

**Laws:** no status change. LAW-02 and LAW-15 now have platform policies backing them.

## 2026-11-18 SB-000 Formal models written (not yet run)

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Wrote `Approvals.tla`, `Rollout.tla` and `Permissions.als` alongside the man pages, using the same state names as `sb-approvals(7)` and `sb-release(7)`. They were reviewed by hand but **not model-checked**: the tool jars couldn't be run in the design environment, so SB-011 to SB-013 start by running them in CI.

**Why this way:** Writing the models while the man pages were still soft meant the models could change the design cheaply, and twice they did (below).

**Learned:**

- **LAW-08 needs a promote guard.** Writing `OneBehindMax` made it obvious that "promote when the new release is good" lets a slow tenant fall two behind. The fix is a guard: promote only when every stable tenant runs the current stable. That rule is now in `sb-release(1)` and SB-507.
- **Edits must void approvals, and so must tampering.** Modelling an approval as a flag on the action was not enough; it has to bind to the payload hash, and `Execute` has to re-check the hash. The `Tamper` action exists to prove the re-check matters (LAW-15).
- **The rollout model is more optimistic than reality.** It treats rollback as atomic across cells; in reality rollback takes up to 10 minutes. Logged as [BUG-0001](BUGS_AND_MITIGATIONS.md#bug-0001-rollout-model-assumes-atomic-rollback).

**Laws:** 16 Stated. LAW-01, 02, 03, 04, 05, 06, 08, 10 and 15 have models written; none is *Modelled* until a clean run plus a mutation check.

**Next:** SB-011, SB-012, SB-013.

## 2026-11-18 SB-000 Plan generated from one data file

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** 8 epics, 37 user stories, 26 slices and 85 tickets across 7 waves, all generated from a single plan file into `epics/**` and the wave section of the PRD. The generator refuses to run if a ticket depends on a later wave or names a man page that doesn't exist.

**Why this way:** The plan, the PRD tables and the ticket files used to drift in every project we've seen. One source makes drift impossible at generation time; after that, tickets are edited by hand as the source of truth.

**Learned:** The policy engine deserved its own epic (E02). In the preliminary PRD it was one ticket inside tenancy, but it's the kernel that every later epic depends on, and it has its own laws.

**Laws:** no change.

## 2026-11-18 SB-000 Design baseline from the preliminary PRD

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Turned the preliminary PRD (v0.1) into the full design set: problem statement, principles, domain model, 16 architecture documents, 14 ADRs, threat model, 31 man pages, examples, living documents and the docs site.

**Why this way:** Manual first (ADR-0009). Every later ticket links to a man page that already exists, so implementation follows a spec instead of inventing one.

**Learned:**

- **The preliminary PRD's eight laws weren't enough.** Kill switches, audit tamper-evidence, deprovisioning, index derivation, uncited labelling, approval binding and manifest drift were promises without laws. There are now 16 laws, each with a check.
- **`EPERM` should cover unknown as well as deny.** Telling a user "unknown" invites retrying until it flips; telling them "deny" leaks that a document exists. Operators get the breakdown instead.
- **Kill switches need a fail-safe direction.** If a cell can't reach the kill-switch store, it should behave as if killed for consequential tools. Losing the ability to be stopped is treated as being stopped.
- **Never cache an allow across requests.** Even 60 seconds breaks LAW-01's "at answer time". Denies can be cached; allows can't (ADR-0002).

**Laws:** LAW-01 to LAW-16 Stated in `sb-laws(7)`.

**Next:** W0 starts with SB-001.

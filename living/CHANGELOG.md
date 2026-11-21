---
title: Changelog
summary: User- and operator-visible changes, by release.
---

# Changelog

All notable changes, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) style. Releases are named `rel_DATE_DIGEST`.

## Unreleased

### Added

- A whole cell on Postgres (S14.4: SB-1508): `database` in sb-cell(5) keeps every store durable, `make cell` runs a cell locally with a development IdP, and `sb tenant verify --cell` runs its checks inside a running cell.

### Fixed

- SCIM user and group ids are `usr_…` and `grp_…`, so legal holds and erasure accept real users (BUG-0025).
- The CLI against a running cell (S14.4: SB-1507): `sb audit verify|tail|export --tenant T`, `sb eval diff`, and `sb factory handoff` and `detach` on the local control plane; no `sb` command answers `ENOTIMPL` any more.
- The updater and apply against a running cell (S14.3: SB-1505, SB-1506): `sb-updater` pulls the channel's release, admits it and deploys each service by digest; `sb tenant apply FILE --cell URL` makes a running cell run a new manifest at once, refusing a held space's removal; the cell reports drift hourly.
- The ingest and jobs services (S14.2: SB-1503, SB-1504): syncs that run from stored cursors with backoff and a staleness alert, every scheduled job with its finish recorded and a watchdog, `sb-api --with ingest,jobs`, `sb-jobs run JOB`; the budget guard, token metering and the weekly report wired; legal holds and reporting lines wired in every service (PT-0011).
- One cell configuration and the api service (S14.1: SB-1501, SB-1502): `sb-api --config FILE` (sb-cell(5)) builds the whole cell and serves every endpoint, with `/healthz`, `/readyz` and graceful shutdown.
- Quick fixes (S13.6): no change lost in the cursor's own second (GitHub, BambooHR, QuickBooks), opted-in repository source backfilled, QuickBooks open-month reports refreshed hourly, conversations tagged with their space for legal holds, and coverage in its own `make coverage` step and CI job.
- Coverage floor (S13.5: SB-1405): every core package at or above 75% over every suite, checked in `make check`. **W13 Hardening built.**
- The manual and the code agree (S13.4: SB-1404): a check in `make lint` that endpoints, commands and flags match the manual both ways; six drifts fixed; CLAUDE.md's state section summarised.
- Clean and concise (S13.3: SB-1403): eleven stricter linters, their findings fixed; the tool bus, the verifier and the API helpers tidied with no behaviour change.
- Fuzzing and races (S13.2: SB-1402): six fuzz targets and `make fuzz`; `make test` runs under the race detector and includes `tests/spaces` and `tests/groupsync`.
- Hardening (S13.1d: SB-1401): re-routing never wedges a sync, sensitive fields go only to the spaces that allow them, allow-list changes re-ingest; GitHub identity and paging, QuickBooks paging, Drive folder cache, KMS key rings, Cloud Run updates and shred order fixed.
- Hardening (S13.1c: SB-1401): governance decisions are atomic and audited, operators can complete access reviews, per-space retention is enforced, and review removals are never dropped.
- Hardening (S13.1b: SB-1401): approval and barrier evasions fixed (edits, negative amounts, rejection, viewing across a barrier, owners by email, barriers without spaces, chat attributes).
- Hardening (S13.1a: SB-1401): the review's two critical findings fixed. Secrets are found across chunk boundaries and in more forms; steward proposals must read back as written; manifest diffs name each line's space.
- Cross-space red team (S12.4b: SB-1208): the plan, the `space-redteam` gating eval, the game day runbook, and a new `sb tenant check` rule refusing a scope that reaches into a restricted space (PT-0003).
- Legal holds (S12.4a: SB-1207): Legal stewards hold a space, custodians or both; retention keeps held records, erasure and space removal are refused with `EHOLD`, and holds and releases are audited first.
- Reorg preview (S12.3b: SB-1206): `sb space diff FILE --idp-snapshot SNAPSHOT` lists per person the spaces a reorg would add and remove, and flags anyone losing every space, gaining a restricted one, or straddling a barrier. Offline; changes nothing.
- Directory attributes and five-minute moves (S12.3a: SB-1205): SCIM `title`, `department`, `division`, `costCenter` and `manager` reach policy at every request, and moves between groups change spaces at the next request. LAW-21 is green.
- Segregation of duties (S12.2b: SB-1204): `duties:` in the manifest asks for two approvers from different groups above a threshold, nobody approves for someone in their reporting line (SCIM `manager`), and only the named group may give each approval.
- Information barriers for everything (S12.2a: SB-1203): ownership from the source, and barrier forbids for drafts, tool calls, proposals and approvals as well as reads. LAW-22 is green.
- Quarterly access reviews (S12.1b: SB-1202): a `review` job per restricted space, certify or remove by stewards, removals as manifest proposals or IdP requests, an overdue alert at 14 days, and an audit-ready evidence export.
- Stewards and delegated space changes (S12.1a: SB-1201): `propose_space` and `approve_space` in Cedar, steward proposals checked against their own space, and hash-bound approval by an operator who isn't the proposer.
- Per-template eval suites in the gate (S11.4: SB-1107): twelve `template-*` suites over a shared world, in the release gate.
- The QuickBooks Online connector (S11.3c: SB-1105), read-only against a recorded fake, and the `accounting` kind; account and routing numbers, tax ids and payroll are never read.
- The BambooHR connector (S11.3b: SB-1104), read-only against a recorded fake, and the `hris` kind; sensitive fields are read only when a space allow-lists them.
- Field-level allow-lists for structured sources (S11.3a: SB-1106): declared fields, a space's `fields` allow-list, and ingest that drops every other field before chunking.
- The GitHub code connector (S11.2: SB-1103), read-only against a recorded fake, the `code` connector kind, and the secret scanner that drops any chunk holding a credential before it is stored.
- Department agents (S11.1b: SB-1102): `incident-summary`, `spec-digest`, `close-checklist` and `runbook-answer`, read-only, bound to their templates' spaces, every claim cited, each with an eval suite in the gate.
- The template catalogue as data (S11.1a: SB-1101): `internal/core/manifest/catalogue.json` pinned in the release bundle, `sb space init` and `sb space templates`.
- Audit and kill switch per space (S10.3c: SB-1008): the `space:NAME` kill scope (`sb kill space`), `space` on audit rows and actions, the admin audit `space=` filter, spaces in the kill drill, and `sb tenant apply` accepting spaces. W10 is built.
- Spaces in the web UI and the chat app (S10.3b: SB-1007): `GET /v1/spaces`, spaces and label on every citation, the "Ask in" switcher and place badges, a space's `channels` in the manifest, and the Slack app asking in a channel's space.
- Ask in a space or company-wide (S10.3a: SB-1006): space, label and barrier checks before the live ACL batch, `?space=` on asks, three trace reasons, spaces in the local eval cell, and `space-leak` in the release gate. LAW-20 is green.
- Ingest routes and labels every document (S10.2b: SB-1005): `internal/core/route`, location and label from every connector (Drive folder path and classification label, Slack private channels), routing on chunks in both indexes, and re-routing on the next sync after a scope change.
- Cedar schema v2 for spaces (S10.2a: SB-1004): `label` and `spaces` on chunks, the space read context, `policy/platform/spaces.cedar` (forbids only), and the narrowing analysis in `make policy`. LAW-19 is green.
- Per-space data keys (S10.1c: SB-1003): `keywrap.Shred` and `keywrap.DataKey`, `seal.Sealer` (AES-256-GCM per space), and keys made and shredded by `spaces.Sync`.
- The space registry and request context (S10.1b: SB-1002): `ports.SpaceStore` with memory and Postgres adapters, `spaces.Sync` and `spaces.Resolve`, `Scope.Space()`, and `?space=` on every API call (`ESPACE`, never a company-wide fallback).
- Spaces in the manifest (S10.1a: SB-1001): `sb tenant check` accepts and checks `spaces:` and `barriers:` (sb-space(5)), `apply --plan` shows them, and `apply` refuses them with `ENOTIMPL` until SB-1006 enforces them.
- Identity Platform sign-in (S9.3b: SB-907): the `identity-platform://` idp and `groups:` manifest field, the `firebase.tenant` check in the `oidc` adapter, `internal/core/groupsync`, and LAW-02 and LAW-12 law tests.
- Firebase Hosting for the web UI (S9.3a: SB-906): `internal/core/hosting` (the generated config and its check), `gcp.Hosting`, Hosting in the GCP fake, and a LAW-02 law test on rewrites.
- Cell endpoints (S9.2b: SB-905): `/metrics` and the verifier endpoints in `internal/api.Verifier`, `runtime.Attestor`, LAW-17 in `verify`, `alerts.Dispatch`, and `sb tenant verify --cell URL`.
- Schedules in the cell (S9.2a: SB-904): `internal/core/schedule` (six jobs and the missed-run watchdog), `gcp.Scheduler` and the WORM `gcp.Bucket`, Cloud Scheduler and Cloud Storage retention in the GCP fake, and law tests for write-once anchors and missed runs.
- Private Cloud SQL with CMEK and a perimeter (S9.1b: SB-902): private-IP, `pgvector`, CMEK instances from the gcp adapter; the perimeter bootstrap script; CMEK, crypto-shredding and VPC Service Controls in the GCP fake; and a LAW-02 law test over two perimeters.
- Cloud Run per cell (S9.1a: SB-901): `sb-runtime(7)`, `internal/core/runtime` (the in-cell updater), `gcp.CloudRun` with Binary Authorization in the GCP fake, and LAW-06 and LAW-02 law tests over the runtime.
- Space laws and models (S8.2: SB-804 to SB-807): LAW-19 to LAW-23 in sb-laws(7) and as red law tests with reference systems and mutations; `proofs/alloy/Spaces.als` with three mutations and a crosscheck mirror; the inside-the-company threats in the threat model; and the `space-leak` eval suite, red on purpose until SB-1006.
- Department spaces, specified (S8.1: SB-801 to SB-803): `sb-spaces(7)`, `sb-space(5)` and `sb-departments(7)`; spaces, labels and barriers in sb-permissions(7) and sb-policy(5); `ESPACE`, `ELABEL` and `EBARRIER`; a "choose your spaces" onboarding step; and `sb tenant check` refusing `spaces:` and `barriers:` by name until SB-1001.
- Hand-off game day (S7.3c: SB-710): the day's checks in sb-factory(7), its runbook, and `tests/handoff`, a full rehearsal of hand-off and detach in `make check`. **W7 built.**
- Model billing (S7.3b: SB-709): monthly cost statements (`internal/core/costs`, `GET /v1/admin/costs` as JSON or CSV) and a tenant-held provider key, read from the vault at every call.
- Detach and export (S7.3a: SB-708): the export package writer and offline verifier, `sb factory verify-export`, the registry's custody history and tombstone, and a LAW-11 law test over tampered packages.
- Releases after hand-off (S7.2d: SB-707): the HTTPS release-store client with a pull token, custody and detach in the release controller, the pause-pulls runbook, and LAW-06 and LAW-08 law tests for tenant-custody cells.
- Access paths (S7.2c: SB-706): `internal/core/access` (Check, Watch, the wrap source), the GCP IAM source with a read-only verifier identity, `getIamPolicy` in the GCP fake, `sb factory access`, and a LAW-17 law test.
- The hand-off controller (S7.2b: SB-705): `internal/core/handoff`, `Handoff.tla` in code and checked against it, with hash-bound dual approval, a custody audit row before every step, cutover with a LAW-17 re-check, abort and detach. The LAW-17 and LAW-18 hand-off law tests are green.
- Key wraps (S7.2a: SB-704): the wrap inventory with verified re-wrap, a destroy and abort that check the kept wrap live, the audit-signing data key made at stamp time, and `key_wraps` in `sb factory status`.
- GCP placement (S7.1c: SB-703): the `gcp` Placement and Keys adapter (Cloud SQL, Cloud KMS, Secret Manager, Pub/Sub over REST), its recorded fake, the tenant's deploy-role bootstrap script, and a LAW-02 law test over GCP projects.
- Factory roles (S7.1b: SB-702): `factory-operator` and `tenant-owner` in Cedar, a `CellCustody` entity and six custody actions; no self-approval of a plan, no content reads across tenants, and custody records must name their tenant.
- Hand-off specified (S7.1a: SB-701): `sb-factory(1)`, the phases and invariants of a hand-off in sb-factory(7), the hand-off runbook, and LAW-17 and LAW-18 law tests (red until SB-705).
- Pen-test readiness (S6.4b: SB-612): `security/` with the scope, the rules of engagement, the findings register and the injection corpus; a `make check` gate on open critical or high findings; law tests for hostile hosts, encoded secrets and the corpus.
- DR game day (S6.4a: SB-613): the scoring rules in sb-observability(7), `internal/core/gameday` (restore from the first good copy into empty stores, and the score), a rehearsal in `tests/gameday` run by `make check`, the game day runbook, and a LAW-11 law test for a restore.
- Retention and deletion (S6.3c: SB-611): the nightly retention job over traces, conversations, feedback and log files (never audit or the index), `POST /v1/admin/subjects/erase` with audit before and after, the `retention` audit kind, daily log files, and a LAW-13 law test showing that retention leaves the index derived.
- HubSpot CRM connector (S6.3b: SB-610), `crm: hubspot`, read-only. It covers companies and deals, never contacts. Record-level access is checked live from the user's role, teams and the record's owner, through a role map held in the vault. Changes are polled through the CRM search API. Tested against the `hubspottest` recorded fake, with a new LAW-01 law test.
- Operations (S6.3a: SB-607 to SB-609): a Sev-1 alert rule and runbook for every law with a production check, a metric catalogue and tenant dashboard, a telemetry rule that keeps content out of labels, histograms, ask and ACL metrics, and a monthly budget and per-user rate limit in the model gateway (`EBUDGET`).
- Onboarding and the weekly report (S6.2: SB-604 to SB-606): every runbook step states what to expect and every phase has a gate checklist, enforced by `tools/check/runbook.py`; the weekly leadership report and `GET /v1/admin/usage`, counts only.
- Admin console and roles (S6.1b: SB-602, SB-603): an operator role as a Cedar policy, the admin API (manifest, diff by JSON Patch through the manifest checker, kill switches, audit paging), `manifest.Format` with a round-trip guarantee, and a web console for kill switches and make-permanent.
- Tenant verify (S6.1a: SB-601): law checks against a cell's own data and users (LAW-01 re-asked live, LAW-02 adversarial ids, LAW-03 injected timeout, LAW-14 claim labels), counts only, unknown fails; runs on the local cell with `tools/evalcell --verify`.
- Attestation and drift (S5.3b: SB-509, SB-510): signed cell attestations checked against the deployed bundle with Sev-1 findings for mismatch, replay and silence; manifest drift detection that names each resource and field.
- Release controller and rollback (S5.3a: SB-507, SB-508): channels, rings, the 48-hour soak, the promote guard, draining rollback and a timed rollback drill, all as in `Rollout.tla` with its invariants checked after every move; `sb release status|promote|rollback` on a local control plane.
- Lockfiles and the CVE path (S5.2b: SB-505): `make lockfiles`, a release-time check that lockfiles are current, a daily advisory watcher (`govulncheck`, `npm audit`), and the patch-release playbook.
- Supply chain (S5.2a: SB-504, SB-506): reproducible builds checked twice, an SPDX 2.3 SBOM as a bundle artefact, cosign keyless signing in a release workflow, Ed25519 development signing, signed gate records, and cell admission that refuses anything unsigned, tampered or ungated (`EUNSIGNED`).
- Eval suites and the gate (S5.1b: SB-501, SB-502): permission, injection, refusal and regression suites beside citation; results stored per release; `python3 -m evals gate` applies `evals/gate.yaml` (failures, scores, drop against stable), and `make evals` gates every run.
- Release bundles (S5.1a: SB-503): `bundle.json` pins artefact digests, prompt hashes, tool schemas, policy hashes, the model and the gate; the release id is derived from its canonical form; `tools/release/bundle` builds it. Canonical JSON moves to `internal/core/canonjson`.
- Backups and the restore drill (S4.4b: SB-412): nightly signed audit export and approvals snapshot to at least three write-once copies on two media with one off-site, 7-year retention, and a drill that verifies every copy's chain and names any failure.
- Secret hygiene (S4.4a: SB-411): canaries and a scanner over logs, prompts and audit, Sev-1 `secret_leak` alerts through an HTTPS webhook (ids only), audit rows redacted before hashing, a gateway `OnSecret` hook, `vault://tenants/T/canary`, and the CI scan in `tests/canary`.
- draft-reply (S4.3c: SB-409): `POST /v1/drafts` writes a cited reply to a live-checked ticket or thread with the bundled `draft` prompt; never sends; `used`/`discarded` outcomes in `sb_drafts_total`; a draft panel in the web UI with Copy and Discard; a tenant-scoped base permit for enabled `draft-reply`.
- The `ticket-update` workflow (S4.3b: SB-410): a strict comment template, a registry built from the manifest, idempotent Linear comments with `Find`, an eval gate in `sb tenant apply` (`EEVAL` without a passing record no more than 30 days old), and `evals run --record`.
- Idempotency store and reconciler (S4.3a: SB-408): receipts kept by idempotency key so nothing runs twice; unconfirmed outcomes looked up by key for 24 hours, never re-executed.
- Actions API and approver view (S4.2b: SB-407): `POST /v1/actions`, `GET /v1/actions/{id}` with policy reason and field diff, approve and reject; a web approver view that echoes the shown hash and needs explicit clicks; id-only notifications.
- Tool bus, approvals and executor (S4.2a: SB-405, SB-406, SB-408): one path for every tool call (kill check, policy, audit row, approval, execute), approvals bound to an RFC 8785 payload hash over payload, target and workflow, no self-approval, 24-hour expiry, receipts required for `Ran`, and a lint that only the bus calls `Act`. Base policies gain tenant-scoped `call_tool`, `propose` and `approve` permits.
- Kill switches (S4.1b: SB-403, SB-404): audited set and clear, a per-cell checker with a 2-second poll, push and a 30-second fail-safe, killed connectors treated as Unknown at the live check, `sb kill|unkill|kill list` on a local control plane, and the release kill drill with its report.
- Audit chain (S4.1a: SB-401, SB-402): hash-chained rows with RFC 8785 canonical form, per-cell Ed25519 anchors, an append-only Postgres store (triggers plus an INSERT/SELECT-only role), signed export, and `sb audit verify --export FILE [--key]` offline. Policy decisions can be written as `decision` rows.
- Pilot tooling (S3.8: SB-311): the pilot-review runbook in sb-onboarding(7), `python3 -m evals.pilot report` for the Gate 1 numbers, and a retro template.
- Feedback (S3.7: SB-310): `POST /v1/feedback` stores ratings, wrong-citation reports and comments against the trace, asker only; a weekly JSON-lines export with no question or answer text; Helpful/Not helpful controls in the web UI.
- Citation-faithfulness eval (S3.6: SB-309): a strict lexical support scorer, a local eval cell running the real ask path over fixture worlds, the shared `citation` suite and per-tenant golden questions in `tenants/<name>/evals/`, all in `make evals`.
- Slack `/ask` (S3.5: SB-308): signed and workspace-checked requests, one-active-user identity mapping by email, ephemeral answers through `response_url`, and a Share button that posts citation links only.
- Web chat UI (S3.4: SB-307): React, Vite and TypeScript, served by the cell at `/` with a self-only CSP; citation chips, an "Uncited" label, notices, and axe-checked accessibility. `make web` and a CI `web` job.
- `POST /v1/ask` and `GET /v1/traces/{id}` (S3.3: SB-306): the ask agent, conversations that re-check every source, traces with decision lists, and the sb-api(3) error shape.
- Cited answers and the model gateway (S3.2: SB-304, SB-305): the answer composer decides citations in code; a provider-neutral gateway with OpenAI-compatible, Anthropic, Groq and OpenRouter adapters refuses prompts holding secrets and meters tokens.
- Retrieval and live authorisation (S3.1: SB-301 to SB-303): pgvector index per tenant, hybrid search with the prefilter in the query, batched live ACL per connector, policy `read` check, a 12-chunk budget.
- Linear tickets connector (SB-211): read-only GraphQL, team-based ACL, polled changes with an exact tie-break; tested against a recorded fake.
- Slack chat connector (SB-204): read-only, channel-membership ACL, DMs never read, Events API with signature checks; tested against a recorded fake.
- Weekly rebuild drill (SB-210): `ingest.Drill` compares the index before and after a rebuild from sources.
- LAW-01 ingest-gap test (SB-209): deletes and permission changes reach the prefilter in one sync; the live check holds in between.
- Durable sync (SB-208): per-connector cursor store, scheduler, and freshness metrics on a Prometheus text endpoint.
- Ingest sanitiser (SB-207): strip hidden text, flag instruction-like spans per chunk, unescapable data blocks; injection corpus in `make check`; a repo-wide guard against invisible characters in source.
- Google Drive docs connector (SB-203): read-only, live ACL as the asker via domain-wide delegation, rate-limit retries, change feed; tested against a recorded fake.
- Ingest pipeline (SB-206): stable chunking, derived chunk ids, provenance, `Embedder` and `Index` ports with local adapters, sync and rebuild.
- Secrets by reference (SB-205): validated `vault://` references, a local vault, and a redacting structured logger (`internal/obs`).
- Scriptable mock connector (SB-202): fault queue, forced verdicts, deletes, no-receipt target.
- Connector contract v1 (SB-201): `APIVersion` and negotiation, the contract kit (`connectorkit.Run`), and a controllable mock source and connector.
- Factory seams (SB-114): `sb factory status TENANT`, custody constants in one place, and the custody lint.
- Policy analysis in CI (SB-112): `make policy` validates and tenant-scopes every policy; pull requests show a permission diff.
- Policy decisions are audited (SB-111): `policy.Kernel` records before returning, samples read allows (`SB_AUDIT_READ_SAMPLE`), and a failed record is `Unknown`.
- Cedar schema v1 and base policies (SB-110): `Document`, a tenant on every entity, tenant condition on every permit (BUG-0006).
- Policy kernel (SB-109): `policy.Authorise`/`Require` with Allow/Deny/Unknown, a cedar-go adapter (vendored), and the `decisions` lint.
- `tenancy.Scope` (SB-108): unforgeable request scope, issued only by the auth middleware; CI rule against other issuers; first scoped port (`People`).
- SCIM 2.0 per cell (SB-107): users and groups, deactivate revokes sessions before responding, groups read at decision time.
- OIDC sign-in per tenant (SB-106): code flow with PKCE, RS256 ID-token checks including the tenant claim, 12-hour sessions re-validated every 60 seconds, and a test IdP.
- Tenant registry and routing (SB-105): Postgres registry adapter (pgx, vendored), cell records with custody, edge router (`EPERM`/`EUNAVAIL`/`EAUTH`/`ETENANT`), `make integration` and the `postgres` CI job.
- ADR-0017 and the module allow-list (`tools/deps/allowed.txt`, checked by `make deps`): cedar-go v1.8.0 and pgx v5.8.0 are admitted.
- `sb tenant apply FILE` (SB-104): idempotent, resumable apply with a write-ahead step journal, on the local control plane (`--endpoint file:///DIR`).
- Cell template (SB-103): `internal/core/cell`, `Placement` and `Keys` ports with envelope keys, adapter contract tests, and a `local` adapter. ADR-0018 (cloud-neutral cells). The law ratchet accepts per-test lines in `laws/expected_red.txt`.
- `sb tenant apply FILE --plan` (SB-102): resource-level dry run against the tenant registry, `--json`; `--endpoint file:///DIR` for a local registry.
- `sb tenant check FILE` (SB-101): strict manifest parser with `file:line:col: field: message` errors, `--json`, and a JSON Schema at `examples/tenants/tenant.schema.json`.
- W0 build: `sb` CLI (every command in `sb(1)`; unbuilt ones exit `ENOTIMPL`), `internal/errs` generated from `sb-errors(7)`, `internal/core/ports`.
- `make check` gate: generated-code check, lint with the dependency rule, unit, law, smoke and regression tests, evals (mock), example and manual checks, cross-check, CI-rule test and docs.
- Law harness with five laws red for the right reason and self-tests proving the checks.
- Eval harness (`python3 -m evals`, `sb eval run --model mock`).
- CI jobs: check, manual-diff, diagrams, proofs (TLC and Alloy, checksum-pinned).
- `ENOTIMPL` error code. ADR-0016 (standard library only in W0).
- `tools/ci/mutations.py`: the 13 model mutation checks, run by `make proofs`.

### Fixed

- Barriers matched groups only when both sides were written `group:NAME`; they now compare names (SB-1107).
- Law tests no longer leave their release stores in `/tmp` (BUG-0013).
- Cutover no longer revokes a retained grant's binding: the hand-off controller keeps paths by the grant they serve (BUG-0012).
- The log redactor scrubs URL-escaped, base64 and hex forms of secrets, and credential-named attributes (BUG-0010). Markdown role labels are flagged as injection (BUG-0011).
- `make proofs` pins TLA+ v1.7.4 (stable) instead of the v1.8.0 prerelease, which upstream rebuilt in place (BUG-0009).
- `make smoke` is never cached, so a rebuilt `sb` is always tested; the smoke test covers the workflow eval gate (BUG-0008).
- Audit `decision` rows no longer repeat the action in `reason` (the kernel's reason already starts with it).
- The ask agent's live check now passes the asker's email to connectors, which ask their sources by email (BUG-0007).

### Changed

- `tools/ci/proofs.sh`: TLC runs from the model directory; Alloy runs each `check`/`run` by name (first real run, SB-013).
- `Rollout.tla`: rollback is no longer atomic; revoked cells drain (BUG-0001).

### Design baseline

- Problem statement, principles, domain model, architecture (17 documents), ADRs, threat model.
- Manual: 32 man pages across sections 1, 3, 5 and 7.
- Plan: 9 epics, 42 stories, 29 slices, 96 tickets in 8 waves.
- Formal models: `Approvals.tla`, `Rollout.tla`, `Permissions.als` (written; TLC and Alloy not yet run).
- `proofs/crosscheck.py`: Python mirror of the three TLA+ models; all invariants hold and every listed mutation is caught.
- Cedar schema and platform base policies, validated against the Cedar validator.
- Examples: manifests, tenant policy, eval suites, a read-only connector.
- Docs site generator `tools/site/build.py`.
- Factory and hand-off design: ADR-0015, architecture doc 16, `sb-factory(7)`, LAW-17 and LAW-18, `Handoff.tla`, epic E08 in wave W7, and v1 seams (SB-114 plus extra acceptance on SB-103, SB-105, SB-401, SB-506, SB-509).

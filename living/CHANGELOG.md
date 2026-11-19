---
title: Changelog
summary: User- and operator-visible changes, by release.
---

# Changelog

All notable changes, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) style. Releases are named `rel_DATE_DIGEST`.

## Unreleased

### Added

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

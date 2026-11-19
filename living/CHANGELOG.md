---
title: Changelog
summary: User- and operator-visible changes, by release.
---

# Changelog

All notable changes, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) style. Releases are named `rel_DATE_DIGEST`.

## Unreleased

### Added

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

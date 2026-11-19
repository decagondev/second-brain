---
title: Manual
summary: Man-page reference for commands, API, file formats and concepts. The manual is the spec.
---

# The Second Brain manual

**The manual is the spec.** Code follows these pages, not the other way round. If the platform does something these pages don't describe, that's a bug in the code or a missing page, and both are fixed in the same PR (ADR-0009).

New here? Take the [tour](sb-tour.7.md) first.

## (7) Concepts

| Page | About |
|---|---|
| [sb-tour(7)](sb-tour.7.md) | Fifteen minutes from manifest to cited answer to approved action |
| [sb-laws(7)](sb-laws.7.md) | The eighteen laws, and how each is checked |
| [sb-tenancy(7)](sb-tenancy.7.md) | Cells, isolation, regions |
| [sb-spaces(7)](sb-spaces.7.md) | Department spaces inside a cell: membership, labels, barriers, company-wide asks (W8 draft) |
| [sb-departments(7)](sb-departments.7.md) | The twelve department templates and the access practices behind them (W8 draft) |
| [sb-factory(7)](sb-factory.7.md) | The factory, custody levels and hand-off |
| [sb-identity(7)](sb-identity.7.md) | Sign-in, SCIM, sessions, deprovisioning |
| [sb-permissions(7)](sb-permissions.7.md) | Allow, deny, unknown; prefilter versus live check |
| [sb-ingest(7)](sb-ingest.7.md) | Sync, sanitising, provenance, freshness, rebuild |
| [sb-citations(7)](sb-citations.7.md) | `doc@version`, claims and `uncited` |
| [sb-agents(7)](sb-agents.7.md) | `ask`, `draft-reply`, workflows and the tool bus |
| [sb-approvals(7)](sb-approvals.7.md) | The action state machine and payload binding |
| [sb-audit(7)](sb-audit.7.md) | What is recorded, and how to prove it wasn't changed |
| [sb-kill-switches(7)](sb-kill-switches.7.md) | Stopping agents, connectors and tenants |
| [sb-evals(7)](sb-evals.7.md) | Suites, thresholds and the release gate |
| [sb-release(7)](sb-release.7.md) | Bundles, channels, rings, rollback, attestation |
| [sb-observability(7)](sb-observability.7.md) | Metrics, alerts, reports, telemetry rules |
| [sb-onboarding(7)](sb-onboarding.7.md) | The day 0 to 90 runbook for a company's IT team |
| [sb-errors(7)](sb-errors.7.md) | Every error code, its status, exit code and remedy |

## (1) Commands

| Page | About |
|---|---|
| [sb(1)](sb.1.md) | The CLI: global options, output, exit codes |
| [sb-tenant(1)](sb-tenant.1.md) | `check`, `apply`, `verify` |
| [sb-kill(1)](sb-kill.1.md) | `kill`, `unkill`, `kill list` |
| [sb-audit(1)](sb-audit.1.md) | `verify`, `tail`, `export` |
| [sb-release(1)](sb-release.1.md) | `status`, `promote`, `rollback` (maintainers) |
| [sb-eval(1)](sb-eval.1.md) | `run`, `diff` |
| [sb-factory(1)](sb-factory.1.md) | `status`, `handoff`, `access`, `detach` (factory operators, tenant owners) |

## (3) API

| Page | About |
|---|---|
| [sb-api(3)](sb-api.3.md) | Conventions: auth, versioning, errors, idempotency, pagination |
| [sb-ask(3)](sb-ask.3.md) | `POST /v1/ask`, traces, feedback |
| [sb-actions(3)](sb-actions.3.md) | Drafts, proposed actions, approvals |
| [sb-admin(3)](sb-admin.3.md) | Manifest diffs, kill switches, audit and usage |
| [sb-connector(3)](sb-connector.3.md) | The four-call connector contract |

## (5) File formats

| Page | About |
|---|---|
| [sb-tenant(5)](sb-tenant.5.md) | The tenant manifest |
| [sb-space(5)](sb-space.5.md) | The `spaces:` and `barriers:` manifest blocks (W8 draft) |
| [sb-policy(5)](sb-policy.5.md) | Cedar policy files and schema |
| [sb-eval(5)](sb-eval.5.md) | Eval suite files and the gate file |
| [sb-audit(5)](sb-audit.5.md) | Audit rows and export format |

## Writing a page

Copy [TEMPLATE.md](TEMPLATE.md). Every page has the same sections in the same order, so readers and tools always know where to look. Code blocks whose info string ends in `run` (for example, a `yaml` block tagged `run`) are executed in CI (SB-009).

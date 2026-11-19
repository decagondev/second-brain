---
title: Engineering Standards
summary: Code layout, the dependency rule, naming, errors, logging and review rules.
---

# Engineering standards

These rules make code easy to read for people, LLM agents and tools. When in doubt, choose the version that's shorter and more obvious (P3).

## Repository layout

One Go module (`go.mod` at the root, module `example.com/second-brain`; rename on first push). A `go.work` workspace isn't needed while there is one module (P3). W0 uses the Go standard library only (ADR-0016).

```text
cmd/
  sb/              CLI: tenant, kill, audit, release, eval
  cell-api/        per-cell HTTP API: ask, actions, admin, scim
  ingestd/         per-cell ingest workers
  controller/      control plane: registry, releases, kill flags, attestation
internal/
  core/
    ports/         interfaces only: Connector, IdentityProvider, Model, Store, Index, Secrets, Queue, Clock
    tenancy/       Scope, TenantID
    policy/        Authorise, Decision, Require
    retrieve/      hybrid search + live authorisation
    compose/       claims, citations, uncited
    agent/         agent loop, limits
    toolbus/       the five-step path
    approvals/     state machine (matches Approvals.tla)
    actions/       executor, receipts, idempotency
    audit/         hash chain, write-ahead
    kill/          flag cache, checks
    manifest/      parse, validate, normalise, diff
    release/       bundles, channels, rings (matches Rollout.tla)
  adapters/
    connectors/<kind>-<vendor>/
    idp/oidc/  idp/scim/
    model/<provider>/
    store/postgres/  store/pgvector/
    vault/<provider>/  queue/<provider>/
laws/              law checks, world generator, self-tests; expected_red.txt is the ratchet (SB-010)
evals/             Python eval runner (python3 -m evals), suites, fixtures, gate.yaml
proofs/            TLA+ and Alloy
policy/            Cedar schema and platform policies
infra/cell/        cell template (IaC)
web/               TypeScript console and chat UI
tests/smoke/  tests/regress/   black-box smoke tests; one regression test per bug, forever
tools/             gen/ (code from the manual), check/ (repo rules), ci/ (CI helpers), site/
manual/  docs/  living/  epics/  examples/  tools/
```

## Make targets

`make check` is the gate and must pass on a clean clone. It runs, in order: `gen-check`, `lint`, `deps`, `build`, `test`, `laws`, `smoke`, `regress`, `evals`, `examples`, `crosscheck`, `ci-rules`, `docs`. `make diagrams` and `make proofs` need a browser and Java, and run as separate CI jobs.

## The dependency rule

`internal/core/**` imports only the standard library, `internal/core/**` and a short allow-list (`rapid` in tests). Never `internal/adapters/**`, never a vendor SDK. Enforced twice (SB-008): by `depguard` in `.golangci.yml`, and by `tools/check/deps.py`, which reads `go list` directly so the rule holds even if lint config drifts. See the table in the [architecture overview](architecture/00-overview.md#layering-and-the-dependency-rule).

## Go style

- `gofmt`, `golangci-lint` v2 (config in `.golangci.yml`) with: `errcheck`, `govet`, `staticcheck`, `depguard`, `exhaustive` (so every `Decision` switch handles `Unknown`), `forbidigo` (bans `fmt.Print*` and `log.*` outside `cmd/`).
- Every exported function that does I/O takes `ctx context.Context` first and `tenancy.Scope` second.
- Zero values must be safe. If the zero value would mean "allow", "ran" or "approved", change the type.
- Prefer small functions with names that read as sentences: `policy.Require(res)`, `audit.WriteAhead(ctx, s, row)`.
- Three lines beat ten: no wrappers that only rename, no interfaces with one implementation outside `ports`.

## Errors

- Errors carry a stable code from [`sb-errors(7)`](../manual/sb-errors.7.md): `errs.New(errs.EPERM, "chunk %s refused", id)`.
- `internal/errs/codes.go` is generated from the manual by `tools/gen/errs.py`; `make gen-check` fails if it drifts. An uncoded error exits 1: an unknown failure is never success.
- The code maps to one HTTP status and one CLI exit code, generated from the registry table (SB-006).
- Messages say what happened and what to do next, in one line.

## Logging

- Structured JSON with `tenant`, `trace`, `release`, `code`. Ids only; never content, prompts, answers or secrets.
- Log at the edge of a decision, not at every step.

## Naming

| Thing | Style | Example |
|---|---|---|
| CLI verbs | short, lowercase, one word | `sb tenant verify` |
| Manifest fields | `snake_case` | `retention_days` |
| API fields | `snake_case` JSON | `"payload_sha256"` |
| Error codes | `E` + uppercase | `EPERM`, `EKILLED` |
| Laws | `LAW-nn` | `LAW-03` |
| Tickets | `SB-nnn` | `SB-302` |
| Ids | prefix + ULID | `tr_01JD…` |

## Review

| Area | Reviewers |
|---|---|
| `policy`, `audit`, `toolbus`, `approvals`, `actions`, `release`, `proofs`, `evals/gate.yaml` | 2, one from security |
| Everything else | 1 |
| Manual-only PRs | 1 |

Every PR title is `SB-nnn: summary`. The PR template is the [Definition of Done](05-living-documents.md#definition-of-done).

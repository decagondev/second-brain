---
title: "SB-109: Policy engine v0: `authorise` returns allow, deny or unknown"
summary: "The kernel has one entry point and three answers; unknown is never allow."
---

# SB-109: Policy engine v0: `authorise` returns allow, deny or unknown

| Field | Value |
|---|---|
| Epic | [E02 Policy Kernel](README.md) |
| Wave / slice | W1 / S1.4 Policy kernel |
| Stories | US-02.1 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-108](../E01-tenancy-identity/SB-108-tenant-scoped-request-context.md) |
| Status | In review |

## Why

The kernel has one entry point and three answers; unknown is never allow.

## Acceptance criteria

- [x] `authorise(principal, action, resource, context)` returns `Allow|Deny|Unknown` with a reason
- [x] Every caller maps `Unknown` to refusal; a lint forbids switch statements that fall through on Unknown
- [x] LAW-03 green over mocks (policy level; retrieval level is SB-302)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Define `policy.Decision` type
- [x] Cedar evaluator adapter
- [x] Caller helper `policy.Require`
- [x] Custom lint for Decision handling
- [x] LAW-03 fault-injection tests
- [x] Write or extend the law tests: LAW-03
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built.
- `internal/core/policy`: `Decision` (the zero value is `Unknown`) and `Authorise(scope, action, resource, context)`. It builds the principal only from the `tenancy.Scope`, denies other-tenant resources before any policy runs, and runs the engine with a 50 ms deadline and panic recovery. Anything but a clean Allow or Deny is `Unknown`, always with a reason. `Require` returns nil only for Allow.
- `internal/adapters/policy/cedar` uses cedar-go v1.8.0 (ADR-0017; vendored now). Policy evaluation errors become `Unknown`. Tests run the platform `base.cedar` through the real kernel.
- `tools/check/decisions.py` (run by `make lint`, with a built-in self-test) refuses `!= policy.Deny` and `default:` in Decision switches. `exhaustive` requires the `Unknown` case.

LAW-03 gains `TestLAW03_PolicyUnknownRefuses`, red first, then green. It injects engine errors, timeouts, panics and invalid values, plus other-tenant resources and missing scopes. A "not denied means allowed" caller is caught.

**Left:** policies are loaded from files for now; loading the tenant's policies from the release bundle is SB-503. Also left: review by the user, and the first green CI run.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

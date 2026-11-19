---
title: "SB-112: Policy analysis in CI"
summary: "A policy change is a code change and gets the same gate."
---

# SB-112: Policy analysis in CI

| Field | Value |
|---|---|
| Epic | [E02 Policy Kernel](README.md) |
| Wave / slice | W1 / S1.4 Policy kernel |
| Stories | US-02.2 |
| Manual | [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-110](../E02-policy/SB-110-cedar-schema-and-base-policies.md) |
| Status | In review |

## Why

A policy change is a code change and gets the same gate.

## Acceptance criteria

- [x] CI validates policies against the schema
- [x] CI rejects any `permit` without a tenant condition
- [x] Policy diffs show added and removed permissions in the PR

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Cedar validate step
- [x] Custom tenant-scope check
- [x] Permission diff reporter
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: `tools/check/policy.py`, run by `make check` as `make policy`, does three things. It validates every policy file against the schema with cedarpy (a missing validator fails). It requires `principal.tenant == resource.tenant` in every permit's `when`. It limits tenant policies to `use_agent`/`propose`/`approve` permits. On pull requests, CI runs `--diff origin/BASE`, which evaluates a fixed matrix (member and each group, times every action and resource kind) against the old and new policies and prints the permissions gained and lost. A built-in self-test runs first, so the rules can't silently stop matching (BUG-0005). Mutation-checked: a tenant-less permit dropped into `examples/policies/` fails with both rule messages. LAW-02's dynamic policy test (SB-110) and this static check guard the same rule from two sides.

**Left:** a second reviewer. Changes to `policy` and `audit` need two (definition of done); the user reviewed on 2026-11-19, and CI is green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

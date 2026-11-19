---
title: "SB-110: Cedar schema and base policies"
summary: "Policies a company operator can read beat policies hidden in code."
---

# SB-110: Cedar schema and base policies

| Field | Value |
|---|---|
| Epic | [E02 Policy Kernel](README.md) |
| Wave / slice | W1 / S1.4 Policy kernel |
| Stories | US-02.2 |
| Manual | [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-109](../E02-policy/SB-109-policy-engine-v0-authorise-returns-allow.md) |
| Status | In review |

## Why

Policies a company operator can read beat policies hidden in code.

## Acceptance criteria

- [x] Schema covers principals, groups, documents, chunks, tools and approvals
- [x] Base policy set: tenant scope on every permit; default deny

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Write `policy/schema.cedarschema`
- [x] Write base policies
- [x] Example tenant policies in `examples/policies/`
- [x] Write or extend the law tests: LAW-02, LAW-03
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: the schema gains `Document` (with `Chunk in [Document]`) and a `tenant` on every entity (`Agent`, `Tool`, `Workflow`). `read` applies to chunks and documents. The base policies now scope every permit to the tenant. A new law test, `TestLAW02_PoliciesNeverCrossTenants`, runs the real platform and example tenant policies through cedar-go with the kernel's guard bypassed. On its first run it was red for a real reason: `permit use_agent ask` had no tenant condition (BUG-0006). It is fixed and green, and a self-test catches a tenant-less permit. Every policy file and docs `cedar` block still validates against the new schema (`make regress`, BUG-0003 test). LAW-03 for policies is covered by the SB-109 kernel test and the adapter test (an erroring policy is `Unknown`).

**Left:** review by the user, and the first green CI run. CI's tenant-condition check and permission diffs are SB-112.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

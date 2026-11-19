---
title: "SB-111: Policy decisions are audited"
summary: "An unexplained allow is a future incident."
---

# SB-111: Policy decisions are audited

| Field | Value |
|---|---|
| Epic | [E02 Policy Kernel](README.md) |
| Wave / slice | W1 / S1.4 Policy kernel |
| Stories | US-02.3 |
| Manual | [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04) |
| Depends on | [SB-109](../E02-policy/SB-109-policy-engine-v0-authorise-returns-allow.md) |
| Status | In review |

## Why

An unexplained allow is a future incident.

## Acceptance criteria

- [x] Every decision writes principal, action, resource, decision, policy ids and reason
- [x] Decision records are sampled for reads and complete for tool calls

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Decision hook in policy engine
- [x] Sampling config
- [x] Tests
- [x] Write or extend the law tests: LAW-04
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: `policy.Kernel` (engine plus `Recorder`) is now the only way to ask; the free `Authorise` is gone. Every `Deny`/`Unknown` and every non-read decision is recorded before it is returned, and read allows are sampled (`SB_AUDIT_READ_SAMPLE`, default 1%). A failed or missing recorder turns the decision into `Unknown`. Records carry tenant, actor, action, subject (`Type::"ID"`) and reason with policy ids: ids only, no content. The rows feed the audit chain when SB-401 lands. LAW-04 gains `TestLAW04_PolicyDecisionsAreRecordedFirst`, red first, then green. A kernel that allows when the audit write fails is caught. The red ratchet line for the existing LAW-04 test is now per-test.

**Left:** the audit adapter that writes `decision` rows into the hash chain (SB-401). Also left: review by the user, (first CI run green on `c0a12be` (all jobs: check, proofs, postgres, diagrams)).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

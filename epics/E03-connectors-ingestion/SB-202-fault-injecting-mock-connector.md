---
title: "SB-202: Fault-injecting mock connector"
summary: "LAW-03 is only tested if unknown, timeout and missing can be produced on demand."
---

# SB-202: Fault-injecting mock connector

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.1 Contract |
| Stories | US-03.5 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-03](../../manual/sb-laws.7.md#law-03), [LAW-05](../../manual/sb-laws.7.md#law-05) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | Open |

## Why

LAW-03 is only tested if unknown, timeout and missing can be produced on demand.

## Acceptance criteria

- [ ] Mock can return allow, deny, unknown, timeout, 404 and no-receipt on demand
- [ ] LAW-03 and LAW-05 suites use it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Scriptable mock
- [ ] Fault matrix tests
- [ ] Write or extend the law tests: LAW-03, LAW-05
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

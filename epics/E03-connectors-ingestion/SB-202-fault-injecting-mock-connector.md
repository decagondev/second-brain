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
| Status | Done |

## Why

LAW-03 is only tested if unknown, timeout and missing can be produced on demand.

## Acceptance criteria

- [x] Mock can return allow, deny, unknown, timeout, 404 and no-receipt on demand
- [x] LAW-03 and LAW-05 suites use it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Scriptable mock
- [x] Fault matrix tests
- [x] Write or extend the law tests: LAW-03, LAW-05
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: done. The mock from SB-201 is now fully scriptable: `QueueFaults` (one fault per coming ACL call), `ScriptVerdict` (force any verdict, including Unknown, for one user and document), `Delete`, `IssueIDs(false)` (no receipt and no effect), and read-only mode. `TestFaultMatrix` asks for each behaviour and checks it. The connector-level LAW-03 test now also covers a scripted Unknown on one ref among truthful ones; LAW-05's runs on the mock too. The retrieval-level suites (SB-302) will use the same mock when they're wired.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-1104: HRIS connector (read-only, restricted)"
summary: "People questions (policies, org chart, who to ask) are some of the most common, and HR data is the most sensitive."
---

# SB-1104: HRIS connector (read-only, restricted)

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.3 People and Finance sources |
| Stories | US-11.3 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), `sb-departments(7)` (new) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-07](../../manual/sb-laws.7.md#law-07), LAW-20 (new, W8) |
| Depends on | [SB-1106](SB-1106-field-level-allow-lists-for-restricted-sources.md) |
| Status | In review |

## Why

People questions (policies, org chart, who to ask) are some of the most common, and HR data is the most sensitive.

## Acceptance criteria

- [x] BambooHR read-only: policies, org chart and directory fields on an allow-list
- [x] Compensation, performance, cases and personal identifiers are never ingested unless named in the allow-list by a steward, and then labelled `restricted`
- [x] Passes the contract kit against a recorded fake (a real BambooHR account stays In review)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Connector
- [x] Field allow-list
- [x] Contract kit
- [x] Write or extend the law tests: LAW-01, LAW-07, LAW-20
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.3b): sb-connector(3) gained `hris: bamboohr`, and `hris` is a kind of the manifest, kill switches, vault references and asks. `internal/adapters/connectors/bamboohr` reads one company with a key from the vault: employee records as structured documents with nine non-sensitive and eleven sensitive declared fields, and company files shared with all employees as text. It asks BambooHR only for the non-sensitive fields and the sensitive ones a space allow-lists, so other sensitive values are never read; ingest (SB-1106) indexes only allowed fields and labels a record with an allowed sensitive field `restricted`. Access is live: an active employee by work email may read records and shared files, anyone else is denied, and a failed directory read is `Unknown`. Changes come from `employees/changed` and from the shared files compared with the cursor (unsharing is a permission-only change). `bamboohrtest` records the API and every field asked for; the connector passes the contract kit. Law tests: `TestLAW03_BambooHRUnknownIsNeverAllow`, `TestLAW01_BambooHRAccessIsLive` and `TestLAW07_BambooHRSensitiveFieldsOnlyWhenAllowed` (a connector that reads every sensitive field is caught).

**Left:** a run against a real BambooHR account, and the runtime wiring of `Allowed` from the spaces' `fields`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

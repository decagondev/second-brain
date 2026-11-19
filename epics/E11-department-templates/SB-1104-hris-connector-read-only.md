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
| Status | Open |

## Why

People questions (policies, org chart, who to ask) are some of the most common, and HR data is the most sensitive.

## Acceptance criteria

- [ ] BambooHR read-only: policies, org chart and directory fields on an allow-list
- [ ] Compensation, performance, cases and personal identifiers are never ingested unless named in the allow-list by a steward, and then labelled `restricted`
- [ ] Passes the contract kit against a recorded fake

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Connector
- [ ] Field allow-list
- [ ] Contract kit
- [ ] Write or extend the law tests: LAW-01, LAW-07, LAW-20
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

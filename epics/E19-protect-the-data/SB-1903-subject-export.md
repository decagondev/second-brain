---
title: "SB-1903: Give a person their data"
summary: "A person can be erased, but a data-protection officer can't give them a copy of what the cell keeps about them."
---

# SB-1903: Give a person their data

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.2 |
| Stories | US-19.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-04, LAW-07 |
| Depends on | — |
| Status | Done |

## Why

A person can be erased, but a data-protection officer can't give them a copy of what the cell keeps about them.

## Acceptance criteria

- [x] `POST /v1/admin/subjects/export` returns, for one person, every record erase would remove: saved and shared questions, steward questions and replies, and feedback
- [x] It is an operator action like erase: policy-checked and audited before it runs; the audit row holds the hash, never the text
- [x] Export, then erase, then export gives an empty export

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Core export over the erase stores
- [x] Endpoint and tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.2): `POST /v1/admin/subjects/export` returns, by store, every record erase would remove for one person: traces, conversations, feedback, saved questions, shares, steward questions with their replies, and the log lines that name them. Every `retention.Store` now has `Export`, by the same selection as its `Erase`, in the memory and Postgres trace stores and the log files. It is an operator action like erase: `kind: retention` audit rows before (subject `export:USER`, the reason) and after, the second holding the count per store and `sha256=` the SHA-256 of the records, never the text. A store that fails fails the whole export, so a partial copy is never handed over. Export, erase, export gives an empty second export (`TestLAW07_SubjectExportIsWhatEraseRemoves`, the Postgres integration test, and the API test).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

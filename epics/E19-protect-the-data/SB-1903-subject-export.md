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
| Status | Open |

## Why

A person can be erased, but a data-protection officer can't give them a copy of what the cell keeps about them.

## Acceptance criteria

- [ ] `POST /v1/admin/subjects/export` returns, for one person, every record erase would remove: saved and shared questions, steward questions and replies, and feedback
- [ ] It is an operator action like erase: policy-checked and audited before it runs; the audit row holds the hash, never the text
- [ ] Export, then erase, then export gives an empty export

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Core export over the erase stores
- [ ] Endpoint and tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-401: Append-only, hash-chained audit log written before execution"
summary: "If the audit row isn't there first, the action didn't happen through us."
---

# SB-401: Append-only, hash-chained audit log written before execution

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.1 Audit and kill |
| Stories | US-05.4 |
| Manual | [`sb-audit(7)`](../../manual/sb-audit.7.md), [`sb-audit(5)`](../../manual/sb-audit.5.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-11](../../manual/sb-laws.7.md#law-11) |
| Depends on | [SB-108](../E01-tenancy-identity/SB-108-tenant-scoped-request-context.md) |
| Status | Open |

## Why

If the audit row isn't there first, the action didn't happen through us.

## Acceptance criteria

- [ ] Audit row committed before any tool call executes (LAW-04 replay test)
- [ ] Each row includes the previous row hash; `UPDATE` and `DELETE` are revoked at the database role
- [ ] Tamper test detects any edit
- [ ] Hourly anchors are signed with a per-cell key, not a factory-wide key; `kind: custody` is a valid audit kind

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Audit table and role grants
- [ ] Chain hashing
- [ ] Write-ahead hook in tool bus
- [ ] Replay and tamper tests
- [ ] Write or extend the law tests: LAW-04, LAW-11
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

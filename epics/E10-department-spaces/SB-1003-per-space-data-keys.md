---
title: "SB-1003: Per-space data keys wrapped by the cell root"
summary: "A space must be removable on its own, including its key."
---

# SB-1003: Per-space data keys wrapped by the cell root

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.1 Space model |
| Stories | US-10.1 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-1002](SB-1002-space-registry-and-space-scoped-request-context.md) |
| Status | Open |

## Why

A space must be removable on its own, including its key.

## Acceptance criteria

- [ ] Each space's chunks and traces are encrypted under its own data key, wrapped by the cell root key
- [ ] Deleting a space destroys its key; its content is unreadable and the rest of the cell is untouched
- [ ] Hand-off re-wraps space keys with the root (LAW-18 still holds)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Keys port extension
- [ ] Encryption at rest per space
- [ ] Crypto-shred test
- [ ] Hand-off test update
- [ ] Write or extend the law tests: LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

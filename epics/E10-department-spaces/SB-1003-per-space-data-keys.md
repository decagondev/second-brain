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
| Status | In review |

## Why

A space must be removable on its own, including its key.

## Acceptance criteria

- [ ] Each space's chunks and traces are encrypted under its own data key, wrapped by the cell root key (`seal.Sealer` is built and tested; the index and trace stores call it once SB-1005 gives each chunk its spaces)
- [x] Deleting a space destroys its key; its content is unreadable and the rest of the cell is untouched
- [x] Hand-off re-wraps space keys with the root (LAW-18 still holds)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Keys port extension
- [x] Encryption at rest per space
- [x] Crypto-shred test
- [x] Hand-off test update
- [x] Write or extend the law tests: LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.1c): sb-tenancy(7) gained "Per-space data keys", and sb-factory(7)'s key wraps name the `space/spc_NAME` purposes and the one exception to never removing a wrap. `keywrap.PurposeSpace`, `keywrap.Shred` (refuses any purpose that isn't a space's, and removes every custodian's wrap of one that is) and `keywrap.DataKey` (unwraps with the first custodian whose wrap works now; a key nobody holds is "not found", a key that doesn't unwrap is an error). `seal.Sealer` seals with AES-256-GCM under a space's key, with a fresh nonce and `TENANT/spc_NAME` as additional data; a record from another space or cell, or tampered, is `ErrOpen`, and a shredded space is `ErrShredded`. `spaces.Sync` takes the cell's wraps: a new space gets its key before its row exists, and a dropped space's key is shredded after its row is gone. Hand-off needs nothing new, because re-wrap covers every purpose. The new law test `TestLAW18_SpaceKeysShredAloneAndSurviveHandoff` adds and drops spaces, seals records, and hands off with a real local key store; the audit key and every live space's key always unwrap, live records always open, dropped records never do, and two mutations (a sync that never shreds, a shred of any purpose) are caught.

**Left:** the index and trace stores sealing each chunk and trace under its space's key, when SB-1005 routes documents to spaces.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

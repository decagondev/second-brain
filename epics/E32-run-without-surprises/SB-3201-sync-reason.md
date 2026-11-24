---
title: "SB-3201: Why a sync fails"
summary: "Health says a sync failed, but not why."
---

# SB-3201: Why a sync fails

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.1 |
| Stories | US-32.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-03 |
| Depends on | — |
| Status | Open |

## Why

Health says a sync failed, but not why.

## Acceptance criteria

- [ ] A sync record keeps its last failure's class: auth, rate_limited, unreachable or source_error, never a URL, body or token
- [ ] Health shows it with the runbook section that fixes it
- [ ] A fake 401 shows auth; an unclassified failure is source_error

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Sync record
- [ ] Health
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-1303: Grant revocation and expiry"
summary: "A company must be able to take consent back, at once."
---

# SB-1303: Grant revocation and expiry

| Field | Value |
|---|---|
| Epic | [E13 Portfolio Department Grants](README.md) |
| Wave / slice | W13 / S13.1 Grants |
| Stories | US-13.2 |
| Manual | `sb-spaces(7)` (new), [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md) |
| Laws | [LAW-10](../../manual/sb-laws.7.md#law-10), LAW-23 (new, W8) |
| Depends on | [SB-1302](SB-1302-aggregate-only-grant-delivery.md) |
| Status | Parked (Q-01 stays no, 2026-11-20) |

## Why

A company must be able to take consent back, at once.

## Acceptance criteria

- [ ] Revoke stops deliveries within 5 minutes; expiry stops them on time
- [ ] The company owner sees every delivery a grant ever made

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Revoke
- [ ] Expiry
- [ ] History view
- [ ] Write or extend the law tests: LAW-10, LAW-23
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

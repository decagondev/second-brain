---
title: "SB-1202: Quarterly access reviews"
summary: "SOX and SOC 2 expect owners to re-certify who has access; the brain should make that a ten-minute job."
---

# SB-1202: Quarterly access reviews

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.1 Stewards |
| Stories | US-12.1 |
| Manual | `sb-spaces(7)` (new), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-1201](SB-1201-stewards-and-delegated-space-changes.md) |
| Status | Open |

## Why

SOX and SOC 2 expect owners to re-certify who has access; the brain should make that a ten-minute job.

## Acceptance criteria

- [ ] Each quarter a steward gets the member list of each restricted space with last-use dates
- [ ] Certify or remove per member; removals become a manifest or IdP change request
- [ ] An uncertified review after 14 days alerts the tenant operator; the evidence export is audit-ready

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Review job
- [ ] Console view
- [ ] Evidence export
- [ ] Write or extend the law tests: LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

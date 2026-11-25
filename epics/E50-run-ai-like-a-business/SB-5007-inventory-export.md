---
title: "SB-5007: Inventory export"
summary: "An auditor asking what AI runs in the company gets a tour of several pages."
---

# SB-5007: Inventory export

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.2 |
| Stories | US-50.2 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-07, LAW-11, LAW-12 |
| Depends on | SB-4707, SB-4004 |
| Status | Open |

## Why

An auditor asking what AI runs in the company gets a tour of several pages.

## Acceptance criteria

- [ ] `sb inventory export` and **Download inventory** write every agent, skill, rhythm, shared routine, trigger and starter question with owner, spaces, cap, targets, last run and last change, as CSV and signed JSON
- [ ] The export matches the manifest and the registry, and the JSON signature verifies
- [ ] A personal routine is never named, only counted per space

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Export
- [ ] CLI and registry button
- [ ] Signature
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

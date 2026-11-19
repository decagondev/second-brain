---
title: "SB-508: One-step rollback"
summary: "Rollback is one command and one audit row."
---

# SB-508: One-step rollback

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.3 Rings |
| Stories | US-06.3 |
| Manual | [`sb-release(1)`](../../manual/sb-release.1.md) |
| Laws | [LAW-08](../../manual/sb-laws.7.md#law-08) |
| Depends on | [SB-507](../E06-evals-release/SB-507-release-controller-with-channels-and-rings.md) |
| Status | Open |

## Why

Rollback is one command and one audit row.

## Acceptance criteria

- [ ] `sb release rollback` restores the previous stable in under 10 minutes (drill)
- [ ] Rolled-back release is marked revoked

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Rollback command
- [ ] Drill
- [ ] Release notes entry
- [ ] Write or extend the law tests: LAW-08
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

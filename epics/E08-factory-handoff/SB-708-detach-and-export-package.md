---
title: "SB-708: Detach and export package"
summary: "A divestiture must not strand a company's knowledge."
---

# SB-708: Detach and export package

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.3 Exit paths |
| Stories | US-08.5 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-audit(5)`](../../manual/sb-audit.5.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-705](../E08-factory-handoff/SB-705-hand-off-workflow-with-dual-approval.md) |
| Status | Open |

## Why

A divestiture must not strand a company's knowledge.

## Acceptance criteria

- [ ] Export contains manifest, policies, evals, signed audit export with anchor key, bundle and SBOM, runbooks
- [ ] After detach, the factory holds zero access paths and keeps only custody history

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Export builder
- [ ] Offline verification of the export
- [ ] Registry tombstone
- [ ] Write or extend the law tests: LAW-11, LAW-17
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

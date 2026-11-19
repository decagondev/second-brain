---
title: "SB-507: Release controller with channels and rings"
summary: "Rings limit blast radius; the promote guard keeps everyone at most one behind."
---

# SB-507: Release controller with channels and rings

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.3 Rings |
| Stories | US-06.3, US-06.4 |
| Manual | [`sb-release(1)`](../../manual/sb-release.1.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-08](../../manual/sb-laws.7.md#law-08) |
| Depends on | [SB-502](../E06-evals-release/SB-502-eval-gate-blocks-regressions.md), [SB-506](../E06-evals-release/SB-506-deploy-time-signature-verification.md) |
| Status | Open |

## Why

Rings limit blast radius; the promote guard keeps everyone at most one behind.

## Acceptance criteria

- [ ] Rings: internal, friendly, all
- [ ] Promotion to stable only when every stable tenant runs current stable (LAW-08)
- [ ] Behaviour matches `Rollout.tla`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Controller
- [ ] Promote guard
- [ ] Ring config
- [ ] LAW-08 test
- [ ] Write or extend the law tests: LAW-06, LAW-08
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

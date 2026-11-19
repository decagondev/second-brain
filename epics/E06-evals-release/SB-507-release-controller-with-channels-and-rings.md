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
| Status | In review |

## Why

Rings limit blast radius; the promote guard keeps everyone at most one behind.

## Acceptance criteria

- [x] Rings: internal, friendly, all
- [x] Promotion to stable only when every stable tenant runs current stable (LAW-08)
- [x] Behaviour matches `Rollout.tla`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Controller
- [x] Promote guard
- [x] Ring config
- [x] LAW-08 test
- [x] Write or extend the law tests: LAW-06, LAW-08
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.3a, with SB-508): `internal/core/rollout` holds exactly the state and moves of `Rollout.tla`: register (only admitted releases are deployable), deploy to testing (ring 0 internal, ring 1 friendly), promote with the 48-hour soak on both rings and the promote guard, deploy to stable (ring 2), rollback with draining, redeploy, and revoke-on-testing. After every move it checks the model's six invariants and refuses any move that would break one. `sb release status` and `sb release promote REL --to stable` work against a local control plane (`DIR/releases/state.json`) and write `kind: release` audit rows first. The smoke test shows the promote guard refusing with the manual's message. `TestLAW08_AtMostOneBehindStable` went red first, then green over random builds, deploys, promotions, rollbacks and revocations, and checks LAW-06 at every step. A controller that promotes past the guard is caught.

**Left:** driving real cells in real rings (W7), where deploys are cell pulls.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

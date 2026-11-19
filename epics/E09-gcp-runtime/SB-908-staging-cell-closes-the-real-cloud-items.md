---
title: "SB-908: A staging cell closes the W1 to W7 items waiting for a real cloud"
summary: "Many tickets are In review only for a staging run: p95 latency, evals on a real model, hand-off and detach against a running cell."
---

# SB-908: A staging cell closes the W1 to W7 items waiting for a real cloud

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.4 Staging |
| Stories | US-09.4 |
| Manual | [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md), [`sb-release(7)`](../../manual/sb-release.7.md), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-08](../../manual/sb-laws.7.md#law-08), [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-903](SB-903-sb-tenant-apply-in-a-real-gcp-project.md), [SB-904](SB-904-schedules-in-the-cell.md), [SB-905](SB-905-metrics-and-the-verifier-endpoint.md) |
| Status | Open |

## Why

Many tickets are In review only for a staging run: p95 latency, evals on a real model, hand-off and detach against a running cell.

## Acceptance criteria

- [ ] Staging cell for `acme-logistics` runs the release train on `testing` then `stable`
- [ ] Ask p95 measured with minimum instances set (SB-303); staging evals run (SB-309, SB-501)
- [ ] `sb factory handoff` and `detach` run against the staging cell (SB-705, SB-708)
- [ ] Every ticket closed by this run is updated

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Staging cell
- [ ] Latency run
- [ ] Eval run
- [ ] Hand-off and detach rehearsal
- [ ] Ticket updates
- [ ] Write or extend the law tests: LAW-06, LAW-08, LAW-17, LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

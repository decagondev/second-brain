---
title: "SB-1508: A whole cell running locally, end to end"
summary: "Before a real cloud project, the whole cell should run on one machine and pass its own verification."
---

# SB-1508: A whole cell running locally, end to end

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.4 |
| Stories | US-15.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | LAW-01 to LAW-22 |
| Depends on | [SB-1507](SB-1507-cli-against-running-cells.md) |
| Status | In review |

## Why

Before a real cloud project, the whole cell should run on one machine and pass its own verification.

## Acceptance criteria

- [x] `make cell` starts api, ingest and jobs against local Postgres with an example tenant
- [x] An end-to-end test asks, drafts, proposes and approves, places a hold, and runs `sb tenant verify --cell` green
- [ ] The same wiring deploys to GCP for SB-903 and SB-908 once a project exists

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] make cell
- [x] End-to-end test
- [ ] GCP path
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.4): every store that was in memory has a Postgres adapter (`internal/adapters/postgres/cellstore`: the directory, sessions kept by a hash of their id, traces, conversations and feedback with retention that honours holds, actions with locked state changes, receipts, proposals, reviews, holds and the chat inbox as a gap-free queue), tested against local Postgres, races included (`make integration`). With `database` in sb-cell(5) the cell keeps all of them, the index, the spaces registry and the audit log in Postgres, and `durable: true` is allowed; the services take turns creating tables. The verifier's checks now run against the cell itself (`wire.Cell.VerifyTarget`), with LAW-03's timeout injected into a copy of the agent only. `TestWholeCellOnPostgres` ingests through a Drive fake, asks, drafts, proposes an action that the requester can't approve and an IT lead can, which reaches the Linear fake, refuses a hold from a non-steward and places one from a legal steward, passes `sb tenant verify --cell`, and finds everything again after a restart. `make cell` starts a development IdP, `sb-api`, `sb-ingest` and `sb-jobs` on local Postgres (run by hand: sign-in, SCIM and an ask work; sync reports its missing Drive credential). The test found BUG-0025. In review: the third criterion, the GCP path, waits for a real project (SB-903, SB-908).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

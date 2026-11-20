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
| Status | Open |

## Why

Before a real cloud project, the whole cell should run on one machine and pass its own verification.

## Acceptance criteria

- [ ] `make cell` starts api, ingest and jobs against local Postgres with an example tenant
- [ ] An end-to-end test asks, drafts, proposes and approves, places a hold, and runs `sb tenant verify --cell` green
- [ ] The same wiring deploys to GCP for SB-903 and SB-908 once a project exists

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] make cell
- [ ] End-to-end test
- [ ] GCP path
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

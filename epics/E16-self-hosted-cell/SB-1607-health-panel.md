---
title: "SB-1607: Cell health in the console"
summary: "Operators should see missed jobs and stale connectors without reading metrics."
---

# SB-1607: Cell health in the console

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.4 |
| Stories | US-16.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-13, LAW-16 |
| Depends on | — |
| Status | Open |

## Why

Operators should see missed jobs and stale connectors without reading metrics.

## Acceptance criteria

- [ ] `GET /v1/admin/health` lists each job's last finish and overdue state and each connector's last success and failures, from the cell's own records
- [ ] The console shows it first, overdue and stale items at the top
- [ ] UI and API tests

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Endpoint
- [ ] Console panel
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

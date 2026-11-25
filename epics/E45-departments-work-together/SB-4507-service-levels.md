---
title: "SB-4507: Department service levels"
summary: "Nobody can see whether a department answers its requests on time."
---

# SB-4507: Department service levels

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.2 |
| Stories | US-45.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-07, LAW-24 |
| Depends on | SB-4104 |
| Status | Open |

## Why

Nobody can see whether a department answers its requests on time.

## Acceptance criteria

- [ ] `GET /v1/spaces/NAME/service` and a home card show first reply, time to close and late share by category and week for 12 weeks
- [ ] A parent sees its children's as counts; rows under 5 show as "fewer than 5"
- [ ] The numbers match the requests' audit trail

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Counts
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

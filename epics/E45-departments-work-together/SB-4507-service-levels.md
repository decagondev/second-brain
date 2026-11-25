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
| Status | Done |

## Why

Nobody can see whether a department answers its requests on time.

## Acceptance criteria

- [x] `GET /v1/spaces/NAME/service` and a home card show first reply, time to close and late share by category and week for 12 weeks
- [x] A parent sees its children's as counts; rows under 5 show as "fewer than 5"
- [x] The numbers match the requests' audit trail

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Counts
- [x] API
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `GET /v1/spaces/NAME/service` (`Stewards.Service`), stewards only: 12 ISO weeks by category (a request's, or `question`), with the median hours to the first claim or reply (`StewardQuestion.FirstClaim`, set on the first claim), the median to the reply, and the share replied to after `due` or after 5 working days; rows under 5 masked; each team counts only, masked the same way. The leader's week adds each team's waits and late count. Home card **Service** for stewards. TestServiceRows and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

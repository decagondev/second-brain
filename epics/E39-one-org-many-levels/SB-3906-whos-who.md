---
title: "SB-3906: Who's who in a branch"
summary: "Nobody can see who looks after a space or who to ask when a team has nobody."
---

# SB-3906: Who's who in a branch

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.2 |
| Stories | US-39.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-12 |
| Depends on | SB-3405 |
| Status | Open |

## Why

Nobody can see who looks after a space or who to ask when a team has nobody.

## Acceptance criteria

- [ ] `GET /v1/spaces/NAME/people` lists stewards, caretakers and owners of its ten most-cited documents this month, checked live
- [ ] Members are counted, not listed; owners appear only for documents the asker can open
- [ ] Who to ask climbs to the department's stewards when no cited owner is active

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

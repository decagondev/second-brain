---
title: "SB-4103: Policies to read"
summary: "A policy everyone must read has no way to ask people to read it or see who has."
---

# SB-4103: Policies to read

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.1 |
| Stories | US-41.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-07, LAW-11 |
| Depends on | SB-3901 |
| Status | Open |

## Why

A policy everyone must read has no way to ask people to read it or see who has.

## Acceptance criteria

- [ ] `POST /v1/spaces/NAME/policies` marks a document as a policy for the branch; members who can open it are told once (`policy`)
- [ ] I've read it is recorded per person and version and audited; a new version asks again
- [ ] Stewards see who hasn't read it by name; members who can't open it are counted

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API and records
- [ ] Notify
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

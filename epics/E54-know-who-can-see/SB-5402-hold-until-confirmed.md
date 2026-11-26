---
title: "SB-5402: Hold until the owner confirms"
summary: "Listing an overshared document doesn't stop the brain citing it to everyone meanwhile."
---

# SB-5402: Hold until the owner confirms

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.1 |
| Stories | US-54.1 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-11, LAW-19 |
| Depends on | SB-5401 |
| Status | Open |

## Why

Listing an overshared document doesn't stop the brain citing it to everyone meanwhile.

## Acceptance criteria

- [ ] A listed document is left out of answers, briefs, routines and MCP results for everyone but its owner and its space's stewards
- [ ] The owner narrows it at the source (the next sync clears the hold) or confirms **Shared on purpose** with a reason, audited, lasting until the sharing or the document changes
- [ ] sb-space(5) `wide_sensitive: warn` shows a warning instead of holding; the default is `hold`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Hold in retrieval as a narrowing filter
- [ ] Confirm and release, audited
- [ ] sb-space(5) `wide_sensitive`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

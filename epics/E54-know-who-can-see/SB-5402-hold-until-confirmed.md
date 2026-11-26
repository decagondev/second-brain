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
| Status | Done |

## Why

Listing an overshared document doesn't stop the brain citing it to everyone meanwhile.

## Acceptance criteria

- [x] A listed document is left out of answers, briefs, routines and MCP results for everyone but its owner and its space's stewards
- [x] The owner narrows it at the source (the next sync clears the hold) or confirms **Shared on purpose** with a reason, audited, lasting until the sharing or the document changes
- [x] sb-space(5) `wide_sensitive: warn` shows a warning instead of holding; the default is `hold`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Hold in retrieval as a narrowing filter
- [x] Confirm and release, audited
- [x] sb-space(5) `wide_sensitive`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `Agent.dropHeld` runs after the retired-document filter in every retrieval (asks, searches, drafts, briefs, routines and MCP all go through it), keeping a held document only for its owner and the stewards of its spaces; `Response.Held` and `wide_warned` count, never name. `Stewards.ConfirmWide` at `POST /v1/documents/sharing/confirm` is the owner's **Shared on purpose**, audited (`shared_on_purpose`), keyed to the version and the ACL copy's hash, so an edit or a sharing change holds it again; an unreadable confirmation releases nothing. sb-space(5) `wide_sensitive: warn` (inherited by teams) uses it with a warning. The web inbox has **Shared on purpose** on a `wide` item.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

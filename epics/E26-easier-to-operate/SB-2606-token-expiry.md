---
title: "SB-2606: Token expiry warning"
summary: "API tokens expire without warning and scripts break."
---

# SB-2606: Token expiry warning

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.2 |
| Stories | US-26.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-07, LAW-12 |
| Depends on | — |
| Status | Done |

## Why

API tokens expire without warning and scripts break.

## Acceptance criteria

- [x] The notify job tells a token's owner once, 7 days before it expires, as a new kind `tokens` they can turn off; the Tokens list shows days left
- [x] The token's name only, never the token; never for a revoked token or a deprovisioned owner
- [x] Told once (fake clock)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify kind
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S25.2; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

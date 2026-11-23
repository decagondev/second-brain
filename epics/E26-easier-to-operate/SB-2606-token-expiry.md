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
| Status | Open |

## Why

API tokens expire without warning and scripts break.

## Acceptance criteria

- [ ] The notify job tells a token's owner once, 7 days before it expires, as a new kind `tokens` they can turn off; the Tokens list shows days left
- [ ] The token's name only, never the token; never for a revoked token or a deprovisioned owner
- [ ] Told once (fake clock)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notify kind
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

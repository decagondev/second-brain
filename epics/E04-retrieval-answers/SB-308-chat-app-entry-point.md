---
title: "SB-308: Chat-app entry point"
summary: "Meet people where they already ask questions."
---

# SB-308: Chat-app entry point

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.4 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md) |
| Status | Open |

## Why

Meet people where they already ask questions.

## Acceptance criteria

- [ ] Ask from the company chat app; replies are visible only to the asker unless they share
- [ ] Same API and laws as web

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Bot adapter
- [ ] Ephemeral replies
- [ ] Identity mapping chat user to IdP user
- [ ] Write or extend the law tests: LAW-01
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

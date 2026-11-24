---
title: "SB-3003: Answers in the asker's language"
summary: "The UI speaks German and French, but answers come in whatever language the model writes."
---

# SB-3003: Answers in the asker's language

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.1 |
| Stories | US-30.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-14, LAW-07 |
| Depends on | — |
| Status | Open |

## Why

The UI speaks German and French, but answers come in whatever language the model writes.

## Acceptance criteria

- [ ] `POST /v1/ask` takes `language` (en, de or fr); the web UI sends the chosen one and chat apps send the manifest's
- [ ] The prompt asks for claims in that language and keeps cites, titles and passages as they are
- [ ] A German ask gets German claims with the same cites; an unknown code changes nothing; a German and a French eval case pass

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ask
- [ ] UI and chat apps
- [ ] Eval cases
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

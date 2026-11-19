---
title: "SB-304: Answer composer with `doc@version` citations and `uncited` in code"
summary: "Citations are structure, not prose; the model proposes, code disposes."
---

# SB-304: Answer composer with `doc@version` citations and `uncited` in code

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.1, US-04.2 |
| Manual | [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-302](../E04-retrieval-answers/SB-302-live-authorisation-of-every-chunk-before.md) |
| Status | Open |

## Why

Citations are structure, not prose; the model proposes, code disposes.

## Acceptance criteria

- [ ] Every claim span maps to an allowed chunk citation or is labelled `uncited` by code
- [ ] Citations that don't match an allowed chunk are stripped and logged

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Claim-span schema
- [ ] Citation validator
- [ ] `uncited` labeller
- [ ] LAW-14 property test
- [ ] Write or extend the law tests: LAW-01, LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

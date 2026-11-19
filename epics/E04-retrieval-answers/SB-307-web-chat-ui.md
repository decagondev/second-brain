---
title: "SB-307: Web chat UI"
summary: "The default entry point for staff."
---

# SB-307: Web chat UI

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.4 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md) |
| Status | Open |

## Why

The default entry point for staff.

## Acceptance criteria

- [ ] Citations open the source at the cited version
- [ ] `uncited` claims are visually distinct
- [ ] Meets WCAG 2.2 AA

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI shell
- [ ] Citation chips
- [ ] Accessibility checks
- [ ] Write or extend the law tests: LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

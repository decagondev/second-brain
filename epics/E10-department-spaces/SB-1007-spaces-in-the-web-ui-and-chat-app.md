---
title: "SB-1007: Spaces in the web UI and the chat app"
summary: "People need to see which brain they're asking and where each citation came from."
---

# SB-1007: Spaces in the web UI and the chat app

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.3 Ask in a space |
| Stories | US-10.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-14](../../manual/sb-laws.7.md#law-14), LAW-20 (new, W8) |
| Depends on | [SB-1006](SB-1006-ask-in-a-space-or-company-wide.md), [SB-307](../E04-retrieval-answers/SB-307-web-chat-ui.md), [SB-308](../E04-retrieval-answers/SB-308-chat-app-entry-point.md) |
| Status | Open |

## Why

People need to see which brain they're asking and where each citation came from.

## Acceptance criteria

- [ ] A space switcher listing only the asker's spaces; each citation shows its space and label
- [ ] In the chat app a channel maps to a space in the manifest; an unmapped channel asks company-wide
- [ ] Accessible (axe passes)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI switcher
- [ ] Citation badges
- [ ] Channel mapping
- [ ] Tests
- [ ] Write or extend the law tests: LAW-14, LAW-20
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

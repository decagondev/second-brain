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
| Status | In review |

## Why

People need to see which brain they're asking and where each citation came from.

## Acceptance criteria

- [x] A space switcher listing only the asker's spaces; each citation shows its space and label
- [x] In the chat app a channel maps to a space in the manifest; an unmapped channel asks company-wide
- [x] Accessible (axe passes; the human WCAG audit of SB-307 covers the switcher too)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI switcher
- [x] Citation badges
- [x] Channel mapping
- [x] Tests
- [x] Write or extend the law tests: LAW-14, LAW-20
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.3b): sb-ask(3) gained `GET /v1/spaces`, `cites[].spaces` and `cites[].label`, the switcher and the place on each chip, and the chat app's channel step; sb-space(5) gained `channels` and rule 10. Every citation carries the names of the spaces holding its document and its label. `api.Spaces` lists only the asker's spaces, never another's or another tenant's, and fails closed. A space's `channels` (`#name`, each in one space only) are parsed, planned and round-tripped. The Slack app asks a mapped channel in its space for members, gives a non-member no answer (never a company-wide one), and asks unmapped channels and DMs company-wide; a registry that can't answer gives no answer. The web UI shows "Ask in" with "Whole company" and the asker's spaces only, sends `?space=`, starts a new conversation on a switch, shows no switcher when there are no spaces or the list fails, and names each source's place ("Finance · restricted") in its chip and accessible name. Web tests: 22, including axe on the switcher and the LAW-14 property with places on chips. The new law test `TestLAW20_ChatChannelsAskOnlyInTheirSpace` drives the real Slack app; a registry read that loses its answer is caught.

**Left:** the human WCAG audit (SB-307's) and a run in a real Slack workspace (SB-308's).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

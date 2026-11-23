---
title: "SB-2307: Notifications by email"
summary: "Sites with neither Slack nor Mattermost hear nothing."
---

# SB-2307: Notifications by email

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.3 |
| Stories | US-23.4 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07 |
| Depends on | SB-2101 |
| Status | Open |

## Why

Sites with neither Slack nor Mattermost hear nothing.

## Acceptance criteria

- [ ] An SMTP adapter (STARTTLS required, credentials in the vault) sends the W20 messages to sites with no chat
- [ ] Plain text with the same content rule: links and counts, no question, answer or document text
- [ ] A fake SMTP server receives each W20 message; one with document text fails a test

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] SMTP adapter
- [ ] Wiring
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

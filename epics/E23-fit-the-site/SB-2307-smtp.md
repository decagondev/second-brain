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
| Status | Done |

## Why

Sites with neither Slack nor Mattermost hear nothing.

## Acceptance criteria

- [x] An SMTP adapter (STARTTLS required, credentials in the vault) sends the W20 messages to sites with no chat
- [x] Plain text with the same content rule: links and counts, no question, answer or document text
- [x] A fake SMTP server receives each W20 message; one with document text fails a test

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] SMTP adapter
- [x] Wiring
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S22.3). `internal/adapters/smtpmail` sends each notification as one plain-text email through sb-cell(5) `email.server` from `email.from`, on a cell with neither Mattermost nor Slack. It always says STARTTLS, refuses a server without it, verifies the certificate, and signs in with `vault://tenants/TENANT/email` when that secret exists. The subject names the kind only and the body is the chat message's text and link and a fixed line on turning the kind off; an address the server refuses (550), or one that isn't one bare address, is skipped. Tests: `smtpmail_test.go` against `smtpmail.Fake` (each W20 kind, no STARTTLS, an untrusted certificate, sign-in, header injection, and a message with document text failing the content rule); `TestLAW07_EmailNotificationsSayNothingOfWhatWasAsked` runs the real notify job through the adapter, with a self-test that a previewed question is caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

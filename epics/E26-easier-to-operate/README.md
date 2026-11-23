---
title: "E26 Easier to Operate"
summary: "Whoever runs the box sets everything from one file, can check each channel before people depend on it, and is told how to fix what is wrong."
---

# E26: Easier to Operate

> Whoever runs the box sets everything from one file, can check each channel before people depend on it, and is told how to fix what is wrong.

**Waves:** W25 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-23 after W22 and accepted by the user the same day, with every recommended decision. The concept is in [12-ask-steward-operate.md](../../docs/12-ask-steward-operate.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-26.1**: As an *IT administrator*, I want to set email in the installer and check notifications, the chat bot and the config before people depend on them, so that nothing fails silently. Delivered by SB-2601, SB-2602, SB-2603, SB-2604.
- **US-26.2**: As an *operator* or *integrator*, I want the doctor to say how to fix things, tokens to warn before they expire, and notifications in health, so that I fix problems before people notice them. Delivered by SB-2605, SB-2606, SB-2607.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S25.1 | Set it up once | Email in the installer, send me a test, check the chat bot, check the config | SB-2601, SB-2602, SB-2603, SB-2604 |
| S25.2 | Know it's working | Fix hints, token expiry warning, notifications in health | SB-2605, SB-2606, SB-2607 |

## Exit gate

- [ ] The email login never leaves the vault
- [ ] Each channel's fake receives exactly the test text
- [ ] A missing scope or permission is named
- [ ] A config typo or missing secret stops an update before any restart
- [ ] Every non-OK doctor verdict names a fix
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2601](SB-2601-install-email.md) | W25 / S25.1 | Email in the installer | LAW-07 |
| [SB-2602](SB-2602-test-notification.md) | W25 / S25.1 | Send me a test | LAW-07 |
| [SB-2603](SB-2603-check-chat.md) | W25 / S25.1 | Check the chat bot | LAW-07 |
| [SB-2604](SB-2604-check-config.md) | W25 / S25.1 | Check the config first | — |
| [SB-2605](SB-2605-doctor-fix.md) | W25 / S25.2 | Doctor says how to fix it | — |
| [SB-2606](SB-2606-token-expiry.md) | W25 / S25.2 | Token expiry warning | LAW-07, LAW-12 |
| [SB-2607](SB-2607-notify-health.md) | W25 / S25.2 | Notifications in health | LAW-07 |

## Decisions for the user

Decided 2026-11-23 ([12-ask-steward-operate.md](../../docs/12-ask-steward-operate.md#decisions-for-the-user)): all of W23 to W25, in order; recent questions kept in the person's browser until sign-out (SB-2405); a soft one-hour claim that never blocks a reply (SB-2502); a pause of up to 30 days, with new questions still reaching the other stewards (SB-2506).

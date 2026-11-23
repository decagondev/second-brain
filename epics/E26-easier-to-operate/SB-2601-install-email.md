---
title: "SB-2601: Email in the installer"
summary: "Email notifications are set by editing the cell's config by hand."
---

# SB-2601: Email in the installer

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.1 |
| Stories | US-26.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Open |

## Why

Email notifications are set by editing the cell's config by hand.

## Acceptance criteria

- [ ] `SB_EMAIL_SERVER`, `SB_EMAIL_FROM` and an optional root-only `SB_EMAIL_LOGIN_FILE` in `install.env`, written to sb-cell(5) `email` and the vault
- [ ] `--doctor` checks the server answers and offers STARTTLS
- [ ] The install test writes the config; `sb secrets scan` finds no password outside the vault

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] install.sh and selfhost.sh
- [ ] Doctor
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

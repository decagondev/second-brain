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
| Status | In review |

## Why

Email notifications are set by editing the cell's config by hand.

## Acceptance criteria

- [x] `SB_EMAIL_SERVER`, `SB_EMAIL_FROM` and an optional root-only `SB_EMAIL_LOGIN_FILE` in `install.env`, written to sb-cell(5) `email` and the vault
- [x] `--doctor` checks the server answers and offers STARTTLS
- [x] The install test writes the config; `sb secrets scan` finds no password outside the vault

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] install.sh and selfhost.sh
- [x] Doctor
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S25.1: SB_EMAIL_* in install.env, sb selfhost check-email, the login in the api and jobs vaults. The install itself waits for a run on a real VM, like SB-1605

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

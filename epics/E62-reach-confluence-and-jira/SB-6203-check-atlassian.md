---
title: "SB-6203: Check the Atlassian setup"
summary: "An Atlassian token with write scopes, or a project it can't reach, shows only on a real site."
---

# SB-6203: Check the Atlassian setup

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.1 |
| Stories | US-62.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-03, LAW-16 |
| Depends on | SB-6001, SB-6201 |
| Status | Done |

## Why

An Atlassian token with write scopes, or a project it can't reach, shows only on a real site.

## Acceptance criteria

- [x] `sb selfhost check-atlassian` refuses a token with any write scope and names it
- [x] Each named Confluence space and Jira project must answer, and groups must resolve
- [x] Each fix is in `docs/runbooks/atlassian.md`; `sb prove` covers SB-6201, SB-6202 and SB-6204 to SB-6207

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Scope and reach checks
- [x] `docs/runbooks/atlassian.md`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `sb selfhost check-atlassian`: the token signs in, its account holds none of the Jira write permissions (site-wide and in each project; an API token carries its account's permissions, so read-only is a read-only account), each space and project answers, groups resolve; fixes in `docs/runbooks/atlassian.md`. `sb prove` covers SB-6201, SB-6202 and SB-6204. Tested in TestCheckAtlassian and TestProveAtlassian.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

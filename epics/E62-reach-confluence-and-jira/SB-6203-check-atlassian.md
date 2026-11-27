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
| Status | Open |

## Why

An Atlassian token with write scopes, or a project it can't reach, shows only on a real site.

## Acceptance criteria

- [ ] `sb selfhost check-atlassian` refuses a token with any write scope and names it
- [ ] Each named Confluence space and Jira project must answer, and groups must resolve
- [ ] Each fix is in `docs/runbooks/atlassian.md`; `sb prove` covers SB-6201, SB-6202 and SB-6204 to SB-6207

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Scope and reach checks
- [ ] `docs/runbooks/atlassian.md`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-6201: Confluence"
summary: "Many teams keep their knowledge in Confluence, and the brain can't read it."
---

# SB-6201: Confluence

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.1 |
| Stories | US-62.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-6202, SB-3002, SB-5401 |
| Status | In review |

## Why

Many teams keep their knowledge in Confluence, and the brain can't read it.

## Acceptance criteria

- [x] A Confluence Cloud connector reads the spaces sb-cell(5) names: pages, blog posts and attachments from a stored cursor, labels as tags, the page tree as sections
- [x] Page restrictions and space permissions resolve to Atlassian groups; a space open to anonymous users counts as wide
- [x] Built against a recorded fake with the standard library; a deleted page is gone at the next sync; a real site run stays In review until `sb prove`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Confluence adapter and recorded fake
- [x] Restrictions to ACLs
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `docs: confluence` (`internal/adapters/atlassian`: `Client`, `Confluence`), standard library only (ADR-0024): spaces' pages and blog posts listed with their versions, storage format read as HTML text with a `Labels:` line, read restrictions as the ACL copy (people and `group:NAME`; none is `confluence:users`, an anonymous space `anyone`, wide), and Confluence's own permission check as the live check. `confluence:users` and `jira:users` join the broad prefilter tokens. Tested in TestConfluence and TestAtlassianThroughARunningCell against the recorded site (`atlassiantest`); `sb prove SB-6201` covers it. In review until a run against a real site.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-5901: SharePoint and OneDrive"
summary: "A company on Microsoft 365 can't connect its documents."
---

# SB-5901: SharePoint and OneDrive

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.1 |
| Stories | US-59.1 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-2003, SB-5205, SB-5401 |
| Status | Open |

## Why

A company on Microsoft 365 can't connect its documents.

## Acceptance criteria

- [ ] A Graph connector for the sites and drives sb-cell(5) names, with delta queries from a stored cursor
- [ ] Live permissions including sharing links (`anonymous` and `organization` count as wide), Office text, pages and slides
- [ ] Built against a recorded Graph fake with the standard library; a real tenant run stays In review

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `connectors/msgraph` documents
- [ ] Recorded Graph fake
- [ ] Permissions and sharing links
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

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
| Status | In review |

## Why

A company on Microsoft 365 can't connect its documents.

## Acceptance criteria

- [x] A Graph connector for the sites and drives sb-cell(5) names, with delta queries from a stored cursor
- [x] Live permissions including sharing links (`anonymous` and `organization` count as wide), Office text, pages and slides
- [ ] Built against a recorded Graph fake with the standard library; a real tenant run stays In review

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `connectors/msgraph` documents
- [x] Recorded Graph fake
- [x] Permissions and sharing links
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built against the recorded Graph; waits for a real tenant. `internal/adapters/msgraph`: one `Client` (client credentials against Entra, the standard library only, `vault://tenants/TENANT/msgraph`, 429 retried) and `SharePoint` (vendor `sharepoint`, kind docs): drive deltas from a stored cursor with `Prefer: deltashowsharingchanges`, deletes as deletes, a sharing change re-read whole; Fetch through `folder.Text` with pages and slides and the review-by property; the ACL copy from `…/permissions` with `anonymous` as `anyone` and `organization` as `domain:` (wide, SB-5401); the live check reads the permissions and the person's nested Entra groups now, 8 at a time within 800 ms, Unknown on anything else. sb-cell(5) `microsoft` (`base`, `login`, `domain`, `drives`). Tested in TestSharePointSync, TestSharePointLiveCheck, TestContractKit (the connector kit), the law tests TestLAW01_SharePointAccessIsLive and TestLAW03_SharePointUnknownIsNeverAllow with TestSelf_SharePointMutationsAreCaught, and TestMicrosoft365ThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-211: Tickets connector (read-only)"
summary: "Tickets hold the how and why of operational work."
---

# SB-211: Tickets connector (read-only)

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.4 Tickets |
| Stories | US-03.5 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | In review |

## Why

Tickets hold the how and why of operational work.

## Acceptance criteria

- [x] Passes the contract kit
- [x] Project permissions map to ACL

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Adapter
- [x] ACL mapping
- [x] Contract kit run
- [x] Write or extend the law tests: LAW-01
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: the user chose Linear. `internal/adapters/connectors/linear` is read-only, standard-library GraphQL with fixed named operations, and passes the contract kit against `lineartest`.
- Each issue (with its comments) is a document, and its version is `updatedAt` in microseconds.
- ACL: private team means members only; public team means full members, and guests only if they're in the team; no account or deactivated means Deny; archived or missing means Unknown. Rate limits are retried.
- Changes are polled. Issues are `updatedAt >= cursor`, with an exact tie-break on ids already seen at the cursor's timestamp; archived means delete. Team privacy and membership changes are permission-only changes for every issue in the team. An empty cursor lists everything.
- `TestLAW03_LinearUnknownIsNeverAllow` (with LAW-01's truth check) runs on it.

The kit found a real bug on the first run: the poll timestamp advanced mid-pagination, so later pages used a different filter and skipped issues. It's fixed. A one-second overlap was tried and rejected, because the kit requires nothing new after the latest cursor; the tie-break is exact instead.

**Left:** a real Linear workspace run; team-membership pages beyond 250 members (a large team's extra members would get a safe false Deny); a test for issues tied at the same timestamp (the fake's clock never ties).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

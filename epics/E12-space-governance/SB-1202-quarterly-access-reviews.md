---
title: "SB-1202: Quarterly access reviews"
summary: "SOX and SOC 2 expect owners to re-certify who has access; the brain should make that a ten-minute job."
---

# SB-1202: Quarterly access reviews

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.1 Stewards |
| Stories | US-12.1 |
| Manual | `sb-spaces(7)` (new), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-1201](SB-1201-stewards-and-delegated-space-changes.md) |
| Status | In review |

## Why

SOX and SOC 2 expect owners to re-certify who has access; the brain should make that a ten-minute job.

## Acceptance criteria

- [x] Each quarter a steward gets the member list of each restricted space with last-use dates
- [x] Certify or remove per member; removals become a manifest or IdP change request
- [x] An uncertified review after 14 days alerts the tenant operator; the evidence export is audit-ready

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Review job
- [ ] Console view (the API is built; the console's screens are left)
- [x] Evidence export
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.1b): sb-spaces(7) gained "Access reviews, exactly", sb-admin(3) four endpoints, sb-runtime(7) the quarterly `review` job and sb-observability(7) the overdue alert; the schema and base policy gained `review_space` for a space's stewards. `internal/core/reviews` opens one review per restricted space per quarter, listing each person in its member groups once with the groups that give the access and their last use from the audit trail, and opens none if a group can't be read; `Decide` refuses a decision on your own access, a removal without a reason, and a second decision; `Requests` turns a complete review's removals into a manifest change for groups in `groups:` and IdP requests for the rest; `Overdue` is open 14 days; `Watch` sets `sb_access_review_overdue{space}` and raises a Sev-2 `review_overdue` alert; `Evidence` is the review with its SHA-256, each decision pointing at its `kind: access` audit row. `api.Reviews` serves it, and the manifest change goes to the operator as a proposal approved by hash (SB-1201), so a review never changes access by itself. The new law test `TestLAW16_AccessReviews` drives the real handler and policies; a review reading only the first member group, an alert a day late, and removals that vanish are caught, and the test fails if no removal ever became a proposal.

**Left:** the console's review screens, and wiring the job and endpoints into a running cell (the cell runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

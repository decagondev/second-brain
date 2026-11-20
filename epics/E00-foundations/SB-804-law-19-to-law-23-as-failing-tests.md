---
title: "SB-804: LAW-19 to LAW-23 as failing tests"
summary: "Space features must be born against their laws, the same way W1 to W7 were."
---

# SB-804: LAW-19 to LAW-23 as failing tests

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.2 Laws red, model green |
| Stories | US-00.7 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-19 (new, W8), LAW-20 (new, W8), LAW-21 (new, W8), LAW-22 (new, W8), LAW-23 (new, W8) |
| Depends on | [SB-802](SB-802-permissions-and-policy-pages-for-spaces.md) |
| Status | Done |

## Why

Space features must be born against their laws, the same way W1 to W7 were.

## Acceptance criteria

- [x] `sb-laws(7)` has LAW-19 to LAW-23 with statements and owners
- [x] One law test each over mocks in `laws/`, failing with 'not implemented'
- [x] `laws/expected_red.txt` lists each with the ticket that turns it green (SB-1004, SB-1006, SB-1205, SB-1203, SB-1302)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Laws in the manual
- [x] Law world gains spaces, labels and barriers
- [x] Five red tests and ratchet lines
- [x] Self-test for each
- [x] Write or extend the law tests: LAW-19, LAW-20, LAW-21, LAW-22, LAW-23
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S8.2): sb-laws(7) has LAW-19 to LAW-23, each with its statement, owner and checks. The law world gained spaces, labels, barriers, moves and grants (`laws/law19.go`, `law21.go`, `law23.go`). Five tests fail with "not implemented", each listed in `laws/expected_red.txt` with the ticket that turns it green: SB-1004, SB-1006, SB-1205, SB-1203 and SB-1302. `selftest_spaces_test.go` has reference systems written from sb-permissions(7). They pass every law, and the space reference really uses documents, so the checks aren't vacuous. Six mutations are caught: skipping the source ACL, restricted content in company-wide asks, ignored barriers, a 15-minute revocation, a grant kept after revocation, and an unapproved metric served.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

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
| Status | Open |

## Why

Space features must be born against their laws, the same way W1 to W7 were.

## Acceptance criteria

- [ ] `sb-laws(7)` has LAW-19 to LAW-23 with statements and owners
- [ ] One law test each over mocks in `laws/`, failing with 'not implemented'
- [ ] `laws/expected_red.txt` lists each with the ticket that turns it green (SB-1004, SB-1006, SB-1205, SB-1203, SB-1302)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Laws in the manual
- [ ] Law world gains spaces, labels and barriers
- [ ] Five red tests and ratchet lines
- [ ] Self-test for each
- [ ] Write or extend the law tests: LAW-19, LAW-20, LAW-21, LAW-22, LAW-23
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-6001: Prove it on the real thing"
summary: "Ten tickets wait In review for a real system, and there's no single run that checks each criterion and leaves evidence a reviewer can trust."
---

# SB-6001: Prove it on the real thing

| Field | Value |
|---|---|
| Epic | [E60 Proven for Real](README.md) |
| Wave / slice | W59 / S59.1 |
| Stories | US-60.1 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4707, SB-5901 |
| Status | Done |

## Why

Ten tickets wait In review for a real system, and there's no single run that checks each criterion and leaves evidence a reviewer can trust.

## Acceptance criteria

- [x] `sb prove TICKET` runs that ticket's acceptance criteria against the configured real system, reading only, in throwaway test spaces it removes
- [x] It writes `state/proofs/TICKET-DATE.json`, signed: pass or fail per criterion, counts, timings and the API version, every id and name hashed, no text; `sb prove --verify` checks it
- [x] Only a person moves a ticket from In review, by attaching its proof; GCP stays out

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `sb prove` runner and the per-ticket criteria
- [x] Signed proof file and `--verify`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `sb prove TICKET --config FILE` (`internal/cell/prove`) runs the criteria of SB-5901 to SB-5907, SB-2603, SB-2601 and SB-1605 against the configured system, reading only; `internal/core/proof` checks a proof holds only fixed names, counts, timings and ids hashed with a per-proof key, signs it with the anchor key as kind `proof`, and `sb prove --verify` checks it. A missing input fails its criteria. The checks are read-only, so no test spaces are made; SB-2005 and SB-2105 post to Mattermost and stay a person's run. Tested in TestProveSharePoint, TestProveMailbox, TestSealRefusesContentAndVerifies and TestLAW07_AProofHoldsNoContent (with TestSelfProofsAreCaught).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

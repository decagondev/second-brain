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
| Status | Open |

## Why

Ten tickets wait In review for a real system, and there's no single run that checks each criterion and leaves evidence a reviewer can trust.

## Acceptance criteria

- [ ] `sb prove TICKET` runs that ticket's acceptance criteria against the configured real system, reading only, in throwaway test spaces it removes
- [ ] It writes `state/proofs/TICKET-DATE.json`, signed: pass or fail per criterion, counts, timings and the API version, every id and name hashed, no text; `sb prove --verify` checks it
- [ ] Only a person moves a ticket from In review, by attaching its proof; GCP stays out

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `sb prove` runner and the per-ticket criteria
- [ ] Signed proof file and `--verify`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

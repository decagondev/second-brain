---
title: "SB-3106: Owners mark their own reviewed"
summary: "Owners hear about their old documents but can't mark one reviewed."
---

# SB-3106: Owners mark their own reviewed

| Field | Value |
|---|---|
| Epic | [E31 Fill the Gaps](README.md) |
| Wave / slice | W30 / S30.2 |
| Stories | US-31.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-11, LAW-12 |
| Depends on | SB-2806 |
| Status | Open |

## Why

Owners hear about their old documents but can't mark one reviewed.

## Acceptance criteria

- [ ] The current owner at the source can mark their own document Reviewed, still right from the Documents tab
- [ ] The mark is the stewards' record and the audit row names the owner
- [ ] Someone else's document is refused with EPERM; a deprovisioned owner can't mark

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

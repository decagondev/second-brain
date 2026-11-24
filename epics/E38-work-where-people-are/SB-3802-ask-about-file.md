---
title: "SB-3802: Ask about a file"
summary: "People can't ask about a file that isn't in the share."
---

# SB-3802: Ask about a file

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.1 |
| Stories | US-38.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-09, LAW-14 |
| Depends on | SB-2003 |
| Status | Open |

## Why

People can't ask about a file that isn't in the share.

## Acceptance criteria

- [ ] The ask takes one file of up to 2 MB of a type the folder connector reads; its text is used for that ask only
- [ ] The answer cites it as your file; the trace keeps only that there was a file, and the index none of it
- [ ] Instruction-like text is flagged as in ingest; a larger file gives EINVAL

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API and agent
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

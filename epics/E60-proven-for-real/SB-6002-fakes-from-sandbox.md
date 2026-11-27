---
title: "SB-6002: Fakes refreshed from a real sandbox"
summary: "Recorded fakes were written from the vendors' docs, and nothing tells us when one drifts from what the vendor really sends."
---

# SB-6002: Fakes refreshed from a real sandbox

| Field | Value |
|---|---|
| Epic | [E60 Proven for Real](README.md) |
| Wave / slice | W59 / S59.1 |
| Stories | US-60.1 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-07, LAW-09 |
| Depends on | SB-6001 |
| Status | Open |

## Why

Recorded fakes were written from the vendors' docs, and nothing tells us when one drifts from what the vendor really sends.

## Acceptance criteria

- [ ] `sb record VENDOR` runs a fixed script of reads against a real sandbox and writes a recording in the fakes' format
- [ ] Every string is replaced by one of the same shape, ids are mapped consistently, people come from `acme-logistics`, `beta-freight` and `gamma-parts`, and `sb secrets scan` runs before writing
- [ ] A shape test fails when a fake drifts from the recording

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Recorder and value replacer
- [ ] Shape test over every recorded fake
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

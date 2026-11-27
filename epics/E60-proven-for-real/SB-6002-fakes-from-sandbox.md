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
| Status | Done |

## Why

Recorded fakes were written from the vendors' docs, and nothing tells us when one drifts from what the vendor really sends.

## Acceptance criteria

- [x] `sb record VENDOR` runs a fixed script of reads against a real sandbox and writes a recording in the fakes' format
- [x] Every string is replaced by one of the same shape, ids are mapped consistently, people come from `acme-logistics`, `beta-freight` and `gamma-parts`, and `sb secrets scan` runs before writing
- [x] A shape test fails when a fake drifts from the recording

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Recorder and value replacer
- [x] Shape test over every recorded fake
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `sb record graph` runs `msgraph.Client.Script` (drive deltas, the first file and its permissions, a person's groups and manager, a shared mailbox's delta) and `internal/core/record` replaces every value by one of the same shape, the same value by the same replacement, addresses by placeholder people, then scans against the vault and secret shapes before writing. `internal/adapters/msgraph/testdata/recording.json` is the Graph fake's recording, and TestFakeMatchesTheRecording fails a fake missing a field. Tested in TestRecordGraph.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

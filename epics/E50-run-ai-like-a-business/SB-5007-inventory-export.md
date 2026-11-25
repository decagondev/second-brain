---
title: "SB-5007: Inventory export"
summary: "An auditor asking what AI runs in the company gets a tour of several pages."
---

# SB-5007: Inventory export

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.2 |
| Stories | US-50.2 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-07, LAW-11, LAW-12 |
| Depends on | SB-4707, SB-4004 |
| Status | Done |

## Why

An auditor asking what AI runs in the company gets a tour of several pages.

## Acceptance criteria

- [x] `sb inventory export` and **Download inventory** write every agent, skill, rhythm, shared routine, trigger and starter question with owner, spaces, cap, targets, last run and last change, as CSV and signed JSON
- [x] The export matches the manifest and the registry, and the JSON signature verifies
- [x] A personal routine is never named, only counted per space

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Export
- [x] CLI and registry button
- [x] Signature
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: `internal/core/inventory`: every agent (the registry's rows), skill, rhythm, shared routine (with its member count), trigger and starter question, with owner, spaces, cap, job and targets, last run and last change, and personal routines only as counts per space. `GET /v1/admin/inventory` gives it signed with the anchor key (`?format=csv` the CSV), the Agents screen has **Download inventory**, and `sb inventory export` reads `GET /v1/cell/inventory`, verifies it and writes `ai-inventory.json` and `ai-inventory.csv`. Signing found that `canonjson` refused amounts, so an agent's record with any real cost couldn't be signed: `canonjson.EncodeDocument` now hashes non-integer numbers as decimal strings for records and the inventory, and approval payloads keep the strict subset. Tested in TestInventory, TestRecordWithAmountsSigns, TestLearnWhatWorksThroughARunningCell and TestInventoryExportAgainstARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

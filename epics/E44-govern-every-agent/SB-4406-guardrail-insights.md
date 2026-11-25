---
title: "SB-4406: Guardrail insights"
summary: "Injection flags and refused tool calls are in the audit log, but nobody sees the trend."
---

# SB-4406: Guardrail insights

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.2 |
| Stories | US-44.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-09 |
| Depends on | SB-207 |
| Status | Done |

## Why

Injection flags and refused tool calls are in the audit log, but nobody sees the trend.

## Acceptance criteria

- [x] `GET /v1/admin/guardrails` and a console card show 12 weeks of injection flags by source and space, flagged uploads, refused and denied tool calls, and dropped uncited claims
- [x] Counts only, rows under 5 shown as "fewer than 5"; no chunk text appears
- [x] A source whose flags double in a week raises one alert to the admin

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] Console
- [x] Alert
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. report.BuildGuardrails: 12 ISO weeks of injection flags (`instruction_like`, `hidden_text_removed`) by source and space from the chunks the index first saw, flagged uploads (Trace.FlaggedFiles), refused and denied tool calls from the audit log, and answers with uncited claims dropped; 1 to 4 shown as "fewer than 5". GET /v1/admin/guardrails for operators and the Guardrails card on the admin console. The notify job raises `guardrail_flags_doubled` once a week for a source whose flags reached 5 and doubled. TestGuardrails and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

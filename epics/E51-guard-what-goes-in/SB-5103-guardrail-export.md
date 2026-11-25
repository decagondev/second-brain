---
title: "SB-5103: Guardrail export"
summary: "Guardrail trends exist, but security can't take the events themselves to their own tools."
---

# SB-5103: Guardrail export

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.1 |
| Stories | US-51.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4406, SB-2302 |
| Status | Done |

## Why

Guardrail trends exist, but security can't take the events themselves to their own tools.

## Acceptance criteria

- [x] Suspect passages, masked outputs and refusals are guardrail events in the audit log and on the audit stream
- [x] `GET /v1/admin/guardrails/events?since=&format=csv` lists time, kind, space, agent, document id and counts
- [x] No document text and no asker appear; the stream carries each event once

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Guardrail events
- [x] Export endpoint
- [x] Stream
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. A new audit kind, `guardrail` (sb-audit(5), law11's kinds), written by `sys_guard`: `suspect:CITE` with `skipped` or `monitored` and `agent=NAME` for each suspect passage of a kept answer (eval runs write none), and `masked:api|notify|slack|mattermost` with `strings=N`. The audit stream carries them like any row. `report.GuardrailEvents` turns these, rejected approvals and denied tool calls into events, and `GET /v1/admin/guardrails/events?since=Nd|RFC3339[&format=csv]` lists them (default 7 days, at most 10,000): time, kind, space, agent, document id and count, never text or the asker. Tested in TestGuardrailEvents and TestGuardWhatGoesInThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

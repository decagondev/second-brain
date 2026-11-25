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
| Status | Open |

## Why

Guardrail trends exist, but security can't take the events themselves to their own tools.

## Acceptance criteria

- [ ] Suspect passages, masked outputs and refusals are guardrail events in the audit log and on the audit stream
- [ ] `GET /v1/admin/guardrails/events?since=&format=csv` lists time, kind, space, agent, document id and counts
- [ ] No document text and no asker appear; the stream carries each event once

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Guardrail events
- [ ] Export endpoint
- [ ] Stream
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

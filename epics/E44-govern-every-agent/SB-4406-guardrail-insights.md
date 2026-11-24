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
| Status | Open |

## Why

Injection flags and refused tool calls are in the audit log, but nobody sees the trend.

## Acceptance criteria

- [ ] `GET /v1/admin/guardrails` and a console card show 12 weeks of injection flags by source and space, flagged uploads, refused and denied tool calls, and dropped uncited claims
- [ ] Counts only, rows under 5 shown as "fewer than 5"; no chunk text appears
- [ ] A source whose flags double in a week raises one alert to the admin

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] Console
- [ ] Alert
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

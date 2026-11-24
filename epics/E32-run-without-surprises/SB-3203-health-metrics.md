---
title: "SB-3203: Health in Prometheus"
summary: "Health is JSON only, so the site's alerts can't see it."
---

# SB-3203: Health in Prometheus

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.1 |
| Stories | US-32.1 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-03 |
| Depends on | — |
| Status | Open |

## Why

Health is JSON only, so the site's alerts can't see it.

## Acceptance criteria

- [ ] `/metrics` has `sb_health_warning{check}` at 0 or 1 for every health line, and `sb_health_unknown` for lines it can't read
- [ ] The runbook gives example alert rules
- [ ] Each warning in health is 1 on its gauge; an unreadable line is never 0 on both

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Metrics
- [ ] Runbook
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

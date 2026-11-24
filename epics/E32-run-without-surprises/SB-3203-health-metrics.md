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
| Status | Done |

## Why

Health is JSON only, so the site's alerts can't see it.

## Acceptance criteria

- [x] `/metrics` has `sb_health_warning{check}` at 0 or 1 for every health line, and `sb_health_unknown` for lines it can't read
- [x] The runbook gives example alert rules
- [x] Each warning in health is 1 on its gauge; an unreadable line is never 0 on both

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Metrics
- [x] Runbook
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

/metrics sets sb_health_warning{check} and sb_health_unknown{check} from health on each scrape, and sb_health_unknown{check=health} when health can't be built; sb-observability(7) gives example alert rules

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

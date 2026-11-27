---
title: "SB-5703: People first when the provider is busy"
summary: "When a provider rate-limits, scheduled work and people's questions compete equally, and one busy space can starve the rest."
---

# SB-5703: People first when the provider is busy

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.1 |
| Stories | US-57.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-03, LAW-14 |
| Depends on | SB-5601, SB-5602 |
| Status | Open |

## Why

When a provider rate-limits, scheduled work and people's questions compete equally, and one busy space can starve the rest.

## Acceptance criteria

- [ ] On a rate limit the gateway queues calls per space, serves a person's ask before any scheduled step, and serves spaces in turn
- [ ] Scheduled work that can't run waits for the next tick; an ask that would wait more than 20 seconds gets passages with "the model is busy"
- [ ] The health report counts rate limits and waits per space

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Per-space queue in the gateway
- [ ] Passages after 20 seconds
- [ ] Health counts
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

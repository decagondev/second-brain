---
title: "SB-606: Weekly leadership report"
summary: "Leaders get evidence, not anecdotes."
---

# SB-606: Weekly leadership report

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.2 Onboarding |
| Stories | US-07.3 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | — |
| Depends on | [SB-607](../E07-rollout-ops/SB-607-per-tenant-observability.md) |
| Status | In review |

## Why

Leaders get evidence, not anecdotes.

## Acceptance criteria

- [x] Weekly active users per seat, cited-answer rate, approve-to-reject ratio, top gaps
- [ ] Delivered automatically (`report.Weekly` builds last week and hands Markdown to a delivery function; the cell's scheduler and mail/chat adapter come with W7)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Report job
- [x] Template
- [ ] Delivery (W7 notification adapter)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.2): `internal/core/report` builds the week (Monday to Monday UTC) from traces, draft outcomes, actions, seats and tokens: active users per seat, asks, cited-answer rate, gaps counted by notice kind (never by question), drafts used and discarded, actions by outcome, and spend at the release's price table. `Markdown` is the report as sent: numbers only, and a test plants content in a trace and checks it never appears. `GET /v1/admin/usage?week=YYYY-Www` serves the same numbers to operators and approvers, and refuses anyone else. sb-observability(7) defines every line.

**Left:** the Monday schedule and delivery through the cell's notification adapter (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

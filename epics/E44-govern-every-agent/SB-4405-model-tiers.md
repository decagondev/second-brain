---
title: "SB-4405: Model tiers"
summary: "Every ask costs the same, though most are short and plain."
---

# SB-4405: Model tiers

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.2 |
| Stories | US-44.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-06, LAW-07 |
| Depends on | SB-305 |
| Status | Open |

## Why

Every ask costs the same, though most are short and plain.

## Acceptance criteria

- [ ] `model.small` takes asks with no agent, no file and a short question; the rest use the main model
- [ ] The release gate runs the eval suites on both tiers, and a failing tier takes no asks; a space can pin `model: main`
- [ ] The costs page shows spend per tier, and the report the share of asks on each

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Config
- [ ] Routing
- [ ] Gate
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

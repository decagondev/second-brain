---
title: "SB-4704: Shadow runs"
summary: "A changed prompt is judged by its eval suite alone before it meets real work."
---

# SB-4704: Shadow runs

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-01, LAW-06 |
| Depends on | SB-4402, SB-4701 |
| Status | Done |

## Why

A changed prompt is judged by its eval suite alone before it meets real work.

## Acceptance criteria

- [x] `shadow: 7d` runs a changed agent beside the current one in the first ring for a week
- [x] Shadow drafts are never delivered, and are gone after the week
- [x] The owner sees scores, lengths and cite counts, and text only for runs whose every cite they can open

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Release
- [x] Runner
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. An agent definition may carry `shadow: { prompt, days }` (at most 7 days; `sb agents lint` rule 8 checks it). On the first ring (the `testing` channel) each draft of that agent also runs the shadow prompt as the same person, in `Stewards.shadow`, and the result is kept only as a shadow run with its own checks (claims, cited, cites, dropped) and length beside the current draft's; it is never stored as a thread, never queued and never rate-limited. `GET /v1/agents/shadow` shows an agent's owners both sides, with text only for runs whose every cite the owner can open (`opensAll`); everyone else sees nothing. The notify job's `SweepShadows` deletes a window's runs when it ends. Shadow runs and windows live in the cell's existing trace store (kinds `shadow` and `shadowwindow`), no new store. The web shows them on Questions for owners (`ShadowRuns`). TestProveItThroughARunningCell checks that the shadow never reaches the queue, only the owner sees it, and the week's end deletes it.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

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
| Status | Open |

## Why

A changed prompt is judged by its eval suite alone before it meets real work.

## Acceptance criteria

- [ ] `shadow: 7d` runs a changed agent beside the current one in the first ring for a week
- [ ] Shadow drafts are never delivered, and are gone after the week
- [ ] The owner sees scores, lengths and cite counts, and text only for runs whose every cite they can open

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Release
- [ ] Runner
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

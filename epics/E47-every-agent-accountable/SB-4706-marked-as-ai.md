---
title: "SB-4706: Marked as AI"
summary: "A published note or message can't be told apart from one a person wrote."
---

# SB-4706: Marked as AI

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-28, LAW-11 |
| Depends on | SB-4402 |
| Status | Done |

## Why

A published note or message can't be told apart from one a person wrote.

## Acceptance criteria

- [x] Every draft, message, email and chat post an agent writes says which agent and version wrote it
- [x] A published note keeps `generated_by` and `approved_by`, and its cite shows the mark; exports and prints keep it
- [x] LAW-28 is a law test and goes green; a note a person rewrote loses the mark

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Mark
- [x] Front matter
- [x] UI and chat
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. Answers carry `agent` and `release`; agent drafts carry `mark`; the web (`Marked.tsx`), Slack and Mattermost show "Written by AGENT, version V", and a shared answer says who wrote it. A note a steward publishes from an agent draft keeps `generated_by: AGENT@VERSION`, `approved_by` and `edited: true` when the reply changed (`markFront`); a reply with none of the draft's lines left (any version) is the steward's own and carries no mark (`Rewrote`). Ingest reads the mark into each chunk (`ingest.MarkOf`, three new index columns), keeping `approved_by` only when it is the document's owner, and cites show it, in prints and exports too. Notifications carry no agent text. LAW-28 is a law test (`laws/law28.go`, TestLAW28_AnAgentsWritingSaysSo) with a selftest that strips the mark and is caught (TestSelf_LAW28CatchesAnUnmarkedNote).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

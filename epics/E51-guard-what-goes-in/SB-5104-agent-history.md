---
title: "SB-5104: Agent history"
summary: "Nobody can see how an agent or skill changed over time, or go back to an earlier version."
---

# SB-5104: Agent history

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.2 |
| Stories | US-51.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-06, LAW-11 |
| Depends on | SB-4402, SB-4707 |
| Status | Done |

## Why

Nobody can see how an agent or skill changed over time, or go back to an earlier version.

## Acceptance criteria

- [x] `GET /v1/admin/agents/NAME/history` lists every version with when, who proposed, who approved and which suite passed
- [x] **View changes** diffs any two versions
- [x] **Restore** proposes the older definition as a manifest diff, which must pass the gate again

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] History from bundles and audit
- [x] Diff
- [x] Restore as a proposal
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `proposals.History` reads every approved proposal's whole manifest and keeps each version whose lines differ, with the applied manifest last (`current`); `proposals.AgentLines` are an agent's settings (enabled, `monthly_usd`, goal, the spaces that list it) and `proposals.SkillLines` a skill's instruction and cases. `GET /v1/admin/agents/NAME/history[?from=&to=]` (with `changes` from `proposals.Changes`) and, for stewards, `GET /v1/spaces/NAME/skills/SKILL/history`. Restore is a proposal: `proposals.RestoreAgent` builds the manifest with the old settings, `Cell.restore` checks it, audits it as `proposed` and stores it open for another operator; a skill's restore is a steward proposal through the usual `submit`. Restoring the current version is `EINVAL`. An agent's prompt and tools belong to the release, so history covers its manifest settings; an older prompt comes back with an older release. Tested in TestAgentHistoryAndRestore and TestRollOutWithCareThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

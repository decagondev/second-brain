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
| Status | Open |

## Why

Nobody can see how an agent or skill changed over time, or go back to an earlier version.

## Acceptance criteria

- [ ] `GET /v1/admin/agents/NAME/history` lists every version with when, who proposed, who approved and which suite passed
- [ ] **View changes** diffs any two versions
- [ ] **Restore** proposes the older definition as a manifest diff, which must pass the gate again

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] History from bundles and audit
- [ ] Diff
- [ ] Restore as a proposal
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

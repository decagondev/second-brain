---
title: "SB-4206: Draft the fix"
summary: "A stale or conflicting document is flagged, but its owner still has to find what changed and write the fix."
---

# SB-4206: Draft the fix

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.2 |
| Stories | US-42.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-14, LAW-16 |
| Depends on | SB-3605 |
| Status | Open |

## Why

A stale or conflicting document is flagged, but its owner still has to find what changed and write the fix.

## Acceptance criteria

- [ ] **Draft a fix** on a stale or conflicting document proposes a diff, each change citing its newer source
- [ ] The suggestion goes to the owner as a message and to their inbox; the brain never writes the source
- [ ] A source the owner can't open is not used; the suggestion is marked done when the source changes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent and prompt
- [ ] API
- [ ] UI
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

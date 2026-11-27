---
title: "SB-5503: Not for me"
summary: "A person can't keep an outdated draft or a noisy document out of their answers for good."
---

# SB-5503: Not for me

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.1 |
| Stories | US-55.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-01, LAW-25 |
| Depends on | SB-3501, SB-3502 |
| Status | Done |

## Why

A person can't keep an outdated draft or a noisy document out of their answers for good.

## Acceptance criteria

- [x] **Don't use this for me** keeps a document out of everything made as the person: answers, briefs, routines, research and MCP results
- [x] The list is on My data, in the person's export and removed by forget
- [x] An excluded document is still cited to others, and undoing it brings it back

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Personal exclusion list beside memory
- [x] Web: My data
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `Agent.Exclude` and `ExcludedFor` at `GET|POST|DELETE /v1/me/exclusions` keep **Don't use this for me** in the person's memory record (`Memory.Excluded`, `Agent.Exclusions`), so the export, erasure and Forget my questions carry it (LAW-25); each change is audited by the document id. `Agent.dropExcluded` runs in every retrieval as the person (asks, searches, drafts, briefs, routines, research, MCP), and an unreadable list fails the ask. Web: the button in the Context panel, and **Not used for me** with Undo on My data.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

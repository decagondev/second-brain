---
title: "SB-4902: Comments on a section"
summary: "A reviewer can only comment on a whole draft, so a redraft can change what was already right."
---

# SB-4902: Comments on a section

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.1 |
| Stories | US-49.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | LAW-09, LAW-14, LAW-16 |
| Depends on | SB-4602 |
| Status | Open |

## Why

A reviewer can only comment on a whole draft, so a redraft can change what was already right.

## Acceptance criteria

- [ ] Reviewers comment on a section of any agent draft; comments are visible to reviewers only and kept in the audit trail
- [ ] **Ask for changes** redrafts only commented sections; an uncommented section is unchanged byte for byte
- [ ] A redraft cites only what the first could, a comment is never a source, and rounds still stop at 3

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Sections
- [ ] Comments
- [ ] Section redraft and diff
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

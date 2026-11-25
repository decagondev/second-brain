---
title: "SB-4501: Ask each department"
summary: "A question that spans departments gets one department's view, and the asker has to ask each one in turn."
---

# SB-4501: Ask each department

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.1 |
| Stories | US-45.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-24, LAW-27 |
| Depends on | SB-3903, SB-4007 |
| Status | Open |

## Why

A question that spans departments gets one department's view, and the asker has to ask each one in turn.

## Acceptance criteria

- [ ] A branch ask consults at most 6 child spaces the asker is a member of, each agent answering as the asker from its own space
- [ ] The answer has a section per department and says which weren't consulted and why
- [ ] A consulted agent never consults or calls a tool; LAW-27 is a law test and goes green

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Consult step
- [ ] Composer
- [ ] Budget per space
- [ ] UI
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-4202: Meeting prep"
summary: "People walk into meetings without the documents, decisions and open questions the brain already holds."
---

# SB-4202: Meeting prep

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.1 |
| Stories | US-42.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-09, LAW-12 |
| Depends on | SB-3906 |
| Status | Open |

## Why

People walk into meetings without the documents, decisions and open questions the brain already holds.

## Acceptance criteria

- [ ] **Prepare** takes a pasted agenda or an event from a read-only calendar connector, read as the person
- [ ] The brief lists related documents, decisions of the last 90 days, open questions and what each tenant attendee owns, each checked as the person
- [ ] Agenda text is data (LAW-09); a deprovisioned attendee isn't shown as an owner

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Calendar connector (read-only, recorded fake)
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

---
title: "SB-5102: Nothing sensitive goes out"
summary: "An answer, notice or MCP result can still carry a key or card number someone pasted into a source."
---

# SB-5102: Nothing sensitive goes out

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.1 |
| Stories | US-51.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-2207, SB-3801 |
| Status | Open |

## Why

An answer, notice or MCP result can still carry a key or card number someone pasted into a source.

## Acceptance criteria

- [ ] Keys, tokens, Luhn-valid card numbers and national ids are masked in answers, drafts, notifications, chat replies, email, MCP results and exports
- [ ] Masking keeps the cite and the rest of the text, and records a guardrail event
- [ ] A law test lists every way text leaves the cell and proves each goes through the one filter

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] One output filter
- [ ] Wire every way out
- [ ] Law test of the paths
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

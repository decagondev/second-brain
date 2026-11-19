---
title: "SB-005: First drafts: sb-connector(3), sb-permissions(7), sb-laws(7)"
summary: "Connectors and permissions carry the platform's hardest guarantees; the laws page is the index of guarantees."
---

# SB-005: First drafts: sb-connector(3), sb-permissions(7), sb-laws(7)

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.2 First pages |
| Stories | US-00.1 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-002](../E00-foundations/SB-002-manual-layout-man-page-template-and-docs.md) |
| Status | In review |

## Why

Connectors and permissions carry the platform's hardest guarantees; the laws page is the index of guarantees.

## Acceptance criteria

- [x] The four connector calls have request, response, error and timeout semantics
- [x] `sb-laws(7)` lists LAW-01 to LAW-16 with how and where each is checked
- [ ] Reviewed by two maintainers and one security reviewer

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Draft `sb-connector(3)`
- [x] Draft `sb-permissions(7)` with the tri-state model
- [x] Draft `sb-laws(7)` from the assurance ledger
- [ ] Review
- [x] Write or extend the law tests: LAW-01, LAW-02, LAW-03
- [x] Devlog entry, ledger and changelog updated

## Progress

Drafts complete. `sb-laws(7)` lists LAW-01 to LAW-18.

**Left:** review by two maintainers and one security reviewer.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

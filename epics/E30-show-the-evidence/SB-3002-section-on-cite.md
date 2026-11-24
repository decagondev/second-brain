---
title: "SB-3002: Section on the cite"
summary: "A cite of a long handbook says only its title."
---

# SB-3002: Section on the cite

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.1 |
| Stories | US-30.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-09, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

A cite of a long handbook says only its title.

## Acceptance criteria

- [x] Each chunk records the nearest heading above it (Markdown, Word, OpenDocument, HTML), at most 120 characters
- [x] The chip, copy and print show Title › Section; a heading can't change rank
- [x] A chunk under ## Carry-over shows it; no headings shows the title only; a rebuild fills it in

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ingest
- [x] Cite field
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S29.1; tested in internal/core/ask/evidence_test.go, internal/core/ingest/section_test.go, internal/adapters/connectors/folder/office_test.go (TestOfficeHeadings), web/src/evidence.test.tsx and the citation suite's cite-006 and cite-007.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

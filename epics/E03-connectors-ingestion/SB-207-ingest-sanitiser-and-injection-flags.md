---
title: "SB-207: Ingest sanitiser and injection flags"
summary: "Ingested text is untrusted input; treat it like user input to a shell."
---

# SB-207: Ingest sanitiser and injection flags

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.3 Pipeline |
| Stories | US-03.3 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | [LAW-09](../../manual/sb-laws.7.md#law-09) |
| Depends on | [SB-206](../E03-connectors-ingestion/SB-206-ingest-pipeline-chunk-embed-tag-provenance.md) |
| Status | Open |

## Why

Ingested text is untrusted input; treat it like user input to a shell.

## Acceptance criteria

- [ ] Instruction-like content is flagged and wrapped as quoted data
- [ ] Injection corpus triggers zero tool calls (eval)
- [ ] Flags visible in citations

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Sanitiser rules
- [ ] Injection corpus in `evals/suites/injection.yaml`
- [ ] Eval wired into CI
- [ ] Write or extend the law tests: LAW-09
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

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
| Status | In review |

## Why

Ingested text is untrusted input; treat it like user input to a shell.

## Acceptance criteria

- [x] Instruction-like content is flagged and wrapped as quoted data
- [ ] Injection corpus triggers zero tool calls (eval)
- [x] Flags visible in citations

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Sanitiser rules
- [x] Injection corpus: `evals/corpus/injection.json` (JSON, because Go reads it with the standard library)
- [x] Eval wired into CI: the corpus runs in `make check` (`TestInjectionCorpus`); the agent-level suite waits for the tool bus
- [x] Write or extend the law tests: LAW-09
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: the sanitiser is built (`internal/core/ingest`, `sanitise.go`) and wired into the pipeline.
- Strip removes invisible format characters (Unicode `Cf`: zero-width, bidi, BOM), HTML comments, elements hidden by style (unclosed ones are dropped to the end), and alt or title text over 100 characters.
- Flag marks instruction-like spans: overrides, new roles, chat-role markers, tool-call JSON, "don't tell the user", and query-string links to non-tenant domains.
- Flags are stored per chunk and are what citations will show. `Wrap` makes a data block the content can't close.
- The 16-case corpus (12 attacks, 4 benign look-alikes) passes.

LAW-09 has its first law test, red first, then green, and a pipeline without a sanitiser is caught. One law fix along the way: zero-width obfuscation doesn't *hide* text; stripping reveals it, so the law now requires it to be flagged rather than removed.

**Left:** "Injection corpus triggers zero tool calls (eval)" needs the agent runtime and tool bus (SB-405, SB-501). A canned mock answer would prove nothing, so this stays open.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

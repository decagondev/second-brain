---
title: "SB-1006: Ask in a space or company-wide"
summary: "People ask their department's brain most days and the whole company's sometimes."
---

# SB-1006: Ask in a space or company-wide

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.3 Ask in a space |
| Stories | US-10.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), `sb-spaces(7)` (new) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), LAW-19 (new, W8), LAW-20 (new, W8) |
| Depends on | [SB-1005](SB-1005-ingest-routes-and-labels-every-document.md), [SB-302](../E04-retrieval-answers/SB-302-live-authorisation-of-every-chunk-before.md) |
| Status | Done |

## Why

People ask their department's brain most days and the whole company's sometimes.

## Acceptance criteria

- [x] `POST /v1/ask` takes a space; company-wide is the union of the asker's spaces minus restricted content they don't hold
- [x] Space, label and barrier checks run before the live ACL batch, and the ACL is still checked for every chunk
- [x] LAW-20 green; the space-leak evals (SB-807) pass

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retrieval filter
- [x] Company-wide union
- [x] Leak evals
- [x] LAW-20 law test green
- [x] Write or extend the law tests: LAW-01, LAW-19, LAW-20
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.3a): sb-ask(3) gained the space parameter (`?space=`, never a body field), the space step of the ask, the three new trace reasons and `ESPACE`; sb-citations(7), sb-spaces(7), sb-eval(5), sb-evals(7) and sb-laws(7) changed with it. `retrieve.Authoriser` takes the cell's spaces and barriers: in a cell with spaces every candidate is decided first by the kernel with its label, holders and the asker's read context (SB-1004), and only the chunks it allows go into the live ACL batch, which still decides each of them. A registry that can't answer fails the request. The trace says `space_denied`, `label_denied` or `barrier_denied`, explained after the kernel's decision, never deciding it. The local eval cell reads a world's `spaces:`, `barriers:`, document `path` and `label`, routes at ingest and resolves each case's `space` as the middleware does. `space-leak` passes (every leak refused, both controls answered), left `evals/expected_red.yaml` and joined the release gate with `failures_max: 0`. `TestLAW20_RestrictedStaysHome` runs on the real retrieval path and left the ratchet; `TestLAW19_AsksOnlyNarrow` covers LAW-19 (and LAW-01: the source ACL decides every chunk) on the same path; a retrieval without its space step is caught, and the ask path agrees with the reference cell on every ask. LAW-22 stays red on its own system until SB-1203 adds drafts, actions and ownership. Apply still refuses a manifest with spaces, now until SB-1008, so no cell runs with spaces until the web UI, the chat app and per-space kill switches enforce them too.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

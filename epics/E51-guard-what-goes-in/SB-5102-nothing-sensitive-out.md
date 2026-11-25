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
| Status | Done |

## Why

An answer, notice or MCP result can still carry a key or card number someone pasted into a source.

## Acceptance criteria

- [x] Keys, tokens, Luhn-valid card numbers and national ids are masked in answers, drafts, notifications, chat replies, email, MCP results and exports
- [x] Masking keeps the cite and the rest of the text, and records a guardrail event
- [x] A law test lists every way text leaves the cell and proves each goes through the one filter

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] One output filter
- [x] Wire every way out
- [x] Law test of the paths
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `ingest.MaskOut` replaces every credential the secret scanner knows with `[removed]` and masks cards, IBANs and national ids but their last four characters; `ingest.MaskJSON` applies it to each JSON string literal only, leaving keys' order, numbers and escapes as they were. Three places cover every way out: `api.MaskOut` wraps the cell's whole HTTP handler (JSON, CSV, Markdown and plain text; `POST /v1/tokens` and `GET /metrics` pass as they are), `notify.Masking` wraps every sender the notify job or `Cell.Notifier` uses, and the Slack and Mattermost apps mask each reply before posting it. Each masked response or message writes a `guardrail` row. Tested in TestMaskOut, TestMaskJSON, TestMaskOutMiddleware, TestMaskingSender, both TestReplyIsMasked, TestGuardWhatGoesInThroughARunningCell and the new LAW-07 property TestLAW07_NothingSensitiveGoesOut (with TestSelfOutFilterIsCaught: no filter, or one that masks numbers only, is caught).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

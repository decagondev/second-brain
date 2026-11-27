---
title: "SB-6101: Read scans and images"
summary: "A scanned contract, a whiteboard photo or a diagram in a slide is invisible to the brain."
---

# SB-6101: Read scans and images

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.1 |
| Stories | US-61.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-06, LAW-09, LAW-28 |
| Depends on | SB-5205, SB-4706, SB-5705 |
| Status | Done |

## Why

A scanned contract, a whiteboard photo or a diagram in a slide is invisible to the brain.

## Acceptance criteria

- [x] PDF pages with no text layer, image files and images in .docx and .pptx are read by sb-cell(5) `model.vision`, gated and in region
- [x] The text is a chunk marked `machine-read`, cited as read from a scan with its page, and counted at the vision tier's price
- [x] Off by default: with `model.vision` unset, images are skipped and counted on health; text in an image is data

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Vision tier in the gateway and evals gate
- [x] Machine-read chunks and cites
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ports.Document.Images` (folder and SharePoint: image files, a text-less PDF's JPEGs as pages, .docx and .pptx pictures with their slide) are read at ingest by sb-cell(5) `model.vision`, a gateway tier of the main provider with no fallback, region-bound, used only while `state/evals/vision-gated` names the release; the evals job gates it by showing three pictures of six random digits (`wire.DigitsPNG`) it must read back. What it reads joins the document after a marker; every chunk holding any of it carries `ReadBy: vision` (Postgres `read_by`) and its page, cites and search results carry `read_by`, the web UI and chat apps say *Read by a model from a scan or picture*. Unread pictures count on health (`unread`). Messages carry an image part for Anthropic and OpenAI-compatible providers. Tested in TestImagesAndMedia, TestPartsAreReadAndMarked, TestScansThroughARunningCell (a fake model that reads the digits by their pixels) and TestLAW28_AnAgentsWritingSaysSo (law28Read, new; TestSelfReadTextIsCaught).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

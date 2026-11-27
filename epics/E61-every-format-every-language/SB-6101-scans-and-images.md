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
| Status | Open |

## Why

A scanned contract, a whiteboard photo or a diagram in a slide is invisible to the brain.

## Acceptance criteria

- [ ] PDF pages with no text layer, image files and images in .docx and .pptx are read by sb-cell(5) `model.vision`, gated and in region
- [ ] The text is a chunk marked `machine-read`, cited as read from a scan with its page, and counted at the vision tier's price
- [ ] Off by default: with `model.vision` unset, images are skipped and counted on health; text in an image is data

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Vision tier in the gateway and evals gate
- [ ] Machine-read chunks and cites
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

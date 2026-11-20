---
title: "SB-1005: Ingest routes and labels every document"
summary: "A document must know which spaces hold it and how sensitive it is before anyone can ask about it."
---

# SB-1005: Ingest routes and labels every document

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.2 Space policy |
| Stories | US-10.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-13](../../manual/sb-laws.7.md#law-13), LAW-20 (new, W8) |
| Depends on | [SB-1004](SB-1004-cedar-schema-v2-spaces-labels-narrowing.md), [SB-206](../E03-connectors-ingestion/SB-206-ingest-pipeline-chunk-embed-tag-provenance.md) |
| Status | In review |

## Why

A document must know which spaces hold it and how sensitive it is before anyone can ask about it.

## Acceptance criteria

- [x] Each document version gets its spaces (from scopes) and a label: the source's label if it has one, else the space default, else `internal`
- [x] A document matched by a restricted space and a non-restricted space takes the stricter label
- [x] Changing a scope or label re-routes on the next sync; a rebuild gives the same result (LAW-13)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Router
- [x] Label mapping from Drive labels and Slack channels
- [x] Rebuild test
- [x] Write or extend the law tests: LAW-13, LAW-20
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.2b): sb-ingest(7) gained "Spaces and labels" and sb-connector(3) "Location and label", with a row per connector. `ports.Document` carries the source's location and label, and `ports.Chunk` its label, holders and routing fingerprint, in the memory index and in Postgres (three columns added in place; tested against the WSL database). `internal/core/route` matches scope patterns (`**`, `*`), takes a known source label, treats an unknown one as `restricted`, and otherwise takes the strictest default of the holders, or `internal`; `Fingerprint` identifies a space configuration. `ingest.Pipeline` routes every version before it writes a chunk, and `Reroute` (the start of every sync) re-ingests documents routed under an older configuration; a cell without spaces does neither. Connectors: Drive reports the folder path (parents read and cached) and maps one tenant-named Drive label field to labels (an unmapped choice is `restricted`), both in `drivetest`; Slack reports the channel and labels private channels `confidential`; Linear the team key; HubSpot the record type; the mock source can place documents. Two law tests run the real pipeline: `TestLAW13_SpaceRoutingIsDerived` (after a sync, a rebuild and a scope change, every chunk carries exactly the reference routing; a router holding an old configuration is caught) and `TestLAW20_IngestNeverLabelsBelowRestricted` (a router that ignores the stricter label is caught).

**Left:** a run against a real Drive with labels (SB-203's sandbox). Sealing each chunk under its space's key (SB-1003) can now use these holders, and waits on keyword search over sealed text.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

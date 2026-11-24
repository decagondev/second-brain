---
title: "SB-3006: Open at the cited lines"
summary: "A GitHub cite opens the file at the top, not where the answer comes from."
---

# SB-3006: Open at the cited lines

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.2 |
| Stories | US-30.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | — |
| Depends on | — |
| Status | Open |

## Why

A GitHub cite opens the file at the top, not where the answer comes from.

## Acceptance criteria

- [ ] A GitHub cite's url ends with `#L12-L20` for the span's lines at that commit
- [ ] Other kinds are unchanged
- [ ] A span in lines 12 to 20 links to #L12-L20

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Connector
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

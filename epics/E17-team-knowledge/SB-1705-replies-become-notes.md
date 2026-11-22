---
title: "SB-1705: Answer once: replies become notes"
summary: "A steward's answer lives in a reply or a chat thread and is found once; the next person asks again."
---

# SB-1705: Answer once: replies become notes

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.2 |
| Stories | US-17.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-04, LAW-05, LAW-09, LAW-13, LAW-15 |
| Depends on | [SB-1704](SB-1704-ask-the-stewards.md) |
| Status | Open |

## Why

A steward's answer lives in a reply or a chat thread and is found once; the next person asks again.

## Acceptance criteria

- [ ] A steward publishes a reply as a note: a Markdown file and its ACL file under `notes/` in the folder share, through the tool bus as `note-publish` on the folder connector's `act`
- [ ] The note is `Ran` only with a receipt, its path and SHA-256 read back after the write; the publish binds the note's hash
- [ ] Readers default to the space's members and can only be narrowed; before publishing, the steward sees how many cites are readable by fewer people than the note and confirms
- [ ] `act` writes only `.md` and `.acl` under `notes/`, opened as a root it can't leave; the rest of the share stays read-only; no model writes a note
- [ ] The next ingest indexes the note, and an ask that matches it gets a cited answer

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notes directory in the self-hosted layout
- [ ] `act` on the folder connector, contract kit
- [ ] Publish from the steward's reply
- [ ] Tests and a smoke check
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Planned 2026-11-22 with W16, accepted by the user the same day ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

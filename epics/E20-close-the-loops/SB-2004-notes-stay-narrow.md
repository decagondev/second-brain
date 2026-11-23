---
title: "SB-2004: Notes stay narrow"
summary: "A note's readers are a snapshot of the space's members when it was published; when the space narrows, the note doesn't."
---

# SB-2004: Notes stay narrow

| Field | Value |
|---|---|
| Epic | [E20 Close the Loops](README.md) |
| Wave / slice | W19 / S19.2 |
| Stories | US-20.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-19, LAW-21 |
| Depends on | — |
| Status | Done |

## Why

A note's readers are a snapshot of the space's members when it was published; when the space narrows, the note doesn't.

## Acceptance criteria

- [x] A steward's review list (SB-1709) includes notes whose readers include someone no longer a member of the space
- [x] Each is checked live as the steward and listed with the number of readers to remove, never their names
- [x] Republishing the note narrowed takes it off the list

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Check over the notes folder
- [x] Review list
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built 2026-11-23. The stewards' review list (`GET /v1/spaces/NAME/review`) lists the space's notes whose readers include someone no longer a member, checked live as the steward, with `wider` (how many readers to remove, never who) and `question`, first in the list. `POST …/questions/ID/note` on a reply that is already a note republishes it narrowed: the same text under its stored title, to its readers now who are still members, owned by its first publisher. The folder connector rewrites a note's ACL file only to fewer readers with the same owner and label, never wider. The steward's screen shows the count and **Republish to members only** with the usual confirmation. Law: `TestLAW19_NotesStayAsNarrowAsTheirSpace` (real connector, pipeline, review list and publish: the count is right and names no one; after republishing the note is off the list and readable by exactly the members who could read it and still can; self-tests catch stale members and a republish that writes nothing). Manual: sb-spaces(7), sb-ask(3), sb-connector(3).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

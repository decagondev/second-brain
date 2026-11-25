---
title: "SB-4303: Memory I control"
summary: "Each ask starts from nothing, so people repeat their role, team and preferred format."
---

# SB-4303: Memory I control

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.1 |
| Stories | US-43.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-25, LAW-07, LAW-09 |
| Depends on | SB-3501 |
| Status | Done |

## Why

Each ask starts from nothing, so people repeat their role, team and preferred format.

## Acceptance criteria

- [x] **Remember** and **About me** keep at most 20 lines of 200 characters the person wrote, sent with their own asks only
- [x] LAW-25: a memory never holds document text and never reaches anyone else's ask; its law test and a breaking selftest
- [x] Memory is in My data's export, cleared by Forget at once, and an admin can turn it off

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Records
- [x] Ask context
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Memory, MemoryStore (memory and Postgres, kind memory: never expired, in the export, erased by Forget through retention.ForgetErased), Agent.Remember/ForgetLine/MemoryFor with fromDocument (a passage they can open, whole or 8 words in a row, is refused), memoryLines sent only with the person's own ask-agent asks as a data block. /v1/me/memory. sb-tenant(5) memory: off. LAW-25 in sb-laws(7), laws/law25.go: TestLAW25_AMemoryIsItsPersonsOwn and TestSelf_LAW25CatchesASharedMemory. Web: About me, Remember something about me.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

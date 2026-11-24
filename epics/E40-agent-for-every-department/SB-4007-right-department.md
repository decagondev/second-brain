---
title: "SB-4007: The right department"
summary: "A company-wide ask doesn't know a department's agent could answer it better."
---

# SB-4007: The right department

| Field | Value |
|---|---|
| Epic | [E40 An Agent for Every Department](README.md) |
| Wave / slice | W39 / S39.2 |
| Stories | US-40.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | SB-3704 |
| Status | Done |

## Why

A company-wide ask doesn't know a department's agent could answer it better.

## Acceptance criteria

- [x] A company-wide ask that matches a department's glossary suggests that space's agent, with a button to ask it there
- [x] Only spaces the asker is a member of, at most one
- [x] It's a suggestion only; a non-member never sees the space named

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

A company-wide ask using a member space's glossary suggests its department agent, at most one, with Ask AGENT in SPACE

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

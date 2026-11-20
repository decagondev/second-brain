---
title: "SB-1404: The manual and the code agree"
summary: "The manual is the spec (P2); anything in the code it doesn't describe is unspecified behaviour."
---

# SB-1404: The manual and the code agree

| Field | Value |
|---|---|
| Epic | [E14 Hardening](README.md) |
| Wave / slice | W13 / S13.4 |
| Stories | US-14.1 |
| Manual | as the change needs (manual first, P2) |
| Laws | LAW-16 |
| Depends on | [SB-1403](SB-1403-clean-and-concise.md) |
| Status | Open |

## Why

The manual is the spec (P2); anything in the code it doesn't describe is unspecified behaviour.

## Acceptance criteria

- [ ] A check that every HTTP endpoint, CLI command and flag, and error code in the code is in the manual, and everything the manual promises exists
- [ ] `make check` fails on an endpoint, command, flag or error code in one and not the other
- [ ] CLAUDE.md's state section is a short summary that points to `epics/README.md`

## Subtasks

- [ ] Update the man pages the change touches first (manual first, P2)
- [ ] The check
- [ ] Fix the drift it finds
- [ ] CLAUDE.md summary
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

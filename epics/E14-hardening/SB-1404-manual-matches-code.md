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
| Status | Done |

## Why

The manual is the spec (P2); anything in the code it doesn't describe is unspecified behaviour.

## Acceptance criteria

- [x] A check that every HTTP endpoint, CLI command and flag, and error code in the code is in the manual, and everything the manual promises exists
- [x] `make check` fails on an endpoint, command, flag or error code in one and not the other
- [x] CLAUDE.md's state section is a short summary that points to `epics/README.md`

## Subtasks

- [x] Update the man pages the change touches first (manual first, P2)
- [x] The check
- [x] Fix the drift it finds
- [x] CLAUDE.md summary
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S13.4): `tools/check/manual_code.py`, with a self-test, in `make lint`. Its first run found six drifts, all fixed in the manual: the web UI's `GET /` and the chat app's two Slack endpoints were served but undocumented (sb-api(3), sb-ask(3)); sb-api(3)'s example used `GET /v1/answers`, which doesn't exist (now `POST /v1/ask`); `sb factory verify-export` was missing from sb(1)'s synopsis. Fakes of outside services are excluded, SCIM paths are read relative to their base, and commands that pick their subcommand inside a one-word case (`sb eval run`) are recognised. Today 46 endpoints, 19 commands and 20 flags match. CLAUDE.md's state section is now a six-line summary that points to `epics/README.md`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

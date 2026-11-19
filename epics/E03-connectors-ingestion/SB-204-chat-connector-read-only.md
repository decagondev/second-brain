---
title: "SB-204: Chat connector (read-only)"
summary: "Decisions get made in chat; that knowledge should be findable by those who could see it."
---

# SB-204: Chat connector (read-only)

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.2 First sources |
| Stories | US-03.1 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md), [SB-205](../E03-connectors-ingestion/SB-205-connector-credentials-via-vault-references.md) |
| Status | In review |

## Why

Decisions get made in chat; that knowledge should be findable by those who could see it.

## Acceptance criteria

- [x] Passes the contract kit
- [x] Channel membership maps to ACL; private channels respected
- [x] Direct messages are excluded by default

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Adapter
- [x] Membership mapping
- [x] DM exclusion test
- [x] Contract kit run
- [x] Write or extend the law tests: LAW-01
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: the user chose Slack. `internal/adapters/connectors/slack` is read-only and standard library only, and it passes the contract kit against `slacktest`, a recorded fake of the Web API and the Events API's signed deliveries.
- Each message is a document (thread replies included), and its version is the edit or post time in microseconds.
- ACL: public channel means any full member (guests need membership); private channel means members only; no Slack account means Deny; a missing message or channel means Unknown.
- DMs and group DMs are never read, and `exclude` patterns match channel names.
- Changes come from verified Events API deliveries (v0 signature, 5-minute window) kept in an inbox. Joins and leaves become permission-only changes, and an empty cursor lists everything.

Tests: kit, membership mapping, exclusions, signature and staleness refusals, deletes. `TestLAW03_SlackUnknownIsNeverAllow` (with LAW-01's truth check) runs on it. While writing it, the law world twice mis-modelled faults (a 404 for one of two channels, then a leftover queued fault). The connector was right both times, and the world now injects one fault per channel in the batch.

**Left:** a real Slack workspace run (needs an app with the listed scopes, and a public URL for the Events API), and a durable inbox in the cell database.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

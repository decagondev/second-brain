---
title: "SB-3005: One-source warning"
summary: "An answer resting on one document looks the same as one backed by five."
---

# SB-3005: One-source warning

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.2 |
| Stories | US-30.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Done |

## Why

An answer resting on one document looks the same as one backed by five.

## Acceptance criteria

- [x] When every cite of an answer with at least two claims is one document, the response carries `notice_single` and the UI and chat apps say so first
- [x] Hidden cites don't count
- [x] Three claims from one document warn; two documents or a one-claim answer don't

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notice
- [x] UI and chat apps
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S29.2; tested in internal/core/ask/evidence_test.go (TestAskAboutDocuments, TestOneSourceWarning), tests/spaces/evidence_test.go, internal/adapters/connectors/github/github_test.go (TestLink) and web/src/evidence.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

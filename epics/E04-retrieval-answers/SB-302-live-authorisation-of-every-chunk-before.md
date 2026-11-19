---
title: "SB-302: Live authorisation of every chunk before context"
summary: "This is the line between 'permission-aware' and 'permission-correct'."
---

# SB-302: Live authorisation of every chunk before context

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.1 Retrieval |
| Stories | US-04.3 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-301](../E04-retrieval-answers/SB-301-tenant-scoped-hybrid-retrieval-with-acl.md), [SB-109](../E02-policy/SB-109-policy-engine-v0-authorise-returns-allow.md) |
| Status | Open |

## Why

This is the line between 'permission-aware' and 'permission-correct'.

## Acceptance criteria

- [ ] Every candidate chunk passes `authorise` with a live source `ACL` call before entering context
- [ ] Unknown, timeout and missing source all drop the chunk
- [ ] Law tests assert that the source was asked (not just the prefilter)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Authorisation stage in the ask pipeline
- [ ] Per-request batching
- [ ] LAW-01 and LAW-03 green
- [ ] Ledger update
- [ ] Write or extend the law tests: LAW-01, LAW-03
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

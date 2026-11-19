---
title: "ADR-0002: The source's live ACL is the authority; the index ACL is only a prefilter"
summary: "The source's live ACL is the authority; the index ACL is only a prefilter."
---

# ADR-0002: The source's live ACL is the authority; the index ACL is only a prefilter

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Permissions copied into an index go stale. A user removed from a group can still match an old copy for minutes or hours.

## Decision

At answer time, every candidate document is checked with the source's live `acl` call, batched per connector with an 800 ms deadline. Unknown, timeout and missing all mean refusal. Allow results are never cached across requests. Deny results may be cached for 60 seconds.

## Alternatives rejected

| Option | Why not |
|---|---|
| Trust the index ACL with frequent sync | Unsound in the gap between syncs; violates P1 |
| Cache allows for a short TTL | Even 60 seconds breaks LAW-01's 'at answer time'; the speed gain doesn't justify a soundness hole |

## Consequences

- LAW-01 holds even when the prefilter is stale
- Ask latency depends on source ACL latency; mitigated by batching and parallel fan-out (SB-303)
- Connectors with no usable ACL API can't be supported until one exists. That's a deliberate limit

---
title: "SB-305: Model gateway adapter"
summary: "The model is a replaceable userspace process behind a port."
---

# SB-305: Model gateway adapter

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.5 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | [LAW-07](../../manual/sb-laws.7.md#law-07) |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Open |

## Why

The model is a replaceable userspace process behind a port.

## Acceptance criteria

- [ ] Provider-agnostic `Model` port; one managed provider adapter
- [ ] Prompts loaded from the release bundle only
- [ ] No secret or raw credential can be placed in a prompt (canary test)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Port and adapter
- [ ] Prompt loader
- [ ] Canary test
- [ ] Token and cost metering
- [ ] Write or extend the law tests: LAW-07
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

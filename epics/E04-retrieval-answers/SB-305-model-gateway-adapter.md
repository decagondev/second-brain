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
| Status | Done |

## Why

The model is a replaceable userspace process behind a port.

## Acceptance criteria

- [x] Provider-agnostic `Model` port; one managed provider adapter
- [x] Prompts loaded from the release bundle only
- [x] No secret or raw credential can be placed in a prompt (canary test)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Port and adapter
- [x] Prompt loader
- [x] Canary test
- [x] Token and cost metering
- [x] Write or extend the law tests: LAW-07
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.2): `ports.Model` and `internal/core/gateway`. The gateway refuses any request containing a registered secret (in the system prompt, a message, or split across them) with `EPERM secret_in_prompt` before anything is sent, pins the release's model id, retries rate limits with backoff only within the deadline, and meters `sb_model_tokens_total`. Four adapters under `internal/adapters/model/` (openaicompat, anthropic, groq, openrouter; OpenRouter fallback routing off) pass one contract test against the recorded fake `modeltest`. Prompts come only from the embedded set in `internal/core/prompts`, each with its SHA-256 for the release bundle (SB-503). `TestLAW07_NoSecretInPrompts` is green, and a mutation that calls the provider without the gateway is caught. Real-provider calls need keys and are not part of CI.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

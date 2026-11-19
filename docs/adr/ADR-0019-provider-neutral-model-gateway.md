---
title: "ADR-0019: A provider-neutral model gateway with adapters for OpenAI-compatible APIs, Anthropic, Groq and OpenRouter"
summary: "A provider-neutral model gateway; OpenAI-compatible, Anthropic, Groq and OpenRouter adapters."
---

# ADR-0019: A provider-neutral model gateway

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-19 | Platform lead (the user) |

## Context

Q-04 left the model provider open. Answers (SB-304), agents (SB-306, SB-409) and evals (SB-501) all need a model, and each portfolio company may have its own contract and data terms (Q-10).

## Decision

- One port, `ports.Model`, with chat completion (messages, tools, response format), usage and a stable error vocabulary. The core never imports a provider.
- Adapters, one folder each under `internal/adapters/model/`: **openaicompat** (any OpenAI-compatible endpoint, including self-hosted), **anthropic**, **groq** and **openrouter**. Groq and OpenRouter reuse the OpenAI-compatible wire format with their own base URLs and quirks.
- Each adapter is standard-library HTTP and is tested against a recorded fake of its API. No vendor SDKs.
- The release bundle pins the provider, the model id and the prompts (ADR-0013). A cell's manifest never names a model.
- API keys live at `vault://tenants/TENANT/model` (an addition to the sb-tenant(5) reference list), resolved at call time.
- Embeddings stay on the local hashing embedder for now (the user's choice); the `Embedder` port takes a provider later.

## Alternatives rejected

| Option | Why not |
|---|---|
| One provider only | Locks every portfolio company to one vendor's terms |
| Vendor SDKs | Four SDKs, many modules, and different error models; the wire formats are small |
| An external LLM proxy | Another service to run per cell, and it would sit outside the audit and kill-switch path |

## Consequences

- Adding a provider is a new adapter that passes a shared model contract test.
- Tool-calling semantics differ by provider; the gateway normalises them, and the tool bus (SB-405) sees one shape.

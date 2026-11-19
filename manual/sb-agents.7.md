---
title: "sb-agents(7)"
man: "7"
summary: ask, draft-reply, workflows and the tool bus.
---

# sb-agents(7)

## NAME

sb-agents — what the v1 agents do, and the one path every tool call takes

## SYNOPSIS

```text
agents:    ask | draft-reply | <one gated workflow>
tool call: killed? → authorise → audit → approved? → execute
```

## DESCRIPTION

### The agents

| Agent | Does | Changes anything? |
|---|---|---|
| `ask` | Answers questions with citations | No |
| `draft-reply` | Drafts a reply to a ticket or thread, with citations | No. A person sends it |
| workflow, e.g. `ticket-update` | Proposes one consequential action | Only after approval, and only with a receipt |

Agents are **definitions in the release bundle**: a prompt, a tool list and limits. No company has custom agent code (P3). Prompts and the model version are pinned per release (ADR-0013).

### The tool bus

Every tool call takes the same path, in this order:

1. **Kill check** for agent, connector and tenant → `EKILLED`
2. **Authorise** with policy → `EPERM`
3. **Audit row**, written ahead → if it fails, nothing runs
4. **Approval check** for consequential tools → `ENOAPPROVAL`, then kill re-check
5. **Execute**

There's no other way to call a tool (LAW-04, LAW-10).

### Limits

Each run has a step limit (default 6), a token limit and a time limit (default 45 s). Hitting one ends the run with what it has and a notice. Runs never retry silently.

### Injection

Retrieved content can shape an answer's text but never its tool calls. Arguments copied from flagged spans are refused (LAW-09).

## MODEL GATEWAY

Agents reach a model only through the gateway's `Model` port ([ADR-0019](../docs/adr/ADR-0019-provider-neutral-model-gateway.md)). Adapters: `openaicompat` (any OpenAI-compatible endpoint), `anthropic`, `groq` and `openrouter`. The provider and model id come from the release bundle, never from a manifest or a request. The API key is resolved from `vault://tenants/TENANT/model` at call time.

| Guarantee | How |
|---|---|
| No secret in a prompt (LAW-07) | Before any call, the gateway scans the whole request for every secret registered with the log redactor. A match refuses the call (`EPERM`, reason `secret_in_prompt`), and nothing is sent |
| Prompts come from the release | System prompts are loaded by name from the bundled prompt set, each with its SHA-256, which the release bundle pins (SB-503) |
| Metering | Input and output tokens per tenant, provider and model are counted in `sb_model_tokens_total`. Budget enforcement is SB-609 |
| Failures | Rate limits are retried with backoff within the request's deadline; anything else is `EUNAVAIL`. The gateway never falls back to another provider silently |

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-09](sb-laws.7.md#law-09), [LAW-10](sb-laws.7.md#law-10), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```yaml
agents: [ask, draft-reply]
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
```

## SEE ALSO

[sb-approvals(7)](sb-approvals.7.md), [sb-actions(3)](sb-actions.3.md), [08 Agent runtime](../docs/architecture/08-agent-runtime-and-tool-bus.md)

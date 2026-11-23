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
| `incident-summary` | Engineering: a timeline of an incident, from tickets, chat and docs, every line cited | No |
| `spec-digest` | Product: what a spec decides and what it leaves open | No |
| `close-checklist` | Finance: what is done and what is open for a month-end close | No |
| `runbook-answer` | IT and security: the steps of a runbook for a question, in order | No |
| `handover` | A brief of the documents one person owns, for whoever takes over from them (SB-1707) | No |
| workflow, e.g. `ticket-update` | Proposes one consequential action | Only after approval, and only with a receipt |

Agents are **definitions in the release bundle**: a prompt, a tool list and limits. No company has custom agent code (P3). Prompts and the model version are pinned per release (ADR-0013).

### Department agents (SB-1102)

`incident-summary`, `spec-digest`, `close-checklist` and `runbook-answer` are read-only agents for one department each. Each is a prompt in the release bundle (`internal/core/prompts/set/NAME.md`) and runs on the `ask` agent's path: the same retrieval, space checks and live ACL, the same model gateway and the same composer. They differ from `ask` in three ways:

1. **Only in their template's spaces.** Each belongs to one template of the catalogue (sb-departments(7)): `incident-summary` to `engineering`, `spec-digest` to `product`, `close-checklist` to `finance` and `runbook-answer` to `it-security`. It runs only in a space made from that template whose `agents` list it (the template's default when the manifest enables it, sb-space(5)), and the manifest's `agents` must enable it. Anywhere else, and in any company-wide ask, it is refused with `EINVAL` before retrieval. A space can't borrow another department's agent.
2. **Every claim cited.** A claim the model leaves uncited is dropped from the answer, and counted in `dropped`, rather than labelled. An answer with nothing left is the one uncited "nothing you can open answers this" claim, with its notice (LAW-14).
3. **No tools.** None of them has a tool, so nothing they read can lead to a tool call (LAW-09).

### Handover (SB-1707)

`handover` is a read-only agent in the release bundle (`internal/core/prompts/set/handover.md`) for when someone leaves or changes role. `POST /v1/handover` `{ "person": "usr_…" }` briefs the documents that person owns at the source (a chunk's owner is one of the person's directory emails) **that the asker can open now**: each owned document takes the same space, label, barrier and live ACL checks as a search, as the asker, and the newest twelve go to the model with the bundled prompt. The brief is grouped by location, one line a document, and **every line is cited**: an uncited line is dropped and counted in `dropped`, as for a department agent. `hidden` counts the person's documents the asker can't open, never their titles, so a manager knows to ask for access to be moved before deprovisioning revokes it (LAW-12): the brief works on the person's documents, never with their access. Policy decides who may ask, with the Cedar action `brief_handover` on the person (sb-policy(5)): the person themselves, their manager (SCIM's `manager`), or a steward of a space whose members include one of the person's groups. Anyone else is `EPERM`, the same as an unknown person. It has no tools (LAW-09), is killed with `sb kill agent handover`, writes a trace like an answer, and has an eval suite, `evals/suites/agent-handover.yaml`, in the gate. It isn't named in the manifest's `agents`: like `ask`, every cell has it.

They are asked through `POST /v1/ask` with `"agent": "NAME"` (sb-ask(3)), are killed like any agent (`sb kill agent NAME`), and each has an eval suite, `evals/suites/agent-NAME.yaml`.

### The tool bus

Every tool call takes the same path, in this order:

1. **Kill check** for agent, connector and tenant → `EKILLED`
2. **Authorise** with policy → `EPERM`
3. **Audit row**, written ahead → if it fails, nothing runs
4. **Approval check** for consequential tools → `ENOAPPROVAL`, then kill re-check
5. **Execute**

There's no other way to call a tool (LAW-04, LAW-10).

**`note-publish` (SB-1705).** The one tool a person, not an agent, calls: a steward publishing a reply as a note (sb-spaces(7)). It is in the registry only when the manifest sets `notes: true`. It isn't consequential in the approvals sense, because the decision for notes is that the steward's publish is the approval: the steward confirms the note's text, readers and the count of narrower cites, and the call carries that payload, so the `Executing` audit row binds the note's hash before the write (LAW-15's binding, without a second approver). `call_tool` on `note-publish` is permitted only to a steward of the note's space: the cell passes the space's stewards and the caller's groups in the policy context, and the platform policy refuses anyone else. No model ever calls it, and an agent's output can't reach it: the payload is the steward's own text from their request (LAW-09). In a cell with information barriers a note is placed before it exists by where it will be written (SB-2001): its target `notes/SPACE/qs_ID.md` is held by the spaces whose `scope` holds that path, and owned by the steward publishing it, so the barrier check runs on it as on any target (sb-spaces(7), Information barriers). A steward publishes in their own space's notes; a note whose path a space across a barrier from them holds is refused (`EPERM`), and so is one the cell can't place because its registry is unreadable.

In code, the bus is `internal/core/toolbus`, and it is the **only** caller of a connector's `Act`. A lint (`tools/check/toolbus.py`) fails the build if any other non-test package calls `.Act(`. Tools come from the release bundle's tool registry. Each has a name, a connector kind, and whether it's consequential. A call to a tool that isn't in the registry is `EPERM`, never "best effort".

| Call | Path |
|---|---|
| Not consequential | kill check → `call_tool` authorise → audit row `tool_call`/`Executing` → `Act` → audit row `Ran` (with receipt) or `DidNotRun` |
| Consequential | kill check → `propose` authorise → audit row `tool_call`/`PendingApproval` with the payload hash → stop. It runs only after an approval (sb-approvals(7)), then: kill re-check → audit row `Executing` with the approval id → `Act` with the approval id as idempotency key → `Ran` or `DidNotRun` |

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

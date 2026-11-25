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
| `policy-answer` | People: the policy that applies to a question, with its version and review date, and who to ask; never a named employee's own case (SB-4001) | No |
| `clause-finder` | Legal: the clauses on a topic in the contracts the asker can open, one cited line each, grouped by contract; never legal advice (SB-4001) | No |
| `account-brief` | Sales: an account's open deals, last contact and people, every line cited (SB-4002) | No |
| `case-brief` | Support: a case's history and how similar cases were solved (SB-4002) | No |
| `brand-check` | Marketing: a draft in the question checked against the brand guide, each finding citing the rule; it suggests and never rewrites (SB-4002) | No |
| `sop-steps` | Operations: a procedure's steps in order, each cited, with its version (SB-4003) | No |
| `design-rationale` | Design: why a design is as it is, from specs, decisions and reviews (SB-4003) | No |
| `exec-brief` | Executive: what the leadership team decided this month and what's open; only in a **restricted** `executive` space (SB-4003) | No |
| `research` | Breaks a question into at most 10 sub-questions, then answers each as the asker into one cited report, only after they press **Run** (SB-4201) | No |
| `meeting-prep` | Answers each item of a meeting's agenda, with the decisions and owners behind it (SB-4202) | No |
| `handover` | A brief of the documents one person owns, for whoever takes over from them (SB-1707) | No |
| workflow, e.g. `ticket-update` | Proposes one consequential action | Only after approval, and only with a receipt |

Agents are **definitions in the release bundle**: a prompt, a tool list and limits. No company has custom agent code (P3). Prompts and the model version are pinned per release (ADR-0013).

### Department agents (SB-1102)

`incident-summary`, `spec-digest`, `close-checklist` and `runbook-answer`, and since SB-4001 to SB-4003 `policy-answer` (`people`), `clause-finder` (`legal`), `account-brief` (`sales`), `case-brief` (`support`), `brand-check` (`marketing`), `sop-steps` (`operations`), `design-rationale` (`design`) and `exec-brief` (`executive`), are read-only agents for one department each, so every department template has one. Each is a prompt in the release bundle (`internal/core/prompts/set/NAME.md`) and runs on the `ask` agent's path: the same retrieval, space checks and live ACL, the same model gateway and the same composer. They differ from `ask` in three ways:

1. **Only in their template's spaces.** Each belongs to one template of the catalogue (sb-departments(7)): `incident-summary` to `engineering`, `spec-digest` to `product`, `close-checklist` to `finance` and `runbook-answer` to `it-security`. It runs only in a space made from that template whose `agents` list it (the template's default when the manifest enables it, sb-space(5)), and the manifest's `agents` must enable it. Anywhere else, and in any company-wide ask, it is refused with `EINVAL` before retrieval. A space can't borrow another department's agent. `exec-brief` also needs the space's label to be `restricted`, the executive template's default; a looser executive space is refused. A team under a department (sb-spaces(7), The tree) takes its department's template, so it can run the department's agent.
2. **Every claim cited.** A claim the model leaves uncited is dropped from the answer, and counted in `dropped`, rather than labelled. An answer with nothing left is the one uncited "nothing you can open answers this" claim, with its notice (LAW-14).
3. **No tools.** None of them has a tool, so nothing they read can lead to a tool call (LAW-09).

### Handover (SB-1707)

`handover` is a read-only agent in the release bundle (`internal/core/prompts/set/handover.md`) for when someone leaves or changes role. `POST /v1/handover` `{ "person": "usr_…" }` briefs the documents that person owns at the source (a chunk's owner is one of the person's directory emails) **that the asker can open now**: each owned document takes the same space, label, barrier and live ACL checks as a search, as the asker, and the newest twelve go to the model with the bundled prompt. The brief is grouped by location, one line a document, and **every line is cited**: an uncited line is dropped and counted in `dropped`, as for a department agent. `hidden` counts the person's documents the asker can't open, never their titles, so a manager knows to ask for access to be moved before deprovisioning revokes it (LAW-12): the brief works on the person's documents, never with their access. Policy decides who may ask, with the Cedar action `brief_handover` on the person (sb-policy(5)): the person themselves, their manager (SCIM's `manager`), or a steward of a space whose members include one of the person's groups. Anyone else is `EPERM`, the same as an unknown person. It has no tools (LAW-09), is killed with `sb kill agent handover`, writes a trace like an answer, and has an eval suite, `evals/suites/agent-handover.yaml`, in the gate. It isn't named in the manifest's `agents`: like `ask`, every cell has it.

They are asked through `POST /v1/ask` with `"agent": "NAME"` (sb-ask(3)), are killed like any agent (`sb kill agent NAME`), and each has an eval suite, `evals/suites/agent-NAME.yaml`.

### The registry (SB-4004)

`GET /v1/admin/agents` lists every agent the manifest enables, and `handover`: its department, the spaces that can run it, its runs and the answers rated unhelpful in the last 30 days, its eval result and whether it is killed (sb-admin(3)). A steward sees the same for their space's agents with `GET /v1/spaces/NAME/agents`, and changes which department agents the space lists with a proposal, which an operator approves as a manifest diff. The list is counted from the manifest and the traces, and holds no question or answer.

### Scheduled and triggered agents (SB-4005, SB-4006)

An agent runs on its own only in these three ways. Each runs on the `ask` agent's path as one person, with every check a search makes, writes a trace and an audit row, has no tools, and **sends and publishes nothing**: its answer waits in a steward queue as a draft, and a steward replies with it and publishes it as a note, or discards it (sb-spaces(7)).

| When | Runs | As | Leaves |
|---|---|---|---|
| A space's `brief` day (sb-space(5)), once a week from 08:00 | The space's `brief.agent`, on the fixed question "What happened in this space this week, and what is still open?" | The space's first active steward, by email | A draft titled "Weekly brief: AGENT" in the space's queue |
| The notify job first sees a transcript (`.vtt`, `.srt`, sb-connector(3)) in a space | `ask`, on "What did this meeting decide, and who owns each action?", narrowed to that transcript | The same | A draft titled "Decisions: TITLE", once per transcript and space |
| SCIM adds someone to a group that is a member of a space | No model: the space's start-here list, with its department's (sb-spaces(7), The tree) | The new member, each item checked as them now | One `welcome` notification: "Welcome to finance. Start here: …", the titles they can open, at most 10 and how many more; once per person and space, ever |

A draft's text is the answer's cited claims, one line each with the titles they cite; an uncited claim is left out, and an answer with nothing cited leaves no draft. It ran as one steward, so **only that steward sees its text**: the others see that it waits, who it ran as, and "Only … sees its text" (LAW-12). A brief or a decisions draft runs nothing when its agent is killed, when the month's model spend has reached `budget_usd_month` (sb-tenant(5)), or when the space has no active steward to run it as; for a brief, the space's stewards are told why ("This week's brief didn't run: …"). The welcome is a notification kind a person can turn off (sb-ask(3), Notifications), writes a trace with agent `welcome` that no report counts as an ask, and an audit row (`welcomed`).

### The update agent (SB-4102)

`update` writes a space's weekly status update. It runs only in a space with `updates: true` (sb-space(5)), only where the `use_agent` policy allows it, and never when it is killed. Every Monday, on the notify job's first run after 08:00 in the `digest` time zone, it answers "What changed in this space last week, what did it decide, and how do its goals stand?" as the space's first active steward, with every check a search makes, and leaves a draft titled "Update: week of DATE" in the space's queue. Its claims start "Week: ". A space whose branch, or department, has spent its `budget_usd_month` skips its brief and its update until the month ends, and its stewards are told why (SB-4105; sb-observability(7), Spend by space). Like every scheduled agent it has no tools and publishes nothing: a steward who publishes it writes a note under `notes/SPACE/updates/` (sb-connector(3), Notes), which the department above lists (sb-ask(3), Goals, updates and policies). Its suite is `agent-update` (sb-evals(7)).

### The legwork agents (SB-4201 to SB-4203)

`research` and `meeting-prep` are read-only agents in the release bundle (`internal/core/prompts/set/research.md` and `meeting-prep.md`), of no template: they run in any space the person asks in, or company-wide, only where the `use_agent` policy allows them, and never when killed. Like a department agent, every claim they keep is cited (an uncited claim is dropped and counted in `dropped`), and they have **no tools** (LAW-09).

- **`research`** writes its plan from the question alone, as at most 10 lines, one sub-question each; nothing from a document goes into the plan. Nothing runs until the person presses **Run**. Each sub-question is then asked by `research` on the `ask` path, as the person, and a report stops at the tenant's `research_usd` or the budget (sb-ask(3), Research reports). Its claims start "Finding: ".
- **`meeting-prep`** answers each agenda item of a pasted agenda or the person's own calendar event; the agenda is data, and an instruction in it is answered as a question at most, never followed. Its claims start "Item: ".
- **Action items** are not an agent: the decisions draft of a new transcript (Scheduled and triggered agents, above) also lists the meeting's action items, read from the text without a model, and they reach their owners only after the steward confirms them (sb-ask(3), Action items).

A report or brief is stored for its owner, and every cite in it is checked again as the reader whenever it is shown (LAW-26). Their suites are `agent-research` and `agent-meeting-prep` (sb-evals(7)).

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

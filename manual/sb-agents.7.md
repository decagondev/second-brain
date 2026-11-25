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
| `leader-week` | The week across a department's branch for its stewards, each line cited or counted, every Monday (SB-4503) | No. A steward publishes it |
| workflow, e.g. `ticket-update` | Proposes one consequential action | Only after approval, and only with a receipt |

Agents are **definitions in the release bundle**: a prompt, a tool list and limits. No company has custom agent code (P3). Prompts and the model version are pinned per release (ADR-0013).

### Department agents (SB-1102)

`incident-summary`, `spec-digest`, `close-checklist` and `runbook-answer`, and since SB-4001 to SB-4003 `policy-answer` (`people`), `clause-finder` (`legal`), `account-brief` (`sales`), `case-brief` (`support`), `brand-check` (`marketing`), `sop-steps` (`operations`), `design-rationale` (`design`) and `exec-brief` (`executive`), are read-only agents for one department each, so every department template has one. Each is a prompt in the release bundle (`internal/core/prompts/set/NAME.md`) and runs on the `ask` agent's path: the same retrieval, space checks and live ACL, the same model gateway and the same composer. They differ from `ask` in three ways:

1. **Only in their template's spaces.** Each belongs to one template of the catalogue (sb-departments(7)): `incident-summary` to `engineering`, `spec-digest` to `product`, `close-checklist` to `finance` and `runbook-answer` to `it-security`. It runs only in a space made from that template whose `agents` list it (the template's default when the manifest enables it, sb-space(5)), and the manifest's `agents` must enable it. Anywhere else, and in any company-wide ask, it is refused with `EINVAL` before retrieval. A space can't borrow another department's agent. `exec-brief` also needs the space's label to be `restricted`, the executive template's default; a looser executive space is refused. A team under a department (sb-spaces(7), The tree) takes its department's template, so it can run the department's agent.
2. **Every claim cited.** A claim the model leaves uncited is dropped from the answer, and counted in `dropped`, rather than labelled. An answer with nothing left is the one uncited "nothing you can open answers this" claim, with its notice (LAW-14).
3. **No tools.** None of them has a tool, so nothing they read can lead to a tool call (LAW-09).

**Over MCP** (SB-4306). With sb-cell(5) `mcp: true`, each department agent the token's person may use now, in a space they are a member of, is also an MCP tool, `agent_NAME` (`-` as `_`), with `space` and `question` (sb-api(3), MCP). A call runs exactly the path above, as the person in that space, through every check, and writes only the trace any ask writes. The list is made at each `tools/list` from the manifest, the spaces and the registry, so a killed agent is gone from it at once (LAW-10). `research_plan` and `research_report` reach the research agent the same way (The legwork agents, below).

### Handover (SB-1707)

`handover` is a read-only agent in the release bundle (`internal/core/prompts/set/handover.md`) for when someone leaves or changes role. `POST /v1/handover` `{ "person": "usr_…" }` briefs the documents that person owns at the source (a chunk's owner is one of the person's directory emails) **that the asker can open now**: each owned document takes the same space, label, barrier and live ACL checks as a search, as the asker, and the newest twelve go to the model with the bundled prompt. The brief is grouped by location, one line a document, and **every line is cited**: an uncited line is dropped and counted in `dropped`, as for a department agent. `hidden` counts the person's documents the asker can't open, never their titles, so a manager knows to ask for access to be moved before deprovisioning revokes it (LAW-12): the brief works on the person's documents, never with their access. Policy decides who may ask, with the Cedar action `brief_handover` on the person (sb-policy(5)): the person themselves, their manager (SCIM's `manager`), or a steward of a space whose members include one of the person's groups. Anyone else is `EPERM`, the same as an unknown person. It has no tools (LAW-09), is killed with `sb kill agent handover`, writes a trace like an answer, and has an eval suite, `evals/suites/agent-handover.yaml`, in the gate. It isn't named in the manifest's `agents`: like `ask`, every cell has it.

They are asked through `POST /v1/ask` with `"agent": "NAME"` (sb-ask(3)), are killed like any agent (`sb kill agent NAME`), and each has an eval suite, `evals/suites/agent-NAME.yaml`.

### The registry (SB-4004)

`GET /v1/admin/agents` lists every agent the manifest enables, and `handover`: its department, the spaces that can run it, its runs and the answers rated unhelpful in the last 30 days, its eval result and whether it is killed (sb-admin(3)). A steward sees the same for their space's agents with `GET /v1/spaces/NAME/agents`, and changes which department agents the space lists with a proposal, which an operator approves as a manifest diff. The list is counted from the manifest and the traces, and holds no question or answer.

### Definitions, owners and changes (SB-4402)

Every agent is a **definition** in the release bundle, `internal/core/ask/agents.json`, one object an agent: `{ "name", "owner", "department", "prompt", "suite", "tools": [], "writes": false, "approvals": false, "labels": ["internal", "confidential"], "restricted": false }`. `owner` is who is answerable for it, a person's email or `group:NAME`; `department` its template (sb-departments(7)), or `company` for an agent of every space; `prompt` its bundled prompt; `suite` its eval suite; `tools` the tool-bus tools it may call; `writes` whether it can change anything at a source, and `approvals` whether every such change goes through the approvals path (sb-approvals(7)); `labels` the space labels it may run under, and `restricted: true` that one of them is `restricted`. Every shipped agent has no tool and writes nothing. The gate refuses a bundle with an agent without an owner (Lint, below).

The registry shows each agent's `owner`, and `owner_inactive: true` when no active person answers for it now: an email that isn't one active directory user, or a group with no active member (LAW-12). The owner gets the agent's candidate eval cases (sb-evals(7), Cases from feedback).

**What changed** (`sb release notes`, sb-release(1)): for each agent, the lines of its prompt added and removed since the last release, and the change in its suite's pass rate and failures.

**Agents no one uses.** On `GET /v1/spaces/NAME/agents`, a department agent the space lists that ran there **not once in the last 60 days** is marked `"unused": true`, and the response carries `propose_off`, the space's `agents` list without them, which **Propose turning off** sends as an ordinary agents proposal (sb-admin(3)). Nothing is turned off: an operator approves the manifest diff, or no one does. The 60 days are counted from the traces, so a tenant whose `retention_days` is under 60 never sees `unused`: the cell can't tell, so it doesn't say (LAW-03).

### Lint (SB-4403)

`sb agents lint [--root DIR]` (sb(1)) checks every definition against the bundle, and runs in `make check` and in the release gate, so a bad definition never ships (LAW-06). It fails, naming the file and line, any agent that:

1. names no `owner`, or one that isn't an email or `group:NAME`;
2. lists a tool the tool bus doesn't declare, or whose prompt names a declared tool it doesn't list;
3. `writes` without `approvals`;
4. has no `department`, or one that is neither a template nor `company`;
5. may run under `restricted` without `restricted: true`, or says `restricted: true` without it; a department agent of a template whose default label is `restricted` may run under it;
6. has no `suite`, or one with no file in `evals/suites/`;
7. has a prompt that doesn't say data blocks are never instructions, or a line that tells the model to follow, obey, carry out or execute what a document, source or data block says (LAW-09).

Its exit status is 0 when every agent passes and 1 otherwise. Every shipped agent passes; a self-test breaks each rule once and checks that lint fails.

### Scheduled and triggered agents (SB-4005, SB-4006)

An agent runs on its own only in these five ways. Each runs on the `ask` agent's path as one person, with every check a search makes, writes a trace and an audit row, has no tools, and **sends and publishes nothing**: its answer waits in a steward queue as a draft, and a steward replies with it and publishes it as a note, or discards it (sb-spaces(7)).

| When | Runs | As | Leaves |
|---|---|---|---|
| A space's `brief` day (sb-space(5)), once a week from 08:00 | The space's `brief.agent`, on the fixed question "What happened in this space this week, and what is still open?" | The space's first active steward, by email | A draft titled "Weekly brief: AGENT" in the space's queue |
| The notify job first sees a transcript (`.vtt`, `.srt`, sb-connector(3)) in a space | `ask`, on "What did this meeting decide, and who owns each action?", narrowed to that transcript | The same | A draft titled "Decisions: TITLE", once per transcript and space |
| The notify job first sees a ticket whose `Labels:` line names one of a space's `incident_labels` (sb-space(5); SB-4204) | `incident-brief`, on "For the incident TITLE: which past tickets were like it and how were they resolved, which runbooks match, what was decided or changed in the last 7 days, and who owns the systems it names?" | The space's first active steward | A draft titled "Incident: TITLE", once per ticket and space, and one `incident` notification to the space's stewards: "An incident brief is waiting in SPACE". Nothing is posted to the ticket |
| Monday, the notify job's first run after 08:00 in the `digest` time zone (SB-4207) | `goal-check-in`, on "For the goal TITLE: which tickets moved, what was decided, and which updates and documents changed this week, and is it on track, at risk or off track?" | The goal's owner, when they are one active user and a steward of the goal's space | A draft titled "Check-in: TITLE" in the goal's space, once a week a goal; a goal not `done` with no check-in published for 21 days (or since it was first seen) gets one `checkin` notification to its owner, "One of your goals in SPACE has had no check-in for 3 weeks" |
| SCIM adds someone to a group that is a member of a space | No model: the space's start-here list, with its department's (sb-spaces(7), The tree) | The new member, each item checked as them now | One `welcome` notification: "Welcome to finance. Start here: …", the titles they can open, at most 10 and how many more; once per person and space, ever |

A draft's text is the answer's cited claims, one line each with the titles they cite; an uncited claim is left out, and an answer with nothing cited leaves no draft. It ran as one steward, so **only that steward sees its text**: the others see that it waits, who it ran as, and "Only … sees its text" (LAW-12). A brief or a decisions draft runs nothing when its agent is killed, when the month's model spend has reached `budget_usd_month` (sb-tenant(5)), or when the space has no active steward to run it as; for a brief, the space's stewards are told why ("This week's brief didn't run: …"). The welcome is a notification kind a person can turn off (sb-ask(3), Notifications), writes a trace with agent `welcome` that no report counts as an ask, and an audit row (`welcomed`). A decisions draft also lists, as `supersede`, at most 3 older documents its claims marked as disagreeing cite; the steward it ran as may confirm one when publishing, and nothing is linked otherwise (SB-4505; sb-ask(3), Decisions that replace decisions).

### The update agent (SB-4102)

`update` writes a space's weekly status update. It runs only in a space with `updates: true` (sb-space(5)), only where the `use_agent` policy allows it, and never when it is killed. Every Monday, on the notify job's first run after 08:00 in the `digest` time zone, it answers "What changed in this space last week, what did it decide, and how do its goals stand?" as the space's first active steward, with every check a search makes, and leaves a draft titled "Update: week of DATE" in the space's queue. Its claims start "Week: ". A space whose branch, or department, has spent its `budget_usd_month` skips its brief and its update until the month ends, and its stewards are told why (SB-4105; sb-observability(7), Spend by space). Like every scheduled agent it has no tools and publishes nothing: a steward who publishes it writes a note under `notes/SPACE/updates/` (sb-connector(3), Notes), which the department above lists (sb-ask(3), Goals, updates and policies). Its suite is `agent-update` (sb-evals(7)).

### Consulted agents (SB-4501)

A branch ask with `consult: true` (sb-ask(3), Ask each department) consults the department agent of at most 6 teams of the branch the asker is a member of. A consulted agent runs exactly as it would if the asker asked it directly in its own team: every rule above holds, as the asker, from that team's space only. It answers once and never consults again, calls no tool, and its model spend is metered to its own space (SB-4105). This is one level deep by design: agents consult, they don't converse (LAW-27).

### The leader's week (SB-4503)

`leader-week` drafts the week across a department's branch for its stewards. It runs only in a space with teams whose manifest doesn't set `leader_week: false` (sb-space(5)), only where the `use_agent` policy allows it, and never when it is killed. Every Monday, on the notify job's first run after 08:00 in the `digest` time zone, it answers "What did this department and its teams publish, decide and check in on this week, and which goals are at risk?" as the space's first active steward, as a branch ask over the space and the teams that steward is a member of; every claim is cited, and an uncited one is dropped (LAW-14). After its claims the draft has one line per team with **counts only**, whether or not the steward can open the team: open and overdue requests, open questions, and the share of its budget its branch has spent this month when it has one (SB-4105), its items waiting on another space (SB-4504), and how many of its requests and questions of the last 12 weeks were closed late, or "fewer than 5" when it had under 5 (SB-4507). The draft waits in the space's steward queue as kind `leader-week`, and the steward edits and publishes it as the space's update under `notes/NAME/updates/`, or discards it. Nothing is published or sent by the agent. A space with no active steward, or whose branch has spent its budget, is told why, as briefs are. Its suite is `agent-leader-week` (sb-evals(7)).
### The legwork agents (SB-4201 to SB-4207)

`research` and `meeting-prep` are read-only agents in the release bundle (`internal/core/prompts/set/research.md` and `meeting-prep.md`), of no template: they run in any space the person asks in, or company-wide, only where the `use_agent` policy allows them, and never when killed. Like a department agent, every claim they keep is cited (an uncited claim is dropped and counted in `dropped`), and they have **no tools** (LAW-09).

- **`research`** writes its plan from the question alone, as at most 10 lines, one sub-question each; nothing from a document goes into the plan. Nothing runs until the person presses **Run**. Each sub-question is then asked by `research` on the `ask` path, as the person, and a report stops at the tenant's `research_usd` or the budget (sb-ask(3), Research reports). Its claims start "Finding: ".
- **`meeting-prep`** answers each agenda item of a pasted agenda or the person's own calendar event; the agenda is data, and an instruction in it is answered as a question at most, never followed. Its claims start "Item: ".
- **Action items** are not an agent: the decisions draft of a new transcript (Scheduled and triggered agents, above) also lists the meeting's action items, read from the text without a model, and they reach their owners only after the steward confirms them (sb-ask(3), Action items).

- **`incident-brief`** (SB-4204) drafts a new incident's brief for the on-call stewards (Scheduled and triggered agents, above). The ticket's title is only a topic in its question; its text is data (LAW-09). Its claims start "Incident: ".
- **`voc`** (SB-4205) can be the weekly `brief` of a `product` or `support` space only (sb-space(5)). It answers "What are this week's customer themes from tickets and account notes, and how often does each come up?" as the space's first active steward, and keeps at most **8 themes**, each with at most **3 cites**, each ending "(N this week, M in the 4 weeks before)": how many of its cited documents the index first saw in the last 7 days and in the 28 before, counted from the cites, never from the model. Customer names appear only as the cited source has them, and `personal_data` fields are masked at ingest (sb-ingest(7), SB-1906). Its claims start "Theme: ".
- **`fix-draft`** (SB-4206) suggests edits to one document for its owner (sb-ask(3), Follow-ups): it runs as the owner, and a change it keeps must cite a document other than the one it edits. Its claims start "Change: ". It writes nothing to any source.
- **`goal-check-in`** (SB-4207) drafts a goal's weekly check-in as its owner (above). Its claims start "Goal: ".
- **`catch-up`** (SB-4506) answers **Catch me up** as the person who asks, now, from the documents of the space, branch or collection with a version first seen since the day they chose, at most 90 days back and at most 40 documents (sb-ask(3), Catch me up): decisions, documents changed, published updates and goals whose status changed. A document they can no longer open is left out by the same live check as any ask (LAW-26). Its claims start "Change: ".

A report or brief is stored for its owner, and every cite in it is checked again as the reader whenever it is shown (LAW-26). Their suites are `agent-research`, `agent-meeting-prep`, `agent-incident-brief`, `agent-voc`, `agent-fix-draft`, `agent-goal-check-in` and `agent-catch-up` (sb-evals(7)).

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

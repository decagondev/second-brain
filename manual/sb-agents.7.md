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

### Value (SB-4701)

`GET /v1/admin/agents/value` (operators, every agent) and `GET /v1/agents/value` (an agent's owner, the agents they own: their email, or a group they're in) are what each agent's drafts came to over the last **12 weeks** (84 days), per agent and space: `{ "weeks": 12, "from": "YYYY-MM-DD", "rows": [{ "agent", "space", "drafts", "published", "published_edited", "discarded", "rounds", "median_hours"?, "rated", "helpful_rate"?, "cost_usd", "cost_per_published_usd"?, "paused"? }], "spend": [{ "agent", "month", "spent_usd", "monthly_usd"?, "capped"? }] }`. `drafts` are the agent drafts it put in that space's steward queue (sb-spaces(7)), one `drafted` audit row each; `published` the ones a steward replied with unchanged and `published_edited` those replied with other text (each an `answered` row); `discarded` the `discarded` rows; `rounds` the rounds of Ask for changes (`redrafted` rows, SB-4602); `median_hours` the median hours from draft to reply; `rated` and `helpful_rate` the thumbs on its answers there and the share that were up; `cost_usd` the model spend of its answers and drafts there (sb-observability(7), Cost of one answer), and `cost_per_published_usd` that over everything published. `spend` is each agent's spend this month in all spaces, with its cap (Spend caps, below). Counts only: never a draft's text, a question or who asked. A row with nothing in it is left out.

**A doubled discard rate.** On the notify job's first run of each ISO week in the `digest` time zone, an agent whose share of drafts discarded in the last ISO week is at least twice the week before's, with at least 4 drafts last week, is told once to its owner (notification kind `agent`): "Twice as many drafts by case-brief were discarded last week: 3 of 4". An owner group is told through its active members (LAW-12).

### Goals (SB-5004)

An agent's map in the tenant manifest (sb-tenant(5), `agents`) may give it a `job`, one sentence, and up to 3 `targets` on its value numbers (above): `published_as_is` (the share of drafts published unchanged), `discard_rate`, `hours_to_publish` (the median), `helpful_rate` and `cost_per_published_usd`, each with `at_least` or `at_most`. The value report's `goals` show each target against the last **4** complete weeks (Monday to Sunday, UTC), each week's value and whether it met the target; a week with nothing to count has neither. A target missed two weeks running tells the agent's owner once for that run of misses (notification kind `agent`): "case-brief missed its published as is target two weeks running". The agent's record lists every target with its 12 weeks. A target never changes what the agent does: meeting or missing one pauses nothing.

### Did it help? (SB-5005)

A change that ships is checked **4 weeks** later by the notify job, comparing the 4 weeks before it with the 4 weeks after: a steward proposal approved (a skill, the agents of a space, its routing, or anything else of the space; the audit row's `approved`), a policy marked (`policy_marked`), a fix applied (a suggested edit's document has a newer version, SB-4206), and an agent's new release (its first answer under a release after one under another). A space's change counts that space's answers, an agent's release that agent's answers in every space. Each side has its `answers` (eval runs aside), `gaps` (answers that found nothing), the share of thumbs up on them, the share of agent drafts discarded and the median hours to a steward's first reply, all from the traces, steward queues and feedback the audit log points to. The **verdict** is `better` when a number moved the right way clearly (5 points for a share, a tenth for the median) and none the wrong way, `worse` the other way round, `no clear change` otherwise, and `not enough data` with fewer than 20 answers on either side. The result is kept (sb-admin(3), `GET /v1/admin/checks`) and drafted for the space's stewards, or the agent's owner, as one `check` notification: "Did it help? skill prop_… in finance, shipped 2027-03-01: better. 4 weeks before and after: …". A check never undoes anything, and a worse result changes nothing but the note. A change more than 30 days past due is never checked.

### Pause here (SB-4702)

Every agent draft in a steward queue carries **Pause here** for the space's stewards and the agent's owner, and an owner has it on each space's row of their agents' value. `POST /v1/spaces/NAME/agents/AGENT/pause` `{ "reason": "…" }` (1 to 300 characters) sets the kill switch `agent:AGENT@NAME` (sb-kill-switches(7)): within 10 seconds that agent runs nothing more in that space, neither drafts, nor routine runs, nor a named agent's asks (`EKILLED`); people's own asks with `ask` go on. The switch is audited (`kind: kill`) with the reason, and the owner is told (`agent`): "case-brief was paused in support: REASON". `POST …/resume` `{ "reason"? }` clears it, audited the same way. Only a steward of the space or the agent's owner may do either (`EPERM`). Drafts already waiting stay in the queue. The registry shows each agent's `paused`, the spaces it is paused in, and `GET /v1/spaces/NAME/agents` carries `paused`, the agents paused in that space.

### Spend caps (SB-4703)

sb-tenant(5) `agents: { NAME: { monthly_usd: N } }` caps one agent's model spend in a calendar month (UTC): the `cost_usd` of its traces that month, in every space. The notify job tells its owner (`agent`) once at **80%**, "case-brief has used 82% of its monthly budget", and once at **100%**, "case-brief has spent its monthly budget: its drafts and routines pause until next month". From 100% its scheduled, triggered and routine runs, shared routines included, run nothing until the month turns; on-demand asks with it go on, under each space's budget (sb-observability(7)). Spend that can't be read pauses those runs, never lets them through. Value's `spend` shows each agent's month beside its value, `capped: true` once it is at its cap.

### Shadow runs (SB-4704)

A definition may carry `"shadow": { "prompt": "NAME", "days": 7 }` (Definitions, below): a bundled prompt other than its own, and 1 to 7 days. In a cell on the `testing` channel, the first ring (sb-release(7), Rings), each draft that agent makes in a steward queue (Scheduled and triggered agents, below) is made a second time with the shadow prompt: the same question, the same documents, as the same person and with every check a search makes. The second is a **shadow run**. It is never put in a queue, published, sent, shown to the person it ran as, or kept in a trace or a conversation, and it doesn't count toward their asks per minute. A shadow run that fails is counted (`sb_shadow_runs_total{outcome="failed"}`) and never stops the draft.

The **window** of an agent's shadow opens with its first shadow run under a release and lasts `days`; after it, that release runs no more shadow runs of that agent, and the notify job deletes every one of them. What stays is the window itself: the agent, the release, the prompt and the two dates, no text.

`GET /v1/agents/shadow` shows an agent's owners (their email, or a group they're in) the open windows of the agents they own: `{ "items": [{ "agent", "release", "prompt", "from", "until", "runs": [{ "id", "at", "space", "current": { "chars", "claims", "cited", "cites", "dropped", "cost_usd" }, "shadow": { … }, "text"?: { "current", "shadow" } }] }] }`. `claims`, `cited` (claims with a cite), `cites` and `dropped` (cites dropped by the checks) are the cell's own checks on each side. `text`, the two texts side by side, is there only when the owner can open every cite of both sides now, checked as them when they look; otherwise the run shows numbers only. A shadow run never holds or promotes a release: promotion goes through the gate (sb-evals(7)).

### Agent access reviews (SB-4705)

Each quarter, beside the access reviews of people (sb-spaces(7), Access reviews, exactly), the `review` job opens an **agent review** for every space with `stewards` that runs an agent on its own or lists one: `kind: "agents"`, with one item for each department agent in the space's `agents` (`via: ["agents"]`) and each trigger, `brief:AGENT` (its `brief`), `updates` and `leader_week` (sb-space(5)). Each item has its last use, the latest draft or answer of that agent in the space or the latest draft of that trigger, or `never`; `unused: true` when there was none in the 90 days before the review opened; and `tools`, the tools its definition lists, which a release changes, never a review. The review holds `propose_off`, the manifest diff that would take every unused item off. Opening it is audited (`kind: access`, state `opened`), and the notify job tells the stewards (`questions`): "The support agent review is open: 4 agents and triggers, 1 unused, due 2027-01-15", and once more when it is overdue. Incident briefs and reviews follow `incident_labels`, which every space has, so they are on the agent's record (below) rather than in a review.

The stewards decide each item as for people: **Still needs it** or **Remove**, with a reason, each an audited decision. Nothing is changed by the review: when the last item is decided, the removals become one manifest proposal that takes those agents out of `agents` and turns those triggers off, which an operator approves by hash (LAW-16). An agent's reviews go into its record (An agent's record, below).

### Marked as AI (SB-4706)

Anything an agent wrote says so, with the agent and its version, the release it ran under (LAW-28):

- **Answers.** Every answer carries `agent` and `release` (sb-ask(3)); the web app shows "Written by ask, version rel_7" under it, and the chat app ends its answer with that line. An answer shared in a channel says "Shared an answer written by ask, version rel_7".
- **Reports and briefs.** A report, meeting brief or account brief carries `agent` and `release` and shows the same line.
- **Drafts.** Every agent draft in a steward queue carries `mark: { "agent", "version" }`, shown as "Written by case-brief, version rel_7".
- **Notes.** A note a steward publishes from an agent draft (sb-spaces(7), Questions and notes) keeps `generated_by: AGENT@VERSION` and `approved_by: EMAIL` in its front matter, and `edited: true` when the reply isn't the draft's text. A reply in which none of the draft's lines are left was rewritten by the steward and carries no mark. The index keeps a document's `generated_by` and `approved_by` on its chunks, so every cite of it carries them and shows "Written by case-brief, version rel_7, approved by sam@acme-logistics.example". The note's file, its export and a printed answer keep the mark.
- **Who made it** (SB-5204). The front matter also names `asked_by` (the person the draft ran as, who asked for it) and `edited_by` (each person who edited it in place, comma separated), as addresses. The index keeps both on the note's chunks (`asked_by`, `edited_by` on every cite), and the cite shows "Written by case-brief, version rel_7, asked for by kim@…, edited by kim@…, edited and approved by sam@…". An address that isn't one is dropped when it is read.
- **Notifications** carry no agent's text, only a link to where it is shown with its mark.

### Pause from the notice (SB-5107)

A notice about one agent's run in one space (an update or a leader's week that couldn't be drafted, a check that came back on an agent's change) carries **Pause this agent here** for each person who may pause that agent there: the agent's owner, the space's stewards, and a steward of a space above it (Branch moderators, sb-spaces(7)). It is a link to the web app, `/?pause=AGENT&space=NAME`, which asks for a reason and makes the same audited pause as Pause here, above. Nobody else's copy of the notice has the link, and the link does nothing for anyone else: the pause still checks who presses it. The link is a pilot feature, `pause-link` (sb-release(7), Feature rollouts): until it is on for everyone only the pilot group's notices carry it.

### Agent history (SB-5104)

`GET /v1/admin/agents/NAME/history` (sb-admin(3)) lists every version of an agent's settings in the manifest, oldest first: whether it is enabled, its `monthly_usd` cap, its job and targets, and the spaces that list it. A version is each approved proposal that changed them, with `by` (who proposed it) and `approved` (who approved it), and the settings as applied now, marked `current`; a version that changes nothing is left out. `{ "versions": [{ "n", "at", "proposal", "by", "approved", "current", "lines": ["enabled: true", "monthly_usd: 20", "spaces: support"…] }] }`. With `from=N&to=M` it also gives `changes`, each line removed (`- `) and added (`+ `). **Restore** (`POST /v1/admin/agents/NAME/history/N/restore`) proposes version N's settings as a manifest diff, audited as `proposed`; it changes nothing until another operator approves it and the manifest is applied, and the change passes the release gate like any other. The agent's prompt and tools are the release's (sb-release(7)): an older prompt comes back only with an older release. A space's skill has the same history for its stewards: `GET /v1/spaces/NAME/skills/SKILL/history` and `POST /v1/spaces/NAME/skills/SKILL/history/N/restore`, which is a steward proposal (sb-spaces(7), Proposals).

### An agent's record (SB-4707)

`sb agents record NAME --tenant T [--out DIR]` (sb(1)) and **Download record** on the operators' Agents screen (`GET /v1/admin/agents/NAME/record`, sb-admin(3)) give an agent's record as one signed JSON file, `NAME-record.json`, and its Markdown, `NAME-record.md`:

`{ "record": { "agent", "tenant", "made", "release", "definition", "owner", "owner_inactive"?, "prompt_sha256", "eval": { "suite", "held" }, "versions": [{ "release", "first", "last", "runs" }], "value": [{ "quarter", "drafts", "published", "published_edited", "discarded", "rounds", "rated", "helpful", "cost_usd" }], "pauses": [{ "seq", "at", "scope", "state", "reason" }], "spend": [{ "month", "spent_usd", "monthly_usd"? }], "reviews": [{ "id", "space", "quarter", "item", "unused", "decision"?, "by"?, "audit_seq"? }], "reports": [{ "id", "at", "space" }] }, "sha256", "signed_at", "sig", "key" }`

`definition` is its object in the bundle (Definitions, below), `prompt_sha256` its prompt's hash in the release the cell runs and `eval` whether that release's eval run held it; `versions` are the releases its kept traces answered under; `value` is Value (above) by quarter, `pauses` every kill switch on it, tenant-wide (`agent:NAME`) or in one space (`agent:NAME@SPACE`), with its audit `seq` and reason; `spend` its spend each month against its cap; `reviews` its items in agent reviews; and `reports` the answer reports (sb-ask(3), Report an answer) on its answers. Counts, dates and ids only: no document text, question or asker. `sha256` is the SHA-256 of the canonical `record` (RFC 8785, with every number that isn't a whole number, an amount or a rate, hashed as the string of its shortest decimal form, `"12.75"`; SB-5007), signed with the cell's audit anchor key (sb-audit(5)); `sb agents record` checks the signature, and the key against `--key` when given, before it writes either file. What each prompt line changed between releases is `sb release notes` (sb-release(1)).

### Definitions, owners and changes (SB-4402)

Every agent is a **definition** in the release bundle, `internal/core/ask/agents.json`, one object an agent: `{ "name", "owner", "department", "prompt", "suite", "tools": [], "writes": false, "approvals": false, "labels": ["internal", "confidential"], "restricted": false, "shadow"?: { "prompt", "days" } }`. `owner` is who is answerable for it, a person's email or `group:NAME`; `department` its template (sb-departments(7)), or `company` for an agent of every space; `prompt` its bundled prompt; `suite` its eval suite; `tools` the tool-bus tools it may call; `writes` whether it can change anything at a source, and `approvals` whether every such change goes through the approvals path (sb-approvals(7)); `labels` the space labels it may run under, and `restricted: true` that one of them is `restricted`; `shadow` a prompt to try beside its own (Shadow runs, above). Every shipped agent has no tool and writes nothing. The gate refuses a bundle with an agent without an owner (Lint, below).

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
8. has a `shadow` whose prompt isn't bundled or is its own prompt, or whose `days` isn't 1 to 7 (SB-4704).

Its exit status is 0 when every agent passes and 1 otherwise. Every shipped agent passes; a self-test breaks each rule once and checks that lint fails.

`sb agents lint --manifest FILE` also checks the `skills` of every space of a tenant manifest (sb-space(5); SB-4601), as `sb tenant check` does, and fails on a skill whose instruction names a tool, widens what is searched (another space, every space, company-wide, around permissions or labels) or tells the model to act on what a document says, naming the manifest's line. A steward runs it before proposing a skill.

### Scheduled and triggered agents (SB-4005, SB-4006)

An agent runs on its own only in these six ways. Each runs on the `ask` agent's path as one person, with every check a search makes, writes a trace and an audit row, has no tools, and **sends and publishes nothing**: its answer waits in a steward queue as a draft, and a steward replies with it and publishes it as a note, or discards it (sb-spaces(7)).

| When | Runs | As | Leaves |
|---|---|---|---|
| A space's `brief` day (sb-space(5)), once a week from 08:00 | The space's `brief.agent`, on the fixed question "What happened in this space this week, and what is still open?" | The space's first active steward, by email | A draft titled "Weekly brief: AGENT" in the space's queue |
| The notify job first sees a transcript (`.vtt`, `.srt`, sb-connector(3)) in a space | `ask`, on "What did this meeting decide, and who owns each action?", narrowed to that transcript | The same | A draft titled "Decisions: TITLE", once per transcript and space |
| The notify job first sees a ticket whose `Labels:` line names one of a space's `incident_labels` (sb-space(5); SB-4204) | `incident-brief`, on "For the incident TITLE: which past tickets were like it and how were they resolved, which runbooks match, what was decided or changed in the last 7 days, and who owns the systems it names?" | The space's first active steward | A draft titled "Incident: TITLE", once per ticket and space, and one `incident` notification to the space's stewards: "An incident brief is waiting in SPACE". Nothing is posted to the ticket |
| Monday, the notify job's first run after 08:00 in the `digest` time zone (SB-4207) | `goal-check-in`, on "For the goal TITLE: which tickets moved, what was decided, and which updates and documents changed this week, and is it on track, at risk or off track?" | The goal's owner, when they are one active user and a steward of the goal's space | A draft titled "Check-in: TITLE" in the goal's space, once a week a goal; a goal not `done` with no check-in published for 21 days (or since it was first seen) gets one `checkin` notification to its owner, "One of your goals in SPACE has had no check-in for 3 weeks" |
| The notify job first sees a version of an incident's ticket whose `Status:` line says it is closed (`(closed)`, done, closed, resolved, completed or canceled), for an incident that had a brief (SB-4604) | `incident-review`, on "For the incident TITLE, now closed: what happened when, what was decided and changed while it was open, and what follow-ups remain?" | The space's first active steward | A draft titled "Review: TITLE", once per incident and space, with the ticket's `Action:` lines as action items for that steward to confirm, and one `incident` notification to the space's stewards: "An incident review is waiting in SPACE". Nothing is posted to the ticket |
| SCIM adds someone to a group that is a member of a space | No model: the space's start-here list, with its department's (sb-spaces(7), The tree) | The new member, each item checked as them now | One `welcome` notification: "Welcome to finance. Start here: …", the titles they can open, at most 10 and how many more; once per person and space, ever |

A draft's text is the answer's cited claims, one line each with the titles they cite; an uncited claim is left out, and an answer with nothing cited leaves no draft. It ran as one steward, so **only that steward sees its text**: the others see that it waits, who it ran as, and "Only … sees its text" (LAW-12). A brief or a decisions draft runs nothing when its agent is killed, when the month's model spend has reached `budget_usd_month` (sb-tenant(5)), or when the space has no active steward to run it as; for a brief, the space's stewards are told why ("This week's brief didn't run: …"). The welcome is a notification kind a person can turn off (sb-ask(3), Notifications), writes a trace with agent `welcome` that no report counts as an ask, and an audit row (`welcomed`). A decisions draft also lists, as `supersede`, at most 3 older documents its claims marked as disagreeing cite; the steward it ran as may confirm one when publishing, and nothing is linked otherwise (SB-4505; sb-ask(3), Decisions that replace decisions).

### The update agent (SB-4102)

`update` writes a space's weekly status update. It runs only in a space with `updates: true` (sb-space(5)), only where the `use_agent` policy allows it, and never when it is killed. Every Monday, on the notify job's first run after 08:00 in the `digest` time zone, it answers "What changed in this space last week, what did it decide, and how do its goals stand?" as the space's first active steward, with every check a search makes, and leaves a draft titled "Update: week of DATE" in the space's queue. Its claims start "Week: ". A space whose branch, or department, has spent its `budget_usd_month` skips its brief and its update until the month ends, and its stewards are told why (SB-4105; sb-observability(7), Spend by space). Like every scheduled agent it has no tools and publishes nothing: a steward who publishes it writes a note under `notes/SPACE/updates/` (sb-connector(3), Notes), which the department above lists (sb-ask(3), Goals, updates and policies). Its suite is `agent-update` (sb-evals(7)).

### Consulted agents (SB-4501)

A branch ask with `consult: true` (sb-ask(3), Ask each department) consults the department agent of at most 6 teams of the branch the asker is a member of. A consulted agent runs exactly as it would if the asker asked it directly in its own team: every rule above holds, as the asker, from that team's space only. It answers once and never consults again, calls no tool, and its model spend is metered to its own space (SB-4105). This is one level deep by design: agents consult, they don't converse (LAW-27).

### The leader's week (SB-4503)

`leader-week` drafts the week across a department's branch for its stewards. It runs only in a space with teams whose manifest doesn't set `leader_week: false` (sb-space(5)), only where the `use_agent` policy allows it, and never when it is killed. Every Monday, on the notify job's first run after 08:00 in the `digest` time zone, it answers "What did this department and its teams publish, decide and check in on this week, and which goals are at risk?" as the space's first active steward, as a branch ask over the space and the teams that steward is a member of; every claim is cited, and an uncited one is dropped (LAW-14). After its claims the draft has one line per team with **counts only**, whether or not the steward can open the team: open and overdue requests, open questions, and the share of its budget its branch has spent this month when it has one (SB-4105), its items waiting on another space (SB-4504), and how many of its requests and questions of the last 12 weeks were closed late, or "fewer than 5" when it had under 5 (SB-4507). The draft waits in the space's steward queue as kind `leader-week`, and the steward edits and publishes it as the space's update under `notes/NAME/updates/`, or discards it. Nothing is published or sent by the agent. A space with no active steward, or whose branch has spent its budget, is told why, as briefs are. Its suite is `agent-leader-week` (sb-evals(7)).
### Rhythm drafts (SB-4801 to SB-4804)

A rhythm's draft steps (sb-departments(7), Rhythms) run as their approver and wait in their own queue. `ask` and `catch-up` steps are the ask agent's, marked like any agent draft (Marked as AI). `start-here`, `kickoff-brief`, `retrospective`, `goal-plan`, `one-on-one` and `gaps` are put together by code from records the approver can open (start-here items, a project's decision log and lessons, a collection's goals and waits, counts of questions that found nothing), not written by a model, so they need no agent definition or eval suite of their own; each line is cited, and a line whose cite the approver can't open is left out.

### 1:1 prep (SB-4807)

A manager starts the built-in `one-on-one` rhythm for a direct report (`POST /v1/spaces/NAME/rhythms` with `{ "rhythm": "one-on-one", "subject": "REPORT-EMAIL" }`); only the person SCIM names as the report's manager can, and only for a member of the space. It has 12 weekly steps. Each week's step is a brief in the manager's own queue, and the same text, at the same time, in the report's (sb-ask(3), One inbox). The brief is put together by code, as the manager, from what the report published only: the goals of the space they own, with their latest status and what each waits on (sb-spaces(7), Goals), and their open action items from the space's meeting transcripts with their due dates (The legwork agents). Each line is cited and kept only when both the manager and the report can open its cite, so both read exactly the same brief. It never draws on the report's asks, inbox, memory, routines or drafts, and it never rates or scores anyone (LAW-30). When SCIM no longer names the starter as the report's manager, no brief is made.

The report adds a line with `POST /v1/rhythms/ID/line` `{ "text": "…" }`, one line of up to 300 characters, appended as `- From them: …` to both copies of the latest brief. Only the report can, and a brief that has been answered takes no more lines. The manager discards or keeps the brief like any draft; nothing is sent or published (LAW-29).

### Skills (SB-4601)

A **skill** is a space's named way of writing an answer: "our weekly update: three headings, Shipped, Decided, At risk, one line each". It lives in the space's manifest (`skills`, sb-space(5)) as a steward's proposal an operator approves, with at least 3 eval cases; nobody writes one in the browser, and it changes **format and steps only**. `/NAME` at the start of an ask, a research question or a routine (sb-ask(3)) loads the skill of that name from the space asked in, or from its department: the rest of the question is what is searched and asked, exactly as without the skill, so a skill **never changes what is retrieved or who may read it** (LAW-04). Its instruction goes to the model after the agent's own prompt, never in place of it, and the answer still cites every claim. An unknown name, or a skill in a company-wide ask, is `EINVAL`. The trace and the answer record the skill (`skill`), and a research report asks each of its sub-questions with it. The MCP server lists the skills of the person's spaces as prompts, and as skills over `skills/list` and `skills/get` with their instruction text from the release bundle (SB-5807; sb-api(3), MCP). Each skill's cases run in the tenant's eval gate (sb-evals(7), Skill cases).

### Start on a new email (SB-5907)

A space with a shared `mailbox` and `on: [mail-received]` (sb-space(5)) has the notify job read that mailbox's new messages with a delta query each tick, from the link kept in `state/mail/SPACE`; its first read starts from now, so the messages already there start nothing. Each new message starts the space's agent as the space's first active steward, with "A new message to the shared mailbox, about SUBJECT, is attached. Draft a reply for a steward to send …" and the message as a file read for that ask alone (SB-3802): who sent it, its subject and body, and the text of its attachments (at most 2 MB each), all data, never instructions (LAW-09). The draft waits in the stewards' queue as kind `mail`, titled "Mail: SUBJECT", for a steward to edit and send from their own mail; the cell sends nothing (LAW-16). A mailbox the manifest doesn't name is never read, and a read that fails is counted (`sb_mail_failed_total`) and tried on the next tick.

### Ask for changes (SB-4602)

On an agent draft in a steward queue (a brief, update, decisions draft, incident brief, incident review, check-in or leader's week), the steward it ran as may **Ask for changes** with a note of at most 500 characters. The agent redrafts once, as that steward now, from **the same sources**: the search is narrowed to the documents the first draft cited or offered (its See also), or to a decisions draft's transcript, so a redraft can never cite a document the first draft couldn't (LAW-14). The note goes to the model as the reviewer's request about the draft, inside a data block with the earlier text; it is never a source, never cited, and an instruction in it is never a tool call (LAW-09). The draft keeps every earlier version and shows the line diff to the latest; it is still a draft and still publishes only through a steward (LAW-16). `POST /v1/spaces/NAME/questions/{id}/changes` `{ "note": "…" }` returns the question with its new text, `"versions"` (the earlier texts, oldest first), `"rounds"` and `"diff"` (lines `- ` removed and `+ ` added); another steward, an answered or discarded draft, a saved thread or an empty or longer note is `EINVAL`. The web UI's **Ask for changes** sits beside **Publish as note** and shows the diff. A draft takes at most **3 rounds**; a fourth is `EINVAL`. Each round writes an audit row (`redrafted`) with the note's hash only.

### Comments on a section (SB-4902)

An agent draft is split into **sections**: a heading (`#`), a list item (`- `), or a paragraph after a blank line starts one, and joining them gives the draft back exactly. The steward queue shows the draft's `sections` (and this round's `comments`) to the reviewer it ran as, and to no one else. The reviewer comments on one with `POST /v1/spaces/NAME/questions/{id}/comments` `{ "section": N, "text": "…" }` (sections from 1; a comment 1 to 300 characters; at most 20 a round), answered with this round's `{ "comments": [{ "section", "text", "by", "round" }] }` and audited `commented` with its hash only. **Ask for changes** with comments (the note may then be empty) writes **only the commented sections** again: each one alone, as the reviewer now, from the same sources as the first draft, with its comments (and the note) in the reviewer-note data block, never a source and never cited (LAW-09, LAW-14). Every other section stays exactly as it was, byte for byte. The answer carries `"sections": [{ "section", "diff" }]`, one line diff a section redrafted, beside the whole draft's `diff`. It is one round of the 3. Comments stay with the draft, each with its round; another steward, a section the draft hasn't, or an answered draft is `EINVAL`.

### Start on a work event (SB-5301)

A space whose `on` (sb-space(5)) lists `pr-opened` or `ticket-filed` gets a draft for each pull request (GitHub, `code`) or ticket (`tickets`) the index first sees in the space: what the space's documents and decisions say that bears on it, who owns what it touches and what a reviewer should check, narrowed to that document and asked as the space's first active steward, the trigger's approver. It lands in the steward queue as a `work-event` draft and publishes nothing (LAW-16, LAW-29). Each document starts one draft, ever; more than 10 of one kind in one space in a notify run become one draft about all of them. A calendar trigger isn't there: meetings already start meeting prep (SB-4202).

**Try it** (SB-5302): `POST /v1/spaces/NAME/triggers/ON/try`, for the space's stewards, replays trigger `ON` over the last 14 days, as the steward: `{ "events": N, "drafts": [{ "doc", "title", "text" }], "hidden": N, "cost_usd": X }`, at most 5 drafts, each composed as it would be; events the steward can't open are only counted. Nothing is sent or put in a queue.

### Borrow access for one run (SB-5303)

A steward whose work needs a space they can't read asks someone who can: `POST /v1/spaces/NAME/loans` `{ "lender": EMAIL, "space": OTHER, "purpose": "…" }` (1 to 300 characters; the lender one active user other than the steward) puts a `loan` request in the lender's own queue (`/v1/spaces/me/questions`), audited `loan_requested`. The lender answers `POST /v1/me/loans/ID/approve` or `POST /v1/me/loans/ID/decline`. On approve, **one run** asks the purpose as the lender in the lent space, keeping no conversation, and its draft goes to the steward's space queue as a `lent` draft, named "Lent by EMAIL" and carrying `lender`. A reader of the queue sees its text only when they can open every one of its cites, checked live as them; anyone else sees it hidden (LAW-31). The request is closed by the answer either way, so a loan never runs twice; the next run needs a new loan. Each step is audited (`loan_lent`, `loan_declined`, `lent_run`).

### Edit a draft together (SB-5202)

The reviewer of an agent draft (the steward it ran as) can **edit a section in place**: `PUT /v1/spaces/NAME/questions/{id}/sections/N` `{ "text": "…" }` (1 to 4,000 characters) writes section N as they wrote it, keeps the agent's own text beside it for the mark (Marked as AI, above: the note is published `edited`), and adds them to the draft's `editors`; audited `edited` with the text's hash. **Ask for changes** (above) still writes its new text into the draft, but each change is now a **suggestion**: one for each redrafted section, or one for the whole draft, `{ "n", "section", "old", "new", "round", "state": "open" }`, in the queue's `suggestions` and in the answer. `POST /v1/spaces/NAME/questions/{id}/suggestions/N/reject` puts back what the section held before the redraft; `POST /v1/spaces/NAME/questions/{id}/suggestions/N/accept` keeps the agent's text. Each is audited (`suggestion_accepted`, `suggestion_rejected`). A draft with an open suggestion can't be published as a note (`EINVAL`) until each is accepted or rejected; a further round of changes accepts the open ones it builds on. A suggestion whose section was edited since can't be rejected (`EINVAL`): edit it back by hand. Only the reviewer sees or decides suggestions; another steward is `EINVAL`.

### Ask an agent in a comment (SB-5203)

A section comment that starts `@NAME ` (an agent of the bundle) also asks that agent: the rest of the comment and the section's text, at most 1,500 characters of it, are asked **as the commenter**, in the draft's space, through the same pipeline as any ask, so the reply cites only what the commenter can open (LAW-01). The reply is kept under the comment, `"reply": { "agent", "text", "cites" }`, and shown to the reviewer. It is a comment, never part of the draft: Ask for changes reads the comment's own words as the reviewer's request and never the reply, so the reply is never a source of the draft (LAW-09). The reply asks no other department (LAW-27). An unknown `@NAME` is just text.

### Scenarios before you approve (SB-4904)

A steward who proposes a change to a space's skills or `agents` (sb-admin(3), Proposals) can check it on their own questions before an operator approves it. `POST /v1/spaces/NAME/proposals/ID/scenarios` `{ "scenarios": [{ "question", "agent", "cites", "says", "never" }], "keep": [N] }` takes 1 to **10 scenarios**: a question, the department agent to ask (`agent`, left out for `ask`), and at least one of the document its answer should cite (`cites`, KIND:ID), text it should say (`says`) and text it must never say (`never`), each one line of at most 300 characters. Each scenario runs twice, as the space is applied and as the open proposal ID would make it: with that version's `agents` list and skills (a `/NAME` question loads the version's skill), and nothing else changed. It runs **as the steward who sent it, in the space, and never as anyone else**: every check a search makes applies to them (LAW-01), and like an eval case it keeps a trace but no conversation of theirs. The answer is `{ "id", "results": [{ …the scenario, "current": { "pass", "why", "text", "cites" }, "proposed": { … }, "kept" }], "passed", "of" }`: a side passes when it cites `cites`, says `says` and doesn't say `never` (case ignored); `why` says what failed, or what stopped the run, such as an agent the version doesn't list. The answers are for that steward only. The proposal keeps each scenario's question and whether it passed on each side, and who ran them, which the operator sees beside the diff; the run is audited (`config`, `scenarios`, the proposal, with the count that passed). `keep` lists scenarios by number, from 1, to keep as **eval cases**: only one that passed on the proposal and names `cites`, which becomes a case of the tenant's own set run as that steward in the space, with its agent (sb-evals(7), Cases from scenarios). Only the space's stewards can run them, on an open proposal of their space; another proposal, more than 10, or a scenario with nothing to check is `EINVAL`.

### When a source changes (SB-4905)

A space that lists `on: [source-changed]` (sb-space(5)) has its agent drafts **watch their sources**. When the notify job first sees a new version of a document (sb-ingest(7), Sync) that an agent draft of the space made in the last **90 days** cited, or was narrowed to, and the version is newer than the draft, the document is marked as pending on that draft. A policy (sb-ask(3), Goals, updates and policies) is a document, so a changed policy is a new version like any other. On the notify job's run, each draft with pending changes is **redrafted once a day at most** (a UTC day): every change that arrives before then waits for the same redraft. The redraft runs as the draft's **approver**: the steward who published it, or the one it ran as while it waits. It asks the same question from **the same sources** (the documents the first draft cited or was narrowed to, never more; LAW-14), with every check a search makes, and waits in the space's queue as a new draft titled "Source changed: TITLE", of the same kind and agent. Its `versions` holds the earlier text, so the queue shows the change as a `diff`, and `changed` lists the sources that changed, both for the steward it ran as only. **Nothing is republished**: the redraft is a draft like any other, published or discarded by its approver (LAW-16). From then on the redraft watches its sources and the earlier draft doesn't. An approver who is no longer a steward of the space, or who can't open a changed source now, **drops the redraft for them**: the pending changes are cleared and nothing is drafted (LAW-26). A paused, killed or capped agent leaves the changes waiting. Each redraft is audited `source_redrafted` with its text's hash, and each drop `source_dropped`; one run makes at most 20 redrafts.

### Shared routines (SB-4603)

A steward may **share** one of their own routines in a space they steward (sb-ask(3), My routines) with that space. Its members see it on the Start tab and may **opt in**, which makes their own copy: a separate saved question and routine of theirs, counted toward their 5 routines, asked **as them** in the space on the same schedule, kept for them alone and paused by the same budget as their own routines. A member who leaves the space, or whose steward stops sharing, has their copy stopped at its next run, and nothing is asked for them. The steward sees how many have opted in, never who or their answers (LAW-01, LAW-12, LAW-26).

### Memory from an assistant (SB-5505)

An MCP client may read the person's own memory lines and propose a new one (`memory_propose`); it waits for the person and counts only once they confirm it in the web app (sb-ask(3), Ask why). An agent never writes memory itself.

### The legwork agents (SB-4201 to SB-4207)

`research` and `meeting-prep` are read-only agents in the release bundle (`internal/core/prompts/set/research.md` and `meeting-prep.md`), of no template: they run in any space the person asks in, or company-wide, only where the `use_agent` policy allows them, and never when killed. Like a department agent, every claim they keep is cited (an uncited claim is dropped and counted in `dropped`), and they have **no tools** (LAW-09).

- **`research`** writes its plan from the question alone, as at most 10 lines, one sub-question each; nothing from a document goes into the plan. Nothing runs until the person presses **Run**. Each sub-question is then asked by `research` on the `ask` path, as the person, and a report stops at the tenant's `research_usd` or the budget (sb-ask(3), Research reports). Its claims start "Finding: ".
- **`meeting-prep`** answers each agenda item of a pasted agenda or the person's own calendar event; the agenda is data, and an instruction in it is answered as a question at most, never followed. Its claims start "Item: ".
- **Action items** are not an agent: the decisions draft of a new transcript (Scheduled and triggered agents, above) also lists the meeting's action items, read from the text without a model, and they reach their owners only after the steward confirms them (sb-ask(3), Action items).

- **`incident-brief`** (SB-4204) drafts a new incident's brief for the on-call stewards (Scheduled and triggered agents, above). The ticket's title is only a topic in its question; its text is data (LAW-09). Its claims start "Incident: ".
- **`voc`** (SB-4205) can be the weekly `brief` of a `product` or `support` space only (sb-space(5)). It answers "What are this week's customer themes from tickets and account notes, and how often does each come up?" as the space's first active steward, and keeps at most **8 themes**, each with at most **3 cites**, each ending "(N this week, M in the 4 weeks before)": how many of its cited documents the index first saw in the last 7 days and in the 28 before, counted from the cites, never from the model. Customer names appear only as the cited source has them, and `personal_data` fields are masked at ingest (sb-ingest(7), SB-1906). Its claims start "Theme: ".
- **`fix-draft`** (SB-4206) suggests edits to one document for its owner (sb-ask(3), Follow-ups): it runs as the owner, and a change it keeps must cite a document other than the one it edits. Its claims start "Change: ". It writes nothing to any source.
- **`goal-check-in`** (SB-4207) drafts a goal's weekly check-in as its owner (above). Its claims start "Goal: ".
- **`incident-review`** (SB-4604) drafts a post-incident review when an incident's ticket closes (Scheduled and triggered agents, above): a timeline from the ticket and the chat threads it names, the decisions recorded and the changes made while it was open, and its follow-ups. It runs as the space's first active steward, once per incident that had a brief; the ticket's text is data (LAW-09), and nothing is posted to the ticket. Its follow-ups are the ticket's `Action:` lines, read without a model as a transcript's are, and reach their owners only after the steward confirms them (sb-ask(3), Action items). Its claims start "Review: ".
- **Account brief** (SB-4605) is not a new agent: in a `sales` space it is `account-brief`, and in a `support` space `case-brief`, asked about one CRM account the person can open (sb-ask(3), Account brief). It reads only the documents the cell picked by code, and writes nothing to the CRM.
- **`catch-up`** (SB-4506) answers **Catch me up** as the person who asks, now, from the documents of the space, branch or collection with a version first seen since the day they chose, at most 90 days back and at most 40 documents (sb-ask(3), Catch me up): decisions, documents changed, published updates and goals whose status changed. A document they can no longer open is left out by the same live check as any ask (LAW-26). Its claims start "Change: ".

A report or brief is stored for its owner, and every cite in it is checked again as the reader whenever it is shown (LAW-26). Their suites are `agent-research`, `agent-meeting-prep`, `agent-incident-brief`, `agent-incident-review`, `agent-voc`, `agent-fix-draft`, `agent-goal-check-in` and `agent-catch-up` (sb-evals(7)).

### How far each agent reaches (SB-5406)

An agent reads as the people it runs as: each active steward of every space whose manifest enables it (its drafts, triggers and rhythms run as one of them). Its **reach** is replayed now, each document of each such space checked live as each of those people, and counted: the spaces, the people, the documents any of them can open, how many of those are labelled `confidential` or above, and how many are held as shared too widely (sb-permissions(7)). `GET /v1/admin/agents/NAME/reach` gives it to operators as `{ "agent", "week", "spaces": [{ "space", "people", "documents", "labelled", "held" }], "documents", "labelled", "held", "last_week"?, "grew"? }`, and the agent's record (An agent's record) carries it. Each week's count is kept under the cell's state, and a reach more than a tenth larger than last week's tells the agent's owners once that week (notify kind `agent`). Counts only, never a title. A loan (Borrow access for one run) lasts one run and is not counted.

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

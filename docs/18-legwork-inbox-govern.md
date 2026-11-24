---
title: Agents That Do the Legwork; Work That Comes to You; Govern Every Agent
summary: Concept for W41 to W43 (proposed 2026-11-24): twenty features from a deeper survey of company OS and second brain tools. Read-only agents that research, prepare and follow up; one inbox, routines and memory for each person; and the controls that keep a growing set of agents honest. No new service, store or Go module.
---

# Agents that do the legwork; work that comes to you; govern every agent

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-24 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [17-org-agents-os.md](17-org-agents-os.md) · [PRD waves](03-prd.md#waves) |

## Summary

W38 to W40 are built and pushed (`c8f2e39`). Spaces are a tree, every department template has a read-only agent, agents run weekly or on two fixed triggers, and goals, updates, policies, requests and budgets live in each space.

The user asked for a deeper and wider look this time: what is good in other company OS and second brain tools, and which further agentic pieces make sense. Earlier rounds surveyed Guru, Glean, Rovo, Slite, Tettra, Notion AI and Tana. This round adds Glean Agents (2026), Microsoft 365 Copilot (Cowork, Researcher, Agent 365), Gemini Enterprise, Atlassian Rovo Studio, Linear, Lattice, Superhuman Docs (Coda), Dust, Slackbot, Guru's knowledge agents, Mem and Reflect.

Three gaps stand out.

1. **Our agents answer, but don't do the legwork.** Others now research for an hour and hand back a cited report, prepare people for meetings, pull action items out of calls and chase them, draft the fix for a stale page, and write a goal's check-in from the work behind it. Each of these is read-only and ends in a draft, so it fits our model.
2. **Work doesn't come to the person.** Drafts to publish, questions to answer, requests, policies to read and approvals each live on their own tab, in each space. Gemini, Copilot and Linear all now give a person one inbox. People also want recurring questions run for them, a brain that remembers how they like answers, and a place to keep the documents for a piece of work.
3. **More agents need more governance.** Glean scans agent definitions before publish and turns feedback into evals, Copilot gives each agent an owner who reviews its feedback, and everyone now sends signed events out, routes cheap asks to cheaper models and shows guardrail trends. With eighteen agents in the catalogue, we need the same, inside the release gate.

Three waves, twenty tickets: **W41 Agents That Do the Legwork** is first, then **W42 Work That Comes to You**, then **W43 Govern Every Agent**.

## What others do, late 2026

| Product | What it does that matters here |
|---|---|
| **Glean Agents (Go, 2026)** | Agents that own a whole workflow and bring a person in only for a real decision. An on-call agent that reads an escalation across Slack, Jira and GitHub and drafts a fix. Voice-of-the-customer and sales agents. **Agent evals** generated before publish, **agent scanning** that flags over-provisioned write tools or vague write instructions, risky changes gated on a moderator, version checkpoints, skills with rollback, and a **Triggers API** of signed webhooks on content events. Learned memories. An inactive-agents export. |
| **Microsoft 365 Copilot** | **Researcher** for long, cited reports. **Cowork automations** that run a task on a schedule or on an event, with approval before anything is written back. **Skills**: saved, reusable instructions. **Workspaces** that replace Notebooks: a shared set of files and chats for a piece of work. Memory with Purview retention. **Insights Agent V2**: an agent's owner reviews its feedback. Agent 365 tracks every agent's access and value. |
| **Gemini Enterprise** | An **Inbox** that groups agent work into "Needs your input", "Errors" and "Completed". A Deep Research agent over the web and internal data, with citations. Human-in-the-loop checkpoints at chosen steps. A **Data Insights** agent that answers with SQL and charts. An agent registry, gateway and per-agent identity. |
| **Atlassian Rovo (Team '26)** | Rovo Studio with roles, approvals, versioning and insights. Event-triggered automations (a Salesforce status change, a Sev-1). **Record for Agent**: a narrated recording becomes a brief. An org-wide agent inventory: who built it, where it runs, how often. |
| **Linear** | **Write with Agent** for project and initiative updates: a draft from what changed and the linked Slack channel, which the person edits and publishes. A **Priority inbox** that keeps what blocks you above the rest. Initiative health and properties. |
| **Lattice** | Auto-drafted weekly updates from Jira, GitHub and Linear. An AI agent in 1:1s that captures **action items** and talking points. Evidence-based drafts with every claim drawn from the record. Quantitative answers over people data. |
| **Guru** | **Knowledge agents** that verify and unverify content by rule, find stale, conflicting and missing knowledge, and **draft the proposed update**. An expert's correction becomes the truth everywhere it's used. |
| **Slackbot, Dust, Superhuman Docs** | Meeting prep from upcoming work, onboarding guides, deep analysis of long files, answers limited to what you can see. Dust Pods with per-pod usage, and per-edit approval for agents in Office files. Coda's AI columns and blocks over tables. |
| **Mem, Reflect, Tana** | Notes that link themselves, meeting briefs, voice memos to notes, daily notes, scheduled agents that prep and draft updates, and MCP for other tools. |

**What they agree on:** long-running research with citations; agents that prepare, capture and follow up rather than just answer; one inbox for a person's agent and human work; reusable recurring tasks; memory a person can see; per-agent owners, evals from real feedback and checks before an agent ships; signed events out; and cost tiers.

**Where we differ, on purpose:**

- **An agent still runs as a person, never with its own identity.** Glean, Gemini and Copilot now give each agent its own credentials. We keep the person's live ACL as the ceiling (ADR-0002), and add a page that shows a person what every agent did as them (SB-4305).
- **A result read later is checked again.** A report or routine that finishes hours after it started could cite something the reader has since lost. Every cite in a stored result is checked again as the reader when they open it (a new law, LAW-26).
- **Memory holds only what the person told it.** Others learn memories from content. Ours never store document text, so a memory can't outlive the access behind it (a new law, LAW-25).
- **Agents stay reviewed definitions in the release bundle.** No one builds an agent in the browser. Governance here is a gate in CI, not a moderator queue.

## Keeping it small

The rules from earlier waves still apply:

1. **No new service the cell must run.** Research reports, routines and new triggers run in the existing `sb-jobs`.
2. **No new store.** Action items, routines, memory, collections and candidate eval cases are `cell_docs` kinds.
3. **No new third-party Go module.** The calendar connector uses the standard library, as the other vendors do.
4. **At most one setting per feature.**
5. **No new page.** The inbox is the existing Questions tab, renamed **Inbox**. Collections live on the Saved tab.
6. **Nothing widens what anyone sees.** Every title, count and list is checked live as the person.
7. **A person approves every write.** Agents draft; a steward, owner or the person publishes. Nothing here writes to a source.
8. **The tree only narrows** (LAW-24).
9. **Agents are definitions in the release bundle**, each with an eval suite in the gate.

Two more rules for these waves:

10. **Stored results are checked when read.** A report, routine answer, brief or inbox item is checked again as the reader each time it is shown (LAW-26).
11. **Every long run has a ceiling.** A research report has at most 10 sub-questions and a cost cap. A routine runs at most daily. Both stop at the space's or the person's budget.

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-4201 | W41 | Research reports | Everyone | **Research** plans up to 10 sub-questions, shows the plan, then answers each as the asker and writes one cited report, delivered when done | Copilot Researcher, Gemini Deep Research, Glean |
| 2 | SB-4202 | W41 | Meeting prep | Everyone | Paste an agenda or pick a calendar event: a brief of the related documents, recent decisions, open questions and what each attendee owns | Slackbot, Mem, Copilot |
| 3 | SB-4203 | W41 | Action items from meetings | Everyone, steward | A transcript's action items, each with an owner and a due date, listed in the space and sent to their owner, who marks them done | Lattice 1:1s, Tana, Teams recap |
| 4 | SB-4204 | W41 | Incident brief | On-call, Engineering | A third fixed trigger: a ticket labelled Sev-1 drafts a brief (similar incidents, runbooks, recent changes, owners) to the on-call space's queue | Glean on-call agent, Rovo |
| 5 | SB-4205 | W41 | Voice of the customer | Product, Support | A weekly `voc` brief: themes across the tickets and CRM notes the steward can open, each with counts and cites | Glean VoC agent |
| 6 | SB-4206 | W41 | Draft the fix | Owner, steward | For a stale or conflicting document, a suggested edit drawn from the newer source, sent to its owner. The owner changes the source themselves | Guru knowledge agents |
| 7 | SB-4207 | W41 | Goal check-ins | Goal owner | Each goal gets a drafted check-in from the tickets, decisions and updates that cite it. A goal silent for 3 weeks is flagged to its owner | Lattice auto-updates, Linear initiatives |
| 8 | SB-4301 | W42 | One inbox | Everyone | **Inbox** joins what waits on me across spaces: drafts to publish, questions and requests, policies to read, approvals, action items, under Needs you, Failed and Done | Gemini Inbox, Linear Priority |
| 9 | SB-4302 | W42 | My routines | Everyone | Up to 5 saved questions run for me daily or weekly, as me, sent to my chat or email | Copilot automations, Notion skills |
| 10 | SB-4303 | W42 | Memory I control | Everyone | The brain keeps what I tell it about me (role, team, format, language), shows it and forgets on request. Never document text | Glean, Copilot memory, Mem |
| 11 | SB-4304 | W42 | Collections | Everyone | A named set of documents and saved threads for a piece of work, shared with named people, asked as one scope, each item checked live | Copilot Workspaces, Gemini Projects, Dust Pods |
| 12 | SB-4305 | W42 | What agents did as me | Everyone | Every scheduled, triggered, routine and research run made as me: when, which agent, what it read (titles I can open), what it drafted | Agent 365, Glean access auditing |
| 13 | SB-4306 | W42 | Agents over MCP | Everyone | The read-only MCP server (SB-3801) offers each department agent I may use, and research reports, as tools | Glean, Rovo MCP, Lattice MCP |
| 14 | SB-4307 | W42 | Numbers from tables | Everyone | A question about a sheet's numbers is answered by code (count, sum, average, filter) with the cells cited. The model never does the arithmetic | Gemini Data Insights, Lattice |
| 15 | SB-4401 | W43 | Feedback becomes evals | Agent owner | An unhelpful or reported answer becomes a candidate eval case. The owner accepts it with the expected answer, and it joins the suite | Glean agent evals, Copilot Insights V2 |
| 16 | SB-4402 | W43 | Agent owners and changes | Admin, agent owner | Every agent names an owner. A release lists each agent's prompt diff and eval change. An agent unused for 60 days is proposed off | Rovo inventory, Glean inactive agents |
| 17 | SB-4403 | W43 | Agent lint in the gate | Platform | `sb agents lint` fails a definition with a tool it doesn't declare, a write without approval, an unscoped department or a restricted space it can't run in | Glean agent scanning |
| 18 | SB-4404 | W43 | Signed events out | Admin | Webhooks for chosen events (decision recorded, goal at risk, request overdue, policy published), signed, carrying ids only | Glean Triggers API, Rovo |
| 19 | SB-4405 | W43 | Model tiers | Admin, steward | Short plain asks go to a smaller model, research and agents to the main one, each tier gated by its own eval run, with spend per tier | Glean "let Glean decide", Notion |
| 20 | SB-4406 | W43 | Guardrail insights | Security | Weekly counts of injection flags by source and space, flagged uploads and refused tool calls, with the trend. Counts only | Glean Protect |

## W41 Agents that do the legwork (epic E42, SB-4201 to SB-4207)

The goal is that agents prepare, research and follow up for people, and every result is still a cited draft or a message to the person themselves.

### S41.1 Research and prepare

**SB-4201 Research reports.** **Research** on the Answer tab (and `POST /v1/research`) asks the `research` agent to break a question into at most 10 sub-questions. The plan is shown first, and runs only when the person presses **Run**. The job then asks each sub-question on the `ask` path as the person, in the space and scope they chose, and composes one report with a summary, a section per sub-question and every claim cited (an uncited claim is dropped, LAW-14). A run costs at most `research_usd` (default $2) and stops at the person's space budget (SB-4105). It is delivered as a notification (`research`), kept for 30 days on the Saved tab, and checked again as the reader each time it is opened (LAW-26). It never calls a tool.
*Laws:* LAW-01, LAW-14, LAW-26 (new). *Acceptance:* the plan waits for **Run**. A sub-question whose documents the person can't open adds nothing. A cite the reader has since lost is dropped on opening.

**SB-4202 Meeting prep.** **Prepare** takes a pasted agenda, or an event from the person's own calendar through a new read-only calendar connector (Google Calendar first, events only, read as the person). The `meeting-prep` agent returns: the documents about each agenda item, decisions on them in the last 90 days (SB-3702), open steward questions on them, and for each attendee who is a tenant user, what they own among the cited documents (SB-3906). Attendees outside the tenant are named only as written in the event. With `prep: true` in a person's preferences, a brief is sent 30 minutes before each event that has an agenda. The event's text is data, not instructions (LAW-09).
*Laws:* LAW-01, LAW-09, LAW-12. *Acceptance:* a brief cites only what the person can open. An instruction in the agenda is ignored. A deprovisioned attendee isn't shown as an owner.

**SB-4203 Action items from meetings.** When a transcript lands (SB-3701), the decisions draft (SB-4006) gains **action items**: a line each, with the owner as said in the meeting matched to a tenant user, and a due date if one was said. The steward confirms them with the decisions. Confirmed items are `cell_docs` records listed on the space's home. Each owner gets one message (`action`, off-able) and can mark an item done. An item past its date appears in the owner's inbox (SB-4301). The brain doesn't sync them to a tracker.
*Laws:* LAW-01, LAW-16. *Acceptance:* nothing reaches an owner before the steward confirms. An unmatched owner stays as written and gets no message.

### S41.2 Follow up

**SB-4204 Incident brief.** A third fixed trigger beside SB-4006: when the tickets connector sees a ticket with a label in the space's `incident_labels` (default `sev-1`), the `incident-brief` agent drafts, as the space's first active steward: similar past tickets and how they were resolved, the runbooks that match, decisions and changes in the last 7 days, and the owners of the systems named. The draft waits in the on-call space's steward queue and is sent to the stewards at once. The ticket's text is data (LAW-09). One brief per ticket.
*Laws:* LAW-01, LAW-09, LAW-16. *Acceptance:* a Sev-1 drafts one brief within one sync. An injected instruction in the ticket is ignored. Nothing is posted to the ticket.

**SB-4205 Voice of the customer.** The `voc` agent (Product and Support templates) can be a space's weekly brief (SB-4005). It groups the last week's tickets and CRM notes that the running steward can open into at most 8 themes, each with a count, a trend against the previous 4 weeks and three cites. Customer names appear only as they are in the cited source, and `personal_data` fields stay masked (SB-1906).
*Laws:* LAW-01, LAW-07, LAW-14. *Acceptance:* the brief's eval suite passes the gate. Counts match the cited tickets. Masked fields stay masked.

**SB-4206 Draft the fix.** On the review list, a stale document (SB-2701) or one side of a conflict (SB-3605) gains **Draft a fix**. The `fix-draft` agent proposes a suggested edit: the passages to change, the new text and the newer source each change comes from, as a diff. It goes to the document's owner as a message and to their inbox. The owner edits the source themselves. The brain never writes the source, and the suggestion is marked done when the source changes.
*Laws:* LAW-01, LAW-14, LAW-16. *Acceptance:* each change cites its newer source. A source the owner can't open is not used. Nothing is written to the source.

**SB-4207 Goal check-ins.** Each Monday, a goal (SB-4101) gets a drafted check-in for its owner from what cites or links it: tickets moved, decisions, updates and documents changed, with a suggested status. The owner edits and publishes it as a note under the goal (SB-1705), or discards it. A goal with no published check-in for 3 weeks is flagged on its owner's inbox and its space's home. Counts only roll up the tree (SB-3905).
*Laws:* LAW-01, LAW-16, LAW-24. *Acceptance:* the draft cites its lines. Nothing is published without the owner. A silent goal is flagged on week 3.

## W42 Work that comes to you (epic E43, SB-4301 to SB-4307)

The goal is that each person sees in one place what waits on them, has recurring questions answered for them, and can keep context for their work, all checked as them.

### S42.1 One inbox

**SB-4301 One inbox.** The Questions tab becomes **Inbox** (`GET /v1/me/inbox`): one list across every space of what waits on me, under **Needs you** (drafts to publish, questions and requests claimed by or routed to me, policies to read, approvals, action items and suggested fixes), **Failed** (a routine or research run that failed, with why) and **Done** (the last 30 days). Each item is checked as me when shown (LAW-26). Notifications keep counts only. Each item links to where it's acted on; the inbox itself doesn't act.
*Laws:* LAW-01, LAW-07, LAW-26. *Acceptance:* a steward of two spaces sees both spaces' drafts. An item whose document I lost disappears. Counts match each tab.

**SB-4302 My routines.** On the Saved tab, a saved question can become a **routine**: daily on working mornings or weekly on a chosen day, at most 5 per person. The notify job asks it as the person, in its saved space and scope, and sends the answer with its cites to their chat or email (SB-2101), with what changed since the last run first. A routine pauses when the person's space budget is spent, and stops when the person is deprovisioned (LAW-12). Each run is listed in SB-4305.
*Laws:* LAW-01, LAW-12, LAW-26. *Acceptance:* a routine's answer carries only what the person can open at send time. A sixth routine is refused. A deprovisioned person's routines stop.

**SB-4303 Memory I control.** **Remember** on an answer, or a line on the new **About me** section of My data (SB-3501), stores a short fact the person states: their role, team, the format they like, their language, a default space. At most 20 lines of 200 characters, sent with each ask as the person's own context, marked as theirs. A new law, **LAW-25**: *a memory holds only what the person wrote, never text from a document, and it is never sent with anyone else's ask.* It's in My data's export, forgotten with **Forget** (SB-3502), and an admin can turn memory off for the tenant.
*Laws:* LAW-25 (new), LAW-07, LAW-09. *Acceptance:* a pasted document passage is refused as a memory. Another person's ask never carries it. Forget removes it at once.

### S42.2 Context for the work

**SB-4304 Collections.** On the Saved tab, **New collection** names a set of up to 50 documents and saved threads, and shares it with named people (SB-1703's shares). **Ask this collection** scopes an ask to its documents, as **Ask about these documents** does (SB-3004). Each person sees only the items they can open, and others' items are counted, not named. A collection is a list of ids, never copies. A collection unused for 90 days is offered for deletion to its owner.
*Laws:* LAW-01, LAW-13. *Acceptance:* a shared collection shows a reader only what they can open. An ask in it cites nothing outside it.

**SB-4305 What agents did as me.** `GET /v1/me/agent-runs` and **Agent runs** on My data list each run made as me in the last 90 days: scheduled briefs I ran as a steward, triggers, routines and research. Each shows when, which agent, which space, the documents it read (titles I can open now, the rest counted) and what it drafted and where it went. It is drawn from traces and the audit log, not a new record.
*Laws:* LAW-01, LAW-11. *Acceptance:* every run in the traces appears. A steward sees each weekly brief that ran as them.

**SB-4306 Agents over MCP.** The read-only MCP server (SB-3801) lists as tools each department agent the token's person may use in the spaces they're in, plus `research_plan` and `research_report` (SB-4201). A tool call runs exactly as `ask` does for that agent, as the person. Nothing writes. A killed agent disappears from the list at once (LAW-10).
*Laws:* LAW-01, LAW-10. *Acceptance:* a person sees only their spaces' agents. A killed agent's tool is gone on the next list.

**SB-4307 Numbers from tables.** For a question about the numbers in a sheet the person can open (SB-2206), the `ask` agent may call one fixed, read-only tool, `table`: filter rows, then count, sum, average, minimum or maximum a column. The tool reads the stored chunks as the asker, and returns the result with the cell range it used, which the answer cites. The model never states a computed number the tool didn't return (an uncited number is dropped, LAW-14).
*Laws:* LAW-01, LAW-14. *Acceptance:* a sum matches the sheet. A number not from the tool is dropped. Rows from a sheet the asker can't open aren't read.

## W43 Govern every agent (epic E44, SB-4401 to SB-4406)

The goal is that every agent has an owner, real feedback improves its evals, the gate catches a bad definition, and admins see the cost and the risk.

### S43.1 Owners and evals

**SB-4401 Feedback becomes evals.** An answer rated unhelpful (SB-310) or reported (SB-3606) on a named agent becomes a **candidate case** in its owner's inbox: the question, the answer and the cites, with the asker's name removed. The owner accepts it with the expected answer or cites (as `ExpectVersion` does, SB-2904), or dismisses it. Accepted cases join the agent's eval suite for the next release. A case whose documents the owner can't open can't be accepted.
*Laws:* LAW-01, LAW-06, LAW-07. *Acceptance:* an accepted case runs in the next gate. The asker isn't named. A candidate on documents the owner can't open can't be accepted.

**SB-4402 Agent owners and changes.** Every agent in the bundle names an `owner` (a person or a group), and the registry (SB-4004) shows it. `sb release notes` and the admin console list, per agent, the prompt diff since the last release and the change in its eval score. An agent turned on in a space and not run for 60 days is proposed off as a manifest diff to the space's stewards (W12). An agent with no active owner is flagged on the registry.
*Laws:* LAW-06, LAW-16. *Acceptance:* the gate refuses an agent with no owner. A prompt change shows in the release notes. An unused agent is proposed off, not turned off.

**SB-4403 Agent lint in the gate.** `sb agents lint`, run in `make check` and the release gate, fails any agent definition that: names a tool it doesn't declare, can write without the approvals path (sb-approvals(7)), has no department or template, runs in a restricted space without `restricted: true`, has no eval suite, or tells the model to act on document text. Each failure names the line. The selftest breaks each rule once and must go red.
*Laws:* LAW-04, LAW-06, LAW-09. *Acceptance:* each broken definition is refused with its line. Every shipped agent passes.

### S43.2 Out, cost and risk

**SB-4404 Signed events out.** sb-tenant(5) `events: { url, kinds: [...] }`, off by default. On a chosen kind (`decision`, `goal_at_risk`, `request_overdue`, `policy_published`, `action_due`) the cell posts a JSON body with the kind, the space, the item's id and the time, signed with HMAC-SHA256 and a key reference from the vault (SB-205). No title, text or name leaves the cell. The receiver fetches anything more with its own read-only token (SB-2306), so the cell checks it as that token's person. Retries with backoff, and each delivery is audited.
*Laws:* LAW-07, LAW-11. *Acceptance:* a body carries ids only. A bad signature is detectable. A fetch with the receiver's token sees only what its person can open.

**SB-4405 Model tiers.** sb-cell(5) `model.small` names a second model. An ask with no agent, no file and a short question goes to it; research, agents, files and long asks go to the main model. The release gate runs the eval suites on both tiers, and a tier that fails takes no asks. A space can pin `model: main`. The costs page shows spend per tier, and the report shows the share of asks on each.
*Laws:* LAW-06, LAW-07. *Acceptance:* a short ask uses the small model. A failing tier takes no asks. A pinned space uses the main model.

**SB-4406 Guardrail insights.** `GET /v1/admin/guardrails` and a security card in the admin console show, by week for 12 weeks: injection flags at ingest (SB-207) by source and space, flagged uploads (SB-3802), refused tool calls and tool calls a policy denied, and answers whose cites were dropped as uncited. Counts only, with rows under 5 shown as "fewer than 5". A source whose flags double in a week is told to the admin.
*Laws:* LAW-07, LAW-09. *Acceptance:* counts match the audit log. No chunk text appears. A doubling raises one alert.

## What we leave out

- **Agents with their own identity** (Glean, Gemini, Copilot). Agents run as a person, with the source's live ACL as the ceiling (ADR-0002). SB-4305 shows what ran as you.
- **Voice capture and narrated recordings** (Rovo Record for Agent, Mem, Lattice). They need a speech model in the cell. Worth a later wave with a local model, as SB-1710 did.
- **A browser extension** (Glean). The web UI, chat apps and MCP already reach people.
- **Agents built in the browser, and agent-to-agent calls** (Rovo Studio, Copilot Studio, A2A). Agents stay reviewed definitions in the release bundle (P3).
- **Autonomous agents that act across tools** (Glean Go, Rovo Max, Copilot Autopilot). Actions still go through approvals.
- **Performance reviews and 1:1s** (Lattice). That's an HR system's job, and evidence about a named person is not a second brain's to draft.
- **A task tracker.** Action items stay light and never sync out (SB-4203).

## Decisions for the user

Waiting on the user. Each has a recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W41, W42, W43. *Other:* W43 first, so agent governance lands before the new agents.
2. **Research reports (SB-4201).** *Recommended:* the plan is shown and runs only on **Run**, at most 10 sub-questions and $2 a report by default. *Other:* run straight away with no plan step.
3. **Meeting prep (SB-4202).** *Recommended:* a pasted agenda plus a new read-only Google Calendar connector, events only, read as the person. *Other:* pasted agendas only, no new connector.
4. **Action items (SB-4203).** *Recommended:* a steward confirms them with the decisions, then each owner gets one message and marks items done. *Other:* list them in the space only, with no message.
5. **Memory (SB-4303).** *Recommended:* on, holding only what each person writes, shown, exported and forgettable, and an admin can turn it off. *Other:* off by default.
6. **Signed events out (SB-4404).** *Recommended:* off by default, ids only, the receiver fetches with its own token. *Other:* include titles checked as a named service person.
7. **Model tiers (SB-4405).** *Recommended:* a fixed rule (no agent, no file, a short question) routes to the small model, each tier gated by its own eval run. *Other:* one model for everything, as now.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Glean: Agents can now work independently, build faster, and stay governed at scale](https://www.glean.com/blog/glean-agents-go-2026)
- [Glean release notes, August 2026](https://docs.glean.com/release-notes/releases/2026-08-18-august-release)
- [Microsoft Copilot: what's new, October 2026](https://techcommunity.microsoft.com/blog/microsoft-copilot-blog/what's-new-in-microsoft-copilot--october-2026/4568528)
- [Google Cloud: what's new in Gemini Enterprise](https://cloud.google.com/blog/products/ai-machine-learning/whats-new-in-gemini-enterprise)
- [Atlassian: Rovo at Team '26](https://www.atlassian.com/blog/company-news/rovo-team-26)
- [Atlassian: Team '26 Europe, Teamwork Collection](https://www.atlassian.com/blog/company-news/team26-europe-teamwork-collection)
- [Linear: agent-assisted project updates](https://linear.app/changelog/2026-06-18-agent-assisted-project-updates)
- [Linear: Priority inbox](https://linear.app/changelog/2026-09-03-priority-inbox)
- [Lattice: everything announced at Lattiverse '26](https://lattice.com/blog/everything-we-announced-at-lattiverse-2026)
- [Lattice: Fall/Winter 2026 release](https://lattice.com/blog/fall-winter-2026-product-release)
- [Guru: knowledge that maintains itself](https://getguru.com/solutions/km-automation)
- [Superhuman: Coda is now Superhuman Docs](https://blog.superhuman.com/introducing-superhuman-docs/)
- [Dust changelog](https://docs.dust.tt/changelog)
- [Notion releases, April 2026](https://www.notion.com/releases/2026-04-14)
- [Slack: meet Slackbot](https://slack.com/blog/news/slackbot-context-aware-ai-agent-for-work)
- [Tana: best second brain apps in 2026](https://tana.inc/blog/best-second-brain-apps-2026)

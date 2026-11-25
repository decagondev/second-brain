---
title: Departments Work Together; Agents You Can Shape; Every Agent Accountable
summary: Concept for W44 to W46 (accepted 2026-11-25): twenty features toward a company OS whose department agents work as a hierarchy. Agents that consult and hand off between departments as the asker, skills and redrafts that stewards shape, and the value, controls and records that make every agent answer for itself. No new service, store or Go module.
---

# Departments work together; agents you can shape; every agent accountable

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-25 | Platform lead | Accepted by the user 2026-11-25, with every recommended decision | [Principles](01-principles.md) · [17-org-agents-os.md](17-org-agents-os.md) · [18-legwork-inbox-govern.md](18-legwork-inbox-govern.md) · [PRD waves](03-prd.md#waves) |

## Summary

W41 to W43 are built and pushed (`e1e6f7d`). Agents research, prepare and follow up; each person has one inbox, routines and memory; and every agent has an owner, a lint, evals from real feedback, model tiers and guardrail trends.

The user asked for the next twenty features with the same cadence, toward a company OS with department agents in a hierarchy. This round looked again at what shipped in October and November 2026: Microsoft Ignite 2026 (Copilot Home, Today, Workspaces, Autopilot, Agent 365 Business Analytics, Work IQ), Glean's November releases (skills over MCP, pause links, version comparison, chat context panel, Departmental Agent Moderators, spend limits), Gemini Enterprise's unified agent (objectives, subagents, a tasks inbox), Atlassian Team '26 (Rovo Max, Studio, Teamwork Graph decision trails and people signals), Notion Custom Agents and Agent Skills, Slackbot (catch-up, customer briefings, agent orchestration), Linear (Loops, shared templates, triage schedules, spend limits), and McKinsey's *seven operating truths of AI-native companies*.

Three gaps stand out.

1. **Our department agents don't work together.** The tree (W38) lets a leader ask across a branch, and each department has its agent (W39), but a question that spans Finance and Legal gets one agent's view, work that belongs to another department needs the asker to start again there, and a leader assembles the week from each team by hand. Gemini's agent delegates to subagents, Slackbot is becoming one front door to many agents, Rovo Max plans across tools, and Atlassian's graph shows decision trails and who is waiting on whom.
2. **Stewards can't shape what agents do.** Agents are reviewed definitions in the bundle, which is right, but a department can't say "write our weekly update this way", can't send a draft back with a note, and can't share a routine with its team. Copilot, Notion, Gemini and Glean all now have **skills**: named, reusable instructions loaded when needed. Copilot Workspaces and Glean artifacts let a reviewer comment and the agent revise.
3. **Agents can't yet answer for themselves.** We know what an agent costs and whether it passes its evals, but not whether its drafts are used. Agent 365 now measures each agent by work completed, Glean and Linear cap spend per agent and per loop, Glean puts a pause link in every notification, and the EU AI Act's transparency duties (Article 50) apply from August 2026. McKinsey's advice is the same: autonomy follows evidence, and every agent needs a name, a job and an escalation path.

Three waves, twenty tickets: **W44 Departments Work Together** is first, then **W45 Agents You Can Shape**, then **W46 Every Agent Accountable**.

## What others do, late 2026

| Product | What it does that matters here |
|---|---|
| **Microsoft 365 Copilot (Ignite 2026)** | **Home** joins chat and Cowork; **Today** gathers mail, calendar and Teams into the day's work and drafts replies for approval. **Workspaces**: a shared canvas where an agent drafts a section, a person comments, and the agent acts on the comment. **Autopilot**, an always-on agent with its own identity. **Auto** routes each request to a model and effort level. **Agent 365 Business Analytics** measures each agent by work completed, not tokens. Copilotforce adds account briefs and follow-ups from Salesforce. |
| **Glean (November 2026)** | **Skills** reusable from any MCP client, and Deep Research as a skill. A **pause-agent link** in every agent notification, with a feedback prompt. **View changes** between agent versions. A **chat context panel**: agents run, outputs made, sources cited. **Departmental Agent Moderators** turn agents on or off. Spend limits per user and application, a guardrails violation export, an AI transparency notice under the composer, and triggers on email, pull requests and calendar meetings. |
| **Gemini Enterprise (October 2026)** | One agent that takes **objectives** and delegates to **subagents**, with a **tasks inbox** showing the plan, the delegation and the skills loaded. Projects for a workstream. Human-in-the-loop checkpoints. Real-time spend caps and smart routing. |
| **Atlassian Rovo (Team '26)** | **Max mode** turns a request into a multi-step plan across Jira and Confluence, shows it for approval and loops a person back in. Studio agents triggered by a new hire, a Sev-1 or a deal. **Teamwork Graph**: related work, people signals (who knows what), **decision trails** (upstream and downstream, who owns a decision), and an org-wide agent inventory. |
| **Notion (2026)** | **Custom Agents** on cheaper models, a credits dashboard showing what's worth running, **Agent Skills** for weekly updates and meeting briefs, a Salesforce connector for call prep and account plans, and admin approval of MCP tools. |
| **Slackbot (GA 2026)** | **Catch me up** on a project, meeting briefs, customer context briefings from Salesforce and Slack, priorities and reminders, and agent orchestration: one conversational front door that routes to the right agent. |
| **Linear (autumn 2026)** | **Loops** triggered by Slack, GitHub and project events, saved as **shared templates**, with spend limits per workspace, user and loop. Triage schedules, the Priority inbox, and agent-drafted project updates. |
| **McKinsey, AI-native operating truths** | Treat agents as teammates with names, responsibilities and escalation paths. Trust precedes autonomy: approval points first, autonomy earned by feedback. Measure work completed and full cycle time, including review. Centralise the platform, decentralise the tasks, and share prompts between teams. |

**What they agree on:** agents that hand work to other agents and loop a person in; one front door that routes to the right department; catch-up and a leader's view of the week; reusable skills; comment-and-revise on agent drafts; shared recurring tasks; measuring agents by work actually used; a pause button wherever an agent speaks; per-agent spend caps; and clear AI labels.

**Where we differ, on purpose:**

- **Agents consult, they don't converse.** Gemini and Slackbot let agents delegate freely. Ours may consult other department agents one level deep, as the asker, in spaces the asker is a member of, and a consulted agent can't consult again (a new law, **LAW-27**). Chains and agent-to-agent negotiation stay out.
- **Skills live in the manifest, not the browser.** A skill is a steward's proposed manifest diff with eval cases, reviewed like every space change (W12). It changes format and steps, never tools or scope.
- **Autonomy isn't earned automatically.** Others widen what an agent may do as its record improves. We measure the record (SB-4701) and show it to the owner, but a person still publishes every draft.
- **Everything an agent wrote says so** (a new law, **LAW-28**), and the label survives publishing.

## Keeping it small

The rules from earlier waves still apply:

1. **No new service the cell must run.** Consultations, reviews and shadow runs run in the existing `sb-api` and `sb-jobs`.
2. **No new store.** Handoffs are requests (SB-4104), dependencies and supersedes are fields on existing records, skills are manifest entries, and values and dossiers are computed from traces and the audit log.
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** The leader's review lives on the space's home, catch-up on the Answer tab, value and dossiers on the existing agent registry.
6. **Nothing widens what anyone sees.** Every title, count and list is checked live as the person.
7. **A person approves every write.** Agents draft; a steward, owner or the person publishes or sends.
8. **The tree only narrows** (LAW-24).
9. **Agents are definitions in the release bundle**, each with an eval suite in the gate.
10. **Stored results are checked when read** (LAW-26).
11. **Every long run has a ceiling.**

Two more rules for these waves:

12. **One level of consultation.** An agent may consult others; a consulted agent answers from its own space only and never consults (LAW-27).
13. **The AI label never comes off by itself.** Only a person rewriting the text removes it (LAW-28).

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-4501 | W44 | Ask each department | Leader, everyone | A branch ask can **consult** the agents of up to 6 child spaces the asker belongs to, each answering as the asker from its own space, then one answer with a section per department | Gemini subagents, Slackbot orchestration, Rovo Max |
| 2 | SB-4502 | W44 | Hand off to a department | Asker | When an answer says the work belongs to another department, the agent drafts a request to that space (category, text, the cites the asker chose to share). The asker sends it | Slackbot routing, Linear Asks |
| 3 | SB-4503 | W44 | The leader's week | Leader, parent steward | Each Monday a parent space's stewards get a drafted review of the branch: published updates, goal check-ins, overdue requests, incidents, budget burn, each line cited or counted | Copilot Today, Linear initiatives, Lattice |
| 4 | SB-4504 | W44 | Waiting on another department | Steward, goal owner | A request, goal or action item can be marked waiting on another space. Both homes show it, and one waiting past its date goes to the nearest common parent's stewards | Rovo decision trails, Linear |
| 5 | SB-4505 | W44 | Decisions that replace decisions | Everyone, steward | A decision can name the one it replaces. An answer citing a replaced decision shows the newer one, and a topic shows its trail | Rovo decision trails |
| 6 | SB-4506 | W44 | Catch me up | Everyone | **Catch me up** on a space, branch or collection since a date or my last visit: decisions, changed documents, updates, goal changes and what waits on me, cited. Offered once after a pause | Slackbot, Copilot Today |
| 7 | SB-4507 | W44 | Department service levels | Steward, leader | Per space and category, 12 weeks of first reply, time to close and overdue share for requests and questions, rolled up the tree as counts | Service desks, Linear triage |
| 8 | SB-4601 | W45 | Skills | Steward, everyone | A space's named skills ("weekly update, our format") as manifest entries with eval cases. `/skill` on an ask, research or routine loads it. Format and steps only, no tools | Copilot, Notion, Gemini, Glean skills |
| 9 | SB-4602 | W45 | Ask for changes | Steward, owner | On any agent draft, **Ask for changes** with a note; the agent redrafts, shows the diff, up to 3 rounds. The note is data about the draft | Copilot Workspaces, Glean artifact comments |
| 10 | SB-4603 | W45 | Shared routines | Steward, everyone | A steward shares a routine with the space. Each member who opts in gets it run as themselves, and sees only their own answer | Linear shared loops, Notion |
| 11 | SB-4604 | W45 | Post-incident review | On-call, Engineering | When a ticket that had an incident brief closes, a drafted review: timeline, decisions, changes, and follow-ups as action items the steward confirms | Glean on-call agent, incident tools |
| 12 | SB-4605 | W45 | Account brief | Sales, Support | For a CRM account the person can open: open tickets, recent notes, contract documents, renewal date and owners, on demand or before a meeting with that account | Slackbot, Copilotforce, Notion Salesforce |
| 13 | SB-4606 | W45 | How this was made | Everyone | On every answer and draft, a panel: the agent and version, skill, model tier, departments consulted, sub-questions, sources read (titles I can open, the rest counted) and cites dropped | Glean context panel, Gemini tasks inbox |
| 14 | SB-4701 | W46 | Agent value | Agent owner, admin | Per agent and space: drafts made, published as is, published edited, discarded, change rounds, time to publish, helpful rate and cost per published draft | Agent 365 Business Analytics, Notion credits |
| 15 | SB-4702 | W46 | Pause from the message | Steward, agent owner | Every agent message and draft carries **Pause here**: the agent stops in that space at once, a reason goes to its owner, and a steward resumes it | Glean pause link |
| 16 | SB-4703 | W46 | Agent spend caps | Admin | `monthly_usd` per agent. At 80% its owner is told; at 100% its scheduled, triggered and shared runs pause until next month | Linear, Glean, Gemini spend caps |
| 17 | SB-4704 | W46 | Shadow runs | Agent owner | A changed agent runs beside the current one for a week in the first ring. Its drafts are never delivered; the owner sees eval scores, lengths and cite counts side by side | Glean version checkpoints, Agent Optimizer |
| 18 | SB-4705 | W46 | Agent access review | Admin, steward | Every 90 days, each agent's spaces, triggers and tools, with what it actually used. Anything unused is proposed off as a manifest diff | Agent 365, Rovo inventory |
| 19 | SB-4706 | W46 | Marked as AI | Everyone | Every draft, message and published note an agent wrote says so, with the agent and the approver, and keeps it when published | EU AI Act Art. 50, Glean notice |
| 20 | SB-4707 | W46 | An agent's record | Admin, auditor | `sb agents record NAME` writes one file: the definition and owner, its versions, eval scores, value, pauses, access reviews and incidents | Agent 365, ISO/IEC 42001 |

## W44 Departments work together (epic E45, SB-4501 to SB-4507)

The goal is that a question or a piece of work that spans departments reaches the right ones, as the asker, and leaders see the week across their branch without assembling it.

### S44.1 Agents across the tree

**SB-4501 Ask each department.** A branch ask (SB-3903) in a parent space may **consult**: the parent's agent picks at most 6 child spaces whose glossary or agents match (the same rule as SB-4007), and only spaces the asker is a member of. Each consulted space's agent answers the question as the asker, from its own space only, with its own cites. The parent composes one answer with a section per department, and says which departments it didn't consult and why (not a member, no match, paused). A consulted agent never consults (one level only), never calls a tool, and its cost counts against the asker's budget in its own space (SB-4105). A new law, **LAW-27**: *a consultation runs as the asker, one level deep, and adds nothing the asker couldn't get by asking that space directly.*
*Laws:* LAW-01, LAW-24, LAW-27 (new). *Acceptance:* a non-member space is never consulted. A consulted agent's answer equals asking it directly. A consulted agent asked to consult is refused.

**SB-4502 Hand off to a department.** When an answer's composer finds that the question is a request another space takes (its `requests` categories, SB-4104), or the asker presses **Hand off**, the agent drafts a request to that space: the category, a short text, and the cites the asker ticks to share. The receiving steward sees only cites they can open, the rest counted. The asker reviews and sends it; nothing is sent by the agent. The request then works as any other: claims, due dates, escalation.
*Laws:* LAW-01, LAW-16. *Acceptance:* nothing reaches the department before the asker sends it. A shared cite the steward can't open shows as a count.

**SB-4503 The leader's week.** Each Monday, the stewards of a space with children get a drafted review of the branch: the week's published updates (SB-4102) and goal check-ins (SB-4207), goals at risk, overdue requests by space, incident briefs, budget burn (SB-4105) and service levels (SB-4507). Each line cites what it came from or, for a child the steward can't open, shows counts only (SB-3905). The steward edits and publishes it as the parent's update, or discards it. A space can set `leader_week: false`.
*Laws:* LAW-01, LAW-16, LAW-24. *Acceptance:* every line is cited or counted. A child the steward can't open shows counts only. Nothing is published without the steward.

### S44.2 Work across departments

**SB-4504 Waiting on another department.** A request, goal or action item can be marked **waiting on** another space, with a short reason and an expected date. Both spaces' homes list it (titles checked as the reader), roll-ups count cross-space waits, and one still waiting a working day past its date goes to the stewards of the two spaces' nearest common parent, the way escalation climbs (SB-3904). Clearing the wait is one click by either side.
*Laws:* LAW-01, LAW-24. *Acceptance:* both homes show the wait. An overdue wait reaches the common parent's stewards once. A reader who can't open the item sees it counted.

**SB-4505 Decisions that replace decisions.** A decision (SB-3702) can name the one it replaces (`supersedes:` in its front matter, or chosen by the steward when confirming). An answer citing a replaced decision shows "replaced by" with the newer one, if the asker can open it. A topic's **trail** lists its decisions in order, with owners. The decisions draft (SB-4006) suggests a replaced decision when a new one contradicts it (SB-3605's check), and the steward confirms.
*Laws:* LAW-01, LAW-14. *Acceptance:* a cite of a replaced decision carries the newer one. A newer decision the asker can't open isn't named. Nothing is linked without the steward.

**SB-4506 Catch me up.** **Catch me up** on a space, a branch or a collection (SB-4304), since a date or the person's last visit, gives one cited summary: decisions, documents changed, published updates, goals whose status changed, and what waits on the person there (SB-4301). After a notification pause (SB-2506) ends, it's offered once on the inbox. It's an ask on the `catch-up` agent, as the person, at most 90 days back.
*Laws:* LAW-01, LAW-14, LAW-26. *Acceptance:* every line is cited. A document the person lost since is left out. The offer after a pause appears once.

**SB-4507 Department service levels.** `GET /v1/spaces/NAME/service` and a card on the space's home show, by category and week for 12 weeks, requests' and questions' time to first reply, time to close, and the share closed late. A parent sees its children's as counts (SB-3905). The leader's week (SB-4503) and the report use it. Counts only; rows under 5 show as "fewer than 5".
*Laws:* LAW-07, LAW-24. *Acceptance:* the numbers match the requests' audit trail. A small row is masked. A parent sees children only as counts.

## W45 Agents you can shape (epic E46, SB-4601 to SB-4606)

The goal is that stewards and owners shape what agents produce, without building agents in the browser, and anyone can see how an answer or draft was made.

### S45.1 Shape the work

**SB-4601 Skills.** A space's manifest can name up to 10 **skills**: a name, a short instruction (at most 1,000 characters) on format and steps, and at least 3 eval cases. A steward proposes one as a manifest diff (W12), and `sb agents lint` (SB-4403) refuses a skill that names a tool, widens a scope or tells the model to act on document text. `/name` on an ask, a research report or a routine loads it; the trace records it. Skills flow down the tree like glossaries (SB-3902), and the MCP server lists them (SB-4306).
*Laws:* LAW-04, LAW-06, LAW-09. *Acceptance:* a skill naming a tool is refused with its line. A skill's eval cases run in the gate. A skill never changes what is retrieved.

**SB-4602 Ask for changes.** On any agent draft (a brief, update, decisions draft, fix, check-in, incident brief or review), **Ask for changes** takes a note of at most 500 characters. The agent redrafts once from the same sources, shows the diff, and keeps both versions. At most 3 rounds a draft. The note is treated as the reviewer's request about the draft, never as a source, and the redraft can't cite anything the first draft couldn't. Rounds are counted for SB-4701.
*Laws:* LAW-09, LAW-14, LAW-16. *Acceptance:* the redraft cites only what the first could. A fourth round is refused. The diff is shown before publishing.

**SB-4603 Shared routines.** A steward can share a routine (SB-4302) with the space: members see it on the Start tab and **opt in**. Each run is a separate ask as each member, in the space and scope, sent to them alone. A shared routine counts towards each member's 5 routines, pauses with the member's budget, and stops for a member who leaves the space. The steward sees how many have opted in, never their answers.
*Laws:* LAW-01, LAW-12, LAW-26. *Acceptance:* two members with different access get different answers. The steward never sees a member's answer. Leaving the space stops it.

### S45.2 Agents that go further

**SB-4604 Post-incident review.** When a ticket that had an incident brief (SB-4204) closes, the `incident-review` agent drafts, as the space's first active steward: a timeline from the ticket and its linked chat threads (SB-3703), decisions recorded during it, changes in the window, and follow-ups as action items (SB-4203) the steward confirms. It waits in the steward queue. The ticket's text is data (LAW-09). One review per incident; nothing is posted to the ticket.
*Laws:* LAW-01, LAW-09, LAW-16. *Acceptance:* a closed Sev-1 drafts one review. Follow-ups reach owners only after the steward confirms. An instruction in the ticket is ignored.

**SB-4605 Account brief.** **Account brief** (Sales and Support templates) takes a CRM account (SB-610) the person can open and returns: open tickets for it, the last 90 days of notes, the contract and proposal documents that name it, the renewal date if the CRM has one, and the owners among the cited documents (SB-3906). With `prep: true` (SB-4202), a meeting whose attendees share the account's domain gets one 30 minutes before. `personal_data` fields stay masked (SB-1906). It never writes to the CRM.
*Laws:* LAW-01, LAW-07, LAW-14. *Acceptance:* an account the person can't open gives nothing. Masked fields stay masked. The brief's eval suite passes the gate.

**SB-4606 How this was made.** Every answer and draft gains **How this was made**, drawn from its trace: the agent and its version, the skill, the model tier (SB-4405), departments consulted and why others weren't (SB-4501), sub-questions (SB-4201), how many sources were read (titles the viewer can open, the rest counted), the cites dropped as uncited, and the cost. It's checked as the viewer each time it's opened (LAW-26).
*Laws:* LAW-01, LAW-11, LAW-26. *Acceptance:* every field matches the trace. A source the viewer can't open is counted, not named.

## W46 Every agent accountable (epic E47, SB-4701 to SB-4707)

The goal is that every agent's use is measured, any steward can stop it where it speaks, its cost is capped, its changes are tried before they ship, and its record can be handed to an auditor.

### S46.1 Value and control

**SB-4701 Agent value.** The registry (SB-4004) and `GET /v1/admin/agents/value` show, per agent and space, for 12 weeks: drafts made, published as is, published after edits, discarded, change rounds (SB-4602), median time to publish, the helpful rate of its answers, and cost per published draft. The owner sees their agents'; an admin sees all. An agent whose discard rate doubles in a week is told to its owner. Counts only, from traces and the audit log.
*Laws:* LAW-07, LAW-11. *Acceptance:* the counts match the audit log. A doubling tells the owner once. No draft text appears.

**SB-4702 Pause from the message.** Every message and draft an agent sends carries **Pause here** for the space's stewards and the agent's owner. It stops that agent in that space at once, as a kill switch scoped to the space (sb-kill-switches(7)), asks for a reason, sends the reason to the owner, and is audited. A steward of the space or the owner resumes it. Pausing never deletes drafts already waiting.
*Laws:* LAW-10, LAW-11. *Acceptance:* a paused agent runs nothing in that space on the next trigger. The owner gets the reason. Resume restores it.

**SB-4703 Agent spend caps.** sb-tenant(5) `agents: { NAME: { monthly_usd } }`. At 80% the agent's owner is told; at 100% its scheduled, triggered and shared runs pause until the month turns, and on-demand asks keep going under each space's budget (SB-4105). The costs page shows spend per agent beside its value (SB-4701).
*Laws:* LAW-07. *Acceptance:* runs pause at the cap and resume next month. On-demand asks still work. The owner is told at 80%.

### S46.2 Prove it

**SB-4704 Shadow runs.** When a release changes an agent's prompt or skill, `shadow: 7d` in sb-release(7) runs the new version beside the current one in the first ring for a week, on the same triggers and asks. Shadow drafts are never delivered or stored past the week. The owner sees, per run, both eval-judge scores, lengths, cite counts and dropped cites, and side-by-side text only for runs whose every cite the owner can open. Promotion still goes through the gate.
*Laws:* LAW-01, LAW-06. *Acceptance:* no shadow draft reaches anyone. A pair with a cite the owner can't open shows numbers only. Shadow data is gone after the week.

**SB-4705 Agent access review.** Every 90 days, beside people's access reviews (SB-3504), each agent gets one: the spaces it's on, its triggers and its tools, with what it used in the period. Anything unused is proposed off as a manifest diff to the space's stewards; nothing is removed without them. The review and its outcome are audited and go into the agent's record (SB-4707).
*Laws:* LAW-11, LAW-16. *Acceptance:* an unused trigger is proposed off. Nothing is changed without a steward. The outcome is audited.

**SB-4706 Marked as AI.** Every draft, message, email and chat post an agent writes carries "Written by AGENT, version V" and, once a person publishes it, "approved by NAME". A published note keeps `generated_by` and `approved_by` in its front matter, and its cite shows the mark. A new law, **LAW-28**: *anything an agent wrote says so wherever it is shown, until a person rewrites it.* Exports and printouts keep the mark.
*Laws:* LAW-28 (new), LAW-11. *Acceptance:* every agent output in the test cell carries the mark. A published note keeps it. A note a person rewrote loses it.

**SB-4707 An agent's record.** `sb agents record NAME` and the registry's **Download record** write one signed JSON file and its Markdown: the definition and owner, every version with its prompt diff and eval scores (SB-4402), value by quarter (SB-4701), pauses and their reasons (SB-4702), spend against its cap, access reviews (SB-4705) and incidents and reports that named it. Counts and ids only; no document text or asker names.
*Laws:* LAW-07, LAW-11. *Acceptance:* the record matches the audit log. It carries no document text or asker names. Its signature verifies.

## What we leave out

- **Agents with their own identity** (Copilot Autopilot, Gemini). Agents still run as a person (ADR-0002); consultation keeps the asker's identity all the way down.
- **Chains of agents and agent-to-agent negotiation.** One level of consultation only (LAW-27).
- **Autonomy that grows on its own** (Autopilot, Rovo Max). SB-4701 shows the record; a person still publishes.
- **Writing to the CRM or tracker** (Copilotforce, Rovo Max). Account briefs and follow-ups stay read-only drafts.
- **Agents and apps built in the browser** (Copilot Code, Rovo Studio). Skills are as far as shaping goes, through manifest diffs.
- **Rescheduling meetings and drafting email replies for the person** (Copilot Today). That's the mail client's job.
- **Voice capture.** Still waiting on a local speech model.

## Decisions for the user

Decided by the user 2026-11-25: every recommended option.

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W44, W45, W46. *Other:* W46 first, so the controls land before agents consult each other.
2. **Agents consulting agents (SB-4501, LAW-27).** *Recommended:* one level, at most 6 child spaces the asker belongs to, each as the asker, no tools. This reverses W41's "no agent-to-agent calls" in that narrow form. *Other:* keep agents separate; leaders keep using branch asks (SB-3903).
3. **Handoffs (SB-4502).** *Recommended:* the agent drafts the request and the asker sends it. *Other:* the agent sends it straight to the department.
4. **Skills (SB-4601).** *Recommended:* steward-proposed manifest entries with at least 3 eval cases, format and steps only. *Other:* people write their own personal skills in the UI.
5. **Account brief (SB-4605).** *Recommended:* yes, read-only from the CRM connector, with the meeting-day brief on `prep: true`. *Other:* leave it to meeting prep (SB-4202).
6. **Shadow runs (SB-4704).** *Recommended:* a week in the first ring, owner sees numbers, and text only where they can open every cite. *Other:* eval suites only, no shadow runs.
7. **Marked as AI (SB-4706, LAW-28).** *Recommended:* always on, kept in published notes' front matter. *Other:* a tenant setting, on by default.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [SiliconANGLE: At Ignite 2026, Microsoft reinvents Copilot as the autonomous engine of enterprise work](https://siliconangle.com/2026/11/18/at-ignite-2026-microsoft-reinvents-copilot-as-the-autonomous-engine-of-enterprise-work/)
- [Constellation Research: At Ignite 2026, Microsoft makes case to be your work operating system](https://www.constellationr.com/insights/news/ignite-2026-microsoft-makes-case-be-your-work-operating-system-intelligence-layer)
- [Glean release notes, November 2026](https://docs.glean.com/release-notes/)
- [Glean: Agents can now work independently, build faster, and stay governed at scale](https://www.glean.com/blog/glean-agents-go-2026)
- [TechCrunch: Google brings agentic AI to Gemini, starting with businesses](https://techcrunch.com/2026/10/08/google-brings-agentic-ai-to-gemini-starting-with-businesses/)
- [Google Cloud: what's new in Gemini Enterprise](https://cloud.google.com/blog/products/ai-machine-learning/whats-new-in-gemini-enterprise)
- [Atlassian: Rovo at Team '26](https://www.atlassian.com/blog/company-news/rovo-team-26)
- [Atlassian: Teamwork Graph at Team '26](https://www.atlassian.com/blog/company-news/teamwork-graph-team-26)
- [Notion releases, April 2026](https://www.notion.com/releases/2026-04-14)
- [Salesforce: Slackbot general availability](https://investor.salesforce.com/news/news-details/2026/Salesforce-Announces-the-General-Availability-of-Slackbot--Your-Personal-Agent-for-Work/default.aspx)
- [Linear changelog](https://www.linear.app/changelog)
- [McKinsey: The seven operating truths of AI-native companies](https://www.mckinsey.com/capabilities/business-building/our-insights/the-seven-operating-truths-of-AI-native-companies)

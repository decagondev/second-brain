---
title: Ways of Working; Agents That Check With You; Run AI Like a Business
summary: Concept for W47 to W49 (accepted 2026-11-25): twenty features toward a company OS that runs the company's recurring moments. Rhythms for onboarding, kickoffs, planning and reorgs; agents that ask, take comments and run longer with checkpoints; and the usage, routing, limits, goals and outcome checks that make AI spend answer for itself. No new service, store or Go module.
---

# Ways of working; agents that check with you; run AI like a business

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-25 | Platform lead | Accepted by the user 2026-11-25, with every recommended decision | [Principles](01-principles.md) · [18-legwork-inbox-govern.md](18-legwork-inbox-govern.md) · [19-departments-shape-account.md](19-departments-shape-account.md) · [PRD waves](03-prd.md#waves) |

## Summary

W44 to W46 are built and pushed (`ede9846`). Department agents consult each other one level deep and hand work across the tree, stewards shape agents with skills and redrafts, and every agent has a value, a pause button, a spend cap, shadow runs, an access review, an AI mark and a record.

The user asked for the next twenty features with the same cadence, to make the application awesome. This round looked at what shipped from September to November 2026: Lattice's **Pace** (rhythms for people and agents, agents with a job, goal and human manager, Pace Checks), Lattice's Fall/Winter '26 release (auto-updates, evidence-based review drafts, calibration signals), Glean's November releases (comments anchored to artifacts, triggers on new artifact versions, Auto mode routing policies, a weekly admin newsletter), Dust (agents that ask clarifying questions, Pods, per-Pod usage analytics, agent and skill exports), Notion's October release (agents that work for up to 24 hours, Auto routing, the AI command center), Linear (loops from events, credit explorer and spend limits per user and loop), and Gemini Enterprise and Rovo (long-running agents, plans shown for approval, scenario builders, prompt libraries).

Three gaps stand out.

1. **The company's recurring moments still run by hand.** We have every part a new joiner, a project or a quarter needs (start-here packs, collections, goals, check-ins, routines, catch-up), but a steward has to remember to use each one at the right time. Lattice Pace ships more than 60 prebuilt **rhythms** for onboarding, reorgs, kickoffs and check-ins; Rovo Studio agents start on a new hire or a deal; Gemini and Notion run long, multi-step work. Nobody yet does a reorg dry run that shows what moves before the tree changes.
2. **Agents guess instead of checking.** An ambiguous ask gets the agent's best reading, a reviewer can only comment on a whole draft, long research has one fixed ceiling and no checkpoints, and an agent doesn't notice when the document it summarised changes. Dust's agents now ask clarifying questions, Glean anchors comments to sections and triggers agents on new versions, Rovo Max and Gemini show a plan for approval, and Rovo has a scenario builder and a prompt library.
3. **AI spend doesn't yet answer to outcomes.** We cap and measure each agent, but an admin can't see usage by feature and tier in one place, can't choose an economy or frontier routing policy per space, can't give a person a monthly limit, can't set an agent a target, and never learns whether a change actually helped. Notion's AI command center, Glean's routing policies and weekly admin newsletter, Linear's per-user limits and Pace's agent goals and checks all aim at this.

Three waves, twenty tickets: **W47 Ways of Working** is first, then **W48 Agents That Check With You**, then **W49 Run AI Like a Business**.

## What others do, late 2026

| Product | What it does that matters here |
|---|---|
| **Lattice Pace (announced 2026-11-12, early access December)** | **Rhythms**: more than 60 prebuilt ways of working for people and AI (onboarding, reorgs, project kickoffs, manager check-ins), tailored to the company's operating model. Each **agent has a job, a goal and a human manager**, and a Trusted Agent Record tracks manager, goal, cost and results. **Pace Checks**: a recurring loop that tracks progress against goals, flags issues, recommends next steps and measures whether changes improved results. Built in Slack. |
| **Lattice Fall/Winter '26 (September 2026)** | **Auto-updates** drafted from Jira, GitHub, Linear and Notion; weekly drafted updates reviewed in Slack; **evidence-based review drafts**; **calibration signals** flagging possible over- and under-rating; meeting notes and action items; an MCP server listed in Glean's marketplace. |
| **Glean (November 2026)** | **Comments anchored to artifacts**. **Agent trigger on new artifact versions**, and real-time triggers on Outlook email. **Auto mode routing policies: Economy, Balanced, Frontier.** Agent ownership drives moderation and notifications. Pinned Library items. A revamped feature rollouts page and a **weekly admin newsletter**. An AI transparency notice. Skills from any MCP client. |
| **Dust (November 2026)** | **Agents ask you questions to clarify their next steps.** **Pods**: shared project spaces with frames and redesigned member management, and **per-Pod usage analytics**. Admins **export agent and skill lists as CSV**. Destructive MCP tools flagged. Office add-ins. |
| **Notion (October 2026)** | Agents that **work on a task for up to 24 hours**, with a "Check for me" mode and safety checks. An **Auto model router** that balances quality, speed and cost. One points system for all AI use, and a coming **AI command center**: usage by person, feature and agent, spending limits and alerts, and which models and agents teams may use. |
| **Linear (October and November 2026)** | Loops started from Slack and GitHub events and saved as workspace templates, with adjustable reasoning levels. An **AI credit usage explorer** and spend limits per workspace, **user** and loop. Bring your own model key. |
| **Gemini Enterprise and Rovo (2026)** | Long-running agents that work for hours or days, an inbox of agent activity, human-in-the-loop checkpoints, Projects and Canvas for a team's workstream. Rovo Max shows a multi-step plan for approval; Rovo adds a **Scenario Builder** for agents, an agent debug view and a **prompt library** in chat. |

**What they agree on:** prebuilt ways of working that start on a company event; agents that ask before guessing and show a plan before a long run; comments on a section, not the whole draft; agents that wake when their sources change; one view of AI use by person, feature and agent; routing policies and per-person limits; agents with goals; and checking whether a change improved anything.

**Where we differ, on purpose:**

- **A rhythm is a schedule of drafts, not an autopilot.** Pace and Rovo Studio run steps on their own. Every step of ours is a draft a named person approves, and a rhythm stops when its subject leaves or its owner stops it (a new law, **LAW-29**).
- **A brief about a person comes only from what they published, and they see it too** (a new law, **LAW-30**). Lattice drafts reviews and calibration signals from evidence; we leave performance ratings out and keep 1:1 prep to shared work.
- **A reorg is planned before it happens.** Nobody else shows what a move in the tree changes; ours is a dry run, and the tree still only narrows (LAW-24).
- **Agents ask once.** Dust's agents can ask as often as they like. Ours ask at most one question with at most four choices, and scheduled runs never ask; they state their assumption instead.
- **Targets inform, they don't widen.** Like W46, an agent meeting its goal earns nothing automatically; the owner sees the record.

## Keeping it small

The rules from earlier waves still apply:

1. **No new service the cell must run.** Rhythms, checks and digests run in the existing `sb-api` and `sb-jobs`.
2. **No new store.** Rhythms are manifest entries with progress in the audit log; comments are fields on drafts; usage, goals and checks are computed from traces and the audit log.
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** Rhythms show on the space's home and the person's inbox; the command centre is a section of the existing costs page.
6. **Nothing widens what anyone sees.** Every title, count and list is checked live as the person.
7. **A person approves every write.** Agents draft; a steward, owner, manager or the person publishes or sends.
8. **The tree only narrows** (LAW-24).
9. **Agents are definitions in the release bundle**, each with an eval suite in the gate.
10. **Stored results are checked when read** (LAW-26).
11. **Every long run has a ceiling.**
12. **One level of consultation** (LAW-27).
13. **The AI label never comes off by itself** (LAW-28).

Two more rules for these waves:

14. **A rhythm only drafts, and stops with its subject** (LAW-29).
15. **A brief about a person draws only on what they published, and they can read it** (LAW-30).

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-4801 | W47 | Rhythms | Steward, everyone | Named, reviewed playbooks in the manifest: a few dated steps (a draft, a routine, a check-in, a request) started for a person, project or space. Every step is a draft a person approves | Lattice Pace rhythms, Rovo Studio |
| 2 | SB-4802 | W47 | A new joiner's first 90 days | New joiner, manager | The onboarding rhythm: day-one start-here pack, people to meet from who's who, catch-ups at weeks 2 and 6, and the questions they asked that had no answer sent to stewards | Pace onboarding, Rovo new-hire agent |
| 3 | SB-4803 | W47 | Project kickoff | Project lead | Starting a project makes a collection, a goal draft and a decision log, plus a kickoff brief of similar past projects and what they decided | Pace kickoffs, Dust Pods, Gemini Projects |
| 4 | SB-4804 | W47 | Project close-out | Project lead, steward | Closing a project drafts a retrospective: goals met, decisions, open waits, and lessons proposed as notes for the next kickoff | Pace, retrospectives |
| 5 | SB-4805 | W47 | Quarterly planning | Steward, leader | Drafted next-quarter goals per space from check-ins, at-risk goals and request trends; the parent sees proposed goals rolled up, with overlaps between departments flagged | Pace, Linear initiatives |
| 6 | SB-4806 | W47 | Reorg dry run | Admin, leader | `sb tree move --plan` shows what a move or merge changes: inherited glossaries, policies, skills, escalation paths, agents, routines and waits. Nothing applies without a manifest diff | Nobody; reorg rhythms in Pace |
| 7 | SB-4807 | W47 | 1:1 prep | Manager, report | Before a 1:1, a brief from the report's published updates, goals, action items and what waits on them. Both see the same brief; asks, inbox and memory never appear | Lattice 1:1 coaching, Pace check-ins |
| 8 | SB-4901 | W48 | Ask before guessing | Everyone | When an ask is ambiguous (two glossary meanings, several spaces, no period), the agent asks one question with up to 4 choices. Scheduled runs state their assumption instead | Dust clarifying questions |
| 9 | SB-4902 | W48 | Comments on a section | Steward, reviewers | Reviewers comment on a section of any agent draft; **Ask for changes** redrafts only the commented sections, with a diff per section | Glean anchored comments, Copilot Workspaces |
| 10 | SB-4903 | W48 | Long research with checkpoints | Everyone | Research shows its plan for approval, then runs up to 4 hours in stages; at each checkpoint the asker can steer, stop or let it go on. A cost ceiling per run | Notion 24-hour agents, Rovo Max, Gemini |
| 11 | SB-4904 | W48 | Scenarios before you approve | Steward | A steward writes scenarios ("asked X, cites Y, never says Z") and runs them against a proposed skill or agent change in the diff review, before approving it | Rovo Scenario Builder |
| 12 | SB-4905 | W48 | When a source changes | Steward, agent owner | An agent can watch documents; a new version (or a policy that changed) redrafts what depended on it, such as an FAQ note or an account brief, for review | Glean trigger on new versions |
| 13 | SB-4906 | W48 | Starter questions | Everyone, steward | A space's best questions, from saved and helpful asks, curated by the steward and shown on the Answer tab and over MCP | Rovo prompt library, Dust |
| 14 | SB-5001 | W49 | AI command centre | Admin | One section of the costs page: use and spend by space, feature, agent and model tier over 12 weeks, with caps and limits beside it. Counts only, small rows masked | Notion AI command center, Linear credit explorer, Dust Pod analytics |
| 15 | SB-5002 | W49 | Routing policies | Admin, steward | `routing:` (economy, balanced or frontier) per space decides which tier each kind of task uses; the gate checks each policy's eval scores | Glean Auto mode policies, Notion Auto |
| 16 | SB-5003 | W49 | Personal limits | Admin, everyone | A monthly limit per person. At 80% they're told; at 100% their asks drop to the economy tier until the month turns, and nothing they're asked to review stops | Linear, Glean per-user limits |
| 17 | SB-5004 | W49 | Agent goals | Agent owner, admin | An owner sets each agent a job and a target (say, 60% of drafts published as is, or cost per published draft under $1). Its record tracks it and a miss two weeks running tells the owner | Pace agent goals |
| 18 | SB-5005 | W49 | Did it help? | Steward, owner | When a fix, skill, policy or agent change ships, a check after 4 weeks compares the numbers it should move (helpful rate, gaps, service levels, discards), before and after | Pace Checks |
| 19 | SB-5006 | W49 | The admin's week | Admin | Each Monday, a drafted digest: new and changed agents, skills and rhythms, caps and limits hit, pauses and their reasons, guardrail trends and checks that came back | Glean weekly admin newsletter |
| 20 | SB-5007 | W49 | Inventory export | Admin, auditor | `sb inventory export` and **Download inventory**: every agent, skill, rhythm, routine and trigger with its owner, spaces, cap and goal, as CSV and JSON | Dust CSV export, Rovo agent inventory |

## W47 Ways of working (epic E48, SB-4801 to SB-4807)

The goal is that the company's recurring moments (a new joiner, a project, a quarter, a reorg, a 1:1) start the right drafts at the right time, without a steward remembering each step.

### S47.1 Rhythms

**SB-4801 Rhythms.** sb-tenant(5) gains `rhythms:`: a named playbook of at most 12 steps over at most 120 days. Each step is one of the parts already built: a draft from an agent (a brief, catch-up, goal draft, update or review), a routine (SB-4302), a goal check-in (SB-4207) or a request (SB-4104), with a day offset and the person who approves it. A steward starts a rhythm for a subject (a person, a collection or a space); its progress shows on the space's home and each step lands in the approver's inbox (SB-4301). Five come built in (onboarding, kickoff, close-out, quarterly planning, 1:1 prep) and a steward proposes others as a manifest diff, linted like skills (SB-4601). A new law, **LAW-29**: *a rhythm only drafts; every step waits for a named person, and a rhythm stops when its subject leaves the space or its owner stops it.*
*Laws:* LAW-16, LAW-26, LAW-29 (new). *Acceptance:* no step publishes or sends without its approver. A rhythm whose subject leaves stops at once. A thirteenth step is refused by the lint.

**SB-4802 A new joiner's first 90 days.** The built-in `onboarding` rhythm starts when SCIM adds a person to a space (SB-107) or a steward starts it. Day 1: the space's start-here pack (SB-1706) and the people to meet, from owners and who's who (SB-3906) the joiner can open. Weeks 2 and 6: catch me up (SB-4506) since they joined. Throughout: questions they asked that got no answer go to stewards as gaps, without their name unless they agree. The manager approves the plan once; each later step goes to the joiner. `onboarding: false` turns it off for a space.
*Laws:* LAW-01, LAW-12, LAW-29. *Acceptance:* every item is one the joiner can open. A gap reaches stewards without the joiner's name. Leaving the space stops it.

**SB-4803 Project kickoff.** **Start a project** in a space creates a collection (SB-4304) with the lead and members, a goal draft (SB-4101), a decision log, and a kickoff brief: up to 5 past collections or topics that look similar (glossary and title match), what they decided and who owned it, cited and checked as the lead. Members from other spaces are added only with their steward's approval, and see only what they could already open. The rest of the `kickoff` rhythm schedules a week-2 check-in and a mid-point catch-up.
*Laws:* LAW-01, LAW-24, LAW-29. *Acceptance:* the kickoff brief cites only what the lead can open. A member from another space is added only after their steward approves. Nothing is published without the lead.

**SB-4804 Project close-out.** **Close project** on a collection drafts a retrospective for the lead: goals met or missed (SB-4207), decisions made and replaced (SB-4505), waits still open (SB-4504), and up to 5 lessons proposed as notes (SB-1704) for the space's stewards to publish. The next kickoff brief (SB-4803) finds those notes. Closing archives the collection read-only; nothing is deleted.
*Laws:* LAW-01, LAW-14, LAW-16. *Acceptance:* every line is cited. A lesson becomes a note only when a steward publishes it. A closed collection stays readable.

### S47.2 Plan and reorganise

**SB-4805 Quarterly planning.** Two weeks before a quarter ends, the `quarterly-planning` rhythm drafts next-quarter goals for each space: goals carried over, goals at risk, and themes from requests and voice of the customer (SB-4205), each cited. The steward edits and proposes them. A parent's stewards see their children's proposed goals rolled up (titles they can open, the rest counted, SB-3905), with possible overlaps between departments flagged by glossary match. Goals only start when each steward publishes.
*Laws:* LAW-01, LAW-16, LAW-24. *Acceptance:* every proposed goal is cited. A child the parent steward can't open shows counts only. An overlap is a flag, never a merge.

**SB-4806 Reorg dry run.** `sb tree move SPACE --to PARENT --plan` and `sb tree merge A B --plan` show what would change: glossaries, policies, skills and settings each space would inherit or lose (SB-3902), escalation paths (SB-3904), roll-ups, agents and their triggers, routines, waits and rhythms in flight, and anyone who would lose access to something they own. The plan is a manifest diff the admin proposes through the usual review (W12); nothing moves without it. A move that would widen any access is refused (LAW-24).
*Laws:* LAW-11, LAW-24. *Acceptance:* the plan matches the tree after applying it in the test cell. A widening move is refused with the path. Nothing changes on `--plan`.

**SB-4807 1:1 prep.** The built-in `one-on-one` rhythm, started by a manager for a direct report (from SCIM's manager field), drafts a brief before each recurring 1:1 on the calendar: the report's published updates (SB-4102), goals they own and their check-ins, open action items (SB-4203), requests and waits they're blocked on, each cited and checked as the manager. It never draws on the report's asks, inbox, memory, routines or anything they haven't published, and it never rates or scores. The report sees the same brief at the same time and can add a line. A new law, **LAW-30**: *a brief about a person draws only on what they published, and they can read it.*
*Laws:* LAW-01, LAW-12, LAW-30 (new). *Acceptance:* the report sees exactly the manager's brief. A private ask or memory entry never appears. A manager without access to a cited item doesn't see it.

## W48 Agents that check with you (epic E49, SB-4901 to SB-4906)

The goal is that agents ask before they guess, take comments where they belong, show a plan before a long run, and notice when their sources change.

### S48.1 Check before acting

**SB-4901 Ask before guessing.** Before composing, the ask pipeline checks for three kinds of ambiguity: a term with two meanings in the glossaries in scope (SB-3902), a question that matches several spaces (SB-4007), and a time-bound question with no period. If one is found, the agent asks **one** question with up to 4 choices and "just answer". Scheduled, triggered and shared runs never ask; they state the assumption at the top. The choice is kept for the conversation, never in memory unless the person saves it (SB-4303).
*Laws:* LAW-09, LAW-26. *Acceptance:* an ambiguous term gets one question. A routine never asks and states its assumption. A second question in one turn is refused.

**SB-4902 Comments on a section.** Any agent draft is split into its sections, and reviewers (stewards, owners, the lead or the approver of a rhythm step) leave comments on a section. **Ask for changes** (SB-4602) then redrafts only the commented sections, from the same sources, with a diff per section; the rest stays as approved. Comments are data about the draft (LAW-09), visible to reviewers only, and kept with the draft's audit trail. Rounds still stop at 3.
*Laws:* LAW-09, LAW-14, LAW-16. *Acceptance:* an uncommented section is unchanged byte for byte. A redraft cites only what the first could. A comment is never a source.

**SB-4903 Long research with checkpoints.** Research (SB-4201) gains `long: true`: the agent first shows its plan (sub-questions, spaces, expected sources and cost estimate) for approval, then runs in at most 6 stages for at most 4 hours. At each checkpoint the asker gets a short note and can steer (add or drop a sub-question), stop, or let it go on; with no answer in 30 minutes it goes on. A run stops at `research.max_usd` (default $5) and counts against the person's and the space's budgets. It runs as the asker throughout, and access is checked again at each stage (LAW-26).
*Laws:* LAW-01, LAW-26. *Acceptance:* nothing runs before the plan is approved. A run stops at its cost ceiling with what it has. A document the asker lost mid-run is dropped at the next stage.

### S48.2 Agents that keep up

**SB-4904 Scenarios before you approve.** In a manifest diff that changes a skill or an agent (W12), a steward can add up to 10 **scenarios**: a question, what it should cite or say, and what it must never say. **Run scenarios** runs them against the current and the proposed version, as the steward, and shows pass or fail side by side before they approve. Passing scenarios can be kept as eval cases (SB-4401). Scenarios never run as anyone else.
*Laws:* LAW-01, LAW-06. *Acceptance:* scenarios run as the steward only. A failing scenario shows both outputs. A kept scenario joins the eval suite in the gate.

**SB-4905 When a source changes.** An agent's trigger list (SB-4006) gains `on: source-changed`: when a new version of a cited document arrives, or a policy (SB-4103) changes, anything the agent drafted from it in the last 90 days (a published note, an FAQ, an account brief, an update) is redrafted for review with the change highlighted. Many changes in a day are batched into one redraft, and a document the original approver can't open any more ends the redraft for them.
*Laws:* LAW-01, LAW-14, LAW-26. *Acceptance:* a changed policy redrafts the note that cited it once a day at most. Nothing republishes without the approver. A source the approver lost drops the redraft.

**SB-4906 Starter questions.** Each space gets up to 12 **starter questions**: suggested by the system from saved answers and asks with a high helpful rate (asker names never kept), curated by the steward, and shown on the Answer tab and listed by the MCP server (SB-4306). A starter question is just a question; its answer is composed fresh, as the person, each time.
*Laws:* LAW-01, LAW-07. *Acceptance:* a suggestion carries no asker's name. Two people with different access get different answers. Only a steward publishes one.

## W49 Run AI like a business (epic E50, SB-5001 to SB-5007)

The goal is that an admin sees what AI is used for and what it costs in one place, chooses how much model each task deserves, limits a person as well as an agent, gives agents targets, and learns whether a change actually helped.

### S49.1 Spend with purpose

**SB-5001 AI command centre.** A section of the existing costs page and `GET /v1/admin/ai/usage`: for 12 weeks, use and spend by space, feature (ask, research, brief, routine, rhythm, consultation), agent and model tier, with each space's budget (SB-4105), each agent's cap (SB-4703) and personal limits (SB-5003) beside it, and a CSV download. People appear only as counts per band; any row under 5 shows as "fewer than 5".
*Laws:* LAW-07, LAW-11. *Acceptance:* totals match the traces. A small row is masked. No person is named.

**SB-5002 Routing policies.** sb-tenant(5) `routing: economy|balanced|frontier` per space (default `balanced`, flowing down the tree like other settings, SB-3902) decides which model tier (SB-4405) each kind of task uses: `economy` keeps everything on the small tier but research, `frontier` sends composing and research to the largest. The gate runs each space's eval suite on its policy; a policy that fails its suite is refused at release, not at run time.
*Laws:* LAW-06, LAW-07. *Acceptance:* each policy picks the documented tier per task. A policy whose eval suite fails is refused by the gate. A child can't choose a higher tier than its parent allows.

**SB-5003 Personal limits.** sb-tenant(5) `limits: { monthly_usd_per_person }`. At 80% the person is told; at 100% their own asks and research move to the economy tier until the month turns, and their routines pause. Anything they're asked to review (rhythm steps, drafts, requests) keeps working, and a steward can lift one person's limit for the month, audited.
*Laws:* LAW-07, LAW-11. *Acceptance:* at the limit, asks still work on the economy tier. Review steps never stop. A lift is audited.

**SB-5004 Agent goals.** An agent's owner (SB-4402) can set its `job:` (one sentence) and up to 3 **targets** on its value numbers (SB-4701): published as is, discard rate, time to publish, helpful rate or cost per published draft. The registry shows each target against the last 4 weeks, and a target missed two weeks running tells the owner once. Targets go in the agent's record (SB-4707). Meeting a target changes nothing else.
*Laws:* LAW-07, LAW-11. *Acceptance:* a target missed twice in a row tells the owner once. A target can only use value numbers. The record lists every target and its history.

### S49.2 Learn what works

**SB-5005 Did it help?** When a fix (SB-4206), skill (SB-4601), routing policy, policy or agent version ships, a **check** is scheduled 4 weeks later. It compares the numbers that change was meant to move (named when it was proposed, or the defaults: helpful rate, open gaps, service levels and discards for that space and agent) for the 4 weeks before and after, and drafts a short result for the steward or owner: better, worse or no clear change, with counts. A check never undoes anything.
*Laws:* LAW-07, LAW-11. *Acceptance:* the numbers match the audit log and traces. Fewer than 20 events either side says "not enough data". Nothing changes on a worse result except the note.

**SB-5006 The admin's week.** Each Monday, the tenant's admins get a drafted digest: agents, skills and rhythms added or changed (with their approvers), caps and limits reached, pauses and their reasons (SB-4702), guardrail trends (SB-4406), access reviews due (SB-4705), checks that came back (SB-5005), and the week's top spaces by spend. Counts and ids, no document text. An admin can turn it off for themselves.
*Laws:* LAW-07, LAW-11. *Acceptance:* every line links to its source record. No document text appears. Turning it off stops only that admin's copy.

**SB-5007 Inventory export.** `sb inventory export` and **Download inventory** on the registry write every agent, skill, rhythm, routine (shared ones only, as counts of members), trigger and starter question, with its owner, spaces, cap, targets, last run and last change, as CSV and signed JSON. Personal routines appear only as counts per space.
*Laws:* LAW-07, LAW-11, LAW-12. *Acceptance:* the export matches the manifest and the registry. A personal routine is never named. The JSON signature verifies.

## What we leave out

- **Performance ratings, calibration and review drafts** (Lattice). 1:1 prep stays on shared work, and the brain never scores a person (LAW-30).
- **Rhythms that act on their own** (Pace, Rovo Studio). Every step is a draft a person approves (LAW-29).
- **Agents with their own computer and browser** (Notion, Gemini). Agents still read only connected sources, as a person.
- **Bring your own model key per person** (Notion, Linear). Model keys stay with the tenant's gateway (SB-305).
- **Office and chat add-ins** (Dust). MCP (SB-3801, SB-4306) is how other tools reach the brain.
- **Agents built in the browser** (Rovo Studio, Gemini Agent Designer). Skills, scenarios and manifest diffs are as far as shaping goes.
- **Voice capture.** Still waiting on a local speech model.

## Decisions for the user

Decided by the user 2026-11-25: every recommended option.

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W47, W48, W49. *Other:* W49 first, so spend controls land before rhythms and long research add use.
2. **Rhythms (SB-4801, LAW-29).** *Recommended:* five built in, others proposed by stewards as manifest diffs, every step a draft a named person approves. *Other:* built-in rhythms only.
3. **1:1 prep (SB-4807, LAW-30).** *Recommended:* include it, from published work only, with the report seeing the same brief. *Other:* leave people-focused briefs out entirely.
4. **Ask before guessing (SB-4901).** *Recommended:* at most one question with up to 4 choices, on-demand asks only. *Other:* never ask; always state the assumption.
5. **Long research (SB-4903).** *Recommended:* plan approval first, up to 4 hours in 6 stages, $5 default ceiling per run. *Other:* keep today's single-pass ceiling.
6. **Routing policies (SB-5002).** *Recommended:* per space, `balanced` by default, children capped by the parent. *Other:* one policy for the whole tenant.
7. **Personal limits (SB-5003).** *Recommended:* at the limit, drop to the economy tier rather than stop, and never stop review steps. *Other:* a hard stop at the limit.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Lattice: Lattice unveils Pace, a new business performance platform for people and AI](https://lattice.com/blog/lattice-unveils-pace-a-new-business-performance-platform-for-people-and-ai)
- [Lattice: Fall/Winter '26 product release](https://lattice.com/blog/fall-winter-2026-product-release)
- [Glean release notes, November 2026](https://docs.glean.com/release-notes/)
- [Dust changelog](https://docs.dust.tt/changelog)
- [Dust: agents can now ask you questions to clarify their next steps](https://docs.dust.tt/changelog/agents-can-now-ask-you-questions-to-clarify-their-next-steps)
- [Notion releases, October 2026](https://www.notion.com/fi/releases/2026-10-19)
- [Linear changelog](https://linear.app/changelog)
- [Google Cloud: what's new in Gemini Enterprise](https://cloud.google.com/blog/products/ai-machine-learning/whats-new-in-gemini-enterprise)
- [Atlassian: Rovo at Team '26](https://www.atlassian.com/blog/company-news/rovo-team-26)
- [Atlassian Community: Rovo updates (Scenario Builder, prompt library, agent usage analytics)](https://community.atlassian.com/forums/Atlassian-AI-Rovo-articles/%EF%B8%8F-Sherif-sclusive-Atlassian-AI-amp-Rovo-Updates-November/ba-p/3159574)
- [SiliconANGLE: At Ignite 2026, Microsoft reinvents Copilot as the autonomous engine of enterprise work](https://siliconangle.com/2026/11/18/at-ignite-2026-microsoft-reinvents-copilot-as-the-autonomous-engine-of-enterprise-work/)

---
title: Guard What Goes In; Write It Down Together; Agents That Act Carefully
summary: Concept for W50 to W52 (accepted 2026-11-25): twenty features. Guards on what agents read and send, safe rollouts and agent history; drafts written together with suggested edits to sources and cites to the page; and agents that start on work events, borrow access for one run, and draft actions in other tools for a person to send. No new service, store or Go module.
---

# Guard what goes in; write it down together; agents that act carefully

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-25 | Platform lead | Accepted by the user 2026-11-25, with every recommended decision | [Principles](01-principles.md) · [20-rhythms-check-run.md](20-rhythms-check-run.md) · [PRD waves](03-prd.md#waves) |

## Summary

W47 to W49 are built and pushed (`28e6744`). The company's recurring moments run as rhythms, agents ask before guessing and run long research with checkpoints, and AI spend answers to routing policies, personal limits, agent goals and outcome checks.

The user asked for the next lot of waves. This round looked at what shipped in November 2026: Glean's releases of 3 and 19 November (artifact-version and pull-request triggers, approvers lending access to independent agents, agent version diffs, a feature rollouts page, departmental agent moderators, guardrail exports, citations to the exact PDF page or slide, a redesigned composer), Dust (built-in prompt-injection and sensitive-data checks, co-edited documents with accept or reject, signed inference hooks, branching conversations), and Atlassian (suggested edits through MCP, managed MCP access, agent attribution in page history, agents mentioned in inline comments).

Three gaps stand out.

1. **What agents read and send is checked only at ingest.** The sanitiser (SB-207) treats documents as data, but a passage that reads like instructions isn't shown to anyone, an answer or notification can still carry a key or card number someone pasted into a source, and new features turn on for everyone at once with no pilot and no history of how an agent changed.
2. **Knowledge flows one way.** The brain finds that a source is wrong or stale, but can only tell the owner, not hand them a suggested fix. Reviewers comment but can't edit a draft with the agent's changes shown as suggestions; cites stop at the line, not the PDF page or slide; and an earlier question can't be branched.
3. **Agents only start on a clock or a source change, and only as one person.** A pull request opening, a ticket filed or a meeting booked can't start an agent; a scheduled agent can't use a source only someone else can read; and an agent can't draft the ticket or message it recommends.

Three waves, twenty tickets: **W50 Guard What Goes In** is first, then **W51 Write It Down Together**, then **W52 Agents That Act Carefully**.

## What others do, November 2026

| Product | What it does that matters here |
|---|---|
| **Glean (3 and 19 November)** | Agent triggers on a new artifact version, a GitHub pull request opening and new Outlook mail. **Independent agents use user-authorised tools, with an approver lending access** through an approval request. **Agent version diffs** ("View changes" before restoring). A **feature rollouts page** to pilot launches with a group before they're on by default, with a weekly digest. **Departmental agent moderators**. **Guardrail violation exports**. Citations to the **exact PDF page or slide**. A composer with `@` and `/`. Slack agents answering mentions. A chat context panel. |
| **Dust (November)** | **Built-in prompt-injection and sensitive-data checks** in Monitor or Block mode. **Real-time co-edited documents**, agent edits accepted or rejected. Signed inference-hook webhooks. Agent and skill CSV export. Branching conversations. |
| **Atlassian (November)** | **Suggested edits** through MCP: an assistant proposes a page edit for a person to accept. **Managed MCP access**: admins decide who may connect which client. **Agent attribution**: page history credits both the agent and the person who invoked it. Agents **mentioned in inline comments**. |
| **Linear (October and November)** | Loops started from Slack and GitHub events; shareable loop templates. |

**What they agree on:** check what goes into and out of a model, not just what was ingested; roll features out to a pilot first; show how an agent changed; let agents propose edits to the source; cite precisely; start agents on work events; and let a person lend access rather than give an agent its own.

**Where we differ, on purpose:**

- **Borrowed access is for one run, and the result goes only where the lender's cites can go** (a new law, **LAW-31**). Glean's approver lends access to an agent; ours lends it to one run, the lender is named on the result, and only people who can open every cite see it.
- **A suggested edit never touches the source.** Connectors stay read-only; the owner gets a drafted change to make themselves, or to dismiss.
- **An action in another tool is a draft a person sends.** No agent files a ticket or posts a message on its own (rule 7).
- **No service accounts for agents.** An agent still runs as a person (the asker, a rhythm's approver or a lender), never as an identity of its own.

## Keeping it small

Rules 1 to 15 from earlier waves still apply (no new service, store or Go module; at most one setting per feature; no new page; nothing widens what anyone sees; a person approves every write; the tree only narrows; agents are definitions in the bundle; stored results are checked when read; every long run has a ceiling; one level of consultation; the AI label never comes off; a rhythm only drafts; a brief about a person draws only on what they published). One more:

16. **Lent access lasts one run and travels with its result** (LAW-31).

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-5101 | W50 | Suspect passages | Steward, security | At read time, a retrieved passage that reads like instructions to the model is skipped and logged; stewards see it on the review list with its document and owner | Dust prompt-injection check |
| 2 | SB-5102 | W50 | Nothing sensitive goes out | Everyone, security | Keys, tokens, card numbers and national ids are masked in every answer, draft, notification, chat reply and MCP result, whatever the source said | Dust sensitive-data check |
| 3 | SB-5103 | W50 | Guardrail export | Security, auditor | Every guardrail event (suspect passage, masked output, refusal) goes to the audit stream and `GET /v1/admin/guardrails/events` as JSON or CSV, counts and ids only | Glean guardrail exports |
| 4 | SB-5104 | W50 | Agent history | Agent owner, steward | Every version of an agent or skill, who proposed and approved it, and a diff between any two; **Restore** proposes the old version as a manifest diff | Glean agent version diff |
| 5 | SB-5105 | W50 | Branch moderators | Steward | A steward of a department can pause any agent and review any skill or routine shared in their branch, not just their own space | Glean departmental moderators |
| 6 | SB-5106 | W50 | Feature rollouts | Admin | A new feature from a release can be turned on for a pilot group first; it goes on for everyone after 2 weeks unless an admin holds it. The admin's week lists rollouts | Glean feature rollouts |
| 7 | SB-5107 | W50 | Pause from the notice | Agent owner, steward | Every agent-run notification carries **Pause this agent here** for the people allowed to pause it | Glean pause link |
| 8 | SB-5201 | W51 | Suggested edits to the source | Owner | When an answer, check or review finds a source stale, contradicted or wrong, a drafted change with the evidence goes to the document's owner; nothing is written to the source | Atlassian suggested edits |
| 9 | SB-5202 | W51 | Edit a draft together | Steward, reviewers | Reviewers edit any agent draft in place; the agent's changes from **Ask for changes** show as suggestions to accept or reject one by one | Dust co-edited documents |
| 10 | SB-5203 | W51 | Ask an agent in a comment | Reviewers | `@agent` in a section comment gets a cited reply under the comment, composed as the commenter | Rovo in inline comments |
| 11 | SB-5204 | W51 | Who made it | Everyone | Every published note's history names the agent, the person who asked for it, each editor and the approver, and the note's page lists them | Atlassian agent attribution |
| 12 | SB-5205 | W51 | Cite the page or slide | Everyone | Cites to PDFs and slide decks name the page or slide, and the passage view opens there | Glean PDF page and slide cites |
| 13 | SB-5206 | W51 | Branch a conversation | Everyone | Ask again from any earlier question in a different direction; both branches keep their own context and can be downloaded | Dust branching conversations |
| 14 | SB-5207 | W51 | `@` and `/` in the ask box | Everyone | `@` picks a space, person or document to ask about; `/` picks a skill or starter question | Glean composer |
| 15 | SB-5301 | W52 | Start on a work event | Agent owner | Triggers on a GitHub pull request opened, a Linear ticket filed or a calendar event booked start a drafting agent, as the trigger's approver | Glean PR trigger, Linear loops |
| 16 | SB-5302 | W52 | Try a trigger first | Agent owner | Replay a new trigger against the last 14 days of events and see what it would have drafted, before turning it on | Nobody; Rovo Scenario Builder |
| 17 | SB-5303 | W52 | Borrow access for one run | Agent owner, lender | A scheduled agent that needs a source its approver can't read asks a named person to lend access for one run; the result shows only to people who can open every cite | Glean approver-lent access |
| 18 | SB-5304 | W52 | Draft the next action | Everyone | An answer or brief that recommends a ticket or message drafts it in Linear or Slack; a person edits and sends it | Atlassian suggested edits, Glean actions |
| 19 | SB-5305 | W52 | Managed MCP access | Admin | Admins choose which groups may connect MCP clients and which tools each group sees; the command centre shows use by client | Atlassian managed MCP, Glean AI Gateway |
| 20 | SB-5306 | W52 | Meeting transcripts from Drive | Everyone | Google Meet transcripts in Drive are read as transcripts, with speakers and action items feeding action items (SB-4203) | Glean Otter connector |

## W50 Guard what goes in (epic E51, SB-5101 to SB-5107)

The goal is that nothing a source says can steer an agent or leak through it, that a new feature reaches a pilot before everyone, and that anyone can see how an agent changed.

### S50.1 Check what goes in and out

**SB-5101 Suspect passages.** Before composing, each retrieved passage is checked for text addressed to the model (instructions to ignore rules, change roles, call tools or send data). A suspect passage is skipped, never quoted, and recorded with its document and owner; the answer notes "1 passage skipped". Stewards see suspect passages on the review list (`section=suspect`) and the owner gets one notice per document. A space can set `suspect: monitor` to log without skipping; the default is `skip`.
*Laws:* LAW-09, LAW-11. *Acceptance:* the injection fixtures are skipped in every path (ask, research, briefs, MCP). A skipped passage's document is still listed for the steward. Monitor mode logs the same events.

**SB-5102 Nothing sensitive goes out.** One output filter on every way text leaves the cell (answers, drafts, notifications, chat replies, email, MCP results and exports) masks API keys and tokens, card numbers that pass Luhn, and national ids (`ingest.nationalIDs`), keeping the cite. A masked output is a guardrail event. Masking can't be turned off.
*Laws:* LAW-07, LAW-11. *Acceptance:* a fixture key planted in a source never appears in any output path. A false positive keeps the rest of the answer intact. Every path goes through the one filter (a law test lists them).

**SB-5103 Guardrail export.** Suspect passages, masked outputs and refusals become guardrail events in the audit log, sent on the audit stream (SB-2302) and listed at `GET /v1/admin/guardrails/events?since=&format=csv`: time, kind, space, agent, document id and counts. No document text and no asker.
*Laws:* LAW-07, LAW-11. *Acceptance:* the export matches the audit log. No text or asker appears. The stream carries each event once.

### S50.2 Roll out with care

**SB-5104 Agent history.** `GET /v1/admin/agents/NAME/history` and the registry's **History** list every version of an agent or skill from the release bundles and the manifest-diff audit: when, who proposed, who approved, which eval suite passed. **View changes** diffs any two versions. **Restore** proposes the older definition as a manifest diff through the usual review; nothing restores directly.
*Laws:* LAW-06, LAW-11. *Acceptance:* the history matches the audit log. Restore creates a proposal, not a change. A restored version must pass the gate again.

**SB-5105 Branch moderators.** A steward of a space with children may pause any agent (SB-4702) and review any shared skill, routine or rhythm in their branch. They see names and counts for children they can't open, never content. Every action is audited with the branch.
*Laws:* LAW-10, LAW-24. *Acceptance:* a department steward pauses a team's agent. A sibling department's agent is refused. Content of a child they can't open never appears.

**SB-5106 Feature rollouts.** Each release lists its features with `rollout: pilot|on`. A `pilot` feature is on only for the tenant's `pilot` group until 14 days pass, then on for everyone unless an admin holds it (`sb rollout hold|release FEATURE`). The admin's week (SB-5006) lists features in pilot, held and released.
*Laws:* LAW-06, LAW-08. *Acceptance:* a pilot feature is off for a non-member. A held feature stays off past 14 days. Release and hold are audited.

**SB-5107 Pause from the notice.** Every notification about an agent run (drafts, briefs, checks, alerts) carries **Pause this agent here** for people who may pause it (owner, the space's stewards, a branch moderator). It asks for a reason and pauses only that agent in that space.
*Laws:* LAW-10. *Acceptance:* only allowed people see the link. A pause from the link is the same audited pause as SB-4702.

## W51 Write it down together (epic E52, SB-5201 to SB-5207)

The goal is that what the brain learns flows back to the sources, drafts are written together, and every answer points to exactly where it came from.

### S51.1 Fix the source

**SB-5201 Suggested edits to the source.** When a check finds a source stale (SB-2701), contradicted by a newer one (SB-3605), replaced (SB-4505) or reported wrong (SB-3606), a drafted suggested edit goes to the document's owner: the passage, the change, and the cites that support it. The owner applies it in the source tool themselves, or dismisses it with a reason; nothing is written through a connector. A dismissed suggestion isn't made again for the same passage.
*Laws:* LAW-01, LAW-14, LAW-16. *Acceptance:* the suggestion cites only what the owner can open. Nothing is written to a source. A dismissal stops repeats.

**SB-5202 Edit a draft together.** Reviewers of any agent draft can edit it in place. **Ask for changes** (SB-4602, SB-4902) now returns its changes as suggestions, each accepted or rejected by a reviewer; the approver publishes only when none are open. Every edit is kept with the draft's audit trail and marks the note `edited` (SB-4706).
*Laws:* LAW-14, LAW-16, LAW-28. *Acceptance:* a rejected suggestion leaves the text unchanged. A draft with open suggestions can't publish. The AI mark stays.

**SB-5203 Ask an agent in a comment.** `@NAME` in a section comment (SB-4902) asks that department agent about the section; the reply appears under the comment, cited and composed as the commenter. It's a comment, never part of the draft until someone accepts it.
*Laws:* LAW-01, LAW-09, LAW-27. *Acceptance:* the reply cites only what the commenter can open. A reply is never a source. Consultation stays one level deep.

**SB-5204 Who made it.** A published note's front matter and history name the agent that drafted it, the person who asked, each editor and the approver; the passage view and the note's page show them. Imported marks (SB-4706) are kept.
*Laws:* LAW-11, LAW-28. *Acceptance:* every published agent draft names all four roles that apply. History matches the audit log.

### S51.2 Point precisely, ask freely

**SB-5205 Cite the page or slide.** Ingest records the PDF page and slide number for each chunk (`Chunk.Page`); cites name them ("Pricing deck, slide 7") and the passage view opens at that page or slide, beside the line range where there is one.
*Laws:* LAW-14. *Acceptance:* the fixture PDF and deck cite the right page and slide. Older chunks without a page still cite by line.

**SB-5206 Branch a conversation.** **Ask from here** on any earlier question starts a branch with the context up to that point; each branch keeps its own last 8 questions (SB-2901) and its own download (SB-3305). Branches live only in the browser, like recent questions (SB-2405).
*Laws:* LAW-01. *Acceptance:* a branch doesn't see later questions of the other branch. Nothing is stored on the server.

**SB-5207 `@` and `/` in the ask box.** In the web UI, `@` offers spaces, people (owners and stewards from who's who) and documents the person can open, and adds them to the ask (SB-3004); `/` offers the skills and starter questions of the space. Each suggestion is checked as the person.
*Laws:* LAW-01. *Acceptance:* a document they can't open is never offered. `/` shows only skills the space has.

## W52 Agents that act carefully (epic E53, SB-5301 to SB-5306)

The goal is that agents start on the work events that matter, can be tried before they're turned on, can borrow access for one run with a named lender, and draft the next action for a person to send.

### S52.1 Start on work

**SB-5301 Start on a work event.** An agent's triggers (SB-4006) gain `on: pr-opened` (GitHub), `on: ticket-filed` (Linear, with labels), and `on: event-booked` (calendar, SB-4202). A trigger runs as its approver, drafts only, and batches more than 10 events an hour into one run. Events come through the signed event intake (SB-4404).
*Laws:* LAW-01, LAW-16, LAW-29. *Acceptance:* a pull request opened starts one draft. A flood of events batches. Nothing publishes without the approver.

**SB-5302 Try a trigger first.** **Try it** on a new or changed trigger replays the last 14 days of matching events, as the approver, and shows what it would have drafted and what it would have cost, before it goes live. Nothing is sent.
*Laws:* LAW-01, LAW-07. *Acceptance:* a replay sends nothing. Its cost matches the run it would have been. Events the approver can't open are counted, not shown.

**SB-5303 Borrow access for one run.** When a scheduled or triggered run needs a space its approver can't read, it can ask a named person who can to **lend access for this run**. The lender sees the plan (sources, purpose, who will see the result) and approves or declines; the run then reads as the lender for those sources only, and the result names the lender and shows only to people who can open every cite. Lending never carries to the next run. A new law, **LAW-31**: *lent access lasts one run, names the lender, and its result reaches only people who can open every cite.*
*Laws:* LAW-01, LAW-26, LAW-31 (new). *Acceptance:* a second run asks again. A reader who can't open a cite doesn't see the result. A declined loan runs without those sources.

### S52.2 Do the next thing

**SB-5304 Draft the next action.** When an answer, brief or review recommends a ticket or a message, **Draft it** prepares a Linear ticket (team, title, description, cites as links) or a Slack or Mattermost message (channel, text). The person edits and presses **Send**, which goes out with their own token through the existing tool bus (SB-4301 pending actions). Agents never send.
*Laws:* LAW-04, LAW-16. *Acceptance:* nothing is sent without the person's press. The sent item is audited. A cite they can't open isn't linked.

**SB-5305 Managed MCP access.** sb-cell(5) `mcp.groups` names which IdP groups may connect MCP clients and which tools each sees (search, ask, agents, skills, research). The command centre (SB-5001) shows use by client (web, chat, MCP, API token).
*Laws:* LAW-01, LAW-07. *Acceptance:* a member outside the groups is refused. A tool not granted isn't listed. Use by client matches the traces.

**SB-5306 Meeting transcripts from Drive.** The Drive connector reads Google Meet transcript documents as transcripts (SB-3701): speakers, times and the meeting from the calendar, with action items offered to SB-4203. Attendees' access on the document stays the ceiling.
*Laws:* LAW-01, LAW-02. *Acceptance:* the recorded fake's transcript parses with speakers. An action item is confirmed by its person before it counts. A non-attendee without access never sees it.

## What we leave out

- **Service accounts for agents** (Glean ServiceNow credentials, Atlassian service accounts). An agent runs as a person; borrowed access (LAW-31) covers the rest.
- **Writing to sources through connectors.** Suggested edits go to owners to apply.
- **Real-time multi-cursor editing** (Dust). Edits are saved in turn, with suggestions accepted one by one.
- **Mail triggers** (Glean Outlook). We have no mail connector; calendar, GitHub and Linear come first.
- **Office add-ins and agents in other editors.** MCP remains the way in.
- **Sharing outside the company** (Confluence access links).

## Decisions for the user

Decided by the user 2026-11-25: every recommended option.

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W50, W51, W52. *Other:* W52 first, for event triggers and drafted actions sooner.
2. **Suspect passages (SB-5101).** *Recommended:* skip and log by default; a space may choose monitor only. *Other:* monitor only by default.
3. **Borrowed access (SB-5303, LAW-31).** *Recommended:* include it, one run at a time, with a named lender and the result limited to people who can open every cite. *Other:* leave it out; agents read only as their approver.
4. **Drafted actions (SB-5304).** *Recommended:* Linear tickets and Slack or Mattermost messages, sent by the person with their own token. *Other:* copy-to-clipboard only, nothing sent through the brain.
5. **Feature rollouts (SB-5106).** *Recommended:* pilot group first, on for everyone after 14 days unless an admin holds it. *Other:* everything stays off until an admin turns it on.
6. **Meeting transcripts (SB-5306).** *Recommended:* Google Meet transcripts through the Drive connector we already have, no new vendor. *Other:* a new meeting-notes vendor (Otter or Fireflies) on a recorded fake.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Glean release notes, 19 November 2026](https://docs.glean.com/release-notes/releases/2026-11-19-november-release)
- [Glean release notes, 3 November 2026](https://docs.glean.com/release-notes/releases/2026-11-03-november-release)
- [Dust changelog](https://docs.dust.tt/changelog)
- [Capable: what's new in Atlassian, week of 16 November 2026](https://www.gocapable.com/blog/whats-new-atlassian-16-november-2026)
- [Capable: what's new in Atlassian, week of 9 November 2026](https://www.gocapable.com/blog/whats-new-atlassian-9-november-2026)
- [Linear changelog](https://linear.app/changelog)

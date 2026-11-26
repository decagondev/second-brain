---
title: Know Who Can See; See and Steer the Context; Keep Answering, Spend Less
summary: Concept for W53 to W55 (accepted 2026-11-26): twenty features. Reports on documents shared too widely and labels that travel with every answer; a context panel and controls that let a person see and steer what an answer used; and answers that keep coming when a provider fails, cost less through reuse, batching and prompt caching. No new service, store or Go module.
---

# Know who can see; see and steer the context; keep answering, spend less

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-26 | Platform lead | Accepted by the user 2026-11-26, with every recommended decision | [Principles](01-principles.md) · [21-guard-write-act.md](21-guard-write-act.md) · [PRD waves](03-prd.md#waves) |

## Summary

W50 to W52 are built and pushed (`90a7249`). What agents read and send is checked, features reach a pilot first, suggested edits go back to owners, cites name the page or slide, and agents start on work events, borrow access for one run and draft the next action for a person to send.

The user asked for the next lot of waves. This round looked at what shipped from late October to November 2026: Glean (releases 566 and 568: a chat context panel, higher limits when a person is blocked, an AI transparency notice), Notion (19 October: agents with smarter context management, a "Review for me" mode, an Auto model, memory sync and one credit system for AI use), Microsoft Ignite 2026 (17 to 20 November: Purview data classification across people and agents, Agent 365 business analytics for AI cost and value, business plugins), and Atlassian Rovo (a debug view for agents).

Three gaps stand out.

1. **The brain trusts source sharing exactly, including its mistakes.** A salary sheet shared with "anyone in the company" is faithfully cited to anyone in the company (LAW-01 holds; the source is wrong). Nobody is told when a document is shared too widely, when sharing widens between syncs, or how far each agent can reach. Labels such as "Confidential" are read nowhere and shown nowhere.
2. **A person can't see or steer what an answer used.** Trace views (SB-4606) show how an answer was made after the fact, but nobody can drop a source, keep one for the rest of a conversation, turn off a document for good, or ask "why didn't you use the pricing deck?".
3. **One model provider is a single point of failure, and every call is paid in full.** When the provider is down, asks fail. The same question asked twice in a day costs twice. Briefs and checks that could wait for tomorrow run at full price now, and the stable part of every prompt is sent and billed again each time.

Three waves, twenty tickets: **W53 Know Who Can See** is first, then **W54 See and Steer the Context**, then **W55 Keep Answering, Spend Less**.

## What others do, late 2026

| Product | What it does that matters here |
|---|---|
| **Glean (releases 566 and 568)** | A **chat context panel** (beta) showing what a chat is working from. **Higher usage limits when blocked.** Comments on artifacts. An **AI transparency notice** on by default from 10 November. New MCP server templates. |
| **Notion (19 October)** | Agents rebuilt with **smarter context management** and a **"Review for me" mode with safety checks**. A cheaper **Auto model**. Bring your own AI. **Memory sync**. All AI use under **one credit system**. Skills over MCP coming. |
| **Microsoft (Ignite, 17 to 20 November)** | **Purview data classification across people and agents.** **Agent 365 Business Analytics** for AI cost and value. Microsoft IQ business plugins bundling knowledge, skills and tools. Copilot Workspaces. |
| **Atlassian Rovo** | A **debug view for agents** showing what an agent retrieved and why. |
| **Model providers** | Batch APIs at about half price for work that can wait a day, and **prompt caching** of a stable prompt prefix (Anthropic, OpenAI). |

**What they agree on:** classify what is sensitive and show it wherever content goes, including into agents; show people the context an answer is built from and let them change it; and make AI spend visible and smaller without making answers worse.

**Where we differ, on purpose:**

- **We never change a source's sharing.** Connectors stay read-only. The brain reports oversharing to the owner and, by default, holds a widely shared sensitive document back from answers until the owner confirms; holding back only narrows (LAW-19).
- **A label never drops** (a new law, **LAW-32**): anything built from a labelled document carries its highest label and goes nowhere that label forbids.
- **A reused answer is checked again as the new asker** (LAW-26), every time, and says it was reused.
- **A fallback model must pass the same eval gate** (LAW-06). When none has, the brain shows passages and says the model is unavailable; it never answers with an ungated model.

## Keeping it small

Rules 1 to 16 from earlier waves still apply (no new service, store or Go module; at most one setting per feature; no new page; nothing widens what anyone sees; a person approves every write; the tree only narrows; agents are definitions in the bundle; stored results are checked when read; every long run has a ceiling; one level of consultation; the AI label never comes off; a rhythm only drafts; a brief about a person draws only on what they published; lent access lasts one run). One more:

17. **A label never drops** (LAW-32), and the brain never changes who can open a source; it tells the owner.

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-5401 | W53 | Shared too widely | Owner, security | Documents open to the whole company or anyone with the link that hold personal data, a masked secret, a sensitive label or sit in a restricted space are listed for their owner and the space's stewards, with how wide they are | Microsoft oversharing reports, Purview |
| 2 | SB-5402 | W53 | Hold until the owner confirms | Owner, everyone | A widely shared sensitive document is left out of answers for everyone but its owner and stewards until the owner confirms the sharing is meant, or narrows it at the source | Nobody; Purview DLP for AI |
| 3 | SB-5403 | W53 | Sharing widened | Owner, security | When a sync sees a document's sharing widen (a new group, the whole company, anyone with the link), its owner gets one notice, and a sensitive one goes on the review list | Purview activity alerts |
| 4 | SB-5404 | W53 | Who can see this | Everyone | Beside a cite, a note or a shared answer: how many people can open it, and which of their groups the person can see; other groups are counted, never named | Microsoft "who has access" |
| 5 | SB-5405 | W53 | Labels travel | Everyone, security | Drive labels and front matter `sensitivity:` (public, internal, confidential, restricted) are read at ingest; every answer, draft, share, download and notification shows the highest label it cites, and confidential content never goes to a chat channel, only to a person (LAW-32) | Purview labels across people and agents |
| 6 | SB-5406 | W53 | How far each agent reaches | Agent owner, security | The agent registry shows each agent's reach: the spaces and documents its approvers, triggers and loans let it read, how many are labelled, and an alert when reach grows by more than a tenth | Agent 365, Purview for agents |
| 7 | SB-5407 | W53 | Sensitive map | Steward, security | Per space, counts of documents by kind (personal data, masked secrets, labels, wide sharing), trended weekly on the review list and in the admin's week; counts only | Purview data classification |
| 8 | SB-5501 | W54 | Context panel | Everyone | Beside each answer: the passages it used, memory lines, skill, collection, earlier questions kept and dropped, departments consulted and model tier, all titles the person can open | Glean chat context panel |
| 9 | SB-5502 | W54 | Keep or drop for this conversation | Everyone | From the panel, **Keep** a document for the rest of the conversation or **Drop** it; both live in the browser with the conversation, and a kept document is still checked as the person each time | Notion context management |
| 10 | SB-5503 | W54 | Not for me | Everyone | **Don't use this document for me** keeps a document out of every answer, brief and routine made as the person, until they undo it on My data | Nobody |
| 11 | SB-5504 | W54 | Memory used, shown | Everyone | An answer that used a memory line (SB-4303) says which, with **Forget this** beside it | Notion memory, ChatGPT memory |
| 12 | SB-5505 | W54 | Memory over MCP | Everyone | MCP clients read the person's memory lines and may propose a new one; it counts only after the person confirms it in the web UI | Notion memory sync |
| 13 | SB-5506 | W54 | Why not this document? | Everyone | Name a document you expected; the brain says why it wasn't used: not indexed, ranked lower (and by what), skipped as suspect, held, dropped by you, or outside the scope. Only for documents the person can open | Rovo agent debug view |
| 14 | SB-5507 | W54 | Retrieval view for stewards | Steward | On a reported or unhelpful answer, the steward sees the ranked passages the ask retrieved, with scores and why each was kept or cut, titles they can open only | Rovo agent debug view |
| 15 | SB-5601 | W55 | Fall back to a gated model | Admin, everyone | sb-cell(5) `model.fallback` names a second provider; when the first fails or times out, the fallback answers only if it passed the same eval gate, and the answer says so | Glean, Notion Auto model |
| 16 | SB-5602 | W55 | Passages when the model is down | Everyone | With no gated model available, the ask box shows the best passages the person can open, labelled "no answer written; model unavailable", and the health report says why | Nobody |
| 17 | SB-5603 | W55 | Answered before, checked now | Everyone | The same question in the same space within 7 days reuses the earlier answer when every cite is unchanged and the asker can open all of them; it says so and offers **Ask fresh** | Glean answer caching |
| 18 | SB-5604 | W55 | Batch what can wait | Admin | Briefs, routines, checks and evals go through the provider's batch API where the adapter has one, with results by the next working morning | Anthropic and OpenAI batch APIs |
| 19 | SB-5605 | W55 | Cache the stable part | Admin | Adapters mark the stable prompt prefix (system prompt, skill, glossary) as cacheable where the provider supports it; traces record cached tokens and costs use the cached price | Anthropic and OpenAI prompt caching |
| 20 | SB-5606 | W55 | What it saved | Admin, finance | The cost statement and the admin's week show what reuse, batching, caching and the economy tier saved this month, from traces | Agent 365 Business Analytics, Notion credits |

## W53 Know who can see (epic E54, SB-5401 to SB-5407)

The goal is that a document shared too widely is noticed and held back before the brain spreads it, that its label goes wherever its content goes, and that everyone can see how far each agent reaches.

### S53.1 Find what is shared too widely

**SB-5401 Shared too widely.** At each sync, a document whose live ACL includes the whole tenant domain, `anyone` or a group above a size the cell sets is checked against what ingest already knows: the `personal_data` flag (SB-1906), national ids (SB-2207), masked secrets (SB-5102), a sensitive label (SB-5405) or a home in a restricted space. Matches appear on the review list (`section=wide`) and on the owner's inbox with the widest grant, the reason and a link to the source's sharing page. The brain changes nothing at the source.
*Laws:* LAW-01, LAW-07, LAW-19. *Acceptance:* the fixture salary sheet shared with the domain is listed with its reason. A wide document with nothing sensitive isn't. No text from the document appears on the list.

**SB-5402 Hold until the owner confirms.** A document listed by SB-5401 is held: left out of answers, briefs, routines and MCP results for everyone except its owner and the stewards of its space. The owner either narrows it at the source (the next sync clears the hold) or confirms **Shared on purpose** with a reason, which is audited and lasts until the sharing or the document changes. A space can set `wide_sensitive: warn` to show a warning instead of holding.
*Laws:* LAW-01, LAW-11, LAW-19. *Acceptance:* a held document is cited to its owner and nobody else. Confirming releases it and is audited. Editing the document holds it again.

**SB-5403 Sharing widened.** Each sync compares a document's ACL with the last one. When it widens (a group added, the domain, anyone with the link), the owner gets one notice naming the change; a sensitive document also goes on the review list. Narrowing is recorded but not announced.
*Laws:* LAW-01, LAW-21. *Acceptance:* a widening between two fake syncs sends one notice. A narrowing sends none. A document the owner can no longer open notifies the space's stewards instead.

**SB-5404 Who can see this.** `GET /v1/documents/{id}/audience` and **Who can see this** beside a cite, a note and a shared answer give the number of people who can open the document now, and the groups behind it that the person may see (their own groups and the space's); other groups are counted, never named. For a shared answer, it is the people who can open every cite.
*Laws:* LAW-01, LAW-02. *Acceptance:* counts match the fake directory. A group the person isn't in and that isn't the space's is never named.

### S53.2 Labels and reach

**SB-5405 Labels travel.** Ingest reads Google Drive labels and front matter `sensitivity:` into `Chunk.Label` (public, internal, confidential, restricted; restricted is LAW-20's label). Every answer, draft, share, download, print view, email and notification shows the highest label among its cites. Content labelled confidential or above goes only to a person (the web, a DM, an email to them), never to a channel or a channel-wide chat reply; a channel answer says "sent to you directly". A new law, **LAW-32**: *a label never drops: anything built from a labelled document carries its highest label and goes nowhere that label forbids.*
*Laws:* LAW-20, LAW-26, LAW-32 (new). *Acceptance:* a confidential cite marks the answer. A channel mention gets a DM instead. A note drafted from a confidential source carries the label into its front matter.

**SB-5406 How far each agent reaches.** For each agent, from its approvers, triggers, rhythms and open loans (SB-5303): the spaces and number of documents it can read, how many are labelled or held, and the change since last week. It shows on the registry (`GET /v1/admin/agents/NAME/reach`) and in the agent's record (SB-4707). Reach growing by more than a tenth in a week alerts the agent owner.
*Laws:* LAW-01, LAW-31. *Acceptance:* reach matches what a replay as each approver can open. A new approver who widens reach alerts once.

**SB-5407 Sensitive map.** Per space, weekly counts of documents by kind: personal data, masked secrets, each label, wide sharing and holds. It shows on the review list (`section=sensitive`), in the admin's week (SB-5006) and at `GET /v1/admin/sensitive`. Counts only, never titles outside the reader's access.
*Laws:* LAW-01, LAW-07. *Acceptance:* counts match the fixtures. A steward sees only their spaces' counts.

## W54 See and steer the context (epic E55, SB-5501 to SB-5507)

The goal is that a person sees what each answer was built from, can change it, and can find out why something they expected wasn't used.

### S54.1 Show the context

**SB-5501 Context panel.** **Context** beside each answer opens a panel listing what went into it, from the trace (SB-4606): passages with their documents, memory lines, the skill, the collection or documents asked about, earlier questions kept and dropped (SB-2901), departments consulted (SB-4501), passages skipped (SB-5101) and the model tier (SB-4405). Titles the person can't open now are counted.
*Laws:* LAW-01, LAW-26. *Acceptance:* the panel matches the trace. A document whose access was removed since the answer shows as a count.

**SB-5502 Keep or drop for this conversation.** In the panel, **Keep** adds a document to every later ask in the conversation (as SB-3004 does) and **Drop** leaves it out. Both are stored with the conversation in the browser (SB-2405), go into its download, and are checked as the person on each ask.
*Laws:* LAW-01. *Acceptance:* a dropped document isn't cited again in that conversation. A kept document the person loses access to stops being used. Nothing is stored on the server.

**SB-5503 Not for me.** **Don't use this for me** on a cite or in the panel keeps that document out of everything made as the person: answers, briefs, routines, research and MCP results. The list is on My data (SB-3501), included in the person's export and removed by forget (SB-3502). It is the person's own, like memory (LAW-25).
*Laws:* LAW-01, LAW-25. *Acceptance:* an excluded document is never cited to that person, and still is to others. Undoing it brings it back.

**SB-5504 Memory used, shown.** An answer that used a memory line says so under the answer, with the line and **Forget this** (SB-4303's forget). An answer that used none says nothing.
*Laws:* LAW-25. *Acceptance:* the line shown is the one in the trace. Forgetting it removes it from the next answer.

### S54.2 Ask why

**SB-5505 Memory over MCP.** The MCP server (SB-3801) offers the person's memory lines as a read-only resource and a `memory_propose` tool. A proposed line goes to the person's inbox and counts only after they confirm it in the web UI. Groups without MCP access (SB-5305) don't see either.
*Laws:* LAW-09, LAW-25. *Acceptance:* a proposed line isn't used until confirmed. Another person's memory is never offered.

**SB-5506 Why not this document?** **Why not…?** under an answer takes a document title or link. For a document the person can open, it says which applies: not indexed yet (and the last sync), outside the space or collection, ranked below the cut (its rank and the passage that came closest), skipped as suspect, held (SB-5402), dropped or excluded by the person, or retired. For any other document it says only "no document you can open by that name".
*Laws:* LAW-01, LAW-09. *Acceptance:* each reason has a fixture. A document the person can't open gets the same reply as one that doesn't exist.

**SB-5507 Retrieval view for stewards.** On a reported answer (SB-3606) or an unhelpful one (SB-3604), the steward of its space sees the ranked passages the ask retrieved: lexical and vector ranks, the fused score, and why each was kept or cut (per-document cap, scope, suspect, hold). Only passages the steward can open are shown; the rest are counted. Runs again as the steward, never as the asker.
*Laws:* LAW-01, LAW-26. *Acceptance:* the view matches a replay of the trace as the steward. The asker's own memory and exclusions are never shown.

## W55 Keep answering, spend less (epic E56, SB-5601 to SB-5606)

The goal is that a provider failing doesn't stop the brain, and that the same quality costs less.

### S55.1 Keep answering

**SB-5601 Fall back to a gated model.** sb-cell(5) `model.fallback` names a second provider and model. When the first returns a server error, a rate limit or a timeout, the ask goes to the fallback only if it passed the release's eval gate (`state/evals`); the answer and trace say "answered by the fallback model". After 3 failures in 5 minutes the cell uses the fallback for 10 minutes before trying the first again, and the health report shows it.
*Laws:* LAW-03, LAW-06. *Acceptance:* a failing fake provider switches to a gated fallback. An ungated fallback is never used. The switch and the return are in the health report.

**SB-5602 Passages when the model is down.** With no gated model available, the ask returns the best passages the person can open, cited, under "No answer written: the model is unavailable." Briefs and routines wait and run when a model comes back. The health report and the notice banner (SB-3207) say why.
*Laws:* LAW-01, LAW-14. *Acceptance:* with every fake provider down, an ask returns passages and no generated text. Nothing is labelled as an answer.

**SB-5603 Answered before, checked now.** An ask whose normalised question, space, scope and language match one answered in the last 7 days reuses that answer when every cite's document is unchanged (same version) and the asker can open all of them now. It says "answered on DATE, checked now" with **Ask fresh**. Memory, exclusions, kept or dropped documents, a file or a collection make an ask unique and never reuse.
*Laws:* LAW-01, LAW-26. *Acceptance:* a changed cite forces a fresh answer. An asker who can't open one cite gets a fresh answer. A reused answer costs no model call.

### S55.2 Spend less

**SB-5604 Batch what can wait.** Scheduled briefs, routines, checks (SB-5005) and evals go to the provider's batch API where the adapter has one (Anthropic, OpenAI), submitted by the notify job and collected on the next tick, with results by the next working morning. Anything a person is waiting for never batches. A provider without a batch API runs them as now.
*Laws:* LAW-05, LAW-16. *Acceptance:* a scheduled brief goes through the fake batch endpoint and lands by morning. An ask never batches. A failed batch reruns once, then normally.

**SB-5605 Cache the stable part.** Prompts are built with the stable part first (system prompt, agent and skill instructions, glossary) and the asker's material after. Adapters mark that prefix as cacheable where the provider supports it; traces record cached and uncached tokens, and costs use the provider's cached price. Nothing the asker sent is ever in the cached prefix.
*Laws:* LAW-02, LAW-07. *Acceptance:* the fake provider sees the same prefix across two asks in a space. No passage, memory or question appears in the prefix. Costs match the fake's prices.

**SB-5606 What it saved.** The cost statement (SB-3505) and the admin's week (SB-5006) gain a "saved" line: reuse, batching, caching and the economy tier, each in dollars this month, worked out from traces against the full price. `GET /v1/admin/costs/saved` gives the same as JSON.
*Laws:* LAW-11. *Acceptance:* the savings add up from the fixture traces. A month with none says so.

## What we leave out

- **Changing a source's sharing for the owner.** Connectors stay read-only; the brain reports and holds back.
- **Our own data-loss prevention for uploads and email.** SB-5102 masks what leaves; anything beyond the brain is the company's existing tools.
- **Bring your own key per department** (Notion). Spend is already attributed per space (SB-4105); one cell, one set of providers.
- **A credit system** (Notion). Dollars stay the unit, as in every budget so far.
- **Agents with their own computer or browser** (Notion, Windows 365 for Agents). Agents stay read-only and draft.
- **Semantic answer caching** across different wordings. Reuse needs the same normalised question; near matches go to suggested questions (SB-3307) instead.

## Decisions for the user

Decided by the user 2026-11-26: every recommended option.

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W53, W54, W55. *Other:* W55 first, for failover and savings sooner.
2. **Widely shared sensitive documents (SB-5402).** *Recommended:* hold them back from everyone but the owner and stewards until the owner narrows or confirms; a space may choose `warn`. *Other:* warn only by default.
3. **Labels (SB-5405, LAW-32).** *Recommended:* read Drive labels and front matter, four levels, and keep confidential content out of chat channels (it goes to the person directly). *Other:* show labels only, and send answers where they were asked.
4. **Answer reuse (SB-5603).** *Recommended:* on, 7 days, same question and space, every cite unchanged and open to the asker, always marked with **Ask fresh**. *Other:* off unless a space turns it on.
5. **Fallback model (SB-5601).** *Recommended:* a second provider, used only if it passed the same eval gate; otherwise passages only (SB-5602). *Other:* no fallback; passages only when the model is down.
6. **Batching (SB-5604).** *Recommended:* scheduled briefs, routines, checks and evals through batch APIs, results by the next working morning. *Other:* keep running them as now.
7. **Memory over MCP (SB-5505).** *Recommended:* read, and propose lines the person confirms in the web UI. *Other:* read only.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Glean release notes (release 566)](https://docs.glean.com/release-notes/)
- [Glean release 568, via Releasebot](https://releasebot.io/updates/glean)
- [Notion releases, 19 October 2026](https://www.notion.com/releases/2026-10-19)
- [SiliconANGLE: at Ignite 2026, Microsoft reinvents Copilot](https://siliconangle.com/2026/11/18/at-ignite-2026-microsoft-reinvents-copilot-as-the-autonomous-engine-of-enterprise-work/)
- [Constellation Research: Ignite 2026](https://www.constellationr.com/insights/news/ignite-2026-microsoft-makes-case-be-your-work-operating-system-intelligence-layer)
- [A Guide to Cloud: Microsoft 365 Copilot, November 2026](https://www.aguidetocloud.com/blog/microsoft-365-copilot-november-2026-updates/)
- [Atlassian community: Rovo updates, November (debug view for agents)](https://community.atlassian.com/forums/Atlassian-AI-Rovo-articles/%EF%B8%8F-Sherif-sclusive-Atlassian-AI-amp-Rovo-Updates-November/ba-p/3159574)

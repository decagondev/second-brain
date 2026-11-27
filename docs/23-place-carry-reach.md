---
title: Right Model, Right Place; Carry the Conversation; Reach Microsoft 365
summary: Concept for W56 to W58 (accepted 2026-11-27): twenty features. Batches that are collected later and work on OpenAI too, people first when a provider is busy, a deeper model only where it helps and model calls kept in the company's region; follow-ups in chat threads, conversations that move to the web, other devices and colleagues, and a nudge back to unfinished work; and SharePoint, OneDrive, Purview labels, Entra, Teams, Outlook calendar and shared mailboxes. No new service, store or Go module.
---

# Right model, right place; carry the conversation; reach Microsoft 365

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-27 | Platform lead | Accepted by the user 2026-11-27, with every recommended decision | [Principles](01-principles.md) · [22-see-steer-spend.md](22-see-steer-spend.md) · [PRD waves](03-prd.md#waves) · [ADR-0023](adr/ADR-0023-microsoft-365.md) |

## Summary

W53 to W55 are built and pushed (`172f77f`, site `60ec72e`). Documents shared too widely are held back, labels travel with every answer (LAW-32), a person sees and steers what an answer used, a gated fallback model keeps answers coming, and reuse, batching and prompt caching cut the cost.

Two things were left open on SB-5604: batches are collected within the run that sends them (it waits up to 10 minutes) rather than on a later tick, and OpenAI's file-based batch API isn't wired, so OpenAI-compatible providers pay full price for work that could wait.

The user asked for the next lot of waves. This round looked at what shipped in November 2026: Glean (releases 567 to 570: a chat context panel going generally available on 24 November, an Auto mode that picks the model and reasoning level for each request, skills shared over MCP through `skills/list` and `skills/get`, Outlook email triggers through webhooks, service credentials for agents, and agents that run when an artifact gets a new version), Notion (agent budgets per agent under its new credit system, agents assigned through people properties, agents that run for more than a day), ChatGPT Enterprise (@ChatGPT in Slack and Teams, chats started in Slack shown in the ChatGPT sidebar, "dots" that carry tasks between conversations), Gemini Enterprise (multi-turn conversations in Slack, an agent that keeps working after the laptop closes) and Slack (regional AI inference, starting with Japan; tasks that start on a message or a reaction).

Three gaps stand out.

1. **Work that can wait still waits in the wrong place, and every ask gets the same model.** A batched brief holds its run open for up to 10 minutes and is lost if the cell restarts. OpenAI-compatible providers never batch. When a provider rate-limits, scheduled work and people's questions compete equally. A hard research synthesis and a one-line lookup use the same tier, and nothing stops a model call from leaving the region a company's data must stay in.
2. **A conversation lives in one place.** A question asked in Slack or Mattermost can't be followed up in its thread, can't be continued on the web, lives only in the browser it was asked in, and can be shared only one answer at a time. Unfinished work (a question waiting on a steward, a research plan waiting for approval) is found only by opening the inbox.
3. **The brain reaches Google's workplace and not Microsoft's.** Drive, Calendar, Meet transcripts, Slack and Mattermost are connected; SharePoint, OneDrive, Purview labels, Entra groups, Teams and Outlook are not, though most companies of our size run on them.

Three waves, twenty tickets: **W56 Right Model, Right Place** is first, then **W57 Carry the Conversation**, then **W58 Reach Microsoft 365**.

## What others do, November 2026

| Product | What it does that matters here |
|---|---|
| **Glean (releases 567 to 570)** | **Auto mode** picks the model and reasoning level per request. **Skills over MCP** through the standard `skills/list` and `skills/get`. **Outlook email triggers** start agents on a new message through webhooks. **Service credentials** for agents. Agents that run when an **artifact gets a new major version**. The **chat context panel** goes generally available on 24 November. |
| **Notion (November)** | **Agent budgets** per agent under one credit system. Agents **assigned through people properties** and mentioned by id. Agents that run for more than a day. |
| **ChatGPT Enterprise (November)** | **@ChatGPT in Slack and Teams**, configured by admins. **Chats started in Slack appear in the ChatGPT sidebar.** **Dots** carry tasks forward between conversations. |
| **Gemini Enterprise (November)** | **Multi-turn conversations in Slack** threads. An agent that **keeps working in the cloud** after the laptop closes. Federated Salesforce data. |
| **Slack (November)** | **Regional AI inference** for Business+ and Enterprise+, starting with Japan. Tasks that start on a **new message or a reaction**. |
| **Model providers** | Batch APIs at about half price with a 24-hour window (Anthropic Message Batches, OpenAI's file-based Batch API), and models with a selectable reasoning effort. |

**What they agree on:** let the product pick the model per request; keep AI processing in the customer's region; follow a person between chat and the web and carry their unfinished work; and reach whichever office suite the company runs.

**Where we differ, on purpose:**

- **Every tier, including a deeper one, passes the same eval gate** (LAW-06). The cell never picks an ungated model to save money or to think harder.
- **A model call stays in its region** (a new law, **LAW-33**): with `model.region` set, no call, whether main, small, deep, fallback or batch, goes to a provider that doesn't declare that region. Unknown means refuse (LAW-03).
- **A conversation is the person's own.** Syncing it across devices is the person's choice, holds only their questions and trace ids, ends with the trace retention and goes with forget. A shared conversation is checked again as each reader (LAW-26), answer by answer.
- **Microsoft 365 stays read-only.** Connectors read; Teams is a place to ask; mail is read only from a shared mailbox a space names, never a person's own inbox. An unmapped Purview label counts as `restricted` until an admin maps it.

## Keeping it small

Rules 1 to 17 from earlier waves still apply (no new service, store or Go module; at most one setting per feature; no new page; nothing widens what anyone sees; a person approves every write; the tree only narrows; agents are definitions in the bundle; stored results are checked when read; every long run has a ceiling; one level of consultation; the AI label never comes off; a rhythm only drafts; a brief about a person draws only on what they published; lent access lasts one run; a label never drops). One more:

18. **A model call stays in its region** (LAW-33), and a new vendor is an adapter against a recorded fake, built with the standard library only.

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-5701 | W56 | Collect batches later | Admin | A batch is submitted by one notify tick and collected by a later one; its id is kept in `state/batches`, a restart loses nothing, and anything not back by the next working morning runs as a normal call | Anthropic and OpenAI batch windows |
| 2 | SB-5702 | W56 | OpenAI batches | Admin | The OpenAI adapter uploads a JSONL file, creates a batch, collects its output file and deletes both; an OpenAI-compatible provider batches only if sb-cell(5) says it supports it | OpenAI Batch API |
| 3 | SB-5703 | W56 | People first when the provider is busy | Everyone, admin | When a provider rate-limits, asks go before scheduled work, each space gets its share, and an ask that would wait more than 20 seconds gets passages instead of an error | Glean, Notion credits |
| 4 | SB-5704 | W56 | Think harder only when it helps | Everyone, admin | sb-cell(5) `model.deep` names a gated deeper model or reasoning effort, used only for research synthesis, tables, consultations and a retry after a failed citation check; the trace says why | Glean Auto mode |
| 5 | SB-5705 | W56 | Keep model calls in region | Admin, security | sb-cell(5) `model.region` and a `region` on every provider; no call leaves the region, and a provider without one is never used (LAW-33) | Slack regional inference |
| 6 | SB-5706 | W56 | Cache hits you can see | Admin | The share of input read from the prompt cache, per space and agent, from traces; a prefix that changes on every ask is named in the admin's week | Anthropic and OpenAI prompt caching |
| 7 | SB-5801 | W57 | Follow up in the thread | Everyone | A reply in the Slack or Mattermost thread under the brain's answer continues the same conversation, re-authorised each turn | Gemini Enterprise in Slack |
| 8 | SB-5802 | W57 | Continue on the web | Everyone | A conversation from a chat app shows in the person's Recent on the web with **Continue here**; every answer is checked again as them | ChatGPT in Slack |
| 9 | SB-5803 | W57 | Your conversations on every device | Everyone | If the person turns it on, their conversation list (questions and trace ids) follows them to every browser, for as long as traces are kept, and goes with forget | ChatGPT, Glean |
| 10 | SB-5804 | W57 | Share a conversation | Everyone | Send a whole conversation to a colleague; each answer is checked as them, one they can't open every cite of is left out and counted, and the highest label goes with it | Glean, Notion |
| 11 | SB-5805 | W57 | Where you left off | Everyone | A new conversation starts with up to 3 of the person's open things: a question waiting on a steward, a research plan to approve, a confirmed action not done, an overdue wait | ChatGPT dots |
| 12 | SB-5806 | W57 | React to save | Everyone, steward | A chosen reaction on a Slack or Mattermost thread offers the person, in a DM, to save the thread for the space's stewards or ask the brain about it; nothing moves until they confirm | Slack reaction tasks |
| 13 | SB-5807 | W57 | Skills over MCP | Everyone | The MCP server answers `skills/list` and `skills/get` beside `prompts/list`, with only the skills of agents the token's person can use | Glean skills over MCP |
| 14 | SB-5901 | W58 | SharePoint and OneDrive | Admin, everyone | A connector over Microsoft Graph delta queries, with live permissions and sharing links, Office text, pages and slides | Glean, Copilot |
| 15 | SB-5902 | W58 | Purview labels | Security, everyone | Sensitivity labels mapped in sb-cell(5) to the four levels of SB-5405; an unmapped label counts as `restricted` until an admin maps it | Purview |
| 16 | SB-5903 | W58 | Entra groups and people | Admin | Entra ID groups, with nested membership, behind every access check, and Entra provisioning revoking a leaver within 5 minutes (LAW-12) | Copilot, Glean |
| 17 | SB-5904 | W58 | Ask in Teams | Everyone | A Teams bot like the Slack and Mattermost apps: mentions and DMs, cites, labels (confidential goes to the person), follow-ups in the thread | ChatGPT in Teams, Copilot |
| 18 | SB-5905 | W58 | Outlook calendar for meeting prep | Everyone | Meeting prep (SB-4202) and `/v1/me/events` read Outlook calendars as they read Google Calendar | Copilot |
| 19 | SB-5906 | W58 | Teams meeting transcripts | Everyone | Transcripts of Teams meetings are ingested as type `transcript`, open only to people who could open the meeting's recording | Copilot, Glean |
| 20 | SB-5907 | W58 | Start on a new email | Steward | `on: [mail-received]` for a shared mailbox a space names: an agent drafts a reply note for a person to send; a person's own inbox is never read | Glean Outlook triggers |

## W56 Right model, right place (epic E57, SB-5701 to SB-5706)

The goal is that every call goes to the cheapest gated model that fits it, in the region the company allows, that people come first when a provider is busy, and that work which can wait is batched on every provider without holding a run open.

### S56.1 Batches and busy providers

**SB-5701 Collect batches later.** Closes SB-5604's first follow-up. A batchable call is submitted by the notify job's tick and its batch id, provider, caller and step are written to `state/batches/ID.json` (never the prompt). Later ticks collect what has ended and hand each result to its step, so the run that submitted it ends at once. A restart between submit and collect loses nothing. A batch not back by 06:00 on the next working morning (cell time) is cancelled and its step runs as a normal call, rebuilt from its source, so results still land by morning. A failed batch reruns once, then normally, as now.
*Laws:* LAW-05, LAW-07. *Acceptance:* a scheduled brief submitted on one tick is delivered on a later one. Killing the cell between the two loses nothing. A batch still running at 06:00 runs normally and is cancelled at the provider. No prompt text is in `state/batches`.

**SB-5702 OpenAI batches.** Closes SB-5604's second follow-up. The `openaicompat` adapter implements `ports.Batcher` against OpenAI's file-based Batch API: it uploads the calls as JSONL with purpose `batch`, creates a batch with the 24-hour window, collects the output and error files, and deletes the input and output files once collected. sb-cell(5) `model.batch: true` says a provider supports it; without it an OpenAI-compatible provider runs every call as before.
*Laws:* LAW-05, LAW-07. *Acceptance:* a brief goes through the recorded OpenAI fake and is charged at the batch rate. Both files are deleted after collection. A provider without `model.batch` never batches.

**SB-5703 People first when the provider is busy.** When a provider returns a rate limit, the gateway queues calls per space and serves a person's ask before any scheduled step, and each space in turn, so one busy space can't starve the others. Scheduled work that can't run waits for the next tick. An ask that would wait more than 20 seconds gets passages (SB-5602) with "the model is busy". The health report counts rate limits and waits per space.
*Laws:* LAW-03, LAW-14. *Acceptance:* in a fake 429 storm, asks finish before scheduled steps and no space takes more than its share while another waits. An ask past 20 seconds gets passages and no generated text.

### S56.2 The right model

**SB-5704 Think harder only when it helps.** sb-cell(5) `model.deep` names a deeper model, or the main model with a higher reasoning effort, of the same provider. It is used only for a research synthesis (SB-4201, SB-4903), a table (SB-4307), a consultation's synthesis (SB-4501) and one retry after a draft fails its citation check, and only while the evals job has gated it (LAW-06). A space's routing policy (SB-5002) can forbid it. The trace records the tier and why, and spend and limits count it at its own price.
*Laws:* LAW-06, LAW-14. *Acceptance:* a table ask uses the deep tier and a plain lookup doesn't. An ungated deep tier is never used. A space that forbids it never gets it.

**SB-5705 Keep model calls in region.** sb-cell(5) `model.region` names where model calls may go (for example `eu`), and every provider entry (main, small, deep, fallback, embedder when remote) declares its `region`. With `model.region` set, a call never goes to a provider of another region or of none: the fallback, batching and every tier obey it, and a failing in-region provider gives passages, not an out-of-region answer. The health report and the inventory (SB-5007) show each provider's region. A new law, **LAW-33**: *a model call stays in its region.*
*Laws:* LAW-03, LAW-33 (new). *Acceptance:* a fallback in another region is never called. A provider with no `region` is never called while `model.region` is set. The inventory lists every provider's region.

**SB-5706 Cache hits you can see.** From traces (SB-5605), the share of input tokens read from the prompt cache per space and agent this month, at `GET /v1/admin/costs/cache` and in the admin's week (SB-5006). A space or agent whose cached share stays under a tenth while it has repeated asks is named, with the part of its prefix that changed between asks (a date, a reordered glossary), never the asker's text.
*Laws:* LAW-07, LAW-11. *Acceptance:* the shares add up from fixture traces. A fixture prefix with a timestamp in it is named. No question, passage or memory appears in the report.

## W57 Carry the conversation (epic E58, SB-5801 to SB-5807)

The goal is that a conversation follows a person from a chat thread to the web, to their other devices and to a colleague, and that work they left unfinished finds them again.

### S57.1 Follow up anywhere

**SB-5801 Follow up in the thread.** In Slack and Mattermost, a reply in the thread under the brain's answer continues the same conversation: one `cv_…` per thread, the last 8 earlier questions sent (SB-2901), each turn re-authorised as the person who replied. A reply from a different person starts their own conversation in the same thread. Labels hold (LAW-32): a confidential answer in a channel thread goes to the person directly.
*Laws:* LAW-01, LAW-32. *Acceptance:* a follow-up in the thread is answered with the earlier question in its context. A second person's reply doesn't see the first person's conversation. A confidential follow-up goes to a DM.

**SB-5802 Continue on the web.** A conversation started in a chat app shows in the person's Recent on the web (SB-2405) with **Continue here**, from the traces the person owns. Opening it shows each answer checked again as them now (LAW-26); one they can no longer open every cite of shows as "no longer available to you".
*Laws:* LAW-01, LAW-26. *Acceptance:* a Slack conversation appears in the same person's Recent and nobody else's. A cite whose access was removed since is not shown.

**SB-5803 Your conversations on every device.** On My data, **Keep my conversations on every device** stores the person's conversation list server-side: each conversation's questions, its kept and dropped documents (SB-5502) and its trace ids, never the answers, which are rebuilt from traces and checked again. Entries go when the traces do (30 days), are in the person's export (SB-3501) and go with forget (SB-3502). Off by default; turning it off deletes the server copy.
*Laws:* LAW-25, LAW-26. *Acceptance:* a conversation started in one browser opens in another after sign-in. Turning it off deletes the copy. Forget removes it. Nobody else can list it, admins included.

**SB-5804 Share a conversation.** **Share conversation** sends a whole conversation to a colleague, as shares of one answer do (SB-1703). The reader sees each answer checked again as them; an answer they can't open every cite of is left out and counted ("2 answers not shared with you"). The highest label of what they see goes with it (LAW-32), and a confidential share is never posted to a channel.
*Laws:* LAW-01, LAW-26, LAW-32. *Acceptance:* a reader who can open half the cites sees only the answers whose cites they can all open. The share carries the label. Revoking the share removes it.

### S57.2 Pick it up again

**SB-5805 Where you left off.** A new conversation on the web, and the person's first DM to the brain in a day, start with up to 3 of their open things from the last 14 days: a question they asked that waits on a steward (SB-1704), a research plan waiting for their approval (SB-4903), an action they confirmed and haven't done (SB-4203), and an overdue wait they set (SB-4504). Each opens where it is. All of it comes from existing stores; nothing new is kept, and **Not now** hides it for the day.
*Laws:* LAW-01, LAW-26. *Acceptance:* each kind has a fixture. An item the person can no longer open isn't shown. Nothing shows when nothing is open.

**SB-5806 React to save.** sb-cell(5) `chat.save_reaction` (default `brain`) names one reaction. A person who adds it to a Slack or Mattermost thread gets a DM offering **Save for the stewards** (SB-3703, into the space the channel belongs to) or **Ask about this thread**. Nothing is read beyond the reaction until they choose, the thread's text is data (LAW-09), and a thread in a channel the brain isn't in is never read.
*Laws:* LAW-01, LAW-09. *Acceptance:* a reaction sends one DM and saves nothing on its own. Confirming saves the thread to the right space's queue. A reaction in a channel without the app does nothing.

**SB-5807 Skills over MCP.** The MCP server (SB-3801) answers `skills/list` and `skills/get` beside `prompts/list` and `prompts/get` (SB-4601): each skill's name, description and instructions, only for agents the token's person can use (SB-4306) and groups allowed MCP (SB-5305). A skill fetched this way is the same text as the cell's, signed in the bundle.
*Laws:* LAW-01, LAW-09. *Acceptance:* a client lists the same skills it could run through prompts. A skill of a restricted department agent is never listed to someone outside it.

## W58 Reach Microsoft 365 (epic E59, SB-5901 to SB-5907)

The goal is that a company on Microsoft 365 gets the same brain as one on Google Workspace: documents with their sensitivity labels, groups behind every check, Teams as a place to ask, and calendars, transcripts and shared mailboxes feeding the same agents (ADR-0023 proposed).

### S58.1 Documents and groups

**SB-5901 SharePoint and OneDrive.** A connector over Microsoft Graph: sites and drives chosen in sb-cell(5), changes read with delta queries from a stored cursor, the live permissions of each item including sharing links (`anonymous` and `organization` scopes count as wide for SB-5401), and text through the Office readers (SB-2003, SB-2206) with pages and slides (SB-5205). Built against a recorded Graph fake with the standard library; a real tenant run stays In review.
*Laws:* LAW-01, LAW-13. *Acceptance:* the fake's site syncs with its permissions, a sharing link widens the audience, and a delete removes every chunk at the next sync.

**SB-5902 Purview labels.** The connector reads each item's sensitivity label id. sb-cell(5) `labels.microsoft` maps label ids to the four levels (public, internal, confidential, restricted). An unmapped label counts as `restricted` until an admin maps it, and the health report lists unmapped labels by id and count, never by document.
*Laws:* LAW-03, LAW-32. *Acceptance:* a mapped confidential label marks the answer. An unmapped label is treated as restricted and shows on health. Changing a mapping relabels at the next sync.

**SB-5903 Entra groups and people.** A directory adapter for Entra ID: groups with transitive membership for access checks and audiences (SB-5404), people's managers for the tree and handovers, and Entra's SCIM provisioning into the existing endpoint (SB-107) so a leaver is revoked within 5 minutes (LAW-12). A group too large to expand is counted, never guessed.
*Laws:* LAW-01, LAW-12. *Acceptance:* nested group membership grants exactly what the fake directory says. A deprovisioned person loses access within 5 minutes on the fake clock.

### S58.2 Teams, calendar and mail

**SB-5904 Ask in Teams.** A Teams app beside the Slack and Mattermost ones (`internal/adapters/chatapp/teamsapp`, `POST /chat/teams`): mentions and DMs, cites with titles the person can open, follow-ups in the reply chain (SB-5801), and labels (a confidential answer goes to the person in a DM). Incoming activities are verified against Microsoft's signing keys with the standard library.
*Laws:* LAW-01, LAW-32. *Acceptance:* a mention in the fake Teams channel is answered with cites. An unsigned or wrongly signed activity is refused. A confidential answer goes to a DM.

**SB-5905 Outlook calendar for meeting prep.** Meeting prep (SB-4202), `/v1/me/events` and the leader's week read Outlook calendars through Graph as they read Google Calendar (`connectors/gcal`): the person's own events only, with attendees resolved through Entra (SB-5903).
*Laws:* LAW-01, LAW-30. *Acceptance:* a fake Outlook meeting gets the same prep as a Google one. Another person's calendar is never read.

**SB-5906 Teams meeting transcripts.** Transcripts of Teams meetings (Graph's VTT) are ingested through `folder.TranscriptText` as type `transcript` (SB-3701), open only to the people who could open the meeting's recording, with speakers and times. A meeting without a transcript is skipped.
*Laws:* LAW-01, LAW-09. *Acceptance:* a fake transcript is cited to an attendee and not to someone outside the meeting. Text in a transcript that looks like an instruction is treated as data.

**SB-5907 Start on a new email.** A space may name one shared mailbox (sb-space(5) `mailbox`, for example `support@`) and `on: [mail-received]` (SB-5301). A new message there, read through Graph change notifications or the next sync, starts the space's agent, which drafts a reply note for a steward to edit and send themselves (SB-5304). The mail is data (LAW-09), its attachments are read like files (SB-3802), and a person's own mailbox is never read.
*Laws:* LAW-09, LAW-16. *Acceptance:* a fake message to the shared mailbox puts one draft in the steward queue and sends nothing. A mailbox not named in the manifest is never read.

## What we leave out

- **Indexing anyone's own mail.** Only a shared mailbox a space names is read, and only to start an agent.
- **Service credentials for agents** (Glean). Agents keep reading as their approvers, narrowed, and borrow access for one run with a named lender (LAW-31); a standing identity of their own would widen what an agent can reach.
- **A general Auto model chooser** that picks any model per request. Tiers stay a short, gated list (small, main, deep, fallback) with fixed rules for when each is used.
- **Agents that run for days.** Long research keeps its ceiling and checkpoints (SB-4903).
- **Confluence, Jira and Notion connectors.** Microsoft 365 reaches more of the companies we serve; these come later if asked.
- **Credits.** Dollars stay the unit.

## Decisions for the user

Decided by the user 2026-11-27: every recommended option.

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W56, W57, W58. *Other:* W58 first, if a company on Microsoft 365 is waiting; or Confluence and Jira instead of Microsoft 365 for W58.
2. **Batches (SB-5701, SB-5702).** *Recommended:* submit on one tick, collect on a later one, with anything not back by 06:00 on the next working morning run normally; OpenAI batches only for providers that declare `model.batch`. *Other:* keep collecting within the run.
3. **Region (SB-5705, LAW-33).** *Recommended:* an optional `model.region`; when it is set, no model call of any tier leaves it, and a provider that declares no region is never used. *Other:* report each call's region without enforcing it.
4. **A deeper tier (SB-5704).** *Recommended:* only for research synthesis, tables, consultations and one retry after a failed citation check, and only while gated. *Other:* leave it out and keep two tiers.
5. **Conversations on every device (SB-5803).** *Recommended:* the person turns it on; questions, kept and dropped documents and trace ids only, for 30 days, gone with forget. *Other:* on for everyone by default; or keep conversations in the browser only and drop SB-5803.
6. **React to save (SB-5806).** *Recommended:* one reaction (default `brain`), and nothing is read or saved until the person confirms in a DM. *Other:* leave it out.
7. **Microsoft 365 (W58, ADR-0023).** *Recommended:* SharePoint and OneDrive, Purview labels with unmapped labels treated as `restricted`, Entra, Teams, Outlook calendar, Teams transcripts and shared-mailbox triggers only; built against recorded fakes, with real-tenant runs In review. *Other:* documents and Entra only (SB-5901 to SB-5903), and the rest later.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Glean release notes](https://docs.glean.com/release-notes/), [19 November 2026](https://docs.glean.com/release-notes/releases/2026-11-19-november-release), [6 November 2026](https://docs.glean.com/release-notes/releases/2026-11-06-november-release), [3 November 2026](https://docs.glean.com/release-notes/releases/2026-11-03-november-release)
- [Notion release notes via Releasebot, November 2026](https://releasebot.io/updates/notion)
- [ChatGPT Enterprise and Edu release notes](https://help.openai.com/en/articles/10128477-chatgpt-enterprise-and-edu-release-notes)
- [Gemini Enterprise release notes](https://docs.cloud.google.com/gemini/enterprise/docs/release-notes?hl=en)
- [Slack release notes via Releasebot, November 2026](https://releasebot.io/updates/slack)

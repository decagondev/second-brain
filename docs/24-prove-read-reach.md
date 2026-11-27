---
title: Proven for Real; Every Format, Every Language; Reach Confluence and Jira
summary: Concept for W59 to W61 (accepted 2026-11-27): twenty features. Proofs and fresh fakes from real systems, a Microsoft 365 setup check, first syncs that survive a big tenant, reactions and direct questions in Mattermost and a list of the AI tools that reach the brain; scans, images and recordings read by gated models and marked, questions that find documents in other languages, translated passages, three more UI languages and an accessible UI; and Confluence, Jira and Jira Service Management, read-only. No new store or Go module.
---

# Proven for real; every format, every language; reach Confluence and Jira

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-27 | Platform lead | Accepted by the user 2026-11-27, with every recommended decision | [Principles](01-principles.md) · [23-place-carry-reach.md](23-place-carry-reach.md) · [PRD waves](03-prd.md#waves) · [ADR-0024](adr/ADR-0024-atlassian.md) |

## Summary

W56 to W58 are built and pushed (`6b4db8e`, site `ccdc268`). Batches are collected on a later tick and work on OpenAI, people come first when a provider is busy, a gated deep tier is used only where it helps, model calls stay in their region (LAW-33), a conversation follows a person from a chat thread to the web, other devices and colleagues, and Microsoft 365 is reached read-only over Graph.

Two things were left open by W58:

- **SB-5901 is In review**: SharePoint and OneDrive, like every Graph adapter, are tested only against the recorded Graph fake. Nothing yet helps a person prove them on a real tenant, or keeps the fake honest once someone does.
- **React to save (SB-5806) and direct questions (SB-5805) are Slack only**: Mattermost's outgoing webhooks carry no reaction events, so `/brain save LINK` stays the Mattermost way.

And older ones still wait on real systems: SB-1605 and SB-2601 (a real Linux VM), SB-2005, SB-2105 and SB-2603 (a real Mattermost), SB-903 and SB-908 (a real GCP project).

This round looked at what shipped in November 2026: Microsoft Ignite 2026 (Copilot Home with Projects that hold conversations, tasks and files; Autopilot agents with their own Entra identities; Entra Agent IDs for local agents such as GitHub Copilot CLI and Claude Code; Agent 365 cost management), Glean (releases 567 to 570: the chat side panel generally available on 24 November, Auto mode, skills over MCP, Outlook triggers, an AI Gateway section in the admin console, a partner that reads millions of images a month), Notion (agents rebuilt on a model router, bring your own model key, an AI Command Center for admins announced) and Gemini Enterprise (federated Salesforce data, Notion connectors in preview).

Three gaps stand out.

1. **What's built against fakes isn't proven against the real thing.** Ten tickets wait In review for a real tenant, server or VM, and when one comes there's no single run that checks each criterion and leaves evidence a reviewer can trust without seeing anyone's documents. A big tenant's first sync has never been tried: Graph throttles, and a restart starts again from the top. Mattermost lags Slack. And nobody can see which AI tools reach the brain on a person's behalf, which Entra now does for local agents.
2. **The brain reads only text, and mostly in one language.** A scanned contract, a whiteboard photo, a diagram in a slide or a recorded all-hands is invisible. A question in German doesn't find the English policy that answers it, a cited passage in French can't be read by someone who doesn't read French, the UI has three languages, and the UI has never been checked for people using a screen reader or a keyboard alone.
3. **Confluence and Jira are left out**, though they are where many engineering and product teams keep their knowledge and their work. W58 skipped them for Microsoft 365, "later if asked".

Three waves, twenty tickets: **W59 Proven for Real** is first, then **W60 Every Format, Every Language**, then **W61 Reach Confluence and Jira**.

## What others do, November 2026

| Product | What it does that matters here |
|---|---|
| **Microsoft Ignite 2026** | **Copilot Home** puts chat and Cowork in one place, with **Projects** grouping conversations, tasks and files. **Autopilot agents** with their own Entra identities run scheduled tasks. **Entra Agent ID** names discovered local agents (GitHub Copilot CLI, Claude Code) so admins can see and govern them. **Agent 365 cost management.** Execution containers limit an agent's file and network access. |
| **Glean (releases 567 to 570)** | The **chat side panel** (agents, outputs and sources used) generally available on 24 November. **Auto mode.** **Skills over MCP.** **Outlook email triggers.** An **AI Gateway** section in the admin console. A partner that **scans up to three million images a month**. |
| **Notion (October and November)** | Agents rebuilt on a model router (**Auto**), **bring your own model key** in private alpha, an improved MCP server, and an **AI Command Center** for usage, spending limits and agent governance, announced as coming. |
| **Gemini Enterprise (November)** | **Federated Salesforce data**, **Notion connectors** in preview, Workflow Builder generally available. |

**What they agree on:** show admins every AI tool and agent that acts for a person; read what's in images and recordings, not just text; and connect whichever wiki and tracker a team already uses.

**Where we differ, on purpose:**

- **Proof without content.** A proof run on a real system keeps pass or fail per criterion, counts, timings and hashes, signed, and never a document's text, a name or an id. A fake refreshed from a real sandbox keeps the shape of the exchange and replaces every value.
- **Text a model read is marked as such.** A scan, an image, a recording or a translation read by a model carries the AI mark (LAW-28) wherever it is cited, and is never shown as the document's own words. Every reading model passes the eval gate (LAW-06) and stays in region (LAW-33).
- **No standing identities for agents.** We don't copy Autopilot's own Entra identities, for the reason W58 gave for Glean's service credentials: agents keep reading as their approvers, narrowed (LAW-31).
- **Unknown access stays deny** (LAW-03). That's why Notion stays out: its API doesn't say who can open a page.

## Keeping it small

Rules 1 to 18 from earlier waves still apply (no new service, store or Go module; at most one setting per feature; no new page; nothing widens what anyone sees; a person approves every write; the tree only narrows; agents are definitions in the bundle; stored results are checked when read; every long run has a ceiling; one level of consultation; the AI label never comes off; a rhythm only drafts; a brief about a person draws only on what they published; lent access lasts one run; a label never drops; a model call stays in its region, and a new vendor is an adapter against a recorded fake). One more:

19. **Text the brain didn't read from the source itself** (a scan or image read by a model, a recording transcribed, a passage translated) **carries the AI mark** (LAW-28), is cited as machine-read, and an optional reader runs as a compose profile the cell can do without, as the local model does (SB-1710).

## The twenty features

| # | Ticket | Wave | Feature | For | What it does | Seen in |
|---|---|---|---|---|---|---|
| 1 | SB-6001 | W59 | Prove it on the real thing | Admin, reviewer | `sb prove TICKET` runs a ticket's acceptance criteria against the configured real system, read-only, and writes signed evidence with no content in it | Pilot sign-offs |
| 2 | SB-6002 | W59 | Fakes refreshed from a real sandbox | Developer | `sb record` captures a real sandbox's exchanges and replaces every value, so a recorded fake matches what the vendor really sends | Contract testing |
| 3 | SB-6003 | W59 | Check the Microsoft 365 setup | Admin | `sb selfhost check-microsoft` checks the app registration, refuses any write permission, checks the mailbox access policy and lists unmapped labels | Copilot admin readiness |
| 4 | SB-6004 | W59 | A big tenant's first sync | Admin | First syncs are paced, honour `Retry-After`, resume from a checkpoint after a restart and show progress on health; answers say a source is still indexing | Glean, Copilot connectors |
| 5 | SB-6005 | W59 | Reactions and direct questions in Mattermost | Everyone | The Mattermost bot listens on its websocket, so react to save and direct questions work as in Slack | Slack reaction tasks |
| 6 | SB-6006 | W59 | The AI tools that reach the brain | Everyone, admin | Every MCP token names its client; My data lists each one with its last use and a revoke, and admins see counts per client and may allow only named ones | Entra Agent ID, Glean AI Gateway |
| 7 | SB-6101 | W60 | Read scans and images | Everyone | Pages with no text layer and images are read by a gated `model.vision`, marked as machine-read; off by default | Glean's image partner, Copilot |
| 8 | SB-6102 | W60 | Recordings become transcripts | Everyone | Audio and video files are transcribed through an optional `transcribe` profile or a gated provider, as type `transcript`, marked | Copilot, Otter |
| 9 | SB-6103 | W60 | Ask in one language, find another | Everyone | The question is also searched in the languages sb-cell(5) names, translated by the gated small tier; cites stay in their own language | Glean, Copilot |
| 10 | SB-6104 | W60 | A glossary in every language | Steward | A glossary term carries its words in each language, used when translating questions and when marking terms | Notion, Confluence glossaries |
| 11 | SB-6105 | W60 | Translate a cited passage | Everyone | **Translate** on a passage shows it in the UI's language beside the original, marked, kept nowhere | ChatGPT, Gemini |
| 12 | SB-6106 | W60 | Spanish, Italian and Dutch | Everyone | The web UI, chat replies and answers in three more languages | Every suite |
| 13 | SB-6107 | W60 | Usable by everyone | Everyone | WCAG 2.2 AA: an automated accessibility check over every tab in `make web`, keyboard-only paths and screen reader names for cites, labels and notices | Accessibility law |
| 14 | SB-6201 | W61 | Confluence | Everyone | Confluence Cloud spaces, pages, blog posts and attachments with their restrictions and space permissions, read-only | Glean, Copilot |
| 15 | SB-6202 | W61 | Atlassian groups and people | Admin | Atlassian groups behind every check, and SCIM from Atlassian Guard revoking a leaver within 5 minutes (LAW-12) | Glean |
| 16 | SB-6203 | W61 | Check the Atlassian setup | Admin | `sb selfhost check-atlassian` checks the token's scopes are read-only, the spaces and projects are reachable and groups resolve; `sb prove` covers W61 | Rovo admin |
| 17 | SB-6204 | W61 | Jira | Everyone | Jira Cloud issues and comments with permission schemes and issue security levels; incident labels and status as for Linear | Glean, Copilot |
| 18 | SB-6205 | W61 | Jira as work events | Steward | `ticket-filed`, incident briefs and incident reviews start on Jira as they do on Linear | Glean triggers |
| 19 | SB-6206 | W61 | Start on a service request | Steward | A Jira Service Management queue a space names starts its agent, which drafts a reply for an agent to send | Glean, Atlassian Rovo |
| 20 | SB-6207 | W61 | Draft a Jira issue | Everyone | The next action drafts a Jira issue the person files with their own token | ChatGPT, Rovo |

## W59 Proven for real (epic E60, SB-6001 to SB-6006)

The goal is that what is built against fakes can be proven on a real system in one run, with evidence a reviewer can trust, and that the parts that lag (Mattermost, big tenants, AI tools acting for people) catch up.

### S59.1 Prove it

**SB-6001 Prove it on the real thing.** `sb prove TICKET` (for SB-5901 to SB-5907, SB-2005, SB-2105, SB-2603, SB-1605 and SB-2601) runs that ticket's acceptance criteria against the real system the cell is configured for, reading only, with throwaway test spaces it creates and removes. It writes `state/proofs/TICKET-DATE.json`, signed with the audit signer (SB-4707): each criterion's pass or fail, counts, timings and the vendor's API version, with every id and name hashed and no document text. `sb prove --verify FILE` checks the signature. A person moves a ticket from In review by attaching its proof to the Progress section; nothing moves a ticket on its own. GCP (SB-903, SB-908) stays out until there is a project.
*Laws:* LAW-07, LAW-11. *Acceptance:* a proof against the recorded fakes passes and verifies. A proof whose criterion fails says which. No text, name or unhashed id is in a proof file.

**SB-6002 Fakes refreshed from a real sandbox.** `sb record VENDOR` runs a fixed script of reads against a real sandbox and writes a recording in the recorded fakes' format, with every string value replaced by one of the same shape and length, every id mapped consistently, people named from `acme-logistics`, `beta-freight` and `gamma-parts`, and `sb secrets scan` run over the result before it is written. A test compares the fake's responses with the recording's shapes, so a fake that drifts from the vendor fails.
*Laws:* LAW-07, LAW-09. *Acceptance:* a recording of the Graph fake round-trips. A real-looking name, mail address or key in a recording fails the scan. A fake missing a field the recording has fails the shape test.

**SB-6003 Check the Microsoft 365 setup.** `sb selfhost check-microsoft` checks the tenant id and app registration, lists the Graph application permissions granted and refuses if any is a write permission (for example `Files.ReadWrite.All` or `Mail.Send`), checks that mail permission is limited by an application access policy to the shared mailboxes the manifest names, checks each named site and drive answers, and lists unmapped Purview labels (SB-5902) by id and count. Each failure names its fix in `docs/runbooks/microsoft.md`.
*Laws:* LAW-03, LAW-16. *Acceptance:* the Graph fake with a write permission granted fails the check and names it. A mail permission without an access policy fails. Every fix points at a real runbook section.

### S59.2 Catch up

**SB-6004 A big tenant's first sync.** Every connector's first sync is paced (sb-cell(5) `sync.max_per_minute`, default 600), honours `Retry-After` and 429s by waiting rather than failing, and writes a checkpoint every 500 items so a restart resumes where it stopped. Health shows each connector's progress ("12,400 items, about 2 hours left"), and an answer from a space whose sources are still in their first sync says so (`notice: still_indexing`).
*Laws:* LAW-05, LAW-13. *Acceptance:* a fake that throttles every tenth call finishes with nothing lost. Killing the cell mid-sync resumes from the checkpoint. The notice shows only during a first sync.

**SB-6005 Reactions and direct questions in Mattermost.** The Mattermost app keeps a websocket open as its bot (`/api/v4/websocket`, a small RFC 6455 client built with the standard library) and reconnects with backoff. A `reaction_added` with `chat.save_reaction` offers save or ask in a DM, as in Slack (SB-5806), and a post in a DM with the bot is a direct question (SB-5805). The outgoing webhook and `/brain` keep working without the websocket.
*Laws:* LAW-01, LAW-09. *Acceptance:* against the recorded Mattermost fake, a reaction sends one DM and saves nothing on its own, and a DM is answered as its sender. A dropped connection reconnects. A reaction in a channel the bot isn't in does nothing.

**SB-6006 The AI tools that reach the brain.** A token made for MCP (SB-3801, SB-5305) names its client (`claude-code`, `cursor` or any short name). My data lists each of the person's tools with its last use, the families it may reach and **Revoke**. `GET /v1/admin/ai/clients` counts tokens, people and calls per client name, never per person, and sb-cell(5) `mcp_clients` may allow only named ones; a token for another client is refused.
*Laws:* LAW-01, LAW-11. *Acceptance:* a revoked tool's next call is refused. The admin's list has counts and no person. With `mcp_clients` set, an unlisted client's token can't be made.

## W60 Every format, every language (epic E61, SB-6101 to SB-6107)

The goal is that the brain reads what's in scans, images and recordings, finds an answer whatever language it's written in, and is usable by everyone, while text a model read is always marked as such.

### S60.1 Every format

**SB-6101 Read scans and images.** A PDF page with no text layer, an image file (PNG, JPEG, WebP) and an image in a .docx or .pptx are read by sb-cell(5) `model.vision`, a gated tier (LAW-06) that obeys `model.region` (LAW-33) and may be the local model profile with a vision model. The text is kept as a chunk marked `machine-read` (LAW-28), cited as "read from a scan" with its page, and counted at the vision tier's price in spend and limits. Off by default; text in an image is data (LAW-09).
*Laws:* LAW-06, LAW-09, LAW-28. *Acceptance:* a scanned fixture page is cited with the mark. An ungated or out-of-region vision tier is never called. With `model.vision` unset, images are skipped and counted on health.

**SB-6102 Recordings become transcripts.** Audio and video files (.mp3, .m4a, .wav, .mp4, .webm) in a source are transcribed by sb-cell(5) `model.transcribe`: either the optional `transcribe` compose profile (a whisper.cpp server, weights pinned in the bundle as for the local model) or a gated provider in region. The result is type `transcript` (SB-3701) with speakers as "Speaker 1, 2", times and the mark, open to whoever can open the file. sb-cell(5) `transcribe.max_minutes` (default 120) caps a file.
*Laws:* LAW-06, LAW-28, LAW-33. *Acceptance:* a fixture recording is cited with its time. A recording over the cap is skipped and counted. Without `model.transcribe`, recordings are skipped.

### S60.2 Every language

**SB-6103 Ask in one language, find another.** sb-cell(5) `languages` names the languages a cell's documents are in. A question is also searched in each of them, translated by the gated small tier with the space's glossary (SB-6104), and the results are fused (SB-2706). Cites stay in their own language and the answer is in the UI's (SB-3003). The trace keeps the translated queries, so the person can see them in the context view (SB-5501).
*Laws:* LAW-06, LAW-14. *Acceptance:* a German question finds the English fixture policy and answers in German. With one language named, nothing is translated. The context view shows the translated query.

**SB-6104 A glossary in every language.** A space glossary term (SB-2203) may carry its words in each named language (`in: {de: Lieferschein, fr: bon de livraison}`). SB-6103 uses them when translating, and marked terms (SB-3302) are found in every language. A proposal can add a translation like any term change (SB-3107).
*Laws:* LAW-16. *Acceptance:* a translated question keeps the glossary's word. A term is marked in a French answer. A translation goes through a proposal.

**SB-6105 Translate a cited passage.** The passage view (SB-3001) has **Translate** when the passage's language isn't the UI's. The gated small tier translates the passage, at most 600 characters, shown beside the original and marked; it is kept nowhere and checked as the reader like the passage itself (LAW-26).
*Laws:* LAW-26, LAW-28. *Acceptance:* a French passage shows in English with the mark. A reader who can't open the passage can't translate it. Nothing is stored.

**SB-6106 Spanish, Italian and Dutch.** The web UI (`web/src/i18n`), chat replies, notices and answers (SB-3003) in Spanish, Italian and Dutch, with the same completeness test as German and French.
*Laws:* none new. *Acceptance:* every string has all six languages. An answer asked in Dutch is in Dutch.

**SB-6107 Usable by everyone.** WCAG 2.2 AA for the web UI: `make web` runs an automated accessibility check over every tab with Playwright and fails on any serious finding; every flow (ask, cite, review, inbox) has a keyboard-only test; cites, labels and notices have screen reader names, and colour is never the only sign of a label or a stale cite.
*Laws:* none new. *Acceptance:* the check passes over every tab. Each named flow passes keyboard-only. A label reads aloud as its level.

## W61 Reach Confluence and Jira (epic E62, SB-6201 to SB-6207)

The goal is that a team on Confluence and Jira gets the same brain as one on Google Drive and Linear, read-only, with Atlassian's own permissions behind every check ([ADR-0024](adr/ADR-0024-atlassian.md) proposed).

### S61.1 Confluence and groups

**SB-6201 Confluence.** A connector for Confluence Cloud over its REST API: spaces chosen in sb-cell(5), pages, blog posts and attachments read from a stored cursor, page restrictions and space permissions resolved to Atlassian groups (SB-6202), labels as tags, the page tree as sections (SB-3002), and text through the existing readers. A space open to anonymous users counts as wide (SB-5401). Built against a recorded fake with the standard library; a real site run stays In review until `sb prove`.
*Laws:* LAW-01, LAW-13. *Acceptance:* the fake's space syncs with its restrictions, a restricted page is cited only to its group, and a deleted page is gone at the next sync.

**SB-6202 Atlassian groups and people.** A directory adapter for Atlassian groups and their members behind every access check and audience (SB-5404), and SCIM from Atlassian Guard into the existing endpoint (SB-107) so a leaver is revoked within 5 minutes (LAW-12). A group the API won't expand is counted, never guessed.
*Laws:* LAW-01, LAW-03, LAW-12. *Acceptance:* membership grants exactly what the fake directory says. A deprovisioned person loses access within 5 minutes on the fake clock.

**SB-6203 Check the Atlassian setup.** `sb selfhost check-atlassian` checks the API token's scopes are read-only (and refuses write scopes), that each named Confluence space and Jira project answers, and that groups resolve; each fix is in `docs/runbooks/atlassian.md`. `sb prove` covers SB-6201, SB-6202 and SB-6204 to SB-6207.
*Laws:* LAW-03, LAW-16. *Acceptance:* a token with a write scope fails and names it. An unreachable project fails. Every fix points at a real runbook section.

### S61.2 Jira

**SB-6204 Jira.** A connector for Jira Cloud: issues, comments and attachments of the projects named, with permission schemes and issue security levels resolved to Atlassian groups and roles, an issue's labels and status read as Linear's are (SB-4204, SB-4604), and links back to the issue.
*Laws:* LAW-01, LAW-13. *Acceptance:* an issue under a security level is cited only to people in it. A closed issue's status shows. A deleted issue is gone at the next sync.

**SB-6205 Jira as work events.** `on: [ticket-filed]` (SB-5301), incident briefs (SB-4204) and incident reviews (SB-4604) start on Jira issues as on Linear ones, from the next sync or a Jira webhook verified with its secret.
*Laws:* LAW-09, LAW-16. *Acceptance:* a filed fake issue starts one draft. An incident label starts a brief and closing it starts a review. An unsigned webhook is refused.

**SB-6206 Start on a service request.** A space may name one Jira Service Management queue (sb-space(5) `service_queue`) and `on: [request-received]`. A new request starts the space's agent, which drafts a reply note for a service agent to edit and send themselves (SB-5304); the request is data (LAW-09), and a queue not named is never read.
*Laws:* LAW-09, LAW-16. *Acceptance:* a fake request puts one draft in the steward queue and sends nothing. A queue not in the manifest is never read.

**SB-6207 Draft a Jira issue.** The next action (SB-5304) can draft a Jira issue (project, type, summary, description with cites) that the person files with their own Atlassian token; the brain never files it. A draft that cites what the project's readers can't open says so before it is filed.
*Laws:* LAW-16, LAW-32. *Acceptance:* a draft is filed only with the person's token. A confidential cite warns before filing. Nothing is filed without the person.

## What we leave out

- **Notion.** Its API doesn't say who can open a page, and unknown access can't be allow (LAW-03). It comes back if Notion's API does.
- **Standing identities for agents** (Copilot Autopilot, Glean service credentials). Agents keep reading as their approvers, narrowed, and borrow access for one run (LAW-31).
- **Writing back to Confluence or Jira**, other than an issue a person files with their own token. Approved notes keep being published where they are now.
- **Projects that group everything** (Copilot Home). Collections (SB-4304) and conversations (SB-5802) already cover it; we don't add a new page.
- **Real GCP** (SB-903, SB-908). It still waits for a project.

## Decisions for the user

Decided by the user 2026-11-27: every recommended option.

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W59, W60, W61. *Other:* W61 first, if a team on Confluence and Jira is waiting; or W59 and W61 only, leaving W60's reading models for later.
2. **Proofs (SB-6001, SB-6002).** *Recommended:* proofs keep pass or fail, counts, timings and hashes, signed, and a person moves a ticket from In review by attaching one; recordings replace every value. *Other:* keep proofs as a runbook checklist a person fills in.
3. **Mattermost (SB-6005).** *Recommended:* a websocket on the bot account, with a small client built with the standard library. *Other:* a Mattermost server plugin (it needs Mattermost's Go module, against rule 1); or leave Mattermost as it is.
4. **AI tools (SB-6006).** *Recommended:* every MCP token names its client, people see and revoke their own, admins see counts per client and may allow only named ones. *Other:* admins see each person's tools.
5. **Reading scans, images and recordings (SB-6101, SB-6102).** *Recommended:* gated `model.vision` and `model.transcribe`, off by default, region kept, text marked as machine-read; whisper.cpp as an optional profile beside llama.cpp. *Other:* leave out recordings; or leave out both.
6. **Languages (SB-6103, SB-6106).** *Recommended:* questions translated by the gated small tier into the languages `languages` names; Spanish, Italian and Dutch added to the UI. *Other:* a multilingual embedder instead of translating questions; other UI languages.
7. **Atlassian (W61, ADR-0024).** *Recommended:* Confluence Cloud, Jira Cloud and Jira Service Management, read-only, built against recorded fakes with real-site runs In review until `sb prove`; Notion left out. *Other:* Confluence only; or Notion with an audience an admin names.

## Still open from earlier waves

- **SB-5901** needs a real Microsoft 365 tenant; SB-6001 and SB-6003 make that run short.
- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Glean release notes](https://docs.glean.com/release-notes/), [19 November 2026](https://docs.glean.com/release-notes/releases/2026-11-19-november-release), [6 November 2026](https://docs.glean.com/release-notes/releases/2026-11-06-november-release), [3 November 2026](https://docs.glean.com/release-notes/releases/2026-11-03-november-release)
- [Microsoft Ignite 2026 recap (Witivio)](https://www.witivio.com/en/blog/microsoft-ignite-2026-announcements/), [Introducing Dynamics 365 in Copilot](https://www.microsoft.com/en-us/dynamics-365/blog/business-leader/2026/11/17/microsoft-ignite-2026-introducing-dynamics-365-in-copilot/)
- [Notion releases](https://www.notion.com/releases), [Agents, rebuilt to do more for less](https://www.notion.com/blog/notion-ai-more-for-less)
- [Gemini Enterprise release notes](https://docs.cloud.google.com/gemini/enterprise/docs/release-notes?hl=en)
- [Glean's newest partner scans up to three million images a month](https://theinference.org/article/glean-s-newest-partner-scans-up-to-three-million-images-a-month)

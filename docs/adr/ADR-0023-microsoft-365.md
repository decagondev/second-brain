---
title: "ADR-0023: Microsoft 365 is reached through Graph adapters built with the standard library, read-only"
summary: "SharePoint, OneDrive, Purview labels, Entra, Teams, Outlook calendars, Teams transcripts and one shared mailbox per space, each an adapter over Microsoft Graph with no new Go module, read-only, built against a recorded fake."
---

# ADR-0023: Microsoft 365 is reached through Graph adapters built with the standard library, read-only

| Status | Date | Deciders |
|---|---|---|
| Proposed (plan, W58) | 2026-11-27 | Platform lead (the user) |

## Context

The cell reaches Google Workspace (Drive, Calendar, Meet transcripts), Slack, Mattermost, Linear, HubSpot, GitHub and file shares. Most companies of the size we serve run on Microsoft 365 instead, and the W56 to W58 review (2026-11-27, [23-place-carry-reach.md](../23-place-carry-reach.md)) found that Glean, Copilot and ChatGPT Enterprise all reach SharePoint, Teams and Outlook. Microsoft offers Go SDKs for Graph, but they pull in a large tree of modules.

## Decision

- **One Graph client in the adapters, standard library only.** OAuth client credentials against Entra, JSON over HTTPS, delta queries and paging. No Microsoft SDK and no new Go module (ADR-0017's allow-list doesn't change). Its secret is at `vault://tenants/TENANT/msgraph`.
- **Read-only, as every connector.** The application permissions asked for are read scopes only (`Sites.Read.All` or `Sites.Selected`, `Files.Read.All`, `Group.Read.All`, `User.Read.All`, `Calendars.Read`, `OnlineMeetingTranscript.Read.All`, `Mail.Read` restricted to the named shared mailboxes by an application access policy). The cell never writes to Microsoft 365; Teams is a place to ask and a reply is the bot's own message.
- **Permissions and labels at sync.** Each item's live permissions, sharing links included, become its ACL copy (LAW-01). Purview label ids map to the four levels of SB-5405; an unmapped id counts as `restricted` (LAW-03).
- **Identity through Entra.** Groups with transitive membership for checks, managers for the tree, and Entra's SCIM provisioning into the existing endpoint for LAW-12.
- **Mail only from a shared mailbox a space names**, only to start that space's agent; nobody's own mailbox is read.
- **Recorded fakes first.** Every adapter is built and tested against a recorded Graph fake; a run against a real tenant stays In review until the user provides one.

## Alternatives rejected

- **Microsoft's Graph SDK for Go.** Faster to write, but it adds dozens of modules to vet and vendor for a few dozen endpoints.
- **Indexing everyone's mail.** It reaches far more personal data than any answer needs and is the hardest thing to hold to LAW-01.
- **Confluence and Jira first.** They matter to fewer of the companies we serve; they can follow the same pattern later.

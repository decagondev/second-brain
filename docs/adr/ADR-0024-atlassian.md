---
title: "ADR-0024: Confluence and Jira are reached through adapters built with the standard library, read-only, and Notion stays out"
summary: "Confluence Cloud, Jira Cloud, Jira Service Management and Atlassian groups, each an adapter over Atlassian's REST APIs with no new Go module, read-only except an issue a person files with their own token, built against a recorded fake. Notion stays out while its API doesn't say who can open a page."
---

# ADR-0024: Confluence and Jira are reached through adapters built with the standard library, read-only, and Notion stays out

| Status | Date | Deciders |
|---|---|---|
| Proposed (plan, W61) | 2026-11-27 | Platform lead (the user) |

## Context

The cell reaches Google Workspace, Microsoft 365 (ADR-0023), Slack, Mattermost, Teams, Linear, HubSpot, GitHub and file shares. W58 left Confluence, Jira and Notion "later if asked". Many engineering and product teams keep their knowledge in Confluence and their work in Jira, and Glean, Copilot and Gemini Enterprise all reach them. The W59 to W61 review (2026-11-27, [24-prove-read-reach.md](../24-prove-read-reach.md)) also looked at Notion, whose public API returns the pages shared with an integration but not who else can open them.

## Decision

- **One Atlassian client in the adapters, standard library only.** OAuth or an API token per site, JSON over HTTPS, cursor paging. No Atlassian SDK and no new Go module (ADR-0017's allow-list doesn't change). Its secret is at `vault://tenants/TENANT/atlassian`.
- **Read-only.** Read scopes only; `sb selfhost check-atlassian` refuses a token with a write scope. The one write is a Jira issue a person files with their own token from a draft (SB-6207), as Linear tickets are today (SB-5304).
- **Permissions at sync.** Confluence page restrictions and space permissions, and Jira permission schemes and issue security levels, become each item's ACL copy through Atlassian groups (LAW-01). A group the API won't expand is counted and grants nothing (LAW-03). A space open to anonymous users counts as wide (SB-5401).
- **Identity through Atlassian groups**, and SCIM from Atlassian Guard into the existing endpoint for LAW-12.
- **Service requests only from a queue a space names**, only to start that space's agent, which drafts a reply for a person to send.
- **Recorded fakes first.** Every adapter is built and tested against a recorded fake; a real-site run stays In review until `sb prove` (SB-6001) passes on one.

## Alternatives rejected

- **Notion now, with an audience an admin names.** The admin would be stating who can open each page instead of the source; unknown access can't be allow. It comes back if Notion's API exposes page permissions.
- **Atlassian's SDKs or a community Go client.** They add modules to vet and vendor for a small set of endpoints.
- **Writing approved notes back into Confluence.** It makes the brain a writer to the source; approved notes keep being published where they are now.

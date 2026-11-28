---
title: Confluence and Jira setup
summary: What sb selfhost check-atlassian checks, and how to fix each failure (SB-6203).
---

# Confluence and Jira setup

`sb selfhost check-atlassian --config FILE` checks the cell's Atlassian setup before the first sync and after any change to the account ([sb(1)](../../manual/sb.1.md), [sb-selfhost(7)](../../manual/sb-selfhost.7.md)). It reads only, prints no secret, and names a section below for each failure. The cell reaches Confluence and Jira read-only ([ADR-0024](../adr/ADR-0024-atlassian.md)).

## api-token

The credentials at `vault://tenants/TENANT/atlassian` sign in to nothing, or the account's permissions can't be read.

1. Sign in to the site as the cell's account and make an API token under **Account settings**, **Security**, **API tokens**.
2. Put `{"email": "…", "api_token": "…", "webhook_secret": "…"}` in the vault under `atlassian`, owner-only.
3. Check `atlassian.base` in sb-cell(5) is the site, such as `https://acme.atlassian.net`.

## read-only-account

An Atlassian API token carries its account's permissions, so the cell's account itself must only read. The check names each Jira write permission the account holds, site-wide or in a named project.

1. Use an account of its own for the cell, never a person's.
2. Give it **Browse projects** in each Jira project and **View** in each Confluence space, through a group of its own, and nothing else.
3. Take it out of every group that grants create, edit, comment, transition, assign or administer, and run the check again.

## spaces-and-projects

A Confluence space in `atlassian.spaces` or a Jira project in `atlassian.projects` doesn't answer.

1. Check each key as Confluence and Jira show it (`FIN`, `OPS`), in upper case.
2. Check the cell's account can view the space and browse the project.

## groups

The account's groups don't resolve, so the live check can't place people in the groups that restrict pages and issues.

1. Check the account can see users and groups (**Browse users and groups** in Jira's global permissions).
2. Check people's email addresses are visible to the account (the organisation's profile visibility settings), since the cell finds people by email.

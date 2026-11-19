---
title: "sb-tour(7)"
man: "7"
summary: Fifteen minutes from manifest to cited answer to approved action.
---

# sb-tour(7)

## NAME

sb-tour — a guided walk through Second Brain

## SYNOPSIS

```sh
sb tenant check acme.yaml && sb tenant apply acme.yaml && sb tenant verify acme.yaml
```

## DESCRIPTION

This tour follows one company, `acme-logistics`, from an empty manifest to its first approved action. Each step links to the page that specifies it.

### 1. Write the manifest

One file describes the whole tenant ([sb-tenant(5)](sb-tenant.5.md)):

```yaml
tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack }
agents: [ask]
pilot_group: group:sb-pilot
```

Six lines and an optional seventh. There are no secrets: connector credentials go into the vault at `vault://tenants/acme-logistics/connectors/docs` and `…/chat`, once. There's no version: `channel: stable` means "the current stable release, always".

### 2. Check it

```text
$ sb tenant check acme.yaml
ok
```

Mistakes come back like compiler errors, with file, line, column and a fix ([sb-tenant(1)](sb-tenant.1.md)).

### 3. Apply it

```text
$ sb tenant apply acme.yaml
+ cell acme-logistics (region eu)     database, index, vault ns, kms key, queue
+ connector docs (sharepoint, read-only)
+ connector chat (slack, read-only)
+ agent ask
applied in 22m14s
```

The cell is yours alone: its own database, index, keys and region ([sb-tenancy(7)](sb-tenancy.7.md)). Sign-in uses your IdP; users and groups arrive over SCIM ([sb-identity(7)](sb-identity.7.md)).

### 4. Wait for the first sync

Connectors pull documents and messages, read-only. Every chunk gets a provenance tag and suspicious, instruction-like text is flagged as data ([sb-ingest(7)](sb-ingest.7.md)).

### 5. Ask

A pilot user asks in chat: *"What's our refund approval limit?"*

```text
Refunds over £500 need a team lead's approval. [1]
This changed in March. (uncited)

[1] Refund policy · docs:4f2a9c11@17 · "Approvals"
```

Behind that answer:

- retrieval searched **only** this tenant's index;
- every candidate document was checked **live** with the source: "can this user open this, right now?" ([sb-permissions(7)](sb-permissions.7.md));
- anything uncertain was dropped. Unknown means no;
- the model proposed citations, and code kept only those pointing at allowed chunks;
- the second sentence had no support, so code labelled it `uncited` ([sb-citations(7)](sb-citations.7.md)).

### 6. Verify the laws on your own data

```text
$ sb tenant verify acme.yaml
LAW-01  pass   LAW-02  pass   LAW-03  pass   LAW-14  pass   evals  pass
```

These are the platform's guarantees ([sb-laws(7)](sb-laws.7.md)), checked on your real users and documents without printing any content.

### 7. Draft

Add `draft-reply` to `agents`, apply, and staff can ask for a cited draft reply to a ticket or thread. The platform never sends it. A person does ([sb-agents(7)](sb-agents.7.md)).

### 8. Approve an action

From day 46, add one gated workflow:

```yaml
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
```

Now an agent can propose a ticket update. An IT lead sees the exact payload and target, then approves **that payload's hash**. The executor runs it once, and it's `Ran` only when the ticket system returns a receipt ([sb-approvals(7)](sb-approvals.7.md)). Every step was written to the audit log first ([sb-audit(7)](sb-audit.7.md)).

### 9. Stop it

Something looks wrong:

```sh
sb kill agent draft-reply --tenant acme-logistics --reason "drafts quoting old policy"
```

Within 10 seconds, nothing new starts ([sb-kill-switches(7)](sb-kill-switches.7.md)).

### 10. Stay current

You never upgrade. New releases pass the eval gate, roll through rings and reach `stable`; no tenant is ever more than one release behind ([sb-release(7)](sb-release.7.md)).

## SEE ALSO

[sb-onboarding(7)](sb-onboarding.7.md) for the full day 0 to 90 runbook. [Architecture overview](../docs/architecture/00-overview.md) for how it works.

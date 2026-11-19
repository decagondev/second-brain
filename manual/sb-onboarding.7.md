---
title: "sb-onboarding(7)"
man: "7"
summary: The day 0 to 90 runbook a company's own IT team follows, unaided.
---

# sb-onboarding(7)

## NAME

sb-onboarding — go live with Second Brain in four phases, without calling the platform team

## SYNOPSIS

```text
Day 0      manifest → check → apply
Days 1–14  connect read-only → pilot (≈20 people) → verify
Days 15–45 everyone → draft-reply → weekly report
Days 46–90 one approval-gated workflow, once its evals pass
```

## DESCRIPTION

Each phase ends with a **gate**. Don't move on until the gate passes. Every step is a command or a console action with an expected result; if what you see differs, that's a bug in this page, so report it.

### Before day 0: prerequisites

- [ ] An OIDC app registered in your IdP, with redirect URI `https://<tenant>.<platform-domain>/auth/callback`
- [ ] A SCIM provisioning app pointed at `https://<tenant>.<platform-domain>/scim/v2`, with a group `sb-pilot`
- [ ] Read-only service credentials for each source, stored at `vault://tenants/<tenant>/connectors/<kind>` (console: **Connectors → Store credential**)
- [ ] A git repository for your manifest and tenant policies
- [ ] Decide what's never ingested (HR, legal, board channels) and list it under `exclude`

### Day 0: create the cell

1. Write `tenant.yaml` from [the pilot example](sb-tenant.5.md#examples).
2. `sb tenant check tenant.yaml` → expect `ok`.
3. `sb tenant apply tenant.yaml --plan` → review: one cell, your connectors, `ask`.
4. `sb tenant apply tenant.yaml` → expect `applied` in under 30 minutes.
5. Sign in as a pilot user → expect the empty-state home page with your company name.

**Gate 0:** a pilot user can sign in; a non-pilot user is refused.

### Days 1–14: pilot

1. Watch first sync in the console (**Sources**). Expect the backlog to drain and freshness under 15 minutes.
2. Add 20 or so pilot users to `sb-pilot` in your IdP. They appear within 5 minutes.
3. Pilot users ask real questions on the web and in chat.
4. Check **Refusals** daily. An `unknown` rate above 1% on a connector means a connector problem; send the console's diagnostic bundle (ids only) to the platform team. This is the one sanctioned contact.
5. Day 10 or later: `sb tenant verify tenant.yaml`.

**Gate 1:** `verify` passes every law; the pilot's thumbs-up rate is at least 70%.

### Days 15–45: everyone, plus drafts

1. Remove `pilot_group`; add `draft-reply` to `agents`; `check`, then `apply`.
2. Announce to staff, with the one-page user guide from the console.
3. The weekly report starts automatically for the approvers group.
4. Day 45: review adoption, refusal rate and draft approve-to-reject ratio with leadership.

**Gate 2:** weekly active users ÷ seats is at least 20%; zero law alerts.

### Days 46–90: one gated workflow

1. Choose **one** workflow (e.g. `ticket-update`) and its approver group.
2. Add your golden examples to `tenants/<tenant>/evals/` ([sb-eval(5)](sb-eval.5.md)).
3. `sb eval run tenant/<tenant>` → expect pass.
4. Add `workflows` and `approval` to the manifest; `check`, then `apply`. Apply refuses if the workflow's evals haven't passed.
5. Train approvers: they approve the **exact payload shown**, and they can't approve their own requests.
6. Rehearse `sb kill agent <workflow>` with your on-call.

**Gate 3:** first 20 approved actions all show `Ran` with receipts; kill rehearsal succeeds.

## LAWS

Every gate includes [`sb tenant verify`](sb-tenant.1.md), which checks [LAW-01](sb-laws.7.md#law-01), [LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03) and [LAW-14](sb-laws.7.md#law-14).

## SEE ALSO

[sb-tour(7)](sb-tour.7.md), [sb-tenant(5)](sb-tenant.5.md), [sb-kill(1)](sb-kill.1.md), [sb-observability(7)](sb-observability.7.md)

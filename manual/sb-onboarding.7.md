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

1. Choose your spaces: pick the departments that need their own brain from [sb-departments(7)](sb-departments.7.md), and for each one the IdP groups that are its members and the folders and channels it covers. → expect a short list, usually 3 to 6 spaces, with People, Finance, Legal and Executive marked `restricted`. Until SB-1006 the list waits in the manifest: `sb tenant check` and `apply --plan` show it, but the pilot runs as one brain until spaces are enforced.
2. Write `tenant.yaml` from [the pilot example](sb-tenant.5.md#examples). → expect six required lines and no secrets in the file.
3. `sb tenant check tenant.yaml` → expect `ok`.
4. `sb tenant apply tenant.yaml --plan` → expect a plan with one cell, your connectors and `ask`, and nothing else.
5. `sb tenant apply tenant.yaml` → expect `applied` in under 30 minutes.
6. Sign in as a pilot user → expect the empty-state home page with your company name.

**Gate 0:** a pilot user can sign in; a non-pilot user is refused.

### Days 1–14: pilot

1. Watch first sync in the console (**Sources**). Expect the backlog to drain and freshness under 15 minutes.
2. Add 20 or so pilot users to `sb-pilot` in your IdP. → expect them under **Users** in the console within 5 minutes.
3. Pilot users ask real questions on the web and with `/ask` in chat. → expect answers with citation chips, and `Uncited` labels where nothing supports a claim.
4. Check **Refusals** daily. → expect an `unknown` rate under 1% per connector. Above that, there's a connector problem: send the console's diagnostic bundle (ids only) to the platform team. This is the one sanctioned contact.
5. Day 10 or later: `sb tenant verify tenant.yaml`. → expect `verify: 5 passed, 0 failed`.

**Gate 1:** `verify` passes every law; the pilot's thumbs-up rate is at least 70%.

- [ ] First sync drained; freshness under 15 minutes
- [ ] Pilot group visible in the console
- [ ] `unknown` refusals under 1% on every connector
- [ ] `sb tenant verify` passes
- [ ] `python3 -m evals.pilot report` passes (thumbs-up at least 70%)
- [ ] Retro held and logged

#### Running the pilot review

The pilot needs about 20 people who will ask real work questions for two weeks (SB-311). Pick them from at least three teams, so that the documents and permissions they hit differ. Then:

1. **Every Monday,** take the feedback export (sb-ask(3), `POST /v1/feedback`) and run `python3 -m evals.pilot report EXPORT.jsonl`. It prints the Gate 1 numbers: feedback count, thumbs-up rate, each wrong-citation report by cite, the share of answers with uncited claims, and the share of answers with refused sources. It exits 1 if the thumbs-up rate is under 70% or if there is no feedback at all.
2. **Each wrong-citation report** is triaged within a week. Open the trace with auditor rights. If the cited chunk doesn't support the claim, add a case to `tenants/<tenant>/evals/` (sb-eval(5)), so the finding becomes a permanent test.
3. **At the end of week two,** hold a retro with the [pilot retro template](../docs/pilot-retro-template.md). Log the findings in the devlog, add new risks to the risk register, and turn each fix into a ticket.

The report holds ids, counts and comments only. It never shows a question or an answer.

### Days 15–45: everyone, plus drafts

1. Remove `pilot_group`; add `draft-reply` to `agents`; `sb tenant check tenant.yaml` → expect `ok`; then `sb tenant apply tenant.yaml` → expect `- setting pilot_group` and `+ agent draft-reply`.
2. Announce to staff, with the one-page user guide from the console (**Help → User guide**). → expect weekly active users to climb in **Usage** within a week.
3. The weekly report starts automatically for the approvers group. → expect it on Monday at 07:00 UTC; the same numbers are at `GET /v1/admin/usage?week=…` (sb-admin(3)).
4. Day 45: review adoption, refusal rate and the drafts' used-to-discarded ratio with leadership, from the weekly report. → expect a decision recorded: go on to a workflow, or not yet.

**Gate 2:** weekly active users ÷ seats is at least 20%; zero law alerts.

- [ ] `pilot_group` removed and `draft-reply` applied
- [ ] Staff announcement sent
- [ ] Weekly report arriving
- [ ] Active users per seat at least 20% (weekly report)
- [ ] No Sev-1 law alerts since day 15

### Days 46–90: one gated workflow

1. Choose **one** workflow (`ticket-update` in v1) and its approver group. → expect the group to exist in your IdP and to appear under **Users → Groups**.
2. Add your golden examples to `tenants/<tenant>/evals/` ([sb-eval(5)](sb-eval.5.md)). → expect `python3 -m evals run --tenant-dir tenants/<tenant> --model extractive` to run them on the local cell.
3. `python3 -m evals run --tenant-dir tenants/<tenant> --model <the release model> --record DIR/evals/<tenant>/ticket-update.json` (sb-eval(1)) → expect every case to pass and the record to be written.
4. Add `workflows` and `approval` to the manifest; `sb tenant check` → expect `ok`; then `sb tenant apply` → expect `+ workflow ticket-update`. Apply refuses with `EEVAL` if the workflow's evals haven't passed in the last 30 days.
5. Train approvers: they approve the **exact payload shown**, and they can't approve their own requests. → expect a practice action approved in the approver view (`/?action=…`), and a self-approval attempt refused with `ENOAPPROVAL`.
6. Rehearse `sb kill agent ticket-update --tenant <tenant> --reason drill` with your on-call. → expect `killed`, new proposals refused with `EKILLED` within 10 seconds, then `sb unkill` clears it.

**Gate 3:** first 20 approved actions all show `Ran` with receipts; kill rehearsal succeeds.

- [ ] Tenant golden questions pass, and the eval record is written
- [ ] Workflow applied
- [ ] Approvers trained (practice approval and refused self-approval)
- [ ] Kill rehearsal done
- [ ] First 20 approved actions show `Ran` with receipts

## LAWS

Every gate includes [`sb tenant verify`](sb-tenant.1.md), which checks [LAW-01](sb-laws.7.md#law-01), [LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03) and [LAW-14](sb-laws.7.md#law-14).

## SEE ALSO

[sb-tour(7)](sb-tour.7.md), [sb-tenant(5)](sb-tenant.5.md), [sb-kill(1)](sb-kill.1.md), [sb-observability(7)](sb-observability.7.md)

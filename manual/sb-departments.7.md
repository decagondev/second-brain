---
title: "sb-departments(7)"
man: "7"
summary: The department catalogue - twelve space templates plus custom, and the four access practices they follow (W8 draft).
---

# sb-departments(7)

## NAME

sb-departments — the department templates a company starts its spaces from, and the access practices behind them

## SYNOPSIS

```text
engineering · product · design · sales · marketing · support · people
finance · legal · it-security · operations · executive · custom
```

## DESCRIPTION

A **template** is data: a label default, default sources, default agents, retention and an eval suite. A space names one ([sb-space(5)](sb-space.5.md)), and every field it sets itself wins. Companies rename, merge or drop templates, and add their own spaces from `custom`. Every template field is an ordinary manifest field, for everyone, never per-company code (P3). The catalogue is data: `internal/core/manifest/catalogue.json`, which `sb tenant check` reads and every release bundle pins by hash (sb-release(7), Bundles; SB-1101). A test holds the file and the table below to each other, so neither can change alone. `sb space init` starts a space from a template (sb-space(1)), and `sb space templates` prints the catalogue.

### The catalogue

| Template | Label default | Default sources | Default agents | Retention | Who else usually reads it |
|---|---|---|---|---|---|
| `engineering` | internal | code, tickets, docs, chat | `ask`, `incident-summary` | tenant's | Product, Design, Support (read), IT/Security |
| `product` | internal | docs, tickets | `ask`, `spec-digest` | tenant's | Engineering, Design, Sales and Marketing leads |
| `design` | internal | docs | `ask`, `design-rationale` | tenant's | Product, Engineering, Marketing |
| `sales` | confidential | crm, docs | `ask`, `draft-reply`, `account-brief` | tenant's | Finance, Support, Executive |
| `marketing` | internal | docs, chat | `ask`, `draft-reply`, `brand-check` | tenant's | Sales, Product |
| `support` | internal | tickets, docs | `ask`, `draft-reply`, `case-brief` | tenant's | Engineering, Product, Sales |
| `people` | restricted | hris, docs | `ask`, `policy-answer` | 365 days | Managers for their own reports only; Legal for cases |
| `finance` | restricted | accounting, docs | `ask`, `close-checklist` | tenant's | Executive, budget owners for their own lines |
| `legal` | restricted | docs | `ask`, `clause-finder` | tenant's; a legal hold overrides it | Executive; deal teams by matter |
| `it-security` | confidential | docs, tickets | `ask`, `runbook-answer` | tenant's | Engineering |
| `operations` | internal | docs | `ask`, `sop-steps` | tenant's | Finance, Sales |
| `executive` | restricted | docs | `ask`, `exec-brief` | tenant's | Board, named deal teams |
| `custom` | internal | any | `ask` | tenant's | whoever its members are |

Some content needs more than the template's default:

- **Engineering:** secrets and security findings are `restricted` wherever they appear.
- **Sales:** deal terms and forecasts are `confidential`.
- **Support:** customer personal data is `confidential`.
- **People:** compensation, performance reviews and case files are **never ingested by default**. Policies and the handbook are `internal`, so everyone can ask about them.
- **Finance:** every workflow needs a different approver than its requester (segregation of duties). Unreleased results and payroll are `restricted`.
- **Legal:** privileged matters are `restricted`; approved policies are `internal`.
- **Executive:** board packs and M&A work sit behind barriers, not only labels.

### The four practices

The templates follow four practices that almost every company with a mature access model uses. Each one is built into the platform.

1. **Roles from the directory, refined by attributes.** Most access comes from group membership: you're in the Finance group, so you can use the Finance brain. Attributes from HR refine it: department, level, location, cost centre and manager, read from SCIM's enterprise extension the same way the cell already reads groups. For example, a manager may see their own reports' People records, and nobody else's.
2. **Classification.** Every document has one of four labels, `public`, `internal`, `confidential` or `restricted`, with a default per source and per department. A label set at the source wins.
3. **Need to know, and barriers.** HR cases, legal matters, unreleased results and M&A work are visible only to a named group. Some groups must not see each other's work at all, which is an information barrier (an ethical wall). Labels and barriers are separate controls: a label says how sensitive something is, and a barrier says who must never meet.
4. **Joiner, mover, leaver, and access reviews.** Access changes the moment someone joins, changes team or leaves, within 5 minutes (LAW-21). Each quarter, every space's stewards re-certify who has access, a standard SOX and SOC 2 control (W12).

### Account brief (SB-4605)

In a `sales` space that runs `account-brief`, or a `support` space that runs `case-brief`, **Account brief** on a CRM account the person can open (sb-ask(3), Account brief) gathers, by code and checked as them: the account's own record, its open tickets, the notes and documents of the last 90 days that name it, and the contracts and proposals that name it, with its renewal date when the CRM record has a `Renewal:` line and the owners among the cited documents. The space's agent writes one cited section of each, as the person. With `prep: true`, a meeting whose attendees share the account's domain gets the brief 30 minutes before. `personal_data` fields stay masked as ingest masked them (sb-ingest(7)), and nothing is written to the CRM.

### Rhythms (SB-4801)

A **rhythm** is a named playbook for a recurring moment: a few dated steps started for a person, a collection or the space, each one a part already built (a draft from an agent, a routine, a goal check-in or a request) offered on its day to one named approver. Five come built in: `onboarding` (SB-4802, sb-onboarding(7)), `kickoff` and `close-out` (SB-4803, SB-4804, sb-spaces(7), Projects), `quarterly-planning` (SB-4805, Quarterly planning below) and `one-on-one` (SB-4807, sb-agents(7), 1:1 prep); a space adds its own, or replaces a built-in one, under `rhythms` in its manifest (sb-space(5)), proposed by a steward and linted like skills: at most 12 steps over at most 120 days, each with an approver.

A steward starts one with `POST /v1/spaces/NAME/rhythms` (`{ rhythm, subject }`, the subject a person's email or a collection id) or from the space's home. A rhythm with a `plan` waits as **proposed** until its plan's approver (the subject's manager from SCIM, or with none the space's stewards) presses **Approve** (`POST /v1/rhythms/ID/approve`); its days count from then. Each step whose day has come lands, as its approver's own draft or prompt, in their queue and their inbox (sb-ask(3), One inbox), drafted as them from what they can open now; a step for the stewards lands in the space's queue. A step is done once its approver answers or discards it. Nothing a step holds is published, sent or created without its approver doing it (LAW-29).

`GET /v1/spaces/NAME/rhythms` lists the space's rhythms with each step's day, approver, due date and state: a steward sees every one, anyone else the ones they started, are the subject of or approve a step of. Each read checks the subject first: a person who has left the space, or a project that has closed, stops the rhythm at once, its waiting steps skipped. Its starter, its subject or a steward stops it with `POST /v1/rhythms/ID/stop`. Every start, approval, step, stop and finish is in the audit log, by id. The jobs service runs the rhythms with the notify job: it starts onboarding for people SCIM added since its last run and offers the steps whose day has come.

### Quarterly planning (SB-4805)

Fourteen days before a quarter ends, the jobs service starts each space's `quarterly-planning` rhythm once, unless the space's manifest replaces or drops it. Its day-0 step is a `goal-plan` draft in the space's steward queue, drafted as the space's first active steward: next quarter's proposed goals, one line for each of this quarter's goals of the space that isn't done (at most 10), marked **carry over**, or **at risk** when its last check-in said at risk or off track, each cited to the goal's document, then the space's requests this quarter counted by category. It is put together by code from the goals and queue the steward can open, not written by a model. The steward edits it and publishes each goal as usual (sb-spaces(7), Goals); a proposed goal starts nothing (LAW-16). A space with no goal to carry over gets no draft.

`GET /v1/spaces/NAME/planning`, for the space's stewards, rolls up the latest planning draft of the space and each of its teams: `{ "space", "quarter", "teams": [{ "space", "proposed": [{ "title", "cite", "why" }], "hidden" }], "overlaps": [{ "teams", "goals", "term" }] }`. A proposed goal is listed only when the steward can open its cite; the rest are counted in `hidden` (LAW-01). Two teams' proposed goals are flagged as a possible overlap when their titles share a glossary term of the space, or two words of four letters or more. An overlap is a flag for the stewards to talk about, never a merge.

## LAWS

LAW-29 A rhythm only drafts, and stops with its subject ([sb-laws(7)](sb-laws.7.md#law-29)).


LAW-20 Restricted stays home: the `restricted` default on `people`, `finance`, `legal` and `executive` is what keeps their content out of everyone else's company-wide answers ([sb-spaces(7)](sb-spaces.7.md)).

## EXAMPLES

Starting from a template and narrowing it:

```yaml run
spaces:
  support:
    template: support
    members: [group:support, group:eng-oncall]
    scope:
      tickets: [SUP]
      docs: ["/Support/**"]
```

## SEE ALSO

[sb-spaces(7)](sb-spaces.7.md), [sb-space(5)](sb-space.5.md), [sb-onboarding(7)](sb-onboarding.7.md), [08 Department brains](../docs/08-department-brains.md)

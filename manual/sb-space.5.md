---
title: "sb-space(5)"
man: "5"
summary: The spaces and barriers blocks of a tenant manifest (checked and planned from SB-1001, applied from SB-1008).
---

# sb-space(5)

## NAME

sb-space — the `spaces:` and `barriers:` blocks of a tenant manifest

## SYNOPSIS

```yaml run
spaces:
  engineering:
    template: engineering
    members: [group:eng]
    scope:
      docs: ["/Engineering/**"]
```

## DESCRIPTION

These blocks sit at the top level of a tenant manifest ([sb-tenant(5)](sb-tenant.5.md)), next to `connectors` and `agents`. What a space means is in [sb-spaces(7)](sb-spaces.7.md). `sb tenant check` accepts both blocks and checks every rule below (SB-1001), and `sb tenant apply --plan` shows each space and barrier as its own line. `sb tenant apply` applies them like any other field since SB-1008, when the last part of the cell that has to enforce spaces (per-space kill switches and audit) was built; until then it refused them, so no cell ran with spaces only partly enforced. Both blocks are optional, and a manifest without them behaves exactly as today.

## FIELDS

### `spaces`

A map from a space name to its definition. A space name is `[a-z][a-z0-9-]{1,30}`, unique in the manifest, and the name `company` is reserved for company-wide asks. At most 50 spaces.

| Field | Type | Required | Default | Rules |
|---|---|---|---|---|
| `template` | string | no | `custom`, or the parent's | One of the templates in [sb-departments(7)](sb-departments.7.md). It supplies the defaults below; any field set here wins. A team space (`parent`) without one takes its department's |
| `parent` | space name | no | none | The department space this one is a team of (SB-3901; sb-spaces(7), The tree). The parent is another space of the manifest, and has no parent of its own, so the tree is at most company, department and team. A team is at least as strict as its department: its `label` is at least the parent's and its `retention_days` at most the parent's, and when it leaves them out it takes the parent's. `members` are never inherited: a team's members are its own groups. A team without its own `glossary`, `start_here`, `review_after_days` or `agents` takes its department's (SB-3902): glossaries merge, the team's own term winning, and so do `skills` (SB-4601); start-here lists join, the department's first, at most ten; agents are the department's that the team's template allows |
| `members` | list of `group:NAME` | yes | | At least one, each once. Groups come from the cell's directory (SCIM). An unknown group is an error at `apply`, never ignored |
| `stewards` | list of `group:NAME` | no | `[]` | May propose changes to this space (W12), and answer the questions its members send to it and publish notes (SB-1704, SB-1705; sb-spaces(7)). Stewards aren't members unless listed in `members` too. A `restricted` space needs at least one |
| `label` | label | no | from the template | The default label for documents in this space whose source carries none: `public`, `internal`, `confidential` or `restricted`. A document's front matter `sensitivity:` only raises it (SB-5405), and whatever cites a document carries its label (LAW-32) |
| `scope` | map kind → list of patterns | yes | | Kinds are the manifest's connector kinds (`docs`, `chat`, `tickets`, `crm`, and from W11 `code`, `hris`, `accounting`). Every kind used must be in `connectors`. Patterns are as in `exclude` (sb-tenant(5)): `/Folder/**` for docs, `#name-*` for chat, a project or repository name for the others |
| `agents` | list | no | from the template | A subset of the manifest's `agents`. An agent not listed in `agents` is an error |
| `retention_days` | int | no | the tenant's | 30 to the tenant's `retention_days`. A space can keep less, never more: the nightly sweep ages the space's records by its own days (SB-1401). A record whose store doesn't know its space (a conversation, a log file) ages by the tenant's |
| `fields` | map kind → list of field names | no | `{}` | For a structured source (sb-connector(3), Fields): the sensitive fields this space lets ingest index, for example `{ hris: [salary_band] }`. A document with one of them is `restricted`. Non-sensitive declared fields need no listing. Every kind used must be in the space's `scope` |
| `channels` | list of `#name` | no | `[]` | The chat-app channels whose `/ask` asks in this space (sb-ask(3), The chat app). A channel belongs to at most one space |
| `review_after_days` | int | no | `365` | 30 to 3650. A document this space holds whose version is older than this, and that answers keep citing, goes on the stewards' review list (sb-ask(3), Documents to review; SB-1709), and its cites show their age |
| `brief` | `{ agent, weekly }` | no | none | The space's one scheduled agent (SB-4005; sb-agents(7), Scheduled and triggered agents): `agent` is `ask`, one of the space's own `agents` (never `draft-reply`), or, in a `product` or `support` space, `voc` (SB-4205), and `weekly` the day it runs, `monday` to `sunday`. Once a week, on the notify job's first run after 08:00 that day in the `digest` time zone (or UTC), the agent answers one fixed question, "What happened in this space this week, and what is still open?", as the space's first active steward (by email), and the answer waits in the space's steward queue as a draft. A space with a brief needs `stewards`. Changed like any field |
| `updates` | bool | no | `false` | (SB-4102) The `update` agent drafts this space's status update every Monday (sb-agents(7), The update agent), and its stewards publish it as a note under `notes/NAME/updates/`, which the space above lists. A space with `updates` needs `stewards` |
| `leader_week` | bool | no | `true` | (SB-4503) In a space with teams and `stewards`, the `leader-week` agent drafts the week across its branch for its stewards every Monday (sb-agents(7), The leader's week). `false` turns it off. Only a space with teams uses it, and it isn't inherited |
| `requests` | `{ categories, due_days }` | no | none | (SB-4104) What members may ask this space for: `categories`, 1 to 10 names of 1 to 40 letters, digits, spaces or `& / : ' -`, none twice (ignoring case), and `due_days`, 1 to 30 working days (default 5). A request goes to the space's steward queue with its category and due date, and climbs the tree once it is past due (sb-ask(3), Requests to a department). A space with requests needs `stewards` to answer them, or they go nowhere: `POST …/requests` is `EINVAL` |
| `budget_usd_month` | int | no | none | (SB-4105) The space's monthly model budget in US dollars, at most the tenant's `budget_usd_month` and, in a team, at most its department's. Its branch's spend (its own and its teams', sb-observability(7), Spend by space) at 80% tells its stewards once a month; at 100% its scheduled briefs and updates, and its teams', pause until the month ends (sb-agents(7)). Asks never stop for a space's budget: only the tenant's refuses a call. Not inherited |
| `incident_labels` | list | no | `[sev-1]` | (SB-4204) The ticket labels that make a ticket an incident in this space: 1 to 10 labels of 1 to 40 letters, digits, spaces or `_ . : / -`, none twice (ignoring case). When the notify job first sees a ticket in the space whose text names one of them on a `Labels:` line among its first five lines (sb-connector(3), `tickets: linear`), the `incident-brief` agent drafts one brief for the stewards (sb-agents(7), Scheduled and triggered agents). Switch it off with the agent's kill switch |
| `mailbox` | an email address | no | | (SB-5907) The one **shared** mailbox whose new messages start the space's agent, with `on: [mail-received]`, for example `support@acme.example`. It is read through Microsoft Graph (`Mail.Read`, which an application access policy limits to the mailboxes spaces name); a person's own mailbox is never named or read. `on` listing `mail-received` without a `mailbox` is an error |
| `routing` | `economy`, `balanced` or `frontier` | no | `balanced` | (SB-5002) How the space's asks pick a model tier. `balanced` is the usual rule (sb-evals(7), Model tiers); `economy` sends everything but research to the small tier; `frontier` sends everything to the main tier. `economy` also never uses the deep tier (SB-5704). A held small tier answers on main whatever the policy. A team inherits its department's, and can't be above it (`economy` < `balanced` < `frontier`). Evals run each space's cases on the tiers its policy uses |
| `on` (work events) | `pr-opened`, `ticket-filed`, `mail-received` | no | none | (SB-5301) Beside `source-changed`: a new pull request on GitHub, or a new ticket, in the space starts a draft about it for the space's first active steward (sb-agents(7), Start on a work event); `mail-received` (SB-5907), with `mailbox`, a new message in the space's shared mailbox (sb-agents(7), Start on a new email) |
| `suspect` | `skip` or `monitor` | no | `skip` | (SB-5101) What happens to a passage that reads like instructions to the model (sb-ingest(7), Suspect passages): `skip` keeps it out of every prompt; `monitor` lets it through wrapped and flagged. Both record it for the stewards. A team inherits its department's |
| `wide_sensitive` | `hold` or `warn` | no | `hold` | (SB-5402) What happens to a sensitive document its source shares too widely (sb-permissions(7), Shared too widely): `hold` leaves it out for everyone but its owner and the space's stewards until the owner narrows it or confirms it; `warn` uses it and says so. A team inherits its department's |
| `model` | `main` | no | none | (SB-4405) Pins every ask in this space to the main model (sb-evals(7), Model tiers) |
| `glossary` | map | no | `{}` | The space's terms and acronyms, `TERM: MEANING` (SB-2203; stewards see the terms its documents define as suggestions, sb-ask(3), SB-3407): at most 200, each term 1 to 40 characters and each meaning 1 to 200, printable, no term twice (ignoring case). Search and answers **in this space** expand a term the question uses with its meaning (sb-ask(3), Glossary); a company-wide ask or another space never does. An entry that reads like an instruction to the model is kept but **flagged and not used**, and the stewards' review list names it. Changed like any field: a steward's proposal an operator approves (sb-spaces(7)) |
| `glossary_in` | map | no | `{}` | (SB-6104) Each glossary term's own words in other languages, `TERM: { LANG: WORD }`, such as `PO: { de: Bestellung, fr: "bon de commande" }`. A term it names must be in the space's glossary or its parent's; a language is one of sb-cell(5) `languages`' codes; a word is 1 to 40 printable characters. A question or an answer that uses the word uses the term: its meaning joins the search and the word is marked in the answer (sb-ask(3), Glossary). A team takes its department's words for the terms it lends. Stewards add a word with a proposal (`POST /v1/spaces/NAME/proposals/term` with `term` and `words`) |
| `skills` | map name → `{ instruction, cases }` | no | `{}` | (SB-4601) The space's named ways of writing an answer, such as `weekly-update` (sb-agents(7), Skills): at most 10, each name `[a-z][a-z0-9-]{0,39}`. `instruction` is one line of at most 1,000 printable characters on **format and steps only**; `cases` are at least 3 and at most 10 eval cases, `"QUESTION": KIND:ID`, the document an answer with the skill must cite (sb-evals(7), Skill cases). `/NAME` at the start of an ask, a research question or a routine loads it. A team takes its department's skills, its own of the same name winning. Changed like any field: a steward's proposal an operator approves (sb-spaces(7)), after `sb agents lint --manifest` |
| `rhythms` | map name → `{ subject, plan, steps }` | no | `{}` | (SB-4801) The space's own playbooks for a recurring moment (sb-departments(7), Rhythms), beside the built-in `onboarding`, `kickoff`, `close-out`, `quarterly-planning` and `one-on-one`; a space's own rhythm of a built-in name replaces it, and a team takes its department's, its own of a name winning. At most 10, each name `[a-z][a-z0-9-]{0,39}`. `subject` is `person`, `collection` or `space`; `plan` is `manager` or `steward` when that person approves the plan before it starts. `steps` are 1 to 12, in day order, each `{ day, kind, ask, approver, agent }`: `day` 0 to 120 from the start, `kind` one of `draft`, `routine`, `checkin` or `request`, `ask` one line of at most 300 printable characters under the skills' content rules, `approver` one of `subject` (a person subject only), `starter`, `manager` or `steward`, and `agent` (a draft only, or `gaps` on a request) one of `ask`, `start-here`, `catch-up`, `kickoff-brief`, `retrospective`, `goal-plan`, `one-on-one` or `gaps`. Every step is a draft in its approver's queue; none publishes or sends (LAW-29). Changed like any field, after `sb agents lint --manifest` |
| `onboarding` | boolean | no | `true` | (SB-4802) `false` turns off the onboarding rhythm for the space and its teams: nobody SCIM adds to it gets one |
| `on` | list | no | `[]` | (SB-4905) The space's opt-in triggers; the only one is `source-changed`: a new version of a document an agent draft of the space cited, or was narrowed to, in the last 90 days redrafts that draft for its approver, once a day at most, and republishes nothing (sb-agents(7), When a source changes). Listed once each; not inherited by teams |
| `start_here` | list | no | `[]` | A new member's reading list, in order (SB-1706): at most ten items, each a document `KIND:ID` (its newest version) or a folder `KIND:/Folder/` (ending in `/`: every document whose location is under it, by location). Each kind must be in the space's `scope`. Changed like any field: a steward's proposal an operator approves (sb-spaces(7)) |

### `barriers`

A list of pairs of groups. Each pair is a barrier between the two groups, in both directions (sb-spaces(7), Information barriers). At most 20 barriers. A barrier covers reads (content held by a space across it, or owned at the source by someone across it), drafts, tool calls and proposals, and approvals (sb-permissions(7), Barriers beyond reads). The kernel turns these pairs into the facts the barrier forbids read; there is no barrier rule to write by hand.

| Rule | |
|---|---|
| Each entry is a list of exactly two `group:NAME` values | `[group:deal-team-orion, group:everyone]` |
| The two groups are different | a group can't be walled off from itself |
| A pair appears once, in either order | duplicates are errors |

## VALIDATION RULES

All of these are errors, reported with line and column like every manifest error ([sb-tenant(5)](sb-tenant.5.md), Validation rules):

1. An unknown field in a space or a barrier.
2. A space with no `members` or no `scope`, or a `scope` kind the manifest has no connector for.
3. A label that isn't one of the four, or a template that isn't in [sb-departments(7)](sb-departments.7.md).
4. A space whose `retention_days` is below 30 or above the tenant's.
5. The space name `company`, or a name that breaks the pattern. More than 50 spaces, or more than 20 barriers.
6. A barrier between two groups that are both members of one space. That space could never answer either side, so it is a mistake, not a policy.
7. A member, steward or barrier group not in `group:NAME` form, or listed twice in one list.
8. A space whose label is `restricted`, set or from its template, with no `stewards`. Someone has to own who can read it.
9. An agent in a space that isn't in the manifest's `agents`.
10. A channel that isn't `#name` (lower case letters, digits, `-` and `_`), or a channel listed in two spaces.
11. A department agent (sb-agents(7)) in a space whose template doesn't list it, such as `close-checklist` in an `engineering` space.
12. A `fields` kind that isn't in the space's `scope`, or a field name that isn't `[a-z][a-z0-9_]*`.
13. A scope pattern that reaches into a **restricted** space's scope (one matches the other, such as `/**` or `/People/Cases/**` against `/People/**`) when this space has members the restricted space doesn't. Holding a restricted space's documents would make those members holders of them (SB-1208; the red team's PT-0003).
14. `barriers` with no `spaces`. The barrier checks run with the space checks, so a cell without spaces couldn't enforce them (SB-1401).
15. A `start_here` with more than ten items, an item listed twice, an item that isn't `KIND:ID` or `KIND:/Folder/`, or a kind outside the space's `scope`.
16. A `glossary` with more than 200 terms, a term listed twice (ignoring case), a term over 40 or a meaning over 200 characters, or a character that isn't printable.
17. A `parent` that isn't another space of the manifest, is the space itself, or has a parent of its own (a fourth level, or a cycle); a team whose `label` is looser than its parent's, or whose `retention_days` is above its parent's (SB-3901, LAW-24).
18. A `skills` entry with a bad name, more than 10 skills, an `instruction` that is empty, over 1,000 characters or not printable, fewer than 3 or more than 10 `cases`, a case whose question is empty or over 4,000 characters or whose expected document isn't `KIND:ID` in the space's `scope`. An `instruction` that names a tool, widens what is searched (another space, every space, company-wide, around permissions) or tells the model to act on what a document says is refused, naming its line (SB-4601, LAW-06, LAW-09).
19. A `rhythms` entry with a bad name, more than 10 rhythms, a `subject` or `plan` not in its list, no steps or more than 12, a step past day 120 or before the step above it, a `kind`, `approver` or `agent` not in its list, a step with no `approver`, an `approver: subject` on a rhythm whose subject isn't a person, an `agent` on a step that isn't a draft, an unknown field, or an `ask` that is empty, over 300 characters, on two lines, or names a tool, an action or a wider scope (SB-4801, LAW-06, LAW-09, LAW-29). A thirteenth step is refused by the lint.

A field a space leaves out takes its template's value: the label, the agents (those of the template's defaults the manifest enables, and always `ask`), and the retention (the template's, or else the tenant's, never more than the tenant's).

## EXIT STATUS / ERRORS

`sb tenant check` exits 1 with `EMANIFEST` on any rule above.

## LAWS

[LAW-16](sb-laws.7.md#law-16): spaces and barriers are manifest only, and no space exists that the manifest doesn't declare. LAW-19 to LAW-22 ([sb-spaces(7)](sb-spaces.7.md)).

## EXAMPLES

Engineering and Product share a spec folder; Finance is restricted; a deal team is walled off:

```yaml run
spaces:
  engineering:
    template: engineering
    members: [group:eng, group:product]
    stewards: [group:eng-leads]
    scope:
      docs: ["/Engineering/**", "/Specs/**"]
      chat: ["#eng-*"]
      tickets: [ENG, OPS]
  product:
    template: product
    members: [group:product]
    scope:
      docs: ["/Product/**", "/Specs/**"]
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    label: restricted
    retention_days: 365
    scope:
      docs: ["/Finance/**"]
barriers:
  - [group:deal-team-orion, group:everyone]
```

## SEE ALSO

[sb-spaces(7)](sb-spaces.7.md), [sb-space(1)](sb-space.1.md), [sb-departments(7)](sb-departments.7.md), [sb-tenant(5)](sb-tenant.5.md), [sb-tenant(1)](sb-tenant.1.md)

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
| `parent` | space name | no | none | The department space this one is a team of (SB-3901; sb-spaces(7), The tree). The parent is another space of the manifest, and has no parent of its own, so the tree is at most company, department and team. A team is at least as strict as its department: its `label` is at least the parent's and its `retention_days` at most the parent's, and when it leaves them out it takes the parent's. `members` are never inherited: a team's members are its own groups. A team without its own `glossary`, `start_here`, `review_after_days` or `agents` takes its department's (SB-3902): glossaries merge, the team's own term winning; start-here lists join, the department's first, at most ten; agents are the department's that the team's template allows |
| `members` | list of `group:NAME` | yes | | At least one, each once. Groups come from the cell's directory (SCIM). An unknown group is an error at `apply`, never ignored |
| `stewards` | list of `group:NAME` | no | `[]` | May propose changes to this space (W12), and answer the questions its members send to it and publish notes (SB-1704, SB-1705; sb-spaces(7)). Stewards aren't members unless listed in `members` too. A `restricted` space needs at least one |
| `label` | label | no | from the template | The default label for documents in this space whose source carries none: `public`, `internal`, `confidential` or `restricted` |
| `scope` | map kind → list of patterns | yes | | Kinds are the manifest's connector kinds (`docs`, `chat`, `tickets`, `crm`, and from W11 `code`, `hris`, `accounting`). Every kind used must be in `connectors`. Patterns are as in `exclude` (sb-tenant(5)): `/Folder/**` for docs, `#name-*` for chat, a project or repository name for the others |
| `agents` | list | no | from the template | A subset of the manifest's `agents`. An agent not listed in `agents` is an error |
| `retention_days` | int | no | the tenant's | 30 to the tenant's `retention_days`. A space can keep less, never more: the nightly sweep ages the space's records by its own days (SB-1401). A record whose store doesn't know its space (a conversation, a log file) ages by the tenant's |
| `fields` | map kind → list of field names | no | `{}` | For a structured source (sb-connector(3), Fields): the sensitive fields this space lets ingest index, for example `{ hris: [salary_band] }`. A document with one of them is `restricted`. Non-sensitive declared fields need no listing. Every kind used must be in the space's `scope` |
| `channels` | list of `#name` | no | `[]` | The chat-app channels whose `/ask` asks in this space (sb-ask(3), The chat app). A channel belongs to at most one space |
| `review_after_days` | int | no | `365` | 30 to 3650. A document this space holds whose version is older than this, and that answers keep citing, goes on the stewards' review list (sb-ask(3), Documents to review; SB-1709), and its cites show their age |
| `glossary` | map | no | `{}` | The space's terms and acronyms, `TERM: MEANING` (SB-2203; stewards see the terms its documents define as suggestions, sb-ask(3), SB-3407): at most 200, each term 1 to 40 characters and each meaning 1 to 200, printable, no term twice (ignoring case). Search and answers **in this space** expand a term the question uses with its meaning (sb-ask(3), Glossary); a company-wide ask or another space never does. An entry that reads like an instruction to the model is kept but **flagged and not used**, and the stewards' review list names it. Changed like any field: a steward's proposal an operator approves (sb-spaces(7)) |
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

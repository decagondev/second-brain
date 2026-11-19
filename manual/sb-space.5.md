---
title: "sb-space(5)"
man: "5"
summary: The spaces and barriers blocks of a tenant manifest (W8 draft; accepted by sb tenant check from SB-1001).
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

These blocks sit at the top level of a tenant manifest ([sb-tenant(5)](sb-tenant.5.md)), next to `connectors` and `agents`. What a space means is in [sb-spaces(7)](sb-spaces.7.md). This page is the W8 draft. **Until SB-1001 lands, `sb tenant check` refuses a manifest with either block**, with `spaces: not built yet (SB-1001)`, so a manifest that declares spaces never runs without them. Both blocks are optional, and a manifest without them behaves exactly as today.

## FIELDS

### `spaces`

A map from a space name to its definition. A space name is `[a-z][a-z0-9-]{1,30}`, unique in the manifest, and the name `company` is reserved for company-wide asks. At most 50 spaces.

| Field | Type | Required | Default | Rules |
|---|---|---|---|---|
| `template` | string | no | `custom` | One of the templates in [sb-departments(7)](sb-departments.7.md). It supplies the defaults below; any field set here wins |
| `members` | list of `group:NAME` | yes | | At least one. Groups come from the cell's directory (SCIM). An unknown group is an error at `apply`, never ignored |
| `stewards` | list of `group:NAME` | no | `[]` | May propose changes to this space (W12). Stewards aren't members unless listed in `members` too |
| `label` | label | no | from the template | The default label for documents in this space whose source carries none: `public`, `internal`, `confidential` or `restricted` |
| `scope` | map kind → list of patterns | yes | | Kinds are the manifest's connector kinds (`docs`, `chat`, `tickets`, `crm`, and from W11 `code`, `hris`, `accounting`). Every kind used must be in `connectors`. Patterns are as in `exclude` (sb-tenant(5)): `/Folder/**` for docs, `#name-*` for chat, a project or repository name for the others |
| `agents` | list | no | from the template | A subset of the manifest's `agents`. An agent not listed in `agents` is an error |
| `retention_days` | int | no | the tenant's | 30 to the tenant's `retention_days`. A space can keep less, never more |

### `barriers`

A list of pairs of groups. Each pair is a barrier between the two groups, in both directions (sb-spaces(7), Information barriers). At most 20 barriers.

| Rule | |
|---|---|
| Each entry is a list of exactly two `group:NAME` values | `[group:deal-team-orion, group:everyone]` |
| The two groups are different | a group can't be walled off from itself |
| A pair appears once, in either order | duplicates are errors |

## VALIDATION RULES

All of these are errors, reported with line and column like every manifest error ([sb-tenant(5)](sb-tenant.5.md), Validation rules):

1. An unknown field in a space or a barrier.
2. A space with no `members` or no `scope`, or a `scope` kind the manifest has no connector for.
3. A label that isn't one of the four.
4. A space whose `retention_days` is above the tenant's.
5. The space name `company`, or a name that breaks the pattern.
6. A barrier between two groups that are both members of one space. That space could never answer either side, so it is a mistake, not a policy.

## EXIT STATUS / ERRORS

`sb tenant check` exits 1 with `EMANIFEST` on any rule above, and, until SB-1001, on any manifest that has a `spaces:` or `barriers:` block at all.

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

[sb-spaces(7)](sb-spaces.7.md), [sb-departments(7)](sb-departments.7.md), [sb-tenant(5)](sb-tenant.5.md), [sb-tenant(1)](sb-tenant.1.md)

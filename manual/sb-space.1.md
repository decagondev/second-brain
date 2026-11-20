---
title: "sb-space(1)"
man: "1"
summary: Add a department space to a tenant manifest from a template.
---

# sb-space(1)

## NAME

sb-space — start a department space from the template catalogue

## SYNOPSIS

```text
sb space init FILE --template NAME --members GROUP[,GROUP…] [--name NAME] [--stewards GROUP[,GROUP…]]
sb space templates
```

## DESCRIPTION

**init** adds one space to the manifest `FILE`, filled in from a template of the catalogue (sb-departments(7)), and writes the file back only if the result passes every rule of `sb tenant check` (sb-space(5)). It changes nothing else in the file: comments and the order of the other fields stay as they are. The new space is added under an existing block-style `spaces:` key, or a new `spaces:` key is added at the end.

The space is named `--name`, or the template's name. Its `members` and `stewards` are the groups given. Its `scope` comes from the template's default sources that the manifest has connectors for: `docs` gets `["/Title/**"]`, `chat` gets `["#name-*"]`, `tickets` gets the name in capitals, and `code`, `hris`, `accounting` and `crm` get `["*"]`, where Title and name are the space's name. A template whose sources the manifest has no connector for is an error that names the connector to add. A restricted template needs `--stewards`, as a restricted space does.

The label, agents and retention are left out of the entry, so they keep coming from the template; set them in the manifest to override them. Then review the new scope patterns, which are only a starting point, and run `sb tenant apply FILE --plan`.

**templates** prints the catalogue this release ships: each template's name, label, default sources, default agents and eval suite.

The catalogue is data, `internal/core/manifest/catalogue.json`, and every release bundle pins its hash (sb-release(7)), so two cells on the same release read the same templates.

## OPTIONS

| Option | Meaning |
|---|---|
| `--template NAME` | One of the catalogue's templates, or `custom` |
| `--members GROUP,…` | The space's member groups, `group:NAME` each |
| `--name NAME` | The space's name; default the template's |
| `--stewards GROUP,…` | The space's stewards; required for a restricted template |

## EXIT STATUS

0 when the space was added. 1 with `EMANIFEST` when the manifest, before or after, fails `sb tenant check`, or the space already exists. 2 with `EUSAGE` for a missing or bad option, or an unknown template.

## LAWS

[LAW-16](sb-laws.7.md#law-16): the space exists only once it is in the manifest and applied.

## EXAMPLES

```text
$ sb space init acme.yaml --template people --members group:people-team --stewards group:people-leads
space people added to acme.yaml (template people, label restricted)
$ sb tenant apply acme.yaml --plan
+ space people (template people, label restricted, members group:people-team, stewards group:people-leads, agents ask, retention 365, scope docs /People/**)
plan: 1 to add, 0 to change, 0 to remove
```

## SEE ALSO

[sb-space(5)](sb-space.5.md), [sb-departments(7)](sb-departments.7.md), [sb-tenant(1)](sb-tenant.1.md), [sb(1)](sb.1.md)

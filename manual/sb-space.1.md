---
title: "sb-space(1)"
man: "1"
summary: Add a department space to a tenant manifest from a template, and preview a reorg's effect on spaces.
---

# sb-space(1)

## NAME

sb-space — start a department space from the template catalogue

## SYNOPSIS

```text
sb space init FILE --template NAME --members GROUP[,GROUP…] [--name NAME] [--stewards GROUP[,GROUP…]]
sb space templates
sb space diff FILE --idp-snapshot SNAPSHOT [--json]
sb space scope-report (--cell URL | --tenant T) [--json]
```

## DESCRIPTION

**init** adds one space to the manifest `FILE`, filled in from a template of the catalogue (sb-departments(7)), and writes the file back only if the result passes every rule of `sb tenant check` (sb-space(5)). It changes nothing else in the file: comments and the order of the other fields stay as they are. The new space is added under an existing block-style `spaces:` key, or a new `spaces:` key is added at the end.

The space is named `--name`, or the template's name. Its `members` and `stewards` are the groups given. Its `scope` comes from the template's default sources that the manifest has connectors for: `docs` gets `["/Title/**"]`, `chat` gets `["#name-*"]`, `tickets` gets the name in capitals, and `code`, `hris`, `accounting` and `crm` get `["*"]`, where Title and name are the space's name. A template whose sources the manifest has no connector for is an error that names the connector to add. A restricted template needs `--stewards`, as a restricted space does.

The label, agents and retention are left out of the entry, so they keep coming from the template; set them in the manifest to override them. Then review the new scope patterns, which are only a starting point, and run `sb tenant apply FILE --plan`.

**templates** prints the catalogue this release ships: each template's name, label, default sources, default agents and eval suite.

**diff** previews a reorg before the IdP changes (SB-1206). `SNAPSHOT` is a JSON file of the people a reorg moves, with their groups before and after:

```json
{"version": 1, "people": [
  {"user": "ana@acme.example", "before": ["group:finance"], "after": ["group:finance", "group:deal-team"]},
  {"user": "bo@acme.example",  "before": ["group:eng"],     "after": []}
]}
```

Unknown fields, a version other than 1, a person listed twice, an empty `user`, or a group not in `group:NAME` form are errors. For each person, `diff` lists the spaces of manifest `FILE` they could use before and after (a person can use a space when one of their groups is one of its `members`), and prints the spaces they **gain** and **lose**. It flags:

- `loses-all`: a person who could use at least one space before and none after;
- `gains-restricted`: a person who gains a space whose label is `restricted` (the template's, unless the manifest sets one);
- `barrier`: a person whose groups after the reorg sit on both sides of one of the manifest's `barriers`, so the barrier would wall them off from themselves.

People whose spaces don't change are counted, not listed. `diff` reads only its two files: it needs no cell, no network and no credentials, and it writes nothing and changes nothing (LAW-16: the change happens only when the IdP and the manifest change). `--json` prints `{"people": [{"user", "gains", "loses", "flags"}], "unchanged": N}`.

**scope-report** (SB-2903) prints, for each space of the cell's applied manifest and each of its scope patterns, how many documents in the cell's index the pattern matches now, out of how many that connector kind has, from `GET /v1/admin/index/scopes` (sb-admin(3)). It names a pattern that matches **none**, which is usually a typo or a folder that moved, and one that matches **more than half** of its connector's documents, which is usually too wide (such as `/**`). It reads only document locations, never their text, and changes nothing: a fix is a manifest change (LAW-16). It exits 0 either way; `--json` prints `{"patterns": [{"space", "kind", "pattern", "matches", "of", "problem"}]}`, with `problem` `none`, `too_wide` or empty.

The catalogue is data, `internal/core/manifest/catalogue.json`, and every release bundle pins its hash (sb-release(7)), so two cells on the same release read the same templates.

## OPTIONS

| Option | Meaning |
|---|---|
| `--template NAME` | One of the catalogue's templates, or `custom` |
| `--members GROUP,…` | The space's member groups, `group:NAME` each |
| `--name NAME` | The space's name; default the template's |
| `--stewards GROUP,…` | The space's stewards; required for a restricted template |
| `--idp-snapshot FILE` | diff: the reorg's people, with their groups before and after |
| `--json` | diff, scope-report: machine-readable output |
| `--cell URL`, `--tenant T` | scope-report: the cell to ask, by URL or by the tenant's registered endpoint |

## EXIT STATUS

0 when the space was added, or `diff` ran (flags are output, not failures). 1 with `EMANIFEST` for a manifest `diff` can't read. 2 with `EUSAGE` for a snapshot that breaks its format. 1 with `EMANIFEST` when the manifest, before or after, fails `sb tenant check`, or the space already exists. 2 with `EUSAGE` for a missing or bad option, or an unknown template.

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

```text
$ sb space diff acme.yaml --idp-snapshot reorg-q1.json
ana@acme.example  gains people  loses -  [gains-restricted]
bo@acme.example  gains -  loses engineering  [loses-all]
1 unchanged
```

## SEE ALSO

[sb-space(5)](sb-space.5.md), [sb-departments(7)](sb-departments.7.md), [sb-tenant(1)](sb-tenant.1.md), [sb(1)](sb.1.md)

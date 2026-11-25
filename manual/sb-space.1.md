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
sb tree move FILE SPACE --to PARENT --plan [--write OUT] [--json]
sb tree merge FILE FROM INTO --plan [--write OUT] [--json]
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

**tree move** and **tree merge** are the reorg dry run (SB-4806). `move` makes `SPACE` a team of `PARENT` (`--to ""` makes it a department of its own); `merge` folds space `FROM` into `INTO`: `FROM`'s scope joins `INTO`'s and its teams become `INTO`'s. It needs the same member groups and a label at least as strict. Each reads manifest `FILE`, makes the manifest the change would leave, checks it with every rule of `sb tenant check`, and prints, for the space, its teams, and its old and new parent (or the space merged into): the parent before and after, the queues a question climbs before and after (the space, its department, company; sb-spaces(7), Escalation), and each setting it would **gain** or **lose** by inheritance: glossary terms, skills, rhythms, start-here, agents, label and retention. A merged space is listed as merged. The output ends with `nothing applied`: the change is a manifest diff the admin proposes through the usual review (sb-tenant(1), apply). `--write OUT` saves that manifest; `FILE` is never touched. A move or merge changes no space's `members`, and a merge needs the two spaces' members to be the same, so nobody loses a space they use or a document they own; a change that would is refused (`sb space diff` previews changes of people's groups).

A change that would let anyone see more is refused, naming the setting and the space: a label that would loosen or a retention that would grow (LAW-24). Neither command needs a cell. What is under way in each space (rhythms in flight, open and waiting queue items, open projects, routines, triggered agents) needs the running cell: `GET /v1/admin/tree/plan` (sb-admin(3)) gives the same plan with `in_flight`. `--plan` is required; there is no form that applies a change.

The catalogue is data, `internal/core/manifest/catalogue.json`, and every release bundle pins its hash (sb-release(7)), so two cells on the same release read the same templates.

## OPTIONS

| Option | Meaning |
|---|---|
| `--template NAME` | One of the catalogue's templates, or `custom` |
| `--members GROUP,…` | The space's member groups, `group:NAME` each |
| `--name NAME` | The space's name; default the template's |
| `--stewards GROUP,…` | The space's stewards; required for a restricted template |
| `--idp-snapshot FILE` | diff: the reorg's people, with their groups before and after |
| `--json` | diff, scope-report, tree: machine-readable output |
| `--to PARENT` | tree move: the new parent; `""` for none |
| `--plan` | tree: print the dry run; required |
| `--write OUT` | tree: write the manifest the change would leave to `OUT` |
| `--cell URL`, `--tenant T` | scope-report: the cell to ask, by URL or by the tenant's registered endpoint |

## EXIT STATUS

0 when the space was added, or `diff` ran (flags are output, not failures). 1 with `EMANIFEST` for a manifest `diff` can't read. 2 with `EUSAGE` for a snapshot that breaks its format. 1 with `EMANIFEST` when the manifest, before or after, fails `sb tenant check`, or the space already exists. 2 with `EUSAGE` for a missing or bad option, or an unknown template. For `tree`: 0 when the plan printed, 2 with `EINVAL` for a space that doesn't exist or a result `sb tenant check` refuses (such as a team that would keep more than its new parent), and 3 with `EPERM` for a change that would widen access: a label that loosens, a retention that grows, or a merge whose member groups differ.

## LAWS

[LAW-16](sb-laws.7.md#law-16): the space exists only once it is in the manifest and applied. [LAW-24](sb-laws.7.md#law-24): a tree change that would widen access is refused (`tree`: 3 with `EPERM`).

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

```text
$ sb tree move acme.yaml support --to customer --plan --write acme-next.yaml
customer  parent - -> -  escalates customer > company -> customer > company
ops  parent - -> -  escalates ops > company -> ops > company
support  parent ops -> customer  escalates support > ops > company -> support > customer > company
  glossary  gains SLA  loses runbook
nothing applied; propose the manifest diff for review
$ sb tree merge acme.yaml support customer --plan
sb tree merge: reorg: the change would widen access: support's documents would reach group:sales [EPERM]
```

## SEE ALSO

[sb-space(5)](sb-space.5.md), [sb-departments(7)](sb-departments.7.md), [sb-tenant(1)](sb-tenant.1.md), [sb(1)](sb.1.md)

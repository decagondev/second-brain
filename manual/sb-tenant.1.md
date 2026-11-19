---
title: "sb-tenant(1)"
man: "1"
summary: Check, apply and verify a tenant manifest.
---

# sb-tenant(1)

## NAME

sb-tenant — validate a manifest, make a cell match it, and prove the laws hold on real data

## SYNOPSIS

```text
sb tenant check  FILE
sb tenant apply  FILE [--plan]
sb tenant verify FILE [--law LAW-nn]... [--sample N]
```

## DESCRIPTION

These three verbs are the whole onboarding story.

**check** parses and validates `FILE` against [sb-tenant(5)](sb-tenant.5.md). It needs no network and changes nothing. A valid file prints `FILE: ok` and exits 0. Otherwise errors go to standard output in compiler format, `file:line:col: field: message`, one per line and sorted by position, so editors and agents can jump to them. A count line (`N errors`) follows, and the command exits 1 with `EMANIFEST` on standard error. With `--json` it prints `{"file": …, "errors": [{"line", "col", "field", "message"}]}` and the same exit status. An unreadable `FILE` is a usage error (exit 2).

**apply** makes the tenant's cell match `FILE`. It creates the cell if it doesn't exist (database, index, vault namespace, KMS key, queue, region pin), enables connectors and agents, and loads approval rules. Apply is **idempotent**: running it twice with the same file changes nothing the second time. It is **resumable**: if interrupted, run it again. `--plan` prints what would change and exits.

Immutable fields (`tenant`, `region`) can't be changed by apply; changing them is a new tenant. A plan that would change `region` fails with `EMANIFEST`.

**How apply works.** Apply computes the plan, then enacts it one change at a time, in plan order. For a new cell, it first stamps the cell from the template into the account named by the cell's custody record (`placement`, [sb-factory(7)](sb-factory.7.md)); the cell is recorded in the registry only after every resource exists. Each later change is recorded in the registry as soon as it is enacted. Before and after each step, apply writes a journal entry (tenant, apply id, step, change, `start` or `done`), never content or secrets. If apply stops at any point, running it again plans only what is left and finishes the job. When there is nothing to do, apply prints `apply: no changes` and writes nothing: no resources, no registry record, no journal entry.

Apply prints each change as it is enacted, in the plan format, and ends with `apply: A added, C changed, R removed`. With `--json` it prints the plan it enacted. **Workflows wait for their evals.** A plan that adds or changes a workflow is refused as a whole, before any step runs, unless that workflow's eval record says it passed. The record is `evals/<tenant>/<workflow>.json` in the control plane, written by `python3 -m evals run --tenant-dir tenants/<tenant> --model … --record FILE` (sb-eval(1)), and it must be at most 30 days old. A missing, failed, stale or unreadable record is `EEVAL` (exit 1) and names the workflow (SB-410). Errors from the registry or the placement are `EUNAVAIL` (exit 7). The changes enacted before the error stay recorded, so a rerun resumes.

**The plan.** `--plan` checks `FILE` as `check` does, reads the tenant's applied state from the control-plane registry, and prints the difference resource by resource. It changes nothing and exits 0, whether or not there are changes. If the registry can't be reached, or its answer can't be read, the plan fails with `EUNAVAIL` (exit 7). It never assumes the tenant has no cell, because that would plan a second one.

| Resource | Name | Shown as |
|---|---|---|
| `cell` | tenant | `cell acme-logistics (region eu)` |
| `setting` | `channel`, `idp`, `retention_days`, `budget_usd_month`, `pilot_group` | `setting channel = stable` |
| `connector` | kind | `connector docs (sharepoint, read-only)`; `read-write` if a workflow writes to it |
| `exclude` | kind | `exclude chat (#hr-*, #legal-*)` |
| `agent` | name | `agent ask` |
| `workflow` | name | `workflow ticket-update (approval group:it-leads)` |

Lines start with `+` (add), `~` (change, showing the old value after `was`) or `-` (remove), in the order of the table, then by name. The last line is `plan: A to add, C to change, R to remove`, or `plan: no changes`. With `--json` the plan is `{"tenant": …, "changes": [{"op", "kind", "name", "attrs", "was"}], "add": A, "change": C, "remove": R}`, with `op` one of `add`, `change`, `remove`. The same manifest and the same applied state always give the same plan, byte for byte.

**verify** runs law checks against the tenant's real data and users, without printing any content:

| Check | What it does |
|---|---|
| LAW-01 | Samples users and questions; asserts every citation is openable by the asker **and that the source was asked** |
| LAW-02 | Issues adversarial queries using another tenant's known document ids; expects `ETENANT` or no results |
| LAW-03 | Injects a timeout on one connector; expects every chunk from it to be dropped |
| LAW-14 | Asserts every claim is cited or labelled `uncited` |
| Evals | Runs the tenant's golden questions (`sb-eval(5)`) |

Any check that returns *unknown* counts as a **fail**.

`--law LAW-17` runs the access-path check in sb-factory(7), Access paths, exactly. It is the same check as `sb factory access`.

How verify decides the other laws, exactly (SB-601):

- **Sampling.** Users are drawn from the directory and questions from the titles of indexed documents, `--sample` of each pairing (default 50), with a seed printed in the report so a run can be repeated. Nothing about a document but its id and version is ever printed.
- **LAW-01.** For every citation in every sampled answer, verify asks the source **again, live**, whether the asker may open that document now. It also checks that the answer's own decision list shows the source was asked. A citation the source doesn't allow, or one with no recorded live check, fails.
- **LAW-02.** Verify takes document ids that belong to **other** tenants (from the control plane's registry, ids only) and asks for them by id and by text. Any result that cites one fails.
- **LAW-03.** Verify makes one connector time out and asks again. Any citation from that connector fails.
- **LAW-14.** Every claim of every sampled answer must have a citation or be labelled `uncited`.
- **Report.** One line per law: `pass`, `fail` or `unknown`, with counts only. The exit status is 0 only if every law passes. `unknown` (for example, no users to sample) is a failure.

Verify runs against a cell through the same `verify.Target` interface whether the cell is local or deployed. Against a deployed cell it needs the cell runtime's verifier endpoint, and until then `sb tenant verify` exits `ENOTIMPL`. The local cell (`tools/evalcell --verify`) runs it over a fixture world today.

## OPTIONS

| Option | Meaning |
|---|---|
| `--plan` | (apply) Show the diff only |
| `--law LAW-nn` | (verify) Run only the given law; repeatable |
| `--sample N` | (verify) Users × questions sampled (default 50) |
| `--json` | Machine output |

## EXIT STATUS

0 success · 1 invalid manifest, apply failed, or any law failed or unknown · 2 usage · 3 not permitted · 7 unavailable. See [sb(1)](sb.1.md).

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03), [LAW-14](sb-laws.7.md#law-14), [LAW-16](sb-laws.7.md#law-16).

## EXAMPLES

```text
$ sb tenant check acme.yaml
acme.yaml:2:1: version: field not allowed; pick a channel (stable|testing) instead
acme.yaml:9:3: workflows[0]: "ticket-update" needs an approval rule; add approval: { ticket-update: group:<name> }
2 errors

$ sb tenant apply acme.yaml --plan
+ cell acme-logistics (region eu)
+ setting budget_usd_month = 1000
+ setting channel = stable
+ setting idp = oidc://login.acme.example
+ setting retention_days = 365
+ connector chat (slack, read-only)
+ connector docs (sharepoint, read-only)
+ agent ask
plan: 8 to add, 0 to change, 0 to remove

$ sb tenant apply acme.yaml --plan     # later: tickets, drafts and one workflow added
~ setting budget_usd_month = 1500 (was 1000)
+ connector tickets (jira, read-write)
+ agent draft-reply
+ workflow ticket-update (approval group:it-leads)
plan: 3 to add, 1 to change, 0 to remove

$ sb tenant verify acme.yaml
LAW-01  pass   50/50 sampled answers, 212 citations, source asked for all
LAW-02  pass   40 adversarial queries, 0 cross-tenant results
LAW-03  pass   chat timeout injected, 0 chat chunks used
LAW-14  pass   612 claims: 571 cited, 41 labelled uncited, 0 unlabelled
evals   pass   golden 38/40 (threshold 36)
verify: 5 passed, 0 failed
```

## SEE ALSO

[sb-tenant(5)](sb-tenant.5.md), [sb-onboarding(7)](sb-onboarding.7.md), [sb-laws(7)](sb-laws.7.md)

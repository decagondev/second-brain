---
title: "sb-kill(1)"
man: "1"
summary: Stop an agent, connector or tenant in 10 seconds or less.
---

# sb-kill(1)

## NAME

sb-kill — set and clear kill switches

## SYNOPSIS

```text
sb kill   agent NAME     --tenant T --reason TEXT
sb kill   connector KIND --tenant T --reason TEXT
sb kill   tenant         --tenant T --reason TEXT
sb kill   agent NAME     --all-tenants --reason TEXT      (maintainers)
sb unkill agent|connector|tenant [NAME] --tenant T --reason TEXT
sb kill   list [--tenant T]
```

## DESCRIPTION

A kill switch stops new work immediately, without a deploy or a manifest change. `sb kill` returns only after the control plane has recorded the switch; it takes effect in every affected cell within **10 seconds**.

| Scope | Stops |
|---|---|
| `agent NAME` | New runs and new tool calls of that agent |
| `connector KIND` | `fetch`, `changes`, `acl` and `act` on that connector. Its chunks become `Unknown`, so answers stop using them |
| `tenant` | Everything except admin, audit read and `sb unkill` |

A tool call that already passed its final kill check finishes. Nothing new starts once the switch is seen (LAW-10).

`--reason` is required and goes into the audit log. Kill switches are runtime state: to make one permanent, remove the agent or connector from the manifest (the console offers this as a diff).

With a local control plane (`--endpoint file:///DIR`), switches live in `DIR/kill/TENANT.json` and their audit rows in `DIR/audit/TENANT.ndjson`. The actor recorded is `SB_ACTOR` when it's set, else `sys_cli_local`. `--all-tenants` needs the control-plane client and exits `ENOTIMPL` until then. `kill list` prints each set switch with its time, actor and reason.

## EXIT STATUS

0 switch recorded · 2 usage (missing `--reason`) · 3 not permitted.

## LAWS

[LAW-10](sb-laws.7.md#law-10), [LAW-04](sb-laws.7.md#law-04).

## EXAMPLES

```sh
sb kill agent draft-reply --tenant acme-logistics --reason "INC-42 drafts quoting retired policy"
sb kill list --tenant acme-logistics
sb unkill agent draft-reply --tenant acme-logistics --reason "INC-42 fixed in rel_2027-03-04_1a2b3c"
```

## SEE ALSO

[sb-kill-switches(7)](sb-kill-switches.7.md), [sb-admin(3)](sb-admin.3.md)

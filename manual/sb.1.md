---
title: "sb(1)"
man: "1"
summary: The Second Brain command-line tool.
---

# sb(1)

## NAME

sb — operate Second Brain tenants, kill switches, audit, releases and evals

## SYNOPSIS

```text
sb [--json] [--tenant NAME] [--endpoint URL] COMMAND [ARGS]

sb tenant  check|apply|verify FILE
sb space   init FILE --template NAME --members GROUP…   (sb-space(1))
sb space   templates
sb space   diff FILE --idp-snapshot SNAPSHOT [--json]
sb kill    agent|connector|space|tenant [NAME] --reason TEXT
sb unkill  agent|connector|space|tenant [NAME] --reason TEXT
sb audit   verify|tail|export
sb release status|promote|rollback       (maintainers)
sb factory status|handoff|access|detach|verify-export  (factory operators and tenant owners, sb-factory(1))
sb eval    run|diff SUITE
sb version
```

## DESCRIPTION

`sb` is a single static binary. It's the only tool a company's IT team needs to onboard and operate a tenant, and the same tool maintainers use for releases.

Every command:

- prints human-readable output by default, and one JSON document with `--json`;
- uses the stable exit codes below, so scripts and agents can branch on them;
- writes nothing locally except `~/.config/sb/credentials` (an OIDC device-flow token);
- is audited server-side when it changes anything.

During the build, every command in the SYNOPSIS exists. A command whose ticket hasn't landed yet exits 2 with `ENOTIMPL` and names the ticket, for example `sb tenant check: not built yet (SB-101)`. It never pretends to succeed.

## OPTIONS

| Option | Default | Meaning |
|---|---|---|
| `--json` | off | Machine output. Errors are `{"error":{"code":…,"message":…}}` on stdout |
| `--tenant NAME` | from manifest or `SB_TENANT` | Tenant to act on |
| `--endpoint URL` | `SB_ENDPOINT` or the control-plane default | Control-plane URL. `file:///DIR` uses a local control plane in a directory instead (development and tests, until SB-105): `DIR/tenants` is the registry, `DIR/accounts` holds the local placement accounts (ADR-0018), and `DIR/journal` the apply journals |
| `-q`, `--quiet` | off | Print only errors |
| `-h`, `--help` | | Help for any command |

## ENVIRONMENT

| Variable | Meaning |
|---|---|
| `SB_TENANT` | Default tenant |
| `SB_ENDPOINT` | Control-plane URL |
| `SB_TOKEN` | Pre-issued token for CI (short-lived, from workload identity); never a long-lived secret |
| `NO_COLOR` | Disable colour |

## EXIT STATUS

| Code | Meaning | Typical error codes |
|---|---|---|
| 0 | Success, or nothing to do | |
| 1 | A check failed: invalid manifest, law failed or unknown, chain broken, eval failed, drift | `EMANIFEST`, `ELAW`, `ECHAIN`, `EEVAL`, `EDRIFT` |
| 2 | Usage error, or command not built yet | `EUSAGE`, `EINVAL`, `ENOTIMPL` |
| 3 | Not signed in or not permitted | `EAUTH`, `EPERM`, `ETENANT` |
| 4 | A kill switch is active | `EKILLED` |
| 5 | Approval missing, expired or mismatched | `ENOAPPROVAL` |
| 6 | Action did not run (no receipt) | `ENORECEIPT` |
| 7 | Dependency unavailable or timed out | `EUNAVAIL` |
| 8 | Budget or rate limit | `EBUDGET` |

The full registry is [sb-errors(7)](sb-errors.7.md).

## EXAMPLES

```sh
sb tenant check acme.yaml && sb tenant apply acme.yaml && sb tenant verify acme.yaml --cell https://api-acme.example
sb --json tenant verify acme.yaml --cell https://api-acme.example | jq '.Results[] | select(.Status != "pass")'
sb kill agent draft-reply --tenant acme-logistics --reason "INC-42 drafts quoting wrong policy"
```

```text
$ sb version
sb 0.0.0-dev (commit unknown)
$ sb --json version
{"version":"0.0.0-dev","commit":"unknown"}
$ sb tenant check acme.yaml
sb tenant check: not built yet (SB-101) [ENOTIMPL]
$ echo $?
2
```

## SEE ALSO

[sb-space(1)](sb-space.1.md), [sb-tenant(1)](sb-tenant.1.md), [sb-kill(1)](sb-kill.1.md), [sb-audit(1)](sb-audit.1.md), [sb-release(1)](sb-release.1.md), [sb-eval(1)](sb-eval.1.md), [sb-factory(1)](sb-factory.1.md), [sb-factory(7)](sb-factory.7.md), [sb-errors(7)](sb-errors.7.md)

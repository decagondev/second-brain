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
sb tenant  retention --plan [--days N] (--cell URL | --tenant T)
sb space   init FILE --template NAME --members GROUP…   (sb-space(1))
sb space   templates
sb space   diff FILE --idp-snapshot SNAPSHOT [--json]
sb space   scope-report (--cell URL | --tenant T) [--json]
sb tree    move FILE SPACE --to PARENT --plan [--write OUT]   (sb-space(1), tree)
sb tree    merge FILE FROM INTO --plan [--write OUT]
sb index   prune [--dry-run] [--reason TEXT] (--cell URL | --tenant T)   (sb-admin(3), index leftovers)
sb index   refresh CITE --reason TEXT (--cell URL | --tenant T)      (sb-admin(3), index refresh)
sb kill    agent|connector|space|tenant [NAME] --reason TEXT
sb unkill  agent|connector|space|tenant [NAME] --reason TEXT
sb audit   verify|tail|export
sb release status|promote|rollback|notes   (maintainers; notes: --from DIR --to DIR)
sb agents  lint [--root DIR]             (sb-agents(7), Lint)
sb agents  record NAME [--tenant T | --cell URL] [--out DIR] [--key ed25519:PUB]  (sb-agents(7), An agent's record)
sb inventory export [--tenant T | --cell URL] [--out DIR] [--key ed25519:PUB]  (sb-admin(3), Inventory)
sb rollout list|hold|release [FEATURE] [--tenant T | --cell URL]  (sb-release(7), Feature rollouts)
sb factory status|handoff|access|detach|verify-export  (factory operators and tenant owners, sb-factory(1))
sb eval    run|gate|diff
sb selfhost doctor [--facts FILE]        (a self-hosted box, sb-selfhost(7))
sb selfhost verify-bundle DIR --trusted ed25519:KEY|--cosign-identity ID [--channel stable]
sb selfhost check-proxy URL | --file FILE
sb selfhost check-dsn FILE [--service OUT]
sb selfhost check-email HOST:PORT [--ca FILE]
sb selfhost check-notify --to EMAIL (--cell URL | --tenant T)
sb selfhost check-chat --slack TOKENFILE [--api URL] | --mattermost URL --token-file FILE [--team ID]
sb selfhost check-microsoft --config FILE [--vault DIR] [--mailbox ADDR]... [--probe ADDR]   (docs/runbooks/microsoft.md)
sb cell check-config FILE [--vault DIR]...
sb prove   TICKET --config FILE [--vault DIR] [--out DIR] [--person EMAIL] [--mailbox ADDR]... [--probe ADDR] [--facts FILE] [--ca FILE]
sb prove   --verify FILE [--key ed25519:PUB] | --list
sb record  graph --config FILE [--vault DIR] --out FILE [--person EMAIL] [--mailbox ADDR]
sb selfhost retire-statement --tenant T --export FILE (--anchor-seed FILE --check | --cell-key KEY --host H --destroyed TEXT... [--kept TEXT]... --out FILE)
sb selfhost verify-retirement FILE --key ed25519:KEY
sb secrets scan [--known PATH]... PATH...   (sb-observability(7))
sb backup keygen --public FILE --private FILE   (sb-selfhost(7), Sealed off-box copies)
sb backup seal --to PUBFILE [--out FILE] FILE
sb backup open --key KEYFILE [--out FILE] FILE.sealed
sb backup push --to s3://BUCKET/PREFIX --keys FILE [--as NAME] FILE   (sb-selfhost(7), Off-box copies to S3)
sb backup pull --from s3://BUCKET/PREFIX --keys FILE --out DIR NAME...
sb backup list --from s3://BUCKET/PREFIX --keys FILE [SUB]
sb backup check --to s3://BUCKET/PREFIX --keys FILE
sb recovery split --tenant T --out DIR [--custodian N=NAME]... ARCHIVE   (sb-selfhost(7), The recovery kit)
sb recovery join --kit KIT.sealed --out FILE SHARE...
sb version
```

## DESCRIPTION

`sb` is a single static binary. It's the only tool a company's IT team needs to onboard and operate a tenant, and the same tool maintainers use for releases.

Every command:

- prints human-readable output by default, and one JSON document with `--json`;
- uses the stable exit codes below, so scripts and agents can branch on them;
- writes nothing locally except `~/.config/sb/credentials` (an OIDC device-flow token);
- is audited server-side when it changes anything.

`sb index prune --dry-run` prints the index's leftovers by cite (`GET /v1/admin/index/leftovers`): documents their source no longer lists, and chunks no space holds. Without `--dry-run` it removes the documents their source no longer lists, which needs `--reason` and is audited (`POST /v1/admin/index/prune`; SB-2905). It never touches a document its source still lists, so chunks no space holds are only listed.

`sb index refresh CITE --reason TEXT` re-reads one document from its source now (`POST /v1/admin/index/refresh`; SB-3202) and prints its version and whether it changed, or that it was removed because the source no longer has it. Audited; admins only.

`sb prove TICKET --config FILE` runs that ticket's acceptance criteria against the real system the cell's configuration names, reading only (SB-6001). It covers SB-5901 to SB-5907 (Microsoft 365: `--person` someone to check access, groups and a calendar for, `--mailbox` each shared mailbox, `--probe` a mailbox that must be refused), SB-2603 (the Mattermost bot), SB-2601 (the mail server; `--ca` the company's CA) and SB-1605 (a self-hosted box: `--facts` from `tools/selfhost/facts.sh`); `sb prove --list` prints them. It writes `state/proofs/TICKET-YYYYMMDD-HHMMSS.json` (or under `--out`), signed with the cell's anchor key as a document of kind `proof`, as an agent's record is (sb-agents(7)): each criterion's fixed name, pass or fail, counts, milliseconds and up to 20 ids hashed with a key made for that proof and never kept. A proof holds no document text, no name and no unhashed id, and one that would is refused before signing. A missing input fails the criteria that need it. It exits 1 when any criterion fails. `sb prove --verify FILE [--key ed25519:PUB]` checks a proof's content rules, hash and signature. A proof doesn't change a ticket: a person moves a ticket from In review by attaching its proof to the Progress section. SB-2005 and SB-2105 post to Mattermost, so a person proves them by hand.

`sb record graph --config FILE --out FILE` runs a fixed script of reads against a real Microsoft 365 sandbox (each drive's delta, the first file and its permissions, `--person`'s groups and manager, `--mailbox`'s delta) and writes a recording in the recorded fakes' format (SB-6002): every string replaced by one of the same shape and length, the same value always by the same replacement, every address by a person at `acme-logistics`, `beta-freight` or `gamma-parts`, and keys, numbers and the protocol's fixed values kept. The recording is scanned before it is written, against the vault's values and secrets' shapes, and any address that isn't a placeholder person refuses it (`ELAW`, nothing written). `internal/adapters/msgraph/testdata/recording.json` is the recording the Graph fake is tested against: the fake must answer every step with at least the recording's fields, of the same types.

`sb selfhost check-microsoft --config FILE` checks a cell's Microsoft 365 setup (SB-6003): the app registration gets a token; every application permission it holds only reads (any other is named and fails); with `Mail.Read`, each shared mailbox the manifest names (and `--mailbox`) reads, and the `--probe` mailbox, such as a person's own, is refused (without `--probe` the access policy can't be confirmed, which fails); each drive in `microsoft.drives` answers; and the tenant's sensitivity labels `labels.microsoft` doesn't map are listed by id, with how many. Each failure names its section of `docs/runbooks/microsoft.md`. It reads only and prints no secret.

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

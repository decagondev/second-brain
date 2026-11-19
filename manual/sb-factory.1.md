---
title: "sb-factory(1)"
man: "1"
summary: Custody, hand-off, access-path checks and detach (factory operators and tenant owners).
---

# sb-factory(1)

## NAME

sb-factory — show a cell's custody, hand it off to its company, check the factory's access, and detach

## SYNOPSIS

```text
sb factory status   TENANT
sb factory handoff  plan    TENANT --to tenant-account://PROVIDER/ID --region REGION [--retain GRANT]...
sb factory handoff  show    PLAN
sb factory handoff  approve PLAN --as factory|tenant
sb factory handoff  execute PLAN
sb factory handoff  abort   PLAN --reason TEXT
sb factory access   TENANT
sb factory detach   TENANT --accept-unsupported --out DIR
```

`PLAN` is a plan hash, `ph_` followed by 64 hex characters, or a unique prefix of at least 12 of them.

## DESCRIPTION

The concepts (custody levels, the hand-off steps, detach) are in [sb-factory(7)](sb-factory.7.md). This page is the command grammar. Every command that changes anything writes a `kind: custody` audit row in the tenant's cell before it acts, and in the factory's own audit log (LAW-04).

### Who may run what

| Command | Role (sb-policy(5), Factory roles) |
|---|---|
| `status`, `handoff show`, `access` | `factory-operator`, or the tenant's `tenant-owner` |
| `handoff plan`, `handoff execute`, `handoff abort` | `factory-operator` |
| `handoff approve --as factory` | `factory-operator`, and not the person who made the plan |
| `handoff approve --as tenant` | `tenant-owner` (member of the tenant IdP group `sb-owners`), signed in through the tenant's own IdP, with step-up |
| `detach` | `tenant-owner`, on a cell in `tenant` custody |

Anything else is `EPERM` (exit 3). Neither role can read tenant content.

### `status TENANT`

Prints the custody record (`custody`, `placement`, `key_custodian`), the channel, and the hand-off in progress, if any, with its phase. It changes nothing.

### `handoff plan TENANT --to tenant-account://PROVIDER/ID --region REGION`

Builds the hand-off plan and prints it with its hash. The plan holds the tenant, its current custody record, the target account and region, every resource to provision, the key steps, **every factory access path to revoke**, and the grants to retain. Access paths are enumerated from IaC state, IAM and the registry (SB-706), never typed in. `--region` must equal the cell's pinned region; a different region is `EINVAL`. `PROVIDER` is one the factory has a placement adapter for (`gcp` first). Each `--retain` adds a grant; `publish-releases` is always retained. The grants are:

| Grant | Lets the factory |
|---|---|
| `publish-releases` | publish signed bundles the cell pulls (always kept) |
| `platform-kill` | set a platform-wide agent kill switch on this cell |
| `support-metrics` | read the cell's content-free metrics |

No grant reads content, and no grant can unwrap the cell's key. The plan is stored and audited as `HandoffPlanned`. A new plan for the same tenant replaces the old one and voids all approvals of it.

### `handoff show PLAN`

Prints the stored plan, its approvals and its phase.

### `handoff approve PLAN --as factory|tenant`

Approves **that hash**. The approval records the approver, the side and the hash. A plan that has changed since it was printed has a different hash, so an approval can only ever be for what the approver saw.

### `handoff execute PLAN`

Runs the hand-off once both sides have approved the current plan: provision in the tenant account, re-wrap the root key under the tenant's KMS, move state, verify, cut over, revoke the factory's unlisted paths, destroy the factory's key wrap. Each step is recorded before it starts, and `execute` resumes from the last finished step if it is interrupted. Before cutover it stops on any failed check and leaves the factory cell serving. After cutover, it finishes the revocation and the key step even when retried.

### `handoff abort PLAN --reason TEXT`

Allowed until cutover. It removes what was provisioned in the tenant account and the tenant's key wrap, and returns the cell to `Managed`. After cutover it is `EINVAL`: going back is a hand-back, with its own plan and approvals.

### `access TENANT`

Enumerates every factory access path to the cell (IAM roles and bindings, deploy credentials, break-glass roles, network routes, key wraps) and compares them with the hand-off record's retained grants. It exits 1 (`ELAW`) and names each unlisted path when one exists (LAW-17). It runs after cutover and monthly.

### `detach TENANT --accept-unsupported --out DIR`

Writes the export package (sb-factory(7), Detach) to `DIR`, then removes the `publish-releases` grant and the registry entry, keeping the custody history. `--accept-unsupported` is required: after detach the factory guarantees nothing about the cell.

## OPTIONS

| Option | Meaning |
|---|---|
| `--to tenant-account://PROVIDER/ID` | The target account (plan) |
| `--region REGION` | Must be the cell's pinned region (plan) |
| `--retain GRANT` | Keep a grant after hand-off; repeatable (plan) |
| `--as factory\|tenant` | Which side approves (approve) |
| `--reason TEXT` | Required for abort; goes into the audit log |
| `--accept-unsupported` | Required for detach |
| `--out DIR` | Where detach writes the export package |

The global options (`--json`, `--endpoint`) are in [sb(1)](sb.1.md).

## EXIT STATUS

0 success · 1 a check failed: unlisted access (`ELAW`), chain or law verification failed during execute (`ECHAIN`, `ELAW`) · 2 usage, a bad plan hash, the wrong region, or not built yet (`EUSAGE`, `EINVAL`, `ENOTIMPL`) · 3 not permitted (`EPERM`, `EAUTH`) · 5 approvals missing or for another hash (`ENOAPPROVAL`) · 7 the registry, an account or a KMS is unreachable (`EUNAVAIL`).

Until its ticket lands, a command exits 2 with `ENOTIMPL` and names it: `handoff` SB-705, `access` SB-706, `detach` SB-708.

## LAWS

[LAW-17](sb-laws.7.md#law-17), [LAW-18](sb-laws.7.md#law-18), [LAW-04](sb-laws.7.md#law-04), [LAW-11](sb-laws.7.md#law-11).

## EXAMPLES

```text
$ sb factory handoff plan gamma-parts --to tenant-account://gcp/gamma-prod-7731 --region eu --retain platform-kill
plan ph_3c09e1d2a4b5…  gamma-parts → tenant-account://gcp/gamma-prod-7731 (region eu)
  provision   postgres, index schema, vault namespace, queue, kms key (tenant-owned)
  key         wrap the root key under the tenant KMS, then destroy the factory wrap
  revoke      7 factory access paths (deploy role, 2 break-glass roles, 4 IAM bindings)
  retain      publish-releases, platform-kill
needs approval from: a factory operator other than you, and the tenant owner (sb-owners)
```

```sh
sb factory handoff approve ph_3c09e1d2a4b5 --as factory
sb factory handoff approve ph_3c09e1d2a4b5 --as tenant      # the tenant owner, from their own IdP
sb factory handoff execute ph_3c09e1d2a4b5
sb factory access gamma-parts                               # exit 0: only the retained grants remain
```

## SEE ALSO

[sb-factory(7)](sb-factory.7.md), [sb-policy(5)](sb-policy.5.md), [the hand-off runbook](../docs/runbooks/handoff.md), [Handoff.tla](../proofs/tla/Handoff.tla)

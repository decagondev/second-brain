---
title: "sb-factory(7)"
man: "7"
summary: The factory that produces every tenant's Second Brain, custody levels, and hand-off.
---

# sb-factory(7)

## NAME

sb-factory — how Second Brains are produced, who has custody of each, and how custody is handed off

## SYNOPSIS

```text
custody:  factory  →  tenant  →  detached
                   ←  (hand-back)

sb factory status   TENANT
sb factory handoff  plan|show|approve|execute|abort ...
sb factory access   TENANT
sb factory detach   TENANT --accept-unsupported --out DIR
```

## DESCRIPTION

The **factory** is the control plane, the cell template and the release train, run by the parent organisation. It stamps one isolated cell per tenant ([sb-tenancy(7)](sb-tenancy.7.md)) and keeps every cell on the release train ([sb-release(7)](sb-release.7.md)). The factory's own organisation is a normal tenant (ring 0). There is no superuser.

### Custody

Each cell has a **custody record** in the factory registry, which says where the cell lives and who holds its keys:

| Field | Values | v1 |
|---|---|---|
| `custody` | `factory`, `tenant`, `detached` | always `factory` |
| `placement` | `factory-account`, `tenant-account://ID` | always `factory-account` |
| `key_custodian` | `factory`, `tenant` | always `factory` |

`sb factory status TENANT` prints the custody record from the registry, the tenant's channel, and whether a hand-off is in progress. It changes nothing. Until the release controller exists (SB-507) it doesn't show the running release. If the registry can't answer, it fails with `EUNAVAIL`, and a tenant with no cell is `EUSAGE`.

The custody values are defined once in code (`internal/core/cell`, `custody.go`). A CI check (`tools/check/custody.py`) fails on any account id, `tenant-account://` address or custody literal anywhere else. Code learns where a cell lives only from its custody record (ADR-0015).

Custody is **not** a manifest field. A manifest containing `custody:`, `placement:` or `key_custodian:` fails `sb tenant check` with a pointer to this page. Custody changes only through the hand-off workflow, with dual approval.

### Levels

- **factory:** the cell runs in the factory's cloud account under a factory-held root key. Company IT operates the manifest, approvals and kill switches; the factory operates infrastructure and releases. This is how every v1 tenant runs.
- **tenant:** the cell runs in the company's own cloud account under a company-held root key. It still **pulls** signed releases from the factory train, so every law still holds. The factory keeps only the grants listed in the hand-off record, at minimum *publish releases*.
- **detached:** the company takes an export package and leaves the train. The factory no longer guarantees anything about the cell.

### Hand-off

1. **plan:** the factory operator generates a plan: target account, resources, key re-wrap, every factory access path to remove, and the grants to retain. The plan has a hash.
2. **approve:** both a factory operator and the tenant owner approve **that hash**. Changing the plan changes the hash and voids both approvals.
3. **execute:** provision in the tenant account, re-wrap the root key under the tenant's KMS, move state, verify laws and the audit chain, cut over, revoke factory paths, then destroy the factory key wrap.
4. **abort:** allowed any time before cutover. The factory cell never stops serving before cutover.

The command grammar, options and exit codes are in [sb-factory(1)](sb-factory.1.md), and the roles that may run each command in [sb-policy(5), Factory roles](sb-policy.5.md#factory-roles-exactly-sb-702). The step-by-step procedure for both sides is [the hand-off runbook](../docs/runbooks/handoff.md).

### The hand-off, exactly (SB-701)

A hand-off moves through the phases of `proofs/tla/Handoff.tla`, and the controller (SB-705) stores the phase and writes a `kind: custody` audit row for each step **before** it acts:

| Phase | Reached by | The factory cell | Key wraps | Factory access |
|---|---|---|---|---|
| `Managed` | start, or abort | serves | factory | all paths |
| `Planned` | `plan` (a new plan voids all approvals) | serves | factory | all paths |
| `Provisioned` | `execute`, once a factory operator other than the planner and the tenant owner have both approved the current hash | serves | factory | all paths |
| `KeyDual` | the root key is wrapped under the tenant's KMS and unwrapped once to prove it | serves | factory **and** tenant | all paths |
| `Moved` | state is restored in the tenant account (the game day's restore, SB-613) and the index is rebuilt | serves | both | all paths |
| `Verified` | `sb tenant verify` passes, the audit chain verifies with the cell's key and continues, and the attestation matches | serves | both | all paths |
| `CutOver` | traffic moves; every path not in the retained grants is revoked | stops | both | retained grants only |
| `TenantOwned` | the factory's wrap is destroyed, and `sb factory access` passes | gone | tenant | retained grants only |
| `Detached` | `detach` | gone | tenant | none |

Three rules hold at every step. They are the model's invariants and the law tests' checks:

1. **The key always has a custodian, and the serving cell can unwrap it** (LAW-18). The tenant's wrap is made and tested before the factory's is destroyed, and the factory's is destroyed only after cutover.
2. **Nothing moves without both approvals of the current plan.** An approval is for a hash, and changing the plan changes the hash.
3. **After cutover the factory holds only the retained grants** (LAW-17), and after detach it holds none.

Abort is allowed in every phase from `Planned` to `Verified`, and it restores `Managed` exactly: factory wrap only, all paths, the factory cell serving. A failed check in `execute` stops before cutover and leaves the phase where it was, so the operator can fix the problem and resume or abort.

### Key wraps, exactly (SB-704)

A cell's data is encrypted under **data keys**, one per purpose. Today the only purpose is `audit-signing`, the seed of the cell's audit signing key (sb-audit(5)), made when the cell is first stamped. Each data key is stored only **wrapped** by a custodian's root key in that custodian's KMS. The **wrap inventory** records, for each purpose, one wrap per custodian: the custodian (`factory` or `tenant`), its KMS key id, and the wrapped bytes. A hand-off never re-encrypts data. It changes who can unwrap the data keys:

1. **Re-wrap** (phase `KeyDual`): for each purpose, unwrap with the factory's key, wrap under the tenant's key, then unwrap the new wrap and compare. A purpose whose new wrap doesn't give back the same key fails the step. Re-wrap is idempotent: a tenant wrap that already unwraps correctly is kept.
2. **Destroy the factory wrap** (phase `TenantOwned`): refused with `ELAW` unless, for **every** purpose, the tenant's wrap unwraps **now** to the same data key as the factory's, and the cell being served is the tenant's. The check runs at the moment of destroying, not from a record of an earlier check.

Abort removes the tenant's wraps and keeps the factory's. Nothing ever removes the last working wrap of a purpose (LAW-18).

`sb factory status TENANT` prints the inventory as checked now, for example `key_wraps audit-signing: factory, tenant`, listing for each purpose the custodians whose wrap unwraps it at that moment. A wrap that fails to unwrap is shown as `factory (broken)`, and that is a Sev-1 (`sb_unwrappable_keys`, LAW-18).

The **hand-off record** is written at `TenantOwned` and kept in the registry's custody history: the plan hash, both approvals, the new custody record (`custody: tenant`, `placement: tenant-account://PROVIDER/ID`, `key_custodian: tenant`), the retained grants, and the `sb factory access` result. LAW-17 is checked against this record.

## LAWS

[LAW-17](sb-laws.7.md#law-17) no unlisted factory access after hand-off · [LAW-18](sb-laws.7.md#law-18) the root key always has a custodian. LAW-01 to LAW-16 continue to hold for `factory` and `tenant` custody.

## EXAMPLES

```text
$ sb factory status gamma-parts
custody        factory
placement      factory-account
key_custodian  factory
key_wraps      audit-signing: factory
channel        stable
handoff        none
```

```text
$ sb factory handoff plan gamma-parts --to tenant-account://cloud/gamma-prod-7731
plan 3c09e1…  target tenant-account://cloud/gamma-prod-7731 (region eu)
  provision   postgres, index schema, vault ns, queue, kms key (tenant-owned)
  key         re-wrap root key under tenant KMS, then destroy factory wrap
  revoke      7 factory access paths (deploy role, 2 break-glass roles, 4 IAM bindings)
  retain      publish-releases
needs approval from: factory operator, tenant owner (group:gamma-it-owners)
```

## SEE ALSO

[sb-factory(1)](sb-factory.1.md), [sb-tenancy(7)](sb-tenancy.7.md), [sb-release(7)](sb-release.7.md), [16 Factory and hand-off](../docs/architecture/16-factory-and-handoff.md), [ADR-0015](../docs/adr/ADR-0015-factory-and-handoff.md)

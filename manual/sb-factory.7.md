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
- **tenant:** the cell runs in the company's own cloud account under a company-held root key. It still **pulls** signed releases from the factory train, so every law still holds (sb-release(7), Cells in a tenant's account). The factory keeps only the grants listed in the hand-off record, at minimum *publish releases*.
- **detached:** the company takes an export package and leaves the train. The factory no longer guarantees anything about the cell.

### Hand-off

1. **plan:** the factory operator generates a plan: target account, resources, key re-wrap, every factory access path to remove, and the grants to retain. The plan has a hash.
2. **approve:** both a factory operator and the tenant owner approve **that hash**. Changing the plan changes the hash and voids both approvals.
3. **execute:** provision in the tenant account, re-wrap the root key under the tenant's KMS, move state, verify laws and the audit chain, cut over, revoke factory paths, then destroy the factory key wrap.
4. **abort:** allowed any time before cutover. The factory cell never stops serving before cutover.

The model-provider contract moves separately and at the tenant's pace. The tenant puts a key from its own provider account at `vault://tenants/TENANT/model`, and the next call uses it (sb-tenant(5), Model billing and cost statements). Monthly cost statements let both sides reconcile the last factory-billed month.

The command grammar, options and exit codes are in [sb-factory(1)](sb-factory.1.md), and the roles that may run each command in [sb-policy(5), Factory roles](sb-policy.5.md#factory-roles-exactly-sb-702). The step-by-step procedure for both sides is [the hand-off runbook](../docs/runbooks/handoff.md).

### The hand-off, exactly (SB-701)

A hand-off moves through the phases of `proofs/tla/Handoff.tla`. The controller (`internal/core/handoff`, SB-705) has the same phases and transitions, and a test reads the model to check that they match. It stores the phase and writes a `kind: custody` audit row for each step **before** it acts:

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

Each step's infrastructure work (provision, move, verify, cutover) goes through a `Target` port, so the same controller drives any cloud. The revocation at cutover re-enumerates the factory's paths afterwards and fails the step if any unlisted path remains. Abort is allowed in every phase from `Planned` to `Verified`, and it restores `Managed` exactly: factory wrap only, all paths, the factory cell serving. A failed check in `execute` stops before cutover and leaves the phase where it was, so the operator can fix the problem and resume or abort.

### Key wraps, exactly (SB-704)

A cell's data is encrypted under **data keys**, one per purpose. The purposes are `audit-signing`, the seed of the cell's audit signing key (sb-audit(5)), made when the cell is first stamped, and one `space/spc_NAME` per department space (sb-tenancy(7), Per-space data keys; SB-1003). Each data key is stored only **wrapped** by a custodian's root key in that custodian's KMS. The **wrap inventory** records, for each purpose, one wrap per custodian: the custodian (`factory` or `tenant`), its KMS key id, and the wrapped bytes. A hand-off never re-encrypts data. It changes who can unwrap the data keys:

1. **Re-wrap** (phase `KeyDual`): for each purpose, unwrap with the factory's key, wrap under the tenant's key, then unwrap the new wrap and compare. A purpose whose new wrap doesn't give back the same key fails the step. Re-wrap is idempotent: a tenant wrap that already unwraps correctly is kept.
2. **Destroy the factory wrap** (phase `TenantOwned`): refused with `ELAW` unless, for **every** purpose, the tenant's wrap unwraps **now** to the same data key as the factory's, and the cell being served is the tenant's. The check runs at the moment of destroying, not from a record of an earlier check.

Abort removes the tenant's wraps and keeps the factory's. Nothing ever removes the last working wrap of a purpose (LAW-18), except crypto-shredding a space the manifest dropped, which removes that space's purpose as a whole and is refused for any other purpose.

`sb factory status TENANT` prints the inventory as checked now, for example `key_wraps audit-signing: factory, tenant`, listing for each purpose the custodians whose wrap unwraps it at that moment. A wrap that fails to unwrap is shown as `factory (broken)`, and that is a Sev-1 (`sb_unwrappable_keys`, LAW-18).

### Access paths, exactly (SB-706)

An **access path** is any way the factory could reach a cell. Paths are enumerated from where they actually live, never from a list someone keeps:

| Source | Paths |
|---|---|
| The cell project's IAM policy | every binding whose member is one of the factory's principals |
| The deploy service account's IAM policy | the factory's right to impersonate it |
| The cell's KMS key's IAM policy | any factory binding on the key |
| The wrap inventory (SB-704) | a factory wrap of any data key |

Each path names the grant it serves, if any. Only three roles serve a grant: the custom role `sbPlatformKill` serves `platform-kill`, `roles/monitoring.viewer` serves `support-metrics`, and nothing in the tenant's project serves `publish-releases`, because the cell pulls bundles from the factory's registry. A key wrap and an impersonation binding serve no grant.

**The check.** `sb factory access TENANT` enumerates every source and compares the paths with the hand-off record's retained grants. A path whose grant isn't retained, or that serves none, is **unlisted**. Any unlisted path fails the check (`ELAW`, exit 1), and the check prints each one. A source that can't be read makes the check fail with `EUNAVAIL` (exit 7): an enumeration that didn't finish proves nothing, so it never passes. For a cell still in `factory` custody, the paths are printed and LAW-17 doesn't apply yet.

**The monthly watch.** Once a month, and right after every cutover, the control plane runs the same check for every `tenant`-custody cell. It sets `sb_unlisted_access_paths{tenant}` to the number of unlisted paths, which raises the LAW-17 Sev-1 when it is above 0. A check that can't finish sets it to 1, so silence never looks like a pass.

The enumeration reads IAM policies with a separate, read-only **verifier** identity that the tenant grants (`resourcemanager.projects.getIamPolicy`, `iam.serviceAccounts.getIamPolicy`, `cloudkms.cryptoKeys.getIamPolicy`), never with the deploy account. The tenant can therefore run the check themselves, and the factory can't hide a path by withdrawing its own read access.

### Detach, exactly (SB-708)

Detach takes a `tenant`-custody cell off the train. `sb factory detach TENANT --accept-unsupported --out DIR` writes the **export package** to `DIR`:

| Path | What |
|---|---|
| `manifest.yaml` | the applied manifest |
| `policies/*.cedar` | the tenant's policies |
| `evals/` | the tenant's eval suites and their latest results |
| `audit/audit.ndjson` | the signed audit export, every row from `seq` 1 (sb-audit(5), Export) |
| `audit/anchor-key.txt` | the cell's anchor public key, to pin when verifying |
| `release/bundle.json`, `bundle.sig`, `gate.json`, `gate.sig`, `sbom.spdx.json` | the running release, its gate record and their signatures |
| `release/release-key.txt` | the release identity the bundle is signed by |
| `runbooks/` | the manual and runbooks at that release |
| `MANIFEST.json` | every file above with its SHA-256, written last |

The index isn't exported, because it rebuilds from the sources (LAW-13). Neither are secrets: connector credentials stay in the tenant's own vault.

The package verifies **offline** with `sb factory verify-export DIR`. It checks that every file in `MANIFEST.json` is present with its SHA-256 and that there are no other files. It checks that the audit export verifies with `audit/anchor-key.txt` from `seq` 1 to its last row (LAW-11), that the bundle's signature verifies with `release/release-key.txt`, and that the gate record is for that bundle and passed. A required item that is missing fails the check.

Detach runs in this order, and stops at the first failure:

1. The hand-off controller moves to `Detached` and writes the custody row (SB-705).
2. The export package is written and verified, as above. Detach never revokes anything on the strength of a package that didn't verify.
3. Every factory access path is revoked, `publish-releases` included, and `sb factory access` must then show **none** (LAW-17).
4. The release controller drops the cell from the train (sb-release(7)).
5. The registry replaces the cell record with a **tombstone**: the cell stops routing, and only the custody history stays (`custody`, `placement` and `key_custodian` with the time of each change). No content, manifest or audit row stays with the factory.

The **hand-off record** is written at `TenantOwned` and kept in the registry's custody history: the plan hash, both approvals, the new custody record (`custody: tenant`, `placement: tenant-account://PROVIDER/ID`, `key_custodian: tenant`), the retained grants, and the `sb factory access` result. LAW-17 is checked against this record.

### Hand-off game day, exactly (SB-710)

Before any company's real hand-off, the factory hands off a friendly tenant's staging cell end to end and detaches a copy of it, following [the game day runbook](../docs/runbooks/handoff-game-day.md). The day passes only if all of these hold:

| Check | Pass |
|---|---|
| The hand-off reaches `TenantOwned` with both approvals of one plan hash | yes |
| `sb factory access` after cutover lists only retained grants (LAW-17) | yes |
| Every data key unwraps with the tenant's key only (LAW-18) | yes |
| The audit chain after the move is the chain before it, continued, and verifies with the cell's key (LAW-11) | yes |
| `sb tenant verify` passes on the tenant's cell | yes |
| The detached copy's export package verifies offline, and the factory holds no path to the copy | yes |

`tests/handoff` rehearses the whole day in `make check`, on the real controller, wrap inventory, backups, restore, access check and registry, with the GCP adapter against its recorded fake.

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

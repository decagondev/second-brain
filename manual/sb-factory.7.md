---
title: "sb-factory(7)"
man: "7"
summary: The factory that produces every tenant's Second Brain, custody levels, and hand-off (commands planned for W7).
---

# sb-factory(7)

## NAME

sb-factory — how Second Brains are produced, who has custody of each, and how custody is handed off

## SYNOPSIS

```text
custody:  factory  →  tenant  →  detached
                   ←  (hand-back)

sb factory status   TENANT                                    (W1, read-only)
sb factory handoff  plan    TENANT --to tenant-account://ID   (W7)
sb factory handoff  approve PLAN-HASH                         (W7)
sb factory handoff  execute PLAN-HASH                         (W7)
sb factory handoff  abort   PLAN-HASH                         (W7)
sb factory detach   TENANT  --accept-unsupported              (W7)
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

## LAWS

[LAW-17](sb-laws.7.md#law-17) no unlisted factory access after hand-off · [LAW-18](sb-laws.7.md#law-18) the root key always has a custodian. LAW-01 to LAW-16 continue to hold for `factory` and `tenant` custody.

## EXAMPLES

```text
$ sb factory status gamma-parts
custody        factory
placement      factory-account
key_custodian  factory
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

[sb-tenancy(7)](sb-tenancy.7.md), [sb-release(7)](sb-release.7.md), [16 Factory and hand-off](../docs/architecture/16-factory-and-handoff.md), [ADR-0015](../docs/adr/ADR-0015-factory-and-handoff.md)

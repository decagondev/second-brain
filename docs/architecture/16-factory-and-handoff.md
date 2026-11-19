---
title: "16 Factory and Hand-off"
summary: One factory produces and runs every tenant's Second Brain, and can hand custody of one to its company.
---

# 16 Factory and hand-off

**Job:** produce an isolated Second Brain for each tenant from one template, run many of them under one release train, and, when a company asks, transfer custody of its Second Brain to it without losing data, audit continuity or the laws.

**Status:** design accepted (ADR-0015). v1 builds the seams; the hand-off itself ships in W7 (epic E08).

## The factory

```mermaid
flowchart TB
    subgraph F["Factory (parent-org account)"]
        direction LR
        TPL["Cell template"]
        REL["Release train<br/>signed bundles"]
        REG["Registry<br/>+ custody records"]
        OPS["Kill switches<br/>+ attestation"]
        HO["Hand-off controller<br/>(W7)"]
    end
    subgraph FA["Cells in factory custody (factory account)"]
        direction LR
        R0["Factory's own tenant<br/>ring 0, not a superuser"]
        C1["acme-logistics"]
        C2["beta-freight"]
    end
    subgraph TA["Cell in tenant custody (tenant's own account)"]
        C3["gamma-parts"]
    end
    DET["Detached cell<br/>off the train"]
    F <-- "stamp · pull bundles · push attestations" --> FA
    F <-- "stamp via Placement port · pull bundles · push attestations" --> TA
    TA -. "detach: export, leave the train" .-> DET
```

"Multi-tenant" here means many isolated cells run by one factory, never one shared store (ADR-0001). The factory itself is tenant ring 0, which keeps it honest: it uses the product exactly as companies do.

## Custody levels

```mermaid
stateDiagram-v2
    direction LR
    [*] --> Factory: sb tenant apply
    Factory --> Tenant: hand-off (dual approval)
    Tenant --> Factory: hand-back (dual approval)
    Tenant --> Detached: detach (tenant owner)
    Detached --> [*]
```

| | `factory` | `tenant` | `detached` |
|---|---|---|---|
| Cloud account | Factory | Tenant | Tenant |
| Root key custodian | Factory KMS | Tenant KMS | Tenant KMS |
| Releases | Pulled from the factory train | Pulled from the factory train | None |
| Laws in force | All | All, LAW-06 and LAW-08 included | The factory guarantees none; the tenant has the code and runbooks |
| Factory access | Operate, release, audited break-glass | Publish releases; anything else only as granted by the tenant, listed in the hand-off record | None |
| Kill switches | Factory and tenant operators | Tenant operators; factory only for platform-wide agent kills, if granted | Tenant |

## The hand-off workflow

```mermaid
sequenceDiagram
    autonumber
    participant FO as Factory operator
    participant TO as Tenant owner
    participant HC as Hand-off controller
    participant TA as Tenant cloud account
    participant CELL as Cell
    participant L as Audit
    FO->>HC: sb factory handoff plan acme-logistics --to tenant-account://…
    HC->>L: custody event: HandoffPlanned (plan hash)
    HC-->>FO: plan: resources, key re-wrap, access paths to remove, retained grants
    FO->>HC: approve plan hash
    TO->>HC: approve plan hash
    HC->>TA: provision via Placement port (least-privilege deploy role)
    HC->>TA: create tenant KMS key, then re-wrap cell root key under it
    Note over HC,TA: both custodians can unwrap here, briefly (LAW-18 forbids zero, not two)
    HC->>CELL: move stateful resources (snapshot + restore in region), rebuild index
    HC->>CELL: verify: laws pass, audit chain verifies, attestation ok
    HC->>CELL: cutover: DNS, revoke every factory path not in retained grants
    HC->>TA: destroy factory wrap of the root key
    HC->>L: custody event: TenantOwned (record of retained grants)
    HC-->>TO: done, and sb tenant verify --law LAW-17 passes
```

Before cutover, the workflow can be aborted, and the factory cell keeps serving throughout. After cutover there is no automatic rollback: going back is a hand-back, with its own dual approval.

## Detach

A `tenant`-custody cell can leave the train, for example after a divestiture. Detach produces an **export package**:

| Item | Form |
|---|---|
| Manifest and tenant policies | Files from the tenant repo |
| Audit log | Signed export ([sb-audit(5)](../../manual/sb-audit.5.md)) plus the cell's anchor public key |
| Eval suites and results | Files |
| Running release bundle and SBOM | Signed artefacts |
| Runbooks | The manual at that release |

The index isn't exported, because it rebuilds from sources (LAW-13). The factory then removes the release-publish grant and deletes its registry entry, keeping only the custody history.

## Laws

| Law | Statement |
|---|---|
| [LAW-17](../../manual/sb-laws.7.md#law-17) | After hand-off, the factory holds no access path to the cell except those listed in the hand-off record |
| [LAW-18](../../manual/sb-laws.7.md#law-18) | A cell's root key always has at least one custodian; data is never orphaned by a hand-off |

Both are modelled in [`proofs/tla/Handoff.tla`](../../proofs/tla/Handoff.tla), along with dual approval and abort-before-cutover.

## Threats specific to hand-off

| Threat | Mitigation |
|---|---|
| Factory insider keeps a hidden access path | Access paths are enumerated from IaC state, IAM and the registry, not from memory; LAW-17 check after cutover and monthly |
| Hand-off hijacked to an attacker's account | Dual approval of a plan hash that includes the target account id; tenant owner approves from their own IdP with step-up |
| Key orphaned mid-transfer | Re-wrap under the tenant key **before** destroying the factory wrap (LAW-18) |
| Audit chain broken by the move | Per-cell anchor key travels with the cell; `sb audit verify` must pass before cutover |
| Detached cell keeps running a vulnerable build | Out of factory scope by definition; the export package includes the SBOM, and detach requires the tenant owner to accept this |

## Tickets

v1 seams: SB-103, SB-105, SB-114, SB-401, SB-506, SB-509. Hand-off: epic E08, wave W7. Manual: [`sb-factory(7)`](../../manual/sb-factory.7.md). Decision: [ADR-0015](../adr/ADR-0015-factory-and-handoff.md).

---
title: "13 Release and Supply Chain"
summary: One trunk, signed reproducible bundles, channels not versions, rings, one-step rollback.
---

# 13 Release and supply chain

**Job:** get a change from trunk to every tenant safely, prove what is running everywhere, and roll back in one step.

## The release bundle

A release is more than code. Everything that changes behaviour is in the bundle and hashed into the release id (SB-503).

```mermaid
flowchart LR
    CODE["service images<br/>(digests)"] --> BUN["release bundle<br/>rel_2027-03-02_9b1c4e"]
    PROMPTS["prompts/*.md<br/>(hashes)"] --> BUN
    TOOLS["tool schemas"] --> BUN
    POL["platform policies<br/>(Cedar)"] --> BUN
    MODEL["model id + params"] --> BUN
    EVAL["eval results"] --> BUN
    SBOM["SBOM (SPDX)"] --> BUN
    BUN --> SIG["signature"]
```

## Pipeline

```mermaid
flowchart LR
    T["trunk commit"] --> RB["hermetic build<br/>lockfile regenerated"]
    RB --> RB2["independent rebuild"]
    RB2 --> EQ{"digests equal?"}
    EQ -- no --> FAIL["fail: not reproducible"]
    EQ -- yes --> SIGN["sign + SBOM"]
    SIGN --> GATE["eval gate"]
    GATE --> R0["ring 0: internal tenant"]
    R0 --> R1["ring 1: friendly tenants<br/>(channel: testing)"]
    R1 --> PROMO{"promote guard"}
    PROMO --> R2["ring 2: all (channel: stable)"]
```

## Channels and the promote guard

Tenants pick `channel: stable` or `channel: testing`, never a version (ADR-0004).

The **promote guard**: a release can become the new stable only when every stable-channel tenant is already running the current stable. So after promotion, nobody is more than one release behind (LAW-08). Without the guard, a slow tenant could fall two behind; [`Rollout.tla`](../../proofs/tla/Rollout.tla) finds that counterexample.

```mermaid
stateDiagram-v2
    direction LR
    [*] --> Built
    Built --> Gated: evals pass
    Gated --> Testing: ring 0 + ring 1 deploy
    Testing --> Stable: promote guard holds
    Stable --> Superseded: next promotion
    Testing --> Revoked: rollback
    Stable --> Revoked: rollback
    Superseded --> [*]
    Revoked --> [*]
```

## Rollback

`sb release rollback` makes the previous stable current and marks the bad release `Revoked` at once. Cells still running it are **draining**: they start no new consequential tool calls (through the kill-switch path) until they've been redeployed, one by one. Target: every cell redeployed in under 10 minutes (SB-508). Revoked releases can never be redeployed. Modelled in `Rollout.tla` (BUG-0001).

## Pull, not push

Cells **pull** signed bundles from the release train and **push** attestations back. The factory never needs inbound access to a cell. That makes deploys simpler to secure, and it means a cell in a company's own cloud account (`custody: tenant`, ADR-0015) stays on the train with no new mechanism.

## Proving what runs

| Check | When | Law |
|---|---|---|
| Signature + SBOM verified before a container starts | Every deploy | LAW-06 |
| Running digests reported by each cell | Hourly and on deploy | LAW-06 |
| Release lag per tenant | Every promotion and hourly | LAW-08 |
| Running config equals applied manifest | Hourly | LAW-16 |

## CVEs

Lockfiles are regenerated every release. A CVE advisory against a dependency triggers a patch release on `testing` within 24 hours, then the normal path. Because nobody is more than one behind, every tenant has the fix within one promotion (SB-505).

## Laws

[LAW-06](../../manual/sb-laws.7.md#law-06) · [LAW-08](../../manual/sb-laws.7.md#law-08) · [LAW-16](../../manual/sb-laws.7.md#law-16).

## Tickets

SB-503 to SB-510. Manual: [`sb-release(7)`](../../manual/sb-release.7.md), [`sb-release(1)`](../../manual/sb-release.1.md).

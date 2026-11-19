---
title: Architecture Overview
summary: Context, containers, cells, layering, and the two request paths that matter.
---

# Architecture overview

Second Brain is a **control plane** shared by all tenants, plus one **cell** per tenant stamped from a single template. The model is a replaceable userspace process; the **policy engine** is the kernel that every read and every tool call passes through.

## Context (C4 level 1)

```mermaid
flowchart TB
    asker["Asker<br/>company staff"]
    op["Operator<br/>company IT"]
    appr["Approver<br/>company leadership"]
    maint["Maintainer<br/>parent-org platform team"]
    sb["Second Brain<br/>cited answers, gated actions"]
    idp["Company IdP<br/>OIDC + SCIM"]
    src["Company sources<br/>docs, chat, tickets, CRM"]
    llm["Model provider<br/>managed API"]
    asker -- asks, drafts --> sb
    appr -- approves --> sb
    op -- manifest, console, kill --> sb
    maint -- releases --> sb
    sb -- sign-in, users, groups --> idp
    sb -- fetch, acl, changes, act --> src
    sb -- prompts with allowed chunks only --> llm
```

## Containers (C4 level 2)

```mermaid
flowchart TB
    subgraph CP["Control plane (shared)"]
        REG["Tenant registry"]
        REL["Release controller<br/>channels + rings"]
        EV["Eval gate"]
        KS["Kill-switch store"]
        ATT["Attestation checker"]
        EDGE["Edge router"]
    end
    subgraph CELL["Tenant cell (one per company, same template)"]
        API["Cell API<br/>ask, actions, admin"]
        POL["Policy engine<br/>Cedar"]
        AG["Agent runtime<br/>+ tool bus"]
        RET["Retriever"]
        CMP["Answer composer"]
        ING["Ingest workers"]
        CON["Connectors"]
        DB[("Cell DB<br/>Postgres")]
        IDX[("Index<br/>pgvector, derived")]
        AUD[("Audit store<br/>append-only")]
        VAULT[("Vault namespace<br/>+ KMS key")]
        Q[["Queue"]]
    end
    EDGE --> API
    REG --> EDGE
    API --> AG
    AG --> RET --> IDX
    AG --> CMP
    AG -- every tool call --> POL
    RET -- candidate chunks --> POL
    POL -- live acl --> CON
    POL --> AUD
    AG --> AUD
    CON --> Q --> ING --> IDX
    CON -.-> VAULT
    API --> DB
    KS -. flags .-> AG
    REL -- signed bundle --> CELL
    EV --> REL
    ATT -. hourly digests .-> CELL
```

**Why cells.** A shared index is one bug away from a cross-company leak. Cells make isolation a property of the template, not of every query being right (ADR-0001). The control plane stores no tenant content: only routing, releases, flags and attestations.

**The factory.** The control plane plus the cell template and release train is the *factory*. Every cell carries a custody record, so a cell can later move to its company's own cloud account without being rebuilt ([16 Factory and hand-off](16-factory-and-handoff.md), ADR-0015).

## The ask path

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant A as Agent (ask)
    participant R as Retriever
    participant P as Policy
    participant S as Source (connector acl)
    participant M as Model
    participant C as Composer
    participant L as Audit
    U->>A: question (token carries tenant + user)
    A->>L: trace opened
    A->>R: retrieve(scope, query)
    R-->>A: candidates (tenant index, ACL prefilter)
    A->>P: authorise(user, read, chunks)
    P->>S: live acl(user, docs) batched
    S-->>P: allow / deny / unknown per doc
    P-->>A: allowed chunks only (unknown dropped)
    P->>L: decisions with reasons
    A->>M: prompt (pinned) + allowed chunks as quoted data
    M-->>A: draft answer with proposed citations
    A->>C: compose(draft, allowed chunks)
    C-->>A: claims mapped to doc@version, rest labelled uncited
    A-->>U: answer + citations + trace id
```

The model never sees a chunk that failed step 6. Citations the model proposes that don't match an allowed chunk are stripped in step 11 (LAW-01, LAW-14).

## The action path

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant A as Agent
    participant B as Tool bus
    participant K as Kill switches
    participant P as Policy
    participant L as Audit
    participant AP as Approver
    participant X as Executor
    participant T as Target (connector act)
    U->>A: "update the ticket with this summary"
    A->>B: propose tool call (payload)
    B->>K: killed? (agent, connector, tenant)
    K-->>B: no
    B->>P: authorise(user, act, target)
    P-->>B: allow
    B->>L: audit row (Drafted to PendingApproval, payload hash)
    B-->>AP: approval request (exact payload, target, expiry)
    AP->>B: approve (apr_ id bound to hash)
    B->>K: killed? (re-check)
    B->>L: audit row (Executing)
    B->>X: execute(apr_ id, payload)
    X->>T: act(payload, idempotency key)
    T-->>X: receipt
    X->>L: audit row (Ran, receipt)
    X-->>U: done, with receipt link
```

No receipt means `DidNotRun`, shown plainly to the user (LAW-05).

## Layering and the dependency rule

```mermaid
flowchart TB
    subgraph Entry["cmd/ (binaries)"]
        CLI["cmd/sb"]
        CAPI["cmd/cell-api"]
        CING["cmd/ingestd"]
        CCTL["cmd/controller"]
    end
    subgraph Adapters["internal/adapters/ (vendors live here)"]
        AC["connectors/*"]
        AI["idp/oidc, idp/scim"]
        AM["model/*"]
        AS["store/postgres, store/pgvector"]
        AV["vault/*, queue/*"]
    end
    subgraph Core["internal/core/ (no vendor imports)"]
        PORTS["ports: Connector, IdentityProvider, Model, Store, Index, Secrets, Queue, Clock"]
        DOM["policy · tenancy · retrieve · compose · agent · toolbus · approvals · actions · audit · kill · manifest · release"]
    end
    Entry --> Adapters
    Entry --> Core
    Adapters --> PORTS
    DOM --> PORTS
```

| Package may import | `core/ports` | `core/*` | `adapters/*` | vendor SDKs |
|---|---|---|---|---|
| `internal/core/ports` | — | only `core/tenancy` (the `Scope` type every port takes, SB-108) | no | no |
| `internal/core/*` | yes | yes (acyclic) | **no** | **no** |
| `internal/adapters/*` | yes | yes | no (siblings) | yes |
| `cmd/*` | yes | yes | yes | yes |

CI enforces this table with `depguard` (SB-008). A new vendor is a new folder in `internal/adapters/`, never an import in the core.

## Technology choices

| Concern | Choice | ADR |
|---|---|---|
| Services, CLI, policy host | Go | [ADR-0007](../adr/ADR-0007-language-and-stack.md) |
| Policy language | Cedar | [ADR-0008](../adr/ADR-0008-policy-language-cedar.md) |
| Evals and data tooling | Python | ADR-0007 |
| Web console and chat UI | TypeScript, server-rendered, minimal JS | ADR-0007 |
| Storage | Managed Postgres + pgvector per cell | [ADR-0006](../adr/ADR-0006-build-vs-buy.md) |
| Runtime | Managed containers, no Kubernetes | [ADR-0011](../adr/ADR-0011-no-kubernetes-in-v1.md) |
| Formal models | TLA+ and Alloy | [ADR-0010](../adr/ADR-0010-formal-methods-from-wave-0.md) |

## Subsystem documents

| # | Subsystem | Laws |
|---|---|---|
| 01 | [Identity and tenancy](01-identity-and-tenancy.md) | LAW-02, LAW-12 |
| 02 | [Policy engine](02-policy-engine.md) | LAW-01, LAW-03 |
| 03 | [Connectors](03-connectors.md) | LAW-01, LAW-03, LAW-05 |
| 04 | [Ingest pipeline](04-ingest-pipeline.md) | LAW-09, LAW-13 |
| 05 | [Storage and index](05-storage-and-index.md) | LAW-02, LAW-13 |
| 06 | [Retrieval and authorisation](06-retrieval-and-authorisation.md) | LAW-01, LAW-02, LAW-03 |
| 07 | [Answer composer and citations](07-answer-composer.md) | LAW-01, LAW-14 |
| 08 | [Agent runtime and tool bus](08-agent-runtime-and-tool-bus.md) | LAW-04, LAW-09, LAW-10 |
| 09 | [Approvals and actions](09-approvals-and-actions.md) | LAW-04, LAW-05, LAW-15 |
| 10 | [Audit log](10-audit-log.md) | LAW-04, LAW-07, LAW-11 |
| 11 | [Kill switches](11-kill-switches.md) | LAW-10 |
| 12 | [Evals and the gate](12-evals-and-gate.md) | LAW-06, LAW-09, LAW-14 |
| 13 | [Release and supply chain](13-release-and-supply-chain.md) | LAW-06, LAW-08 |
| 14 | [Manifest, CLI and console](14-manifest-cli-console.md) | LAW-16 |
| 15 | [Observability, backup and DR](15-observability-backup-dr.md) | LAW-11, LAW-13 |
| 16 | [Factory and hand-off](16-factory-and-handoff.md) | LAW-17, LAW-18 |

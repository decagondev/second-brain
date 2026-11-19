---
title: "ADR-0018: Cells are cloud-neutral; each cloud is an adapter behind the Placement and Keys ports"
summary: "Cells are cloud-neutral; a local adapter first, then AWS, GCP and Azure adapters behind the same ports."
---

# ADR-0018: Cells are cloud-neutral; each cloud is an adapter behind the Placement and Keys ports

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-19 | Platform lead (the user) |

## Context

SB-103 needs a cell template, and open question Q-04 asked which cloud and which infrastructure-as-code tool to use. Portfolio companies may end up on different clouds, and ADR-0015 already requires that placement is a parameter behind a port, with no hard-coded account ids. Picking one cloud now would bake its resource model into the core.

## Decision

- The cell template is declared once, in the core (`internal/core/cell`), as cloud-neutral resources: database, index schema, vault namespace, root key, queue, all pinned to the manifest's region.
- Two ports carry it out. **`Placement`** creates or finds a cell's resources in one account; **`Keys`** holds the cell root key and wraps data keys (envelope encryption, so the root can be re-wrapped at hand-off, LAW-18). Ports stay small (one job each), and the core depends only on them (dependency inversion, ADR-0006 and SB-008).
- **W1 ships a `local` adapter** (`internal/adapters/placement/local`) for development and tests. It also stamps cells into two different mock accounts to prove that no account id is hard-coded.
- **AWS, GCP and Azure adapters come later**, one folder each under `internal/adapters/`, and each passes the same port contract tests. A new cloud is a new adapter, never a change to the core.
- The IaC tool is chosen per cloud adapter when it is built; the template doesn't depend on it. This answers the cloud half of Q-04. The tool half is deferred to the first cloud adapter.

## Alternatives rejected

| Option | Why not |
|---|---|
| Pick one cloud now | Locks portfolio companies to one vendor and leaks that vendor's model into the core |
| One multi-cloud IaC codebase as the template | The template would be IaC text that can't be unit-tested in Go, and isolation would become a property of discipline, not of the template |
| Wait for the decision before building SB-103 | Blocks all of W1; the local adapter proves isolation and placement now |

## Consequences

- SB-103 and SB-104 can be built and tested end to end against `local`. Real-cloud acceptance (for example "new cell in under 30 minutes") stays *In review* until a cloud adapter exists.
- Every adapter must pass one shared contract test: idempotent ensure, a stable id per resource, no resource shared between cells, every resource carrying its cell's region, and "unknown" reported as an error, never as "absent".
- Three cloud adapters is more work than one. We accept that so that cloud choice stays reversible per company.

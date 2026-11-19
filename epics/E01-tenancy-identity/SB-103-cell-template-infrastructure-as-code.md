---
title: "SB-103: Cell template (infrastructure as code)"
summary: "Isolation is a property of the template, not of discipline."
---

# SB-103: Cell template (infrastructure as code)

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.2 Cell |
| Stories | US-01.1 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-101](../E01-tenancy-identity/SB-101-manifest-schema-parser-and-sb-tenant-check.md) |
| Status | In review |

## Why

Isolation is a property of the template, not of discipline.

## Acceptance criteria

- [x] Template creates database, index schema, vault namespace, KMS key, queue and region pin
- [x] No resource is shared between two cells except the control plane
- [x] Template has a test that stamps two cells and checks no shared resource ids
- [x] Placement is a template parameter behind the `Placement` port; no account id is hard-coded; a test stamps cells into two different mock accounts (ADR-0015)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Write IaC module `infra/cell/`: deferred to the first cloud adapter, which picks its IaC tool (ADR-0018)
- [x] Parameterise only by manifest fields
- [x] Isolation test stamping two cells
- [x] Document cell anatomy in `sb-tenancy(7)`
- [x] Envelope key hierarchy: cell root key in KMS wraps data keys, so the root can be re-wrapped later
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: the user decided the cloud question: cells are cloud-neutral, with a local adapter now and AWS, GCP and Azure adapters later ([ADR-0018](../../docs/adr/ADR-0018-cloud-neutral-cells-behind-ports.md)). Built against that decision:
- `internal/core/cell` declares the template (database, index schema, vault namespace, root key, queue), taking only `tenant` and `region` from the manifest. `Stamp` makes it real through the new `ports.Placement` and `ports.Keys`, and it is idempotent.
- `NewDataKey` gives envelope encryption: data keys are wrapped under the cell root key.
- `internal/core/cell/celltest` is the contract every adapter must pass: idempotent ids, no shared ids in the same or different accounts, and a data key that another cell's root key can't unwrap.
- `internal/adapters/placement/local` passes the contract and refuses unreadable records ("unknown" is not "absent"), identity mismatches and unsafe names.
- No account id appears anywhere. Cells are stamped into the mock accounts `acct-a` to `acct-d`.

LAW-02 gains a cell-level law test, `TestLAW02_CellsShareNothing`, which was red first (`not implemented`) and is green now. The red ratchet takes per-test lines so the retrieval-level test can stay red until SB-301.

**Left:** the IaC module and real-cloud stamping arrive with the first cloud adapter (ADR-0018). Also left: review by the user, (first CI run green on `c0a12be` (all jobs: check, proofs, postgres, diagrams)).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

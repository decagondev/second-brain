---
title: "SB-703: Placement adapter for tenant-owned cloud accounts"
summary: "Stamp a cell into a company's own account with least privilege."
---

# SB-703: Placement adapter for tenant-owned cloud accounts

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.1 Custody model |
| Stories | US-08.2 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-114](../E08-factory-handoff/SB-114-factory-seams-custody-record-placement-port.md), [SB-702](../E08-factory-handoff/SB-702-factory-roles-and-policies.md) |
| Status | In review |

## Why

Stamp a cell into a company's own account with least privilege.

## Acceptance criteria

- [ ] Cell stamped into a tenant account via a scoped deploy role the tenant creates (against the recorded fake, with the deploy role's exact permissions; a run in a real GCP project needs the user's project and credentials)
- [x] Same template; no tenant-specific code

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Tenant-account adapter behind `Placement`
- [x] Bootstrap script for the tenant's deploy role
- [x] Two-account test
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.1c): sb-tenancy(7) gained "Placement on GCP, exactly". `internal/adapters/placement/gcp` implements `Placement` and `Keys` over Google's REST APIs with the standard library only, so no new module was needed. The template's resources map to a Cloud SQL instance and database, the `idx` schema made by migrations, a Secret Manager prefix with a marker secret, a Cloud KMS key ring and key, and a Pub/Sub topic, all in the region's one location (`eu` is `europe-west1`, `us` is `us-central1`). Every resource is labelled with its tenant, kind and location, and one with another identity is refused. Only a 404 means absent. `Wrap` and `Unwrap` are KMS encrypt and decrypt, with the key's name as AAD. `deploy/gcp/bootstrap-deploy-role.sh` creates the custom role `sbCellDeployer`, granting exactly `gcp.DeployPermissions` (a test compares the two), and a service account only the factory may impersonate. `gcptest` is a recorded fake that enforces one IAM permission per call and per-project tokens. The adapter passes the shared placement contract (`celltest`). `TestTwoAccounts` shows one tenant's deployer can't create, read or decrypt in another's project. `TestEveryPermissionIsNeeded` removes each of the 14 permissions in turn and shows the stamp, wrap or unwrap then fails. The new law test `TestLAW02_GCPCellsShareNothing` runs the cell-isolation law over GCP projects.

**Left:** a run in a real GCP project with the user's project and credentials.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

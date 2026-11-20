---
title: "SB-901: Cloud Run services and jobs per cell"
summary: "A cell needs to run somewhere real; Cloud Run gives a managed runtime without Kubernetes (ADR-0011)."
---

# SB-901: Cloud Run services and jobs per cell

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.1 Run a cell |
| Stories | US-09.1 |
| Manual | [`sb-tenancy(7)`](../../manual/sb-tenancy.7.md), `sb-runtime(7)` (new) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-703](../E08-factory-handoff/SB-703-placement-adapter-for-tenant-owned-cloud.md) |
| Status | In review |

## Why

A cell needs to run somewhere real; Cloud Run gives a managed runtime without Kubernetes (ADR-0011).

## Acceptance criteria

- [ ] `api`, `ingest` and `web` services and the job runner deploy into the cell's project, each with its own service account and least privilege (against the recorded fake; granting the roles in a real project is SB-903)
- [x] Only images the release controller verified are admitted (Binary Authorization), and an unsigned image is refused (LAW-06)
- [x] The deploy is pulled by the cell, never pushed by the factory (ADR-0015)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `sb-runtime(7)` first
- [x] Runtime port and Cloud Run adapter
- [x] Admission policy
- [x] Contract test against a recorded fake
- [x] Write or extend the law tests: LAW-02, LAW-06
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S9.1a): the new man page `sb-runtime(7)` has the services, their accounts and roles, the pulled update, and Binary Authorization as a second guard. It also drafts the SB-902 data section. `internal/core/runtime` has the services and the `Updater`, which reads the channel's target release, admits it with the four checks of sb-release(7), and changes nothing if it fails. It then attests each image and deploys every service by digest, never by tag, labelled with the release id. `gcp.CloudRun` is the `Deployer` over the Cloud Run Admin v2 and Container Analysis APIs, and `gcp.ServiceRoles` and `gcp.UpdaterPermissions` define the accounts' rights. The GCP fake now runs Cloud Run services and jobs with a Binary Authorization policy: only images the project attested, and only service accounts from the same project. The new law test `TestLAW06_CellsRunOnlyAdmittedImages` deploys random signed, unsigned and ungated releases and shows services only ever run admitted images, and that a sideloaded image is refused. A self-test catches an updater that skips admission. `TestLAW02_RuntimeStaysInItsProject` shows every account is the cell's own and that one cell's updater can't deploy into another's project. Building the law world surfaced BUG-0013: the law tests' release stores leaked into `/tmp` until its inodes ran out. They are now removed when the tests end.

**Left:** deploying into a real GCP project (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-1601: Container images for every service"
summary: "Nothing builds the cell's services as images yet: GCP's Cloud Run and a self-hosted box both need them, pinned by digest."
---

# SB-1601: Container images for every service

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.1 |
| Stories | US-16.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06, LAW-08 |
| Depends on | — |
| Status | Open |

## Why

Nothing builds the cell's services as images yet: GCP's Cloud Run and a self-hosted box both need them, pinned by digest.

## Acceptance criteria

- [ ] One image per service (`api`, `ingest`, `jobs`, `updater`), static Go on a distroless base pinned by digest, non-root, no shell
- [ ] `make images` builds them reproducibly and the release bundle lists their digests (`image/api`, ...); the SBOM covers them
- [ ] A smoke test starts each image and finds `/healthz` (or the job's exit 0)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Dockerfiles and make images
- [ ] Bundle digests and SBOM
- [ ] Image smoke test
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

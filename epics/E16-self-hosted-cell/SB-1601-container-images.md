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
| Status | Done |

## Why

Nothing builds the cell's services as images yet: GCP's Cloud Run and a self-hosted box both need them, pinned by digest.

## Acceptance criteria

- [x] One image per service (`api`, `ingest`, `jobs`, `updater`), static Go on a distroless base pinned by digest, non-root, no shell
- [x] `make images` builds them reproducibly and the release bundle lists their digests (`image/api`, ...); the SBOM covers them
- [x] A smoke test starts each image and finds `/healthz` (or the job's exit 0)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Dockerfiles and make images
- [x] Bundle digests and SBOM
- [x] Image smoke test
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S15.1): `make images` (`tools/selfhost/images.sh`, `deploy/images/Dockerfile`) builds `sb-api`, `sb-ingest`, `sb-jobs` and `sb-updater` (and the demo IdP) as static Go binaries with the reproducible build's flags and puts each on `gcr.io/distroless/static-debian12` pinned by digest, as user 65532, with no shell; the `api` image carries `web/dist`. The images are 16 to 25 MB. A rebuild of the same commit gives the same image digests: files and the image's dates are set to the commit's time (`SOURCE_DATE_EPOCH`) and BuildKit's provenance attestation, which carries the build's own time, is off. Each digest goes to `build/images/digests.txt` and each binary gets its own SPDX SBOM; `tools/release/bundle --image NAME=sha256:…` records them as `image/NAME` (`TestImages`). `tools/selfhost/smoke.sh` starts every image as part of the self-hosted cell and checks each service is up, non-root and read-only.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

---
title: "SB-906: Firebase Hosting for the web UI"
summary: "Each company should reach its brain at its own address, served close to its people."
---

# SB-906: Firebase Hosting for the web UI

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.3 Edge and identity |
| Stories | US-09.3 |
| Manual | `sb-runtime(7)` (new), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-901](SB-901-cloud-run-services-per-cell.md) |
| Status | In review |

## Why

Each company should reach its brain at its own address, served close to its people.

## Acceptance criteria

- [ ] One Hosting site per tenant serving the built `web/`, rewriting `/v1/*` to that cell's Cloud Run `api` only (the config and its release against the fake; uploading the built files to a real site is SB-903)
- [x] A site can never rewrite to another tenant's cell (tested)
- [x] Optional: a tenant may serve the UI from its own host instead

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Hosting adapter
- [x] Rewrite config generated from the manifest
- [x] LAW-02 test on rewrites
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S9.3a): sb-runtime(7) gained "Web UI hosting, exactly", and sb-ask(3) points to it. `internal/core/hosting` generates the one allowed configuration for a tenant's site, `sb-TENANT` in the cell's own project. It rewrites `/v1/**` to `api` in the cell's region, serves the app's `index.html` for everything else, and sends the CSP of sb-ask(3). `hosting.Check` refuses another project or site name, another service or region, a function, an absolute or scheme-less URL, a missing api rewrite, and the api on anything but `/v1/**`; nine cases are tested. `gcp.Hosting` publishes through the Firebase Hosting API (create version, finalize, release) and never publishes a config `Check` refuses. The GCP fake serves Hosting and, like the real service, refuses a rewrite to a Cloud Run service outside the site's project. The new law test `TestLAW02_SitesReachOnlyTheirOwnCell` tries four ways for one tenant to point its site at the other's cell and finds none released, and a self-test catches a publisher that skips the check. A tenant's own host is documented as the alternative.

**Left:** uploading the built `web/` files to a real site (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

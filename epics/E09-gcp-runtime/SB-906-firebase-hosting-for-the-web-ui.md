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
| Status | Open |

## Why

Each company should reach its brain at its own address, served close to its people.

## Acceptance criteria

- [ ] One Hosting site per tenant serving the built `web/`, rewriting `/v1/*` to that cell's Cloud Run `api` only
- [ ] A site can never rewrite to another tenant's cell (tested)
- [ ] Optional: a tenant may serve the UI from its own host instead

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Hosting adapter
- [ ] Rewrite config generated from the manifest
- [ ] LAW-02 test on rewrites
- [ ] Write or extend the law tests: LAW-02
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

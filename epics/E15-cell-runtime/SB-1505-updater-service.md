---
title: "SB-1505: The updater job: releases pulled and admitted"
summary: "The updater's logic exists against the GCP fake; it needs a binary and a local deployer."
---

# SB-1505: The `updater` job: releases pulled and admitted

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.3 |
| Stories | US-15.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06, LAW-08 |
| Depends on | [SB-1502](SB-1502-api-service.md) |
| Status | Open |

## Why

The updater's logic exists against the GCP fake; it needs a binary and a local deployer.

## Acceptance criteria

- [ ] `cmd/sb-updater` reads the channel, admits the bundle, attests and deploys each service by digest, local or GCP
- [ ] An unsigned or ungated bundle changes nothing and raises a Sev-1

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Binary
- [ ] Local deployer
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

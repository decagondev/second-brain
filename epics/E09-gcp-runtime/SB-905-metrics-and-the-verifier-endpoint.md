---
title: "SB-905: /metrics and the verifier endpoint"
summary: "Operators and `sb tenant verify` need to read a running cell."
---

# SB-905: `/metrics` and the verifier endpoint

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.2 Runtime jobs |
| Stories | US-09.2 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md), [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-901](SB-901-cloud-run-services-per-cell.md) |
| Status | Open |

## Why

Operators and `sb tenant verify` need to read a running cell.

## Acceptance criteria

- [ ] `/metrics` in Prometheus text format, scraped by Managed Service for Prometheus, with Sev-1 rules wired to the webhook adapter
- [ ] The verifier endpoint serves the running-build attestation (SB-509) and answers `sb tenant verify`
- [ ] Neither endpoint returns content, prompts or secrets (LAW-07)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Metrics endpoint
- [ ] Verifier endpoint
- [ ] Alert rules
- [ ] Close the runtime items on SB-509, SB-601, SB-607
- [ ] Write or extend the law tests: LAW-06, LAW-17
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

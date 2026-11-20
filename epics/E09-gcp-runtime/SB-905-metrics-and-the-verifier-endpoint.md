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
| Status | In review |

## Why

Operators and `sb tenant verify` need to read a running cell.

## Acceptance criteria

- [ ] `/metrics` in Prometheus text format, scraped by Managed Service for Prometheus, with Sev-1 rules wired to the webhook adapter (the endpoint and `alerts.Dispatch` are built; the scrape config in a real project is SB-903)
- [x] The verifier endpoint serves the running-build attestation (SB-509) and answers `sb tenant verify`
- [x] Neither endpoint returns content, prompts or secrets (LAW-07)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Metrics endpoint
- [x] Verifier endpoint
- [x] Alert rules
- [ ] Close the runtime items on SB-509, SB-601, SB-607 (when a real cell answers, SB-903 and SB-908)
- [x] Write or extend the law tests: LAW-06, LAW-17
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S9.2b): sb-runtime(7) gained "Endpoints, exactly", sb-tenant(1) `--cell`, and sb-observability(7) the dispatch. `internal/api.Verifier` serves `GET /metrics` (the `obs` registry, so labels obey the telemetry rule), `GET /v1/verifier/attestation` and `POST /v1/verifier/verify`. The two verifier endpoints accept only the control plane's identity (`EAUTH` without one, `EPERM` for anyone else), and an unknown law is `EINVAL`. `runtime.Attestor` signs what Cloud Run actually runs, read at the moment of the call. Its sequence numbers are milliseconds and always increase, so a restarted updater never replays one. `verify` gained LAW-17 through an optional `AccessTarget`: a fail if any path is unlisted, `unknown` if enumeration fails. `alerts.Dispatch` sends each firing Sev-1 rule through the alert sender with the law and runbook. `sb tenant verify FILE --cell URL` posts to the verifier with `SB_TOKEN`, accepts only https (or a loopback test server), and exits 1 unless every law passes. Two new law tests: `TestLAW06_AttestationReportsWhatRuns`, where a half-applied update is an attestation mismatch and a mutation that attests intent is caught, and `TestLAW17_VerifyReportsAccessPaths`, where a mutation that counts unknown as a pass is caught.

**Left:** the Managed Prometheus scrape and a real cell answering verify (SB-903, SB-908); closing the runtime items on SB-509, SB-601 and SB-607.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

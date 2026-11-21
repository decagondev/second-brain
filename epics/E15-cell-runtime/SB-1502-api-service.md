---
title: "SB-1502: The api service: every endpoint, mounted"
summary: "The handlers exist one by one; the cell needs them served together behind one sign-in."
---

# SB-1502: The `api` service: every endpoint, mounted

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.1 |
| Stories | US-15.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-01, LAW-02, LAW-04, LAW-12 |
| Depends on | [SB-1501](SB-1501-cell-configuration-and-wiring.md) |
| Status | Done |

## Why

The handlers exist one by one; the cell needs them served together behind one sign-in.

## Acceptance criteria

- [x] `cmd/sb-api` serves ask, drafts, actions, admin, stewards, access reviews, legal holds, spaces, SCIM, the chat app, the web UI, `/metrics` and the verifier, behind the auth middleware
- [x] Health and readiness endpoints; graceful shutdown that finishes in-flight requests and never half-writes an audit row
- [x] `tools/check/manual_code.py` sees every route the binary mounts, and a smoke test drives one request of each kind against a local cell

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Binary and routes
- [x] Health and shutdown
- [x] Smoke test
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.1): `cmd/sb-api --config FILE` builds the cell with `internal/cell/wire` and serves it; `make build` writes `bin/sb-api`. Public routes: `GET /healthz`, `GET /readyz`, `GET /login`, `GET /callback`, SCIM, `/metrics`, the verifier (guarded: `EPERM` to everyone until the control plane's identity is wired, SB-1505), the Slack chat app and Events API when the chat connector is Slack, and the web UI when `web` is set. Everything under `/v1` is behind the auth middleware. On SIGTERM readiness goes first, then `http.Server.Shutdown` finishes the requests in flight (`cmd/sb-api`, `TestGracefulShutdown`: a request whose body arrives after the signal is still answered). `tests/cell` signs a SCIM-provisioned user in through the test IdP and sends one request to each family of routes; nothing answers 5xx and nothing behind sign-in answers without it. `manual_code.py` now reads `/readyz` and `/connectors/...` routes too, and found four undocumented routes, now in sb-identity(7), sb-connector(3) and sb-api(3).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

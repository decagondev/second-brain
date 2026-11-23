---
title: "SB-1806: Outbound proxy"
summary: "Most company networks reach the internet only through a proxy, and the cell has no setting for one."
---

# SB-1806: Outbound proxy

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.3 |
| Stories | US-18.3 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Done |

## Why

Most company networks reach the internet only through a proxy, and the cell has no setting for one.

## Acceptance criteria

- [x] `SB_HTTPS_PROXY` and `SB_NO_PROXY` in install.env reach `api`, `ingest`, `jobs` and `updater` only; `postgres`, `model` and the data network stay without a route out
- [x] A proxy URL with credentials is refused; credentials go in a file (`SB_HTTPS_PROXY_FILE`) and are never logged
- [x] `--check` reports the proxy setting

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] install.env and compose
- [x] selfhost.sh
- [x] Docs
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.3): `SB_HTTPS_PROXY` reaches `api`, `ingest`, `jobs` and the updater only (`x-cell`'s environment in compose: `HTTPS_PROXY`, `NO_PROXY`, `SB_HTTPS_PROXY_FILE`); `postgres`, the model profile and `worm` are on the `data` network and get none. A URL carrying credentials is refused (`box.CheckProxy`, `sb selfhost check-proxy`) and the refusal never repeats the password; `SB_HTTPS_PROXY_FILE` holds such a URL, copied into the four vaults as `https-proxy`, and each service's `main` reads it before any HTTP client (`wire.ProxyFromFile`), so it is never in a container's environment. The cell's own names are always in `NO_PROXY`; `SB_NO_PROXY` adds the company's. `install.sh --check` reports the proxy. A changed vault recreates the services on the next install. Run on the demo box: with a proxy file the services started and answered, `docker inspect` and every log held no part of the credentials, and clearing it removed the file from the vaults. Tests: `TestCheckProxy`, `TestProxyFromFile`, `TestSelfhostCheckProxyAndDSN`, `TestLAW07_ProxyCredentialsStayInAFile` with a self-test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

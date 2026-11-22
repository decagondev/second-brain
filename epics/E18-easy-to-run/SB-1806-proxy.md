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
| Status | Open |

## Why

Most company networks reach the internet only through a proxy, and the cell has no setting for one.

## Acceptance criteria

- [ ] `SB_HTTPS_PROXY` and `SB_NO_PROXY` in install.env reach `api`, `ingest`, `jobs` and `updater` only; `postgres`, `model` and the data network stay without a route out
- [ ] A proxy URL with credentials is refused; credentials go in a file (`SB_HTTPS_PROXY_FILE`) and are never logged
- [ ] `--check` reports the proxy setting

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] install.env and compose
- [ ] selfhost.sh
- [ ] Docs
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

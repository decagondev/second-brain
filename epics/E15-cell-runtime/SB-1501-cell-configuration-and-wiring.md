---
title: "SB-1501: Cell configuration and one composition root"
summary: "Every part of a cell is built and tested, but nothing builds the whole cell from one configuration."
---

# SB-1501: Cell configuration and one composition root

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.1 |
| Stories | US-15.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md) (new), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-02, LAW-03, LAW-16 |
| Depends on | — |
| Status | Done |

## Why

Every part of a cell is built and tested, but nothing builds the whole cell from one configuration.

## Acceptance criteria

- [x] A cell's configuration (`sb-cell(5)`: tenant, manifest, state, vault, policies, listen address, sign-in, model, connector overrides) is parsed strictly; unknown fields, missing fields and wrong types are errors
- [x] `internal/cell/wire` builds every port and service of a cell from it in one place, with nothing built twice. Local adapters today; the Postgres stores are wired with SB-1508 (`make cell` against local Postgres), and until then `durable: true` refuses to start
- [x] A missing or unreadable dependency stops the cell from starting; it never runs with a part switched off

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Configuration format and parser
- [x] Composition root
- [x] Start-up checks
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.1): `internal/cell/wire/config.go` parses sb-cell(5) strictly (unknown fields, missing required fields, wrong types, a listen address that isn't `host:port`, a URL that isn't http(s), an unknown model provider, a gateway provider without `model.base` and `model.id`). `wire.Build` builds the whole cell once: the manifest (checked, and for this tenant), the Cedar policies, audit, kill switches, logs, the directory, sessions, spaces (synced from the manifest, held spaces kept), sign-in (the IdP's discovery and keys fetched at start), SCIM, connectors by vendor with vault credentials, the model (extractive, or one of ADR-0019's gateways), retrieval, the ask and draft agents, the tool bus with barriers and reporting lines, retention with legal holds read when they're needed, and every API handler. The cell's tenancy Issuer is made by `auth.WithIssuer`, so only `internal/auth` makes one (SB-108). It refuses to start, with `ErrConfig` and the reason, on `durable: true` while stores are in memory, another tenant's manifest, missing policies or state, an unreachable IdP, an unknown connector vendor, or Slack chat without `chat.team_id` (`tests/cell`, `TestRefusesToStart`). The usage reports answer `EUNAVAIL` until the jobs service keeps usage (SB-1504), never made-up numbers.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).

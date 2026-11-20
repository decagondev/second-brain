---
title: "E15 Cell Runtime"
summary: "Every part of a cell runs together as real services: one configuration, the api, ingest, jobs and updater, apply against a running cell, and a whole cell end to end."
---

# E15: Cell Runtime

> Every part of a cell runs together as real services: one configuration, the api, ingest, jobs and updater, apply against a running cell, and a whole cell end to end.

**Waves:** W14 · **Tickets:** 9 · **Stories:** 1

Planned 2026-11-20 with the user, after W13 hardening: the last piece between "built and tested" and "deployable".

## User stories

- **US-15.1**: As a *platform owner*, I want a cell to run as real services from one configuration, so that the next step is deploying it, not assembling it. Delivered by every ticket below.

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1501](SB-1501-cell-configuration-and-wiring.md) | W14 / S14.1 | Cell configuration and one composition root | LAW-02, LAW-03, LAW-16 |
| [SB-1502](SB-1502-api-service.md) | W14 / S14.1 | The `api` service: every endpoint, mounted | LAW-01, LAW-02, LAW-04, LAW-12 |
| [SB-1503](SB-1503-ingest-service.md) | W14 / S14.2 | The `ingest` service: syncs that run | LAW-07, LAW-13 |
| [SB-1504](SB-1504-jobs-service.md) | W14 / S14.2 | The `jobs` service: every schedule, run | LAW-11, LAW-13, LAW-16, LAW-17 |
| [SB-1505](SB-1505-updater-service.md) | W14 / S14.3 | The `updater` job: releases pulled and admitted | LAW-06, LAW-08 |
| [SB-1506](SB-1506-apply-to-a-running-cell.md) | W14 / S14.3 | `sb tenant apply` against a running cell | LAW-16 |
| [SB-1507](SB-1507-cli-against-running-cells.md) | W14 / S14.4 | The CLI commands that need a running cell | LAW-11, LAW-17, LAW-18 |
| [SB-1508](SB-1508-local-cell-end-to-end.md) | W14 / S14.4 | A whole cell running locally, end to end | LAW-01 to LAW-22 |
| [SB-1509](SB-1509-console-screens.md) | W14 / S14.5 | Console screens for stewards and access reviews | LAW-15, LAW-16 |

---
title: "E16 Self-hosted Cell"
summary: "A whole cell on one machine the company owns, in containers that mirror GCP: for sovereign data, on-prem sites, demos and testing; plus four easy wins."
---

# E16: Self-hosted Cell

> A whole cell on one machine the company owns, in containers that mirror GCP: for sovereign data, on-prem sites, demos and testing; plus four easy wins.

**Waves:** W15 · **Tickets:** 9 · **Stories:** 1

Planned 2026-11-21 with the user, after the hosting review: the user chose the self-hosted cell and left the easy wins to the plan. The decision is ADR-0022.

## User stories

- **US-16.1**: As a *company that keeps its data on its own hardware*, I want a whole cell on one machine I run, so that nothing leaves my site; and as a *platform owner*, I want the same cell for demos and testing. Delivered by every ticket below.

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1601](SB-1601-container-images.md) | W15 / S15.1 | Container images for every service | LAW-06, LAW-08 |
| [SB-1602](SB-1602-compose-layout.md) | W15 / S15.1 | A self-hosted cell in containers that mirror GCP | LAW-02, LAW-07, LAW-16 |
| [SB-1603](SB-1603-object-locked-worm.md) | W15 / S15.2 | Write-once audit storage on the box | LAW-11 |
| [SB-1604](SB-1604-upgrades-without-the-socket.md) | W15 / S15.2 | Upgrades without giving the cell the Docker socket | LAW-06 |
| [SB-1605](SB-1605-sovereign-install.md) | W15 / S15.3 | A sovereign install on a VM or an on-prem server | LAW-04, LAW-11, LAW-12, LAW-17, LAW-18 |
| [SB-1606](SB-1606-folder-connector.md) | W15 / S15.3 | A folder connector for file shares and demos | LAW-01, LAW-03, LAW-13 |
| [SB-1607](SB-1607-health-panel.md) | W15 / S15.4 | Cell health in the console | LAW-13, LAW-16 |
| [SB-1608](SB-1608-budget-warning-and-report-delivery.md) | W15 / S15.4 | Budget warning at 80% and the weekly report delivered | LAW-16 |
| [SB-1609](SB-1609-unhelpful-answer-queue.md) | W15 / S15.4 | A queue of answers people marked unhelpful | LAW-14 |

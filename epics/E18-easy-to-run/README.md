---
title: "E18 Easy to Run"
summary: "A company's IT team runs, upgrades, restores and gets help for a cell without reading the source, on one machine or two VMs."
---

# E18: Easy to Run

> A company's IT team runs, upgrades, restores and gets help for a cell without reading the source, on one machine or two VMs.

**Waves:** W17 · **Tickets:** 8 · **Stories:** 3

Proposed 2026-11-22 after W16 and accepted by the user the same day, with every recommended decision. The concept is in [10-run-protect-close.md](../../docs/10-run-protect-close.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, and no new page.

## User stories

- **US-18.1**: As an *IT administrator*, I want to see in one place what's wrong with the box and how to fix it, so that problems are found before people do. Delivered by SB-1801, SB-1802.
- **US-18.2**: As an *IT administrator*, I want to restore, upgrade offline and roll back with one command each, so that recovering the box doesn't depend on reading a runbook under pressure. Delivered by SB-1803, SB-1804, SB-1805.
- **US-18.3**: As an *IT administrator*, I want the cell to fit my network: a proxy, our own Postgres, and a support bundle I can safely send, so that it runs the way the rest of our systems do. Delivered by SB-1806, SB-1807, SB-1808.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S17.1 | Know it's healthy | `--doctor` and host health in the console | SB-1801, SB-1802 |
| S17.2 | Upgrade and recover | One-command restore and drill, verified offline upgrades, rollback | SB-1803, SB-1804, SB-1805 |
| S17.3 | Fit the network | Outbound proxy, the company's Postgres, a support bundle | SB-1806, SB-1807, SB-1808 |

## Exit gate

- [x] `--doctor` is green on a running box, and each check has a test that makes it fail
- [x] The console shows the box's disk, certificate and backup ages, unknown when it can't read them
- [x] A restore is one command, and the monthly drill proves the newest dump restores
- [x] A tampered or unsigned offline bundle loads nothing; a rollback starts only admitted digests
- [ ] A proxy, an external Postgres and a support bundle work, and the bundle carries no secret
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1801](SB-1801-doctor.md) | W17 / S17.1 | `install.sh --doctor`: check the running box | LAW-03, LAW-07 |
| [SB-1802](SB-1802-host-health.md) | W17 / S17.1 | Host health in the console | LAW-03 |
| [SB-1803](SB-1803-restore.md) | W17 / S17.2 | One-command restore and a monthly restore drill | LAW-11, LAW-13 |
| [SB-1804](SB-1804-offline-upgrade.md) | W17 / S17.2 | Offline upgrade, verified before it's loaded | LAW-06 |
| [SB-1805](SB-1805-rollback.md) | W17 / S17.2 | Roll back the box in one command | LAW-06, LAW-08 |
| [SB-1806](SB-1806-proxy.md) | W17 / S17.3 | Outbound proxy | LAW-07 |
| [SB-1807](SB-1807-external-postgres.md) | W17 / S17.3 | Use the company's Postgres: two VMs | LAW-02 |
| [SB-1808](SB-1808-support-bundle.md) | W17 / S17.3 | A support bundle with no content | LAW-07 |

## Decisions for the user

Decided 2026-11-22 ([10-run-protect-close.md](../../docs/10-run-protect-close.md#decisions-for-the-user)): all of W17 to W19, in order; a 3-of-5 recovery kit on paper (SB-1902); sealing on by default (SB-1901); Mattermost (SB-2005).

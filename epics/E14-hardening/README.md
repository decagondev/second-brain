---
title: "E14 Hardening"
summary: "Before calling it feature complete: review, fuzz, tidy, reconcile the manual with the code, and close coverage gaps."
---

# E14: Hardening

> Before calling it feature complete: review, fuzz, tidy, reconcile the manual with the code, and close coverage gaps.

**Waves:** W13 · **Tickets:** 5 · **Stories:** 1

Planned 2026-11-20 with the user, who kept Q-01 as no: the portfolio grants of [E13](../E13-portfolio-grants/README.md) are parked, and W13 is this hardening wave instead.

## User stories

- **US-14.1**: As the *platform owner*, I want the code reviewed, fuzzed, tidied and matched to its manual before we call it feature complete, so that what we hand to companies is as solid as its laws say. Delivered by every ticket below.

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1401](SB-1401-fresh-eyes-review.md) | W13 / S13.1 | Fresh-eyes review of W8 to W12 | LAW-19, LAW-20, LAW-22 |
| [SB-1402](SB-1402-fuzzing-and-races.md) | W13 / S13.2 | Fuzz the parsers and run every test under the race detector | LAW-03, LAW-07 |
| [SB-1403](SB-1403-clean-and-concise.md) | W13 / S13.3 | Clean and concise: stricter lint and tidier big files | none (no behaviour change) |
| [SB-1404](SB-1404-manual-matches-code.md) | W13 / S13.4 | The manual and the code agree | LAW-16 |
| [SB-1405](SB-1405-coverage-gaps.md) | W13 / S13.5 | Close the coverage gaps in the core | every law a thin package carries |

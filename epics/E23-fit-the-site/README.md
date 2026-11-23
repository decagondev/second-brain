---
title: "E23 Fit the Site"
summary: "The cell speaks the company's language, logs where the company logs, stores where it stores, and stays fair on small hardware."
---

# E23: Fit the Site

> The cell speaks the company's language, logs where the company logs, stores where it stores, and stays fair on small hardware.

**Waves:** W22 · **Tickets:** 7 · **Stories:** 4

Proposed 2026-11-23 after W19 and accepted by the user the same day, with every recommended decision. The concept is in [11-loop-own-fit.md](../../docs/11-loop-own-fit.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, and no new page.

## User stories

- **US-23.1**: As a *person at an EU site* or *security team*, I want the UI in our language and the audit rows in our SIEM, so that the brain fits the way we already work. Delivered by SB-2301, SB-2302.
- **US-23.2**: As an *IT administrator*, I want off-box copies in our own S3 bucket, so that we don't need a NAS. Delivered by SB-2303.
- **US-23.3**: As an *asker* on a CPU box, I want sources at once and a fair share of the model, so that a slow box still feels usable. Delivered by SB-2304, SB-2305.
- **US-23.4**: As an *integrator* or *site without chat*, I want read-only tokens and email notifications, so that scripts and email work without a browser or chat. Delivered by SB-2306, SB-2307.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S22.1 | Language and logs | English, German and French, the SIEM stream, S3 off-box copies | SB-2301, SB-2302, SB-2303 |
| S22.2 | Small boxes | Sources first, asks per minute | SB-2304, SB-2305 |
| S22.3 | Integrations | Read-only API tokens, SMTP notifications | SB-2306, SB-2307 |

## Exit gate

- [x] No UI string outside the catalog; axe in each language
- [x] A fake SIEM gets every row in order across a restart
- [x] The restore drill passes from S3 (against the S3 fake in `TestBackupToAnS3Bucket`; a company bucket waits with SB-1605's real VM)
- [x] A burst of ten asks gets six answers and four ERATEs
- [x] A token can only ask and search, and dies with its person
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2301](SB-2301-i18n.md) | W22 / S22.1 | The UI in the company's language | LAW-14 |
| [SB-2302](SB-2302-siem-stream.md) | W22 / S22.1 | Audit stream to the SIEM | LAW-11, LAW-07 |
| [SB-2303](SB-2303-offsite-s3.md) | W22 / S22.1 | Off-box copies to S3 | LAW-07 |
| [SB-2304](SB-2304-sources-first.md) | W22 / S22.2 | Search results while the answer is written | LAW-01, LAW-14 |
| [SB-2305](SB-2305-fair-share.md) | W22 / S22.2 | Fair share on a small box | — |
| [SB-2306](SB-2306-api-tokens.md) | W22 / S22.3 | Read-only API tokens | LAW-12, LAW-01, LAW-02 |
| [SB-2307](SB-2307-smtp.md) | W22 / S22.3 | Notifications by email | LAW-07 |

## Decisions for the user

Decided 2026-11-23 ([11-loop-own-fit.md](../../docs/11-loop-own-fit.md#decisions-for-the-user)): all of W20 to W22, in order; notifications by Slack or Mattermost direct message with email as the fallback (SB-2101, SB-2307); English, German and French (SB-2301); read-only API tokens, 90 days at most (SB-2306).

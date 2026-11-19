---
title: "E13 Portfolio Department Grants"
summary: "If the parent reads company data at all, it gets consented, aggregate-only, revocable grants on named spaces, never a superuser."
---

# E13: Portfolio Department Grants

> If the parent reads company data at all, it gets consented, aggregate-only, revocable grants on named spaces, never a superuser.

**Waves:** W13 · **Tickets:** 4 · **Stories:** 2

## User stories

- **US-13.1**: As a *parent-org finance lead*, I want consented, aggregate-only roll-ups from each company's Finance space, so that I see the portfolio without a superuser or raw documents. Delivered by [SB-1301](SB-1301-space-grant-model-and-consent.md), [SB-1302](SB-1302-aggregate-only-grant-delivery.md), [SB-1304](SB-1304-finance-roll-up-pilot.md).
- **US-13.2**: As a *company owner*, I want to revoke a grant and see every delivery it made, so that consent stays ours. Delivered by [SB-1303](SB-1303-grant-revocation-and-expiry.md).

## Ticket graph

```mermaid
flowchart LR
  SB1301["SB-1301<br/>Space grant model and…"]
  SB1302["SB-1302<br/>Aggregate-only grant…"]
  SB1303["SB-1303<br/>Grant revocation and expiry"]
  SB1304["SB-1304<br/>Finance roll-up pilot with…"]
  SB1301 --> SB1302
  SB1302 --> SB1303
  SB1303 --> SB1304
```

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1301](SB-1301-space-grant-model-and-consent.md) | W13 / S13.1 | Space grant model and consent | LAW-02, LAW-23 |
| [SB-1302](SB-1302-aggregate-only-grant-delivery.md) | W13 / S13.1 | Aggregate-only grant delivery | LAW-02, LAW-23 |
| [SB-1303](SB-1303-grant-revocation-and-expiry.md) | W13 / S13.1 | Grant revocation and expiry | LAW-10, LAW-23 |
| [SB-1304](SB-1304-finance-roll-up-pilot.md) | W13 / S13.2 | Finance roll-up pilot with friendly tenants | LAW-23 |

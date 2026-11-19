---
title: "ADR-0004: Tenants choose a channel, never a version; nobody is more than one behind"
summary: "Tenants choose a channel, never a version; nobody is more than one behind."
---

# ADR-0004: Tenants choose a channel, never a version; nobody is more than one behind

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Per-tenant version pinning grows into dozens of live versions, unpatched CVEs and support for every one of them.

## Decision

The manifest has `channel: stable|testing`. A `version:` field is rejected by `sb tenant check`. The release controller only promotes a new stable when every stable tenant runs the current stable, so no tenant is ever more than one release behind (LAW-08).

## Alternatives rejected

| Option | Why not |
|---|---|
| Version pinning per tenant | Unbounded support matrix; CVE fixes stall |
| Everyone on head | No soak time; one bad release hits everyone |

## Consequences

- At most two live releases at any time (stable and testing), three during a promotion
- A tenant can't opt out of an upgrade; they can only choose to be early

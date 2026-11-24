---
title: Connector syncs
summary: Why a connector's sync fails, by the class health shows, and how to fix it (SB-3201).
---

# Connector syncs

`GET /v1/admin/health` names the class of a connector's last failed sync in `failure`, and links the section below in `fix` ([sb-admin(3)](../../manual/sb-admin.3.md)). The cell keeps only the class: look at the source's own admin console for the detail. Every class is a warning until the next sync succeeds.

## auth

The source answered 401 or 403, or said the credentials were revoked. The connector's credentials have expired, been rotated, or lost a permission.

1. Check in the source's admin console that the connector's app, token or service account still exists and still has read access.
2. Put new credentials in the vault under the connector's name, as at install ([sb-connector(3)](../../manual/sb-connector.3.md)).
3. Wait for the next sync, about a minute, and check that health shows a new `last_success`.

## rate_limited

The source answered 429: the connector asked faster than the source allows. Syncs back off and repeat their work, so nothing is lost.

1. If it lasts more than an hour, check whether another tool shares the same app or token and its quota.
2. Ask the source's admin for a higher quota, or give the connector its own app.

## unreachable

The cell got no answer: a name that doesn't resolve, a TLS error, a refused connection or a timeout.

1. Check the source's status page.
2. From the cell's network, check that the source's host resolves and answers on 443.
3. On a self-hosted cell, check the box's outbound proxy and clock; a clock far off breaks TLS.

## source_error

The source answered with a 5xx, or the sync failed in a way the cell can't place. It is never treated as fine.

1. Check the source's status page.
2. If the source is up, run `sb index refresh CITE --reason TEXT` on one document the connector should have, to see whether one document or the whole sync fails ([sb(1)](../../manual/sb.1.md)).
3. If it keeps failing, send `install.sh --support` (self-hosted) or tell the maintainers the connector's kind and the time of `last_failure`.

## Alert rules

Health is also on `/metrics` as `sb_health_warning{check}` and `sb_health_unknown{check}` ([sb-observability(7)](../../manual/sb-observability.7.md#health-as-gauges-exactly-sb-3203)). A connector line is `check="connector:KIND"`.

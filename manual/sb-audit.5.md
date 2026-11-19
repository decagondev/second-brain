---
title: "sb-audit(5)"
man: "5"
summary: Audit row schema and export format.
---

# sb-audit(5)

## NAME

sb-audit — audit row fields, hash rules and the NDJSON export format

## SYNOPSIS

```json
{"seq":1234,"id":"aud_000000001234","at":"2027-03-02T10:41:07.512Z","tenant":"acme-logistics","actor":"usr_01JD…","kind":"tool_call","subject":"tc_01JD…","state":"Executing","approval":"apr_01JD…","payload_sha256":"5d1f…","release":"rel_2027-03-02_9b1c4e","reason":"policy:workflow-ticket-update permit","prev":"a3c9…","hash":"77e0…"}
```

## DESCRIPTION

### Fields

| Field | Type | Always | Meaning |
|---|---|---|---|
| `seq` | int | yes | Gap-free sequence per cell, starting at 1 |
| `id` | string | yes | `aud_` + zero-padded `seq` |
| `at` | RFC 3339, ms, UTC | yes | Commit time |
| `tenant` | string | yes | The cell's tenant |
| `actor` | id | yes | `usr_…`, `svc_…` or `sys_…` |
| `kind` | enum | yes | `tool_call`, `decision`, `approval`, `kill`, `config`, `release`, `access`, `custody` |
| `subject` | id | yes | What the row is about (`tc_…`, `apr_…`, …) |
| `state` | string | for state changes | e.g. `PendingApproval`, `Ran` |
| `approval` | id | consequential tool calls | Approval id |
| `payload_sha256` | hex | when a payload exists | Hash of the canonical payload. **Payloads are never stored here** |
| `receipt` | string | `Ran` | Target-issued receipt id |
| `release` | id | yes | Release running when written |
| `reason` | string | yes | Policy ids, kill reason, or error code |
| `prev` | hex | yes | `hash` of row `seq-1` (all zeros for row 1) |
| `hash` | hex | yes | SHA-256 of `prev` concatenated with the RFC 8785 canonical JSON of the row without `hash` |

### Canonical form and hash, exactly

The canonical form of a row is RFC 8785 JSON of the row's fields **without `hash`**: keys sorted, no whitespace, integers as plain digits, and fields that aren't set left out. Rows hold ids, hashes, codes and timestamps only, so every value is a string or an integer. A string with a control character, a `U+2028`/`U+2029`, or more than 512 bytes is refused at write time, so every implementation agrees on the bytes.

`hash = hex(SHA-256(prev ‖ canonical))`, where `prev` is the previous row's `hash` as 64 hex characters (64 zeros for row 1).

### Storage

The cell keeps rows in its own database, schema `audit_<tenant>`. The table refuses `UPDATE`, `DELETE` and `TRUNCATE` with a trigger, and the cell's audit role holds only `INSERT` and `SELECT`. Writers serialise on a per-tenant lock, so `seq` stays gap-free and each row's `prev` is the row before it.

### Anchors

Every hour, and at the end of every export, the cell signs its latest row:

| Field | Meaning |
|---|---|
| `seq`, `hash` | The row being anchored |
| `signed_at` | RFC 3339, ms, UTC |
| `sig` | Ed25519 signature, base64, over `sb-audit-anchor:v1:<tenant>:<seq>:<hash>:<signed_at>` |
| `key` | `ed25519:` + base64 public key |

The signing key belongs to **the cell**. Its seed is a data key wrapped by the cell's root key (`Keys.Wrap`, ADR-0015), never a factory-wide key. The public key is published in the cell's custody record, so auditors can pin it. Hourly anchors also go to WORM storage, a `kind: custody` audit row records every key hand-off, and `custody` is a valid `kind`.

### Forbidden content

Rows must never contain document text, prompts, answers, drafts, secrets or tokens. The canary scanner checks every row in CI and production (LAW-07).

### Export

`sb audit export` writes NDJSON, one row per line, ordered by `seq`. The first line is a signed anchor on the row **before** the first exported row (`seq` 0 with the zero hash for a full export), so a dropped first row is detected. Every 10,000 rows, and on the last row, there is an anchor line:

```json
{"anchor":true,"seq":10000,"hash":"…","signed_at":"…","sig":"…","key":"…"}
```

The file is encrypted with the cell's KMS key for transport. `sb audit verify --export FILE` checks it offline.

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-07](sb-laws.7.md#law-07), [LAW-11](sb-laws.7.md#law-11).

## SEE ALSO

[sb-audit(1)](sb-audit.1.md), [sb-audit(7)](sb-audit.7.md)

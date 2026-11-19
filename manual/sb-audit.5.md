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

### Forbidden content

Rows must never contain document text, prompts, answers, drafts, secrets or tokens. The canary scanner checks every row in CI and production (LAW-07).

### Export

`sb audit export` writes NDJSON, one row per line, ordered by `seq`. Every 10,000 rows, and at the end, there is an anchor line:

```json
{"anchor":true,"seq":10000,"hash":"…","signed_at":"…","sig":"…","key":"…"}
```

The file is encrypted with the cell's KMS key for transport. `sb audit verify --export FILE` checks it offline.

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-07](sb-laws.7.md#law-07), [LAW-11](sb-laws.7.md#law-11).

## SEE ALSO

[sb-audit(1)](sb-audit.1.md), [sb-audit(7)](sb-audit.7.md)

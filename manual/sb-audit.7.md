---
title: "sb-audit(7)"
man: "7"
summary: What is recorded, and how to prove it wasn't changed.
---

# sb-audit(7)

## NAME

sb-audit — the append-only, hash-chained record of everything consequential

## SYNOPSIS

```text
row(n).prev = row(n-1).hash      hash = SHA-256(prev ‖ canonical(row))
hourly: sign(latest hash) → WORM storage
```

## DESCRIPTION

### What is recorded

Every tool-call state change; every policy decision on a tool call; every read refusal and a sample of read allows; every approval event; every kill and unkill; every manifest apply and console change; every deploy, rollback and attestation; and any access from outside the tenant (break-glass).

### Policy decisions

The policy kernel writes a `decision` row for every decision it records, **before** it returns the decision to the caller:

| Field | Value |
|---|---|
| `actor` | the scope's user |
| `subject` | the resource, as `Type::"ID"` |
| `state` | `allow`, `deny` or `unknown` |
| `reason` | the action, then the deciding policy ids and the kernel's reason, e.g. `call_tool: permitted by base.cedar#3` |
| `space` | the space the request was in, when it was in one (SB-1008) |

What is recorded: every `Deny` and `Unknown`, and every decision on `call_tool`, `propose`, `approve`, `admin` and `use_agent`. `Allow` on `read` is sampled, 1% by default (`SB_AUDIT_READ_SAMPLE`, 0 to 1). If the row can't be written, the decision becomes `Unknown`: nothing that should have been audited goes ahead unaudited (LAW-04). Rows hold ids and policy ids only, never content (LAW-07).

### When

**Before** the thing it records takes effect. If the audit write fails, the action doesn't happen (LAW-04).

### What is never recorded

Document content, prompts, answers, drafts, secrets and tokens. Payloads appear only as SHA-256 hashes (LAW-07).

### Tamper evidence

The audit role can only insert and select. Each row chains to the previous row's hash, and the latest hash is signed and written to WORM storage every hour. `sb audit verify` recomputes everything (LAW-11).

### Retention and backup

Seven years by default, independent of `retention_days`. Backed up 3-2-1, with a monthly restore drill that verifies the chain.

Exactly (SB-412):

- **What.** Every night, the backup job writes two objects for each cell: the full signed audit export (sb-audit(5)) and a snapshot of the approval store (every action with its state, hashes and receipts, as JSON lines). Each object is named `<tenant>/<YYYY-MM-DD>/audit.ndjson` or `.../approvals.jsonl`, with a `.sha256` beside it.
- **Where.** At least **three** copies, on at least **two** kinds of media, with at least **one** off-site. The job refuses to run against a target set that breaks any of these, and reports which rule.
- **WORM.** Each copy is write-once: an object can't be overwritten or deleted before its retention date (7 years). A second write to the same name is refused, so a nightly run never replaces an earlier night. On GCP that is a bucket with a locked retention policy (sb-runtime(7)); on a self-hosted cell it is an S3 store with compliance-mode object lock (sb-selfhost(7), Write-once storage), which the cell refuses unless the lock is in compliance mode for at least seven years, and which is proved at install by a probe the store must refuse to delete.
- **Sealed off-box copy.** A copy with `seal` (sb-cell(5), `backups`) has every object sealed to the recovery key's public half before it is written, as `NAME.sealed` (SB-1901, sb-selfhost(7), Sealed off-box copies); the other copies are unchanged. Drilling that copy needs the recovery key: `sb backup open` each object first.
- **Restore drill.** Monthly, and after any restore, the drill fetches the latest audit export from **every** copy. It checks each one's `.sha256`, verifies the chain offline with the cell's pinned key (`sb audit verify --export`), and checks that the copies agree with each other and that none is behind the live chain by more than a day. The drill passes only if every copy passes, and the report names each copy's result. A failure is Sev-1, like a chain break (LAW-11).

### Streaming to a SIEM (SB-2302)

With `audit.stream: syslog+tls://HOST:PORT` in sb-cell(5), the jobs service forwards every audit row to the company's SIEM as it is written, within a few seconds, as syslog over TLS (RFC 5425, octet-counted frames). The SIEM's certificate is verified for its host name against the system's roots, which on a self-hosted cell include the company's CA in `/etc/sb/ca`; TLS 1.2 at least.

Each row is one RFC 5424 message: facility `log audit` (13), severity info, the row's `at` as the timestamp, the cell's host name, `second-brain` as the app, the row's `kind` as the message id, the structured data `[sb@32473 tenant="…" seq="…" hash="…"]`, and as its text the row exactly as the export writes it (sb-audit(5)): ids, hashes, decisions and reasons, never content (LAW-07). A SIEM can check every row it gets as the export's tooling does, its hash and its `prev` link to the row before (LAW-11).

The audit log is the queue: the stream keeps only a cursor, the `seq` of the last row the SIEM took, in `state/audit/stream.json`. A SIEM that is down never blocks a write; the stream tries again every 30 seconds and resumes from the cursor, after a restart too. Before each message it checks that the SIEM still holds the connection, so a row sent to a SIEM that hung up is sent again, never lost; a SIEM should drop a `seq` it already has. Rows queue up to a day: when the oldest row still to send is an hour old, a Sev-2 `audit_stream_behind` alert fires (and again each hour it stays so); rows older than a day when the SIEM comes back are not sent, and one `gap` message (severity warning, `[sb@32473 tenant="…" from="…" to="…"]`) names them, with a Sev-2 `audit_stream_gap` alert, so the SIEM reads them from the export. An unreadable cursor stops the stream, never restarts it from row 1. `sb_audit_stream_rows_total` counts the rows sent and `sb_audit_stream_behind_seconds` says how far behind it is.

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-07](sb-laws.7.md#law-07), [LAW-11](sb-laws.7.md#law-11).

## EXAMPLES

```sh
sb audit verify --tenant acme-logistics
sb audit export --tenant acme-logistics --since 2027-01-01 --out acme-2027Q1.ndjson
sb audit verify --export acme-2027Q1.ndjson       # offline, for auditors
```

## SEE ALSO

[sb-audit(1)](sb-audit.1.md), [sb-audit(5)](sb-audit.5.md), [10 Audit log](../docs/architecture/10-audit-log.md)

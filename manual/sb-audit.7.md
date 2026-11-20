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
- **WORM.** Each copy is write-once: an object can't be overwritten or deleted before its retention date (7 years). A second write to the same name is refused, so a nightly run never replaces an earlier night.
- **Restore drill.** Monthly, and after any restore, the drill fetches the latest audit export from **every** copy. It checks each one's `.sha256`, verifies the chain offline with the cell's pinned key (`sb audit verify --export`), and checks that the copies agree with each other and that none is behind the live chain by more than a day. The drill passes only if every copy passes, and the report names each copy's result. A failure is Sev-1, like a chain break (LAW-11).

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

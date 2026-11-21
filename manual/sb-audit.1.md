---
title: "sb-audit(1)"
man: "1"
summary: Verify, follow and export the audit log.
---

# sb-audit(1)

## NAME

sb-audit — verify the audit hash chain, tail it, or export it

## SYNOPSIS

```text
sb audit verify [--tenant T | --cell URL] [--since TIME] [--key ed25519:PUB] | --export FILE [--key ed25519:PUB]
sb audit tail   [--tenant T | --cell URL] [--kind KIND]... [--actor ID] [--space NAME] [--from SEQ] [--no-follow]
sb audit export [--tenant T | --cell URL] --since TIME [--until TIME] --out FILE
```

## DESCRIPTION

**verify** recomputes the hash chain from the first row (or from `--since`, starting at the nearest signed anchor) and checks every hourly anchor signature. With `--export FILE`, it verifies an exported file offline, so auditors don't need access to the platform. It checks that `seq` starts where it should and has no gaps, that every `prev` and `hash` holds, that every anchor's hash matches its row and its signature verifies, and that the file ends with an anchor on its last row. `--key` pins the cell's public key (from its custody record). Without it, the key in the file is used and `verify` prints `key not pinned`, because a forger could sign with a key of their own.

The forms without `--export` talk to a running cell (SB-1507): `--cell URL`, or `--tenant T`, whose cell the control plane's registry names. They call the cell's control-plane routes (sb-api(3)) with the control plane's identity, `SB_TOKEN`; any other caller gets `EAUTH` or `EPERM`. `verify` fetches the cell's signed export and checks it exactly as `--export` does, so a running cell is checked by the same code as a file.

**tail** prints the matching rows from `--from SEQ` (default the first), then streams new ones every two seconds until interrupted; `--no-follow` stops after the rows already there. With `--json` each row is one NDJSON line, otherwise a short human line. Filter by `--kind` (`tool_call`, `decision`, `approval`, `kill`, `config`, `release`, `access`) or by `--space NAME`, which keeps the rows written in that space and the `kill` rows about it (`subject` `space:NAME`). The admin API takes the same filter, `GET /v1/admin/audit?space=NAME` (sb-admin(3)).

**export** writes signed NDJSON in the [sb-audit(5)](sb-audit.5.md) format for the rows from `--since` to `--until`, anchored on the row before the first and on the last, so the file verifies on its own; `sb` checks it before writing it and says so. It travels over TLS; on a local cell the file itself isn't encrypted, and encrypting it with the cell's KMS key comes with the GCP adapter (SB-908).

Rows never contain document content, prompts or secrets: payloads appear only as SHA-256 hashes.

## EXIT STATUS

0 chain intact · 1 `ECHAIN`: a break was found (the first bad `seq` is printed) · 3 not permitted.

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-07](sb-laws.7.md#law-07), [LAW-11](sb-laws.7.md#law-11).

## EXAMPLES

```text
$ sb audit verify --tenant acme-logistics
chain   ok   1,204,551 rows, seq 1 to 1204551
anchors ok   2,184 hourly anchors, all signatures valid

$ sb audit tail --tenant acme-logistics --kind approval
10:41:07 apr_01JD3QD9 approved  by usr_01JD3Q8Z  tc_01JD3QC7 ticket-update  hash 5d1f…
```

## SEE ALSO

[sb-audit(7)](sb-audit.7.md), [sb-audit(5)](sb-audit.5.md)

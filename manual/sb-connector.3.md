---
title: "sb-connector(3)"
man: "3"
summary: The four-call connector contract every source adapter implements.
---

# sb-connector(3)

## NAME

sb-connector — `Fetch`, `ACL`, `Changes`, `Act`: the connector contract, API version 1

## SYNOPSIS

```go
package ports

const ConnectorAPIVersion = 1

type Connector interface {
    Fetch(ctx context.Context, ref SourceRef) (Document, error)
    ACL(ctx context.Context, user UserRef, refs []SourceRef) ([]Verdict, error)
    Changes(ctx context.Context, cursor Cursor) (ChangeSet, Cursor, error)
    Act(ctx context.Context, a Action, key IdempotencyKey) (Receipt, error)
}
```

## DESCRIPTION

A connector is an adapter between one kind of company system and the platform. It holds no state except a sync cursor and does nothing but these four calls.

### Fetch

Returns the document's text and structure, its **version** (monotonic per document, as reported by the source), and an ACL snapshot used only as a prefilter. The same `(ref, version)` must always return the same content hash.

### ACL

The **authority** on who may see what. Called live, at answer time, for a batch of up to 200 refs. Returns one `Verdict` per ref, in order:

| Verdict | When |
|---|---|
| `VerdictAllow` | The source says this user can open this document now |
| `VerdictDeny` | The source says they can't |
| `VerdictUnknown` | **Anything else**: timeout, error, 404, rate limit, unmapped permission model, partial response |

A connector must never return `VerdictAllow` because it couldn't tell. The caller's deadline is 800 ms. A late answer counts as `Unknown`.

### Changes

Returns everything that changed since `cursor`: creates, updates, deletes **and permission changes**. The cursor must be resumable, and replaying a change must be harmless.

### Act

Performs one write for a gated workflow. Must be idempotent per `key` (the approval id) and must return a `Receipt` with the **target-issued** id of the change. If the target gives no id, `Act` reads the change back to obtain one or returns an error. Connectors that don't support writes return `ENOTSUP`.

### Credentials

Resolved at call time from `vault://tenants/<tenant>/connectors/<kind>` through the `Secrets` port. Never stored in a struct field, never logged.

## ERRORS

| Error | Meaning |
|---|---|
| `ENOTSUP` | Call not supported (usually `Act`) |
| `EUNAVAIL` | Source unreachable; on `ACL` this becomes `VerdictUnknown` |
| any other | Treated as `VerdictUnknown` on `ACL`, as `DidNotRun` on `Act` |

## CONTRACT KIT

Run `go test ./internal/adapters/connectors/<name>/... -run Kit` against a sandbox. All cases must pass before a connector can be named in any manifest. The cases are listed in [03 Connectors](../docs/architecture/03-connectors.md#the-contract-kit).

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-03](sb-laws.7.md#law-03), [LAW-05](sb-laws.7.md#law-05), [LAW-07](sb-laws.7.md#law-07).

## EXAMPLES

A minimal connector skeleton is in [examples/connector/](../examples/connector/README.md).

## SEE ALSO

[sb-permissions(7)](sb-permissions.7.md), [sb-ingest(7)](sb-ingest.7.md), [sb-approvals(7)](sb-approvals.7.md)

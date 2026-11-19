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
    APIVersion() int
    Fetch(ctx context.Context, ref SourceRef) (Document, error)
    ACL(ctx context.Context, user UserRef, refs []SourceRef) ([]Verdict, error)
    Changes(ctx context.Context, cursor Cursor) (ChangeSet, Cursor, error)
    Act(ctx context.Context, a Action, key IdempotencyKey) (Receipt, error)
}
```

## DESCRIPTION

A connector is an adapter between one kind of company system and the platform. It holds no state except a sync cursor and does nothing but these four calls.

### Version negotiation

`APIVersion` returns the contract version the connector implements. The platform loads a connector only if `ports.Negotiate` accepts its version: today exactly `1`. A connector built for a newer or older contract is refused at load time with `ENOTSUP`, never run in a degraded mode.

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

The kit is `connectorkit.Run(t, sandbox)` (package `internal/core/connectorkit`). A connector's test supplies a `Sandbox`: the connector under test, an existing document, a user who can open it and one who can't, a way to inject a fault into the next `ACL` call (`500`, timeout, `404`), a way to edit a document and to change its permissions, 200 existing refs for the batch case, and, for writable connectors, a valid `Action`. The kit also checks `APIVersion` against `ports.Negotiate`.

## MOCK CONNECTOR

`internal/adapters/connectors/mock` is a scriptable source system and connector for laws, evals and tests. Nothing in it is random; every behaviour is asked for:

| Ask for | How |
|---|---|
| Allow or Deny | the document's viewers (`Put`, `SetViewers`) |
| Unknown for one user and document | `ScriptVerdict(email, id, VerdictUnknown)` (any verdict can be forced this way) |
| A failing ACL call | `QueueFaults(500, timeout, 404, …)`: one fault per coming call, in order |
| A deleted document | `Delete` (Fetch fails, ACL is Unknown) |
| A target with no receipt | `IssueIDs(false)`: `Act` returns `ENORECEIPT` and changes nothing |
| Read-only | `New(src, false)`: `Act` returns `ENOTSUP` |

It passes the contract kit in both modes. The connector-level LAW-03 and LAW-05 tests run on it.

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-03](sb-laws.7.md#law-03), [LAW-05](sb-laws.7.md#law-05), [LAW-07](sb-laws.7.md#law-07).

## EXAMPLES

A minimal connector skeleton is in [examples/connector/](../examples/connector/README.md).

## SEE ALSO

[sb-permissions(7)](sb-permissions.7.md), [sb-ingest(7)](sb-ingest.7.md), [sb-approvals(7)](sb-approvals.7.md)

---
title: Example connector
summary: A minimal read-only connector showing the four-call contract.
---

# Example connector: `notes`

[`notes.go`](notes.go) is a complete, minimal, read-only connector for a fictional notes system. It compiles against the `ports` types in [sb-connector(3)](../../manual/sb-connector.3.md) and shows the rules every connector follows:

| Rule | Where in the code |
|---|---|
| Credentials fetched per call from the fixed vault path, never stored | `token()` |
| `ACL` answers a batch, with an 800 ms deadline | `ACL()` |
| Anything uncertain is `VerdictUnknown`, the zero value | `make([]ports.Verdict, …)` and every early `return out, nil` |
| `Changes` reports deletes and permission-only changes | `Changes()` |
| Read-only connectors return `ENOTSUP` from `Act` | `Act()` |
| States its contract version, which the platform negotiates | `APIVersion()` |
| Compile-time proof it implements the port | `var _ ports.Connector = (*Connector)(nil)` |

A real connector also passes the contract kit: its test calls `connectorkit.Run(t, sandbox)`. See `internal/adapters/connectors/mock/mock_test.go` for a complete sandbox.

Note what `ACL` does on error: it returns all-`Unknown` and **no error**. The caller would treat an error as `Unknown` anyway, so there's one obvious path (P3).

To turn this into a real connector: put it under `internal/adapters/connectors/docs-<vendor>/`, implement `Client` against the vendor SDK, and make the contract kit pass.

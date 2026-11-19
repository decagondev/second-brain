---
title: "ADR-0017: First third-party Go modules: cedar-go and pgx, pinned, vendored and allow-listed"
summary: "First third-party Go modules: cedar-go and pgx, pinned, vendored and allow-listed."
---

# ADR-0017: First third-party Go modules: cedar-go and pgx, pinned, vendored and allow-listed

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-19 | Platform lead (the user) |

## Context

ADR-0016 kept W0 on the standard library and said the first module would arrive with its own ADR. W1 needs two: a Cedar evaluator for the policy kernel (SB-109, ADR-0008) and a Postgres driver for the tenant registry and cell databases (SB-105). Every module is supply-chain surface (ADR-0005), so how modules get in matters as much as which ones.

## Decision

- Add **`github.com/cedar-policy/cedar-go` v1.8.0** (the Cedar team's Go evaluator) and **`github.com/jackc/pgx/v5` v5.8.0** (Postgres driver and pool).
- These are the newest releases that build with the pinned **Go 1.24** (`tools/versions.env`). cedar-go v1.9 and later needs Go 1.26, and pgx v5.9 and later needs Go 1.25. Moving to them is a separate Go-version bump, recorded in a later ADR.
- **Allow-list.** `tools/deps/allowed.txt` lists every module that may appear in `go.mod`, direct and indirect, with its exact version and `go.sum` hash. `make deps` fails if `go.mod` or `go.sum` contain anything that isn't on the list, or a different version or hash. Adding a module means changing that file in review.
- **Vendored.** Each module is vendored (`go mod vendor`) in the change that first imports it, so builds don't need the network and reviewers see the code. CI builds with `-mod=vendor`.
- **Only adapters import vendors.** cedar-go is used only by `internal/adapters/policy/cedar` and pgx only by `internal/adapters/postgres/...`. The core sees them through ports, and depguard (SB-008) already enforces this.
- **SBOM.** These modules are the first entries for the SPDX SBOM that SB-504 generates. Until SB-504, the allow-list is the inventory.

The full transitive set at these versions is: `github.com/jackc/pgpassfile` v1.0.0, `github.com/jackc/pgservicefile` v0.0.0-20240606120523-5a60cdf6a761, `github.com/jackc/puddle/v2` v2.2.2, `golang.org/x/exp` v0.0.0-20220921023135-46d9e7742f1e, `golang.org/x/sync` v0.17.0 and `golang.org/x/text` v0.29.0. The test-only modules of those modules are not built into anything.

## Alternatives rejected

| Option | Why not |
|---|---|
| Write our own Cedar evaluator | Policy semantics must match the Cedar spec and the analysis tools (SB-112); re-implementing them is a correctness risk |
| Call Cedar through the Rust CLI or a sidecar | A process boundary on every decision, and a second toolchain in every cell |
| `database/sql` with `lib/pq` | lib/pq is in maintenance mode; pgx is the maintained driver with a native pool |
| Bump Go to 1.26 now for the newest versions | It changes the toolchain for everything and is a separate decision; the 1.24-compatible versions do what W1 needs |
| No allow-list, trust `go.sum` | `go.sum` proves integrity, not intent. The allow-list makes adding a module a reviewed decision |

## Consequences

- `go.mod` gets its first `require` lines when SB-105 and SB-109 import these modules. ADR-0016's "no require lines" rule ends there.
- We run releases one or two minor versions behind upstream until the Go bump. Upstream security fixes on newer lines must be checked against these versions (SB-505's CVE rebuild path).
- Vendored code makes the repository larger. That's accepted in exchange for offline, reviewable builds.

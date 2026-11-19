---
title: "ADR-0016: Go standard library only in W0; property tests use an in-repo helper"
summary: "Go standard library only in W0; property tests use an in-repo helper."
---

# ADR-0016: Go standard library only in W0; property tests use an in-repo helper

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead |

## Context

ADR-0007 named `pgregory.net/rapid` for property testing. When SB-001 started, the build environment couldn't reach the Go module proxy (policy denial), so no third-party module could be fetched. Separately, every dependency is supply-chain surface (ADR-0005), and W0 needs very little.

## Decision

- W0 Go code uses the **standard library only**. `go.mod` has no `require` lines.
- Property tests use `laws.Check`: deterministic seeds 1..n, `SB_SEED` to reproduce a failure, no shrinking. It is about 40 lines.
- This **supersedes the `rapid` line in ADR-0007's consequences**. The rest of ADR-0007 stands.
- The first third-party module will be the Cedar engine for SB-109. It arrives with its own ADR: fetched from a trusted source, vendored (`go mod vendor`), checksum-pinned and listed in the SBOM.

## Alternatives rejected

| Option | Why not |
|---|---|
| Fetch `rapid` by another route | The proxy refusal is a policy decision; routing around it isn't acceptable |
| Block W0 until the proxy is available | Nothing in W0 needs a dependency |
| Write property tests later | LAW-01 to LAW-05 must exist red in W0 (SB-010) |

## Consequences

- No shrinking: a failing seed shows the whole generated world. Acceptable because worlds are small; revisit if failures get hard to read.
- The helper's own correctness is covered by the law self-tests (a reference system passes every law and six mutations are caught).
- One fewer dependency to sign, scan and update.

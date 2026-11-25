---
title: "sb-release(1)"
man: "1"
summary: Release status, promotion and one-step rollback (maintainers).
---

# sb-release(1)

## NAME

sb-release — see what runs where, promote a release to stable, roll back

## SYNOPSIS

```text
sb release status   [--tenant T]
sb release promote  REL --to stable
sb release rollback [--reason TEXT]
sb release notes    --from DIR --to DIR
```

## DESCRIPTION

**status** lists the current `stable` and `testing` releases, each ring's state, and for every tenant: channel, running release, attested digest and lag behind stable.

**promote** makes `REL` the new stable. It refuses (exit 1) unless:

1. `REL` passed the eval gate and is signed;
2. `REL` has soaked on ring 0 and ring 1 for the minimum time (default 48 h);
3. **every stable-channel tenant is running the current stable** (the promote guard, LAW-08).

**rollback** makes the previous stable current, marks the bad release `Revoked`, and redeploys every cell running it. Target: under 10 minutes for all cells. A revoked release can never be promoted or redeployed.

**notes** (SB-4402) says what changed for each agent between two releases. `DIR` is a release's agent bundle as `make release-build` writes it to `build/release/agents/`: `agents.json`, `prompts/NAME.md` and `evals/SUITE.json`, the gate's results. For each agent in either, it prints its owner, `added` or `removed` if it is only in one, the prompt's lines added (`+`) and removed (`-`), and its suite's pass rate and failures before and after, or `no results` when a side has none. It reads files only, and needs no control plane.

Only maintainers can run status, promote and rollback. Company operators see the same information read-only in the console.

**How the controller decides, exactly (SB-507, SB-508).** The controller is `internal/core/rollout`. Its state and moves are those of `proofs/tla/Rollout.tla`, and after every move it checks the model's invariants (`OnlyGatedRuns`, `OneBehindMax`, `StableNotAhead`, `StableNotRevoked`, `RevokedOnlyWhileDraining`, `DrainingOnlyRevoked`). A move that would break one is refused.

| Move | Allowed when |
|---|---|
| register | a release passed admission (signed and gated, sb-release(7)); others are never deployable |
| deploy to `testing` (ring 0 is the internal tenant, ring 1 the friendly testing tenants) | the release is gated, not revoked, and not older than stable |
| promote | the release is gated, not revoked and newer than stable; it has run on ring 0 **and** ring 1 for the soak time (48 h); and every stable tenant runs the current stable (the promote guard) |
| deploy to `stable` (ring 2) | a stable tenant isn't on the current stable |
| rollback | there is a previous stable: stable is revoked, the previous stable becomes current, and every cell on the revoked release starts **draining** (no new consequential tool calls) until it is redeployed |
| revoke on testing | the release is newer than stable; its cells drain |

Every move writes a `kind: release` audit row. With a local control plane (`--endpoint file:///DIR`), the state is `DIR/releases/state.json`. A rollback drill (SB-508) rolls back and redeploys every draining cell, then reports the time taken against the 10-minute target.

## EXIT STATUS

0 done · 1 guard not met (reason printed) · 3 not permitted.

## LAWS

[LAW-06](sb-laws.7.md#law-06), [LAW-08](sb-laws.7.md#law-08).

## EXAMPLES

```text
$ sb release status
stable   rel_2027-03-02_9b1c4e   promoted 2027-03-04
testing  rel_2027-03-09_4f7a20   ring 1, soaking 31h of 48h
tenant             channel  running                 lag
acme-logistics     stable   rel_2027-03-02_9b1c4e   0
beta-freight       testing  rel_2027-03-09_4f7a20   -
gamma-parts        stable   rel_2027-02-23_77c0d1   1   (deploying)

$ sb release promote rel_2027-03-09_4f7a20 --to stable
refused: promote guard: gamma-parts runs rel_2027-02-23_77c0d1, not current stable (LAW-08)
```

## SEE ALSO

[sb-release(7)](sb-release.7.md), [sb-evals(7)](sb-evals.7.md)

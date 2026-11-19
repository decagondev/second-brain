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
```

## DESCRIPTION

**status** lists the current `stable` and `testing` releases, each ring's state, and for every tenant: channel, running release, attested digest and lag behind stable.

**promote** makes `REL` the new stable. It refuses (exit 1) unless:

1. `REL` passed the eval gate and is signed;
2. `REL` has soaked on ring 0 and ring 1 for the minimum time (default 48 h);
3. **every stable-channel tenant is running the current stable** (the promote guard, LAW-08).

**rollback** makes the previous stable current, marks the bad release `Revoked`, and redeploys every cell running it. Target: under 10 minutes for all cells. A revoked release can never be promoted or redeployed.

Only maintainers can run these commands. Company operators see the same information read-only in the console.

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

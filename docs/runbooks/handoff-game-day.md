---
title: "Hand-off game day runbook"
summary: "Hand a friendly tenant's cell to its own cloud account end to end, then rehearse detach on a copy (SB-710)."
---

# Hand-off game day runbook

With a friendly tenant who has agreed, in their staging project, before any company hand-off for real. The scoring is in [sb-factory(7), Hand-off game day, exactly](../../manual/sb-factory.7.md#hand-off-game-day-exactly-sb-710), and the hand-off itself follows [the hand-off runbook](handoff.md). Rehearse first: `go test ./tests/handoff/...` must pass.

## Plan (two weeks before)

1. Agree the date, the tenant's staging project and the people for each role in the hand-off runbook. Expect: names for planner, factory approver, tenant owner and security reviewer in the game-day ticket.
2. The tenant runs the deploy-role bootstrap and grants the verifier identity. Expect: the deploy account's email and the verifier's grants recorded in the ticket.
3. Snapshot what the day will compare against: the audit chain's last seq, the index's `(cite, ACL)` set and the list of factory paths from `sb factory access`. Expect: all three saved outside the cell.

## Run

1. Hand off the cell with the hand-off runbook, from plan to `TenantOwned`. Expect: every step's expected result, and a custody audit row for each.
2. Check LAW-17: `sb factory access TENANT`. Expect: exit 0, and only the retained grants' paths listed.
3. Check LAW-18: `sb factory status TENANT`. Expect: `key_wraps` shows only `tenant` for every data key, none broken.
4. Check the audit chain across the move: export it from the tenant cell and run `sb audit verify --export FILE --key KEY`. Expect: it verifies, its first rows are the snapshot's rows unchanged, and it continues past the snapshot's last seq.
5. Check the laws on the tenant cell: `sb tenant verify`. Expect: every law passes.
6. Make a copy of the tenant cell (restore its backups into a scratch project) and detach the copy: `sb factory detach COPY --accept-unsupported --out DIR`. Expect: the export package in `DIR`, and `sb factory verify-export DIR` passing on a machine with no factory credentials.
7. After the copy's detach, run `sb factory access COPY`. Expect: no paths at all.

## Retro (the same day)

Copy into the devlog: the plan hash, both approvers, the time of each phase, the `access` result, the `verify` result, and whether the chain and the export verified. Each surprise becomes a `BUG-nnnn` entry with the test that would have caught it, and a fix to this runbook or the hand-off runbook.

The rehearsal already found one such bug before any game day: BUG-0012, a cutover that revoked a retained grant's binding.

Laws: [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18), [LAW-11](../../manual/sb-laws.7.md#law-11).

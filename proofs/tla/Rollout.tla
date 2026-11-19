----------------------------- MODULE Rollout -----------------------------
(***************************************************************************)
(* Releases, channels, promotion and rollback.                             *)
(* Laws: LAW-06 only gated (signed) releases run; LAW-08 no stable tenant  *)
(* is more than one promotion behind stable.                               *)
(* Releases are numbered 0..MaxRel; release 0 is the initial stable.       *)
(* Rollback is not atomic (BUG-0001): it revokes at once, then cells are   *)
(* redeployed one by one. Cells still on a revoked build are "draining" and *)
(* start no consequential tool calls (the kill-switch path, SB-508).        *)
(***************************************************************************)
EXTENDS Naturals, FiniteSets

CONSTANTS StableTenants, TestingTenants, MaxRel

ASSUME StableTenants \cap TestingTenants = {} /\ MaxRel \in Nat

Tenants == StableTenants \cup TestingTenants

VARIABLES latest, gated, revoked, stable, prevStable, running, draining

vars == <<latest, gated, revoked, stable, prevStable, running, draining>>

TypeOK ==
  /\ latest \in 0..MaxRel
  /\ gated \subseteq 0..MaxRel
  /\ revoked \subseteq 0..MaxRel
  /\ stable \in 0..MaxRel
  /\ prevStable \in 0..MaxRel
  /\ running \in [Tenants -> 0..MaxRel]
  /\ draining \subseteq Tenants

Init ==
  /\ latest = 0
  /\ gated = {0}
  /\ revoked = {}
  /\ stable = 0
  /\ prevStable = 0
  /\ running = [t \in Tenants |-> 0]
  /\ draining = {}

\* Build the next release; it either passes the eval gate (and is signed) or not.
Build ==
  /\ latest < MaxRel
  /\ latest' = latest + 1
  /\ \/ gated' = gated \cup {latest + 1}
     \/ gated' = gated
  /\ UNCHANGED <<revoked, stable, prevStable, running, draining>>

\* Ring 0/1: testing tenants take any gated, unrevoked release at or after stable.
DeployTesting(t, r) ==
  /\ t \in TestingTenants \ draining
  /\ r \in gated \ revoked
  /\ r >= stable
  /\ running' = [running EXCEPT ![t] = r]
  /\ UNCHANGED <<latest, gated, revoked, stable, prevStable, draining>>

\* Promote guard: every stable tenant already runs the current stable.
Promote(r) ==
  /\ r \in gated \ revoked
  /\ r > stable
  /\ \A t \in StableTenants : running[t] = stable
  /\ prevStable' = stable
  /\ stable' = r
  /\ UNCHANGED <<latest, gated, revoked, running, draining>>

\* Ring 2: a stable tenant catches up to the current stable.
DeployStable(t) ==
  /\ t \in StableTenants \ draining
  /\ running[t] # stable
  /\ running' = [running EXCEPT ![t] = stable]
  /\ UNCHANGED <<latest, gated, revoked, stable, prevStable, draining>>

\* One-step rollback: revoke stable and restore the previous one at once. Every
\* cell on the revoked release starts draining; Redeploy moves them one by one.
Rollback ==
  /\ prevStable < stable
  /\ revoked' = revoked \cup {stable}
  /\ stable' = prevStable
  /\ draining' = draining \cup {t \in Tenants : running[t] = stable}
  /\ UNCHANGED <<latest, gated, prevStable, running>>

\* A draining cell is redeployed to the current stable.
Redeploy(t) ==
  /\ t \in draining
  /\ running' = [running EXCEPT ![t] = stable]
  /\ draining' = draining \ {t}
  /\ UNCHANGED <<latest, gated, revoked, stable, prevStable>>

\* Revoke a release that is only on the testing channel.
RevokeTesting(r) ==
  /\ r \in gated \ revoked
  /\ r > stable
  /\ revoked' = revoked \cup {r}
  /\ draining' = draining \cup {t \in Tenants : running[t] = r}
  /\ UNCHANGED <<latest, gated, stable, prevStable, running>>

Next ==
  \/ Build
  \/ \E t \in Tenants, r \in 0..MaxRel : DeployTesting(t, r)
  \/ \E r \in 0..MaxRel : Promote(r) \/ RevokeTesting(r)
  \/ \E t \in Tenants : DeployStable(t) \/ Redeploy(t)
  \/ Rollback

Spec == Init /\ [][Next]_vars

---------------------------------------------------------------------------
\* LAW-06: every running release passed the gate.
OnlyGatedRuns == \A t \in Tenants : running[t] \in gated

\* A cell runs a revoked release only while it is draining (BUG-0001).
RevokedOnlyWhileDraining == \A t \in Tenants : running[t] \in revoked => t \in draining

\* Draining is only ever for revoked releases.
DrainingOnlyRevoked == \A t \in draining : running[t] \in revoked

\* LAW-08: serving stable tenants run stable or the previous stable, never older.
OneBehindMax == \A t \in StableTenants \ draining : running[t] \in {stable, prevStable}

\* Serving stable tenants never run ahead of stable.
StableNotAhead == \A t \in StableTenants \ draining : running[t] <= stable

\* Stable is never a revoked release.
StableNotRevoked == stable \notin revoked
==========================================================================

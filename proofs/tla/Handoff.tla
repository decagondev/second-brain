----------------------------- MODULE Handoff -----------------------------
(***************************************************************************)
(* Hand-off of one cell's custody from the factory to its tenant.          *)
(* Laws: LAW-17 no unlisted factory access after hand-off; LAW-18 the root *)
(* key always has a custodian. Also: dual approval of the exact plan, the  *)
(* serving cell can always unwrap its key, abort before cutover is safe.   *)
(* Hand-back is not modelled yet (W7 follow-up).                           *)
(***************************************************************************)
EXTENDS FiniteSets

CONSTANTS Paths, Retained, Plans, NoPlan

ASSUME Retained \subseteq Paths /\ NoPlan \notin Plans

Phases == {"Managed", "Planned", "Provisioned", "KeyDual", "Moved",
           "Verified", "CutOver", "TenantOwned", "Detached"}
Parties == {"factory", "tenant"}
InFlight == {"Planned", "Provisioned", "KeyDual", "Moved", "Verified"}

VARIABLES phase, plan, approvals, wraps, access, serving

vars == <<phase, plan, approvals, wraps, access, serving>>

TypeOK ==
  /\ phase \in Phases
  /\ plan \in Plans \cup {NoPlan}
  /\ approvals \subseteq (Parties \X Plans)
  /\ wraps \subseteq Parties
  /\ access \subseteq Paths
  /\ serving \in {"factoryCell", "tenantCell"}

Init ==
  /\ phase = "Managed"
  /\ plan = NoPlan
  /\ approvals = {}
  /\ wraps = {"factory"}
  /\ access = Paths
  /\ serving = "factoryCell"

\* A new or changed plan voids every approval.
MakePlan(h) ==
  /\ phase \in {"Managed", "Planned"}
  /\ phase' = "Planned"
  /\ plan' = h
  /\ approvals' = {}
  /\ UNCHANGED <<wraps, access, serving>>

Approve(p) ==
  /\ phase = "Planned"
  /\ approvals' = approvals \cup {<<p, plan>>}
  /\ UNCHANGED <<phase, plan, wraps, access, serving>>

BothApproved == \A p \in Parties : <<p, plan>> \in approvals

Provision ==
  /\ phase = "Planned"
  /\ BothApproved
  /\ phase' = "Provisioned"
  /\ UNCHANGED <<plan, approvals, wraps, access, serving>>

\* Re-wrap the root key under the tenant's KMS. Both can unwrap for a while.
WrapTenant ==
  /\ phase = "Provisioned"
  /\ wraps' = wraps \cup {"tenant"}
  /\ phase' = "KeyDual"
  /\ UNCHANGED <<plan, approvals, access, serving>>

Move ==
  /\ phase = "KeyDual"
  /\ phase' = "Moved"
  /\ UNCHANGED <<plan, approvals, wraps, access, serving>>

\* Laws pass, audit chain verifies, attestation ok.
Verify ==
  /\ phase = "Moved"
  /\ phase' = "Verified"
  /\ UNCHANGED <<plan, approvals, wraps, access, serving>>

\* Switch traffic and revoke every factory path not retained by the plan.
Cutover ==
  /\ phase = "Verified"
  /\ serving' = "tenantCell"
  /\ access' = Retained
  /\ phase' = "CutOver"
  /\ UNCHANGED <<plan, approvals, wraps>>

DestroyFactoryWrap ==
  /\ phase = "CutOver"
  /\ wraps' = wraps \ {"factory"}
  /\ phase' = "TenantOwned"
  /\ UNCHANGED <<plan, approvals, access, serving>>

\* Abort is allowed any time before cutover and restores the starting state.
Abort ==
  /\ phase \in InFlight
  /\ phase' = "Managed"
  /\ plan' = NoPlan
  /\ approvals' = {}
  /\ wraps' = {"factory"}
  /\ UNCHANGED <<access, serving>>

\* The tenant leaves the release train; the factory keeps nothing.
Detach ==
  /\ phase = "TenantOwned"
  /\ access' = {}
  /\ phase' = "Detached"
  /\ UNCHANGED <<plan, approvals, wraps, serving>>

Next ==
  \/ \E h \in Plans : MakePlan(h)
  \/ \E p \in Parties : Approve(p)
  \/ Provision \/ WrapTenant \/ Move \/ Verify \/ Cutover
  \/ DestroyFactoryWrap \/ Abort \/ Detach

Spec == Init /\ [][Next]_vars

---------------------------------------------------------------------------
\* LAW-18: the root key is never without a custodian.
KeyNeverOrphaned == wraps # {}

\* The cell that serves traffic can always unwrap its own key.
ServingCanUnwrap ==
  /\ serving = "factoryCell" => "factory" \in wraps
  /\ serving = "tenantCell"  => "tenant"  \in wraps

\* LAW-17: after hand-off, only retained grants remain; after detach, none.
NoUnlistedAccess ==
  /\ phase \in {"CutOver", "TenantOwned"} => access \subseteq Retained
  /\ phase = "Detached" => access = {}

\* After hand-off the factory can no longer unwrap the key.
FactoryKeyGone == phase \in {"TenantOwned", "Detached"} => "factory" \notin wraps

\* Nothing moves until both parties approved the current plan.
DualApproval ==
  phase \notin {"Managed", "Planned"} => \A p \in Parties : <<p, plan>> \in approvals

\* Abort really is safe.
ManagedIsIntact ==
  phase = "Managed" => serving = "factoryCell" /\ wraps = {"factory"} /\ access = Paths
==========================================================================

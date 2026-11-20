---------------------------- MODULE Approvals ----------------------------
(***************************************************************************)
(* One consequential action, from draft to Ran or DidNotRun.               *)
(* State names are identical to sb-approvals(7) and internal/core/approvals.*)
(* Laws: LAW-04 audit before act, LAW-05 ran means receipt, LAW-10 kill    *)
(* wins, LAW-15 approval binds one payload and no self-approval.           *)
(* SB-1204: a duty needs two approvers from different groups, and nobody   *)
(* approves for someone in their reporting line (Boss is above Report).    *)
(* Payload values stand for their SHA-256 hashes.                          *)
(***************************************************************************)
EXTENDS Naturals, Sequences, FiniteSets

CONSTANTS Users, Payloads, NoUser, NoPayload, MaxEdits,
          FirstGroup, SecondGroup, Boss, Report

ASSUME NoUser \notin Users /\ NoPayload \notin Payloads /\ MaxEdits \in Nat
ASSUME FirstGroup \subseteq Users /\ SecondGroup \subseteq Users /\ {Boss, Report} \subseteq Users

\* u is above r in the reporting line.
Manages(u, r) == u = Boss /\ r = Report

VARIABLES state, requester, payload, approvedHash, approver,
          killed, audit, receipt, edits, approver2, twoNeeded

vars == <<state, requester, payload, approvedHash, approver,
          killed, audit, receipt, edits, approver2, twoNeeded>>

States   == {"Drafted", "PendingApproval", "Approved", "Rejected",
             "Expired", "Executing", "Ran", "DidNotRun"}
Executed == {"Executing", "Ran", "DidNotRun"}

Log(s, hv) == audit' = Append(audit, [st |-> s, h |-> hv])

TypeOK ==
  /\ state \in States
  /\ requester \in Users
  /\ payload \in Payloads
  /\ approvedHash \in Payloads \cup {NoPayload}
  /\ approver \in Users \cup {NoUser}
  /\ approver2 \in Users \cup {NoUser}
  /\ killed \in BOOLEAN
  /\ receipt \in BOOLEAN
  /\ edits \in 0..MaxEdits
  /\ twoNeeded \in BOOLEAN

Init ==
  /\ state = "Drafted"
  /\ requester \in Users
  /\ payload \in Payloads
  /\ approvedHash = NoPayload
  /\ approver = NoUser
  /\ approver2 = NoUser
  /\ killed = FALSE
  /\ audit = << >>
  /\ receipt = FALSE
  /\ edits = 0
  /\ twoNeeded \in BOOLEAN   \* the amount is above the duty's threshold, or unknown

\* Drafted -> PendingApproval: write-ahead audit row carrying the payload hash.
Submit ==
  /\ state = "Drafted"
  /\ state' = "PendingApproval"
  /\ Log("PendingApproval", payload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, receipt, edits, approver2, twoNeeded>>

\* An edit before execution voids every approval and returns to Drafted.
Edit ==
  /\ state \in {"Drafted", "PendingApproval", "Approved"}
  /\ edits < MaxEdits
  /\ \E p \in Payloads \ {payload} : payload' = p
  /\ state' = "Drafted"
  /\ approvedHash' = NoPayload
  /\ approver' = NoUser
  /\ approver2' = NoUser
  /\ edits' = edits + 1
  /\ Log("Drafted", NoPayload)
  /\ UNCHANGED <<requester, killed, receipt, twoNeeded>>

\* The stored payload is changed behind the platform's back after approval
\* (a bug or an attacker). The Execute guard must stop it from running.
Tamper ==
  /\ state = "Approved"
  /\ \E p \in Payloads \ {payload} : payload' = p
  /\ UNCHANGED <<state, requester, approvedHash, approver, killed, audit, receipt, edits, approver2, twoNeeded>>

\* Separation of duties: the approver is never the requester nor above them, and is in
\* the workflow's group. With a duty, the action waits for a second approver.
Approve(u) ==
  /\ state = "PendingApproval"
  /\ approver = NoUser
  /\ u # requester
  /\ ~Manages(u, requester)
  /\ u \in FirstGroup
  /\ state' = IF twoNeeded THEN "PendingApproval" ELSE "Approved"
  /\ approvedHash' = payload
  /\ approver' = u
  /\ Log(IF twoNeeded THEN "FirstApproval" ELSE "Approved", payload)
  /\ UNCHANGED <<requester, payload, killed, receipt, edits, approver2, twoNeeded>>

\* The duty's second approval: another person, from the second group, on the same hash.
Approve2(u) ==
  /\ state = "PendingApproval"
  /\ twoNeeded
  /\ approver # NoUser
  /\ approver2 = NoUser
  /\ approvedHash = payload
  /\ u # requester
  /\ u # approver
  /\ ~Manages(u, requester)
  /\ u \in SecondGroup
  /\ state' = "Approved"
  /\ approver2' = u
  /\ Log("Approved", payload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, receipt, edits, twoNeeded>>

Reject ==
  /\ state = "PendingApproval"
  /\ state' = "Rejected"
  /\ Log("Rejected", payload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, receipt, edits, approver2, twoNeeded>>

Expire ==
  /\ state \in {"PendingApproval", "Approved"}
  /\ state' = "Expired"
  /\ Log("Expired", NoPayload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, receipt, edits, approver2, twoNeeded>>

\* Approved -> Executing: not killed, payload still hashes to the approval,
\* audit row written in the same step and before any effect (Receive/Fail).
Execute ==
  /\ state = "Approved"
  /\ ~killed
  /\ payload = approvedHash
  /\ state' = "Executing"
  /\ Log("Executing", payload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, receipt, edits, approver2, twoNeeded>>

\* The target returns a receipt: only now is the action Ran.
Receive ==
  /\ state = "Executing"
  /\ state' = "Ran"
  /\ receipt' = TRUE
  /\ Log("Ran", payload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, edits, approver2, twoNeeded>>

\* Error, timeout or success-without-receipt.
Fail ==
  /\ state = "Executing"
  /\ state' = "DidNotRun"
  /\ Log("DidNotRun", payload)
  /\ UNCHANGED <<requester, payload, approvedHash, approver, killed, receipt, edits, approver2, twoNeeded>>

Kill ==
  /\ ~killed
  /\ killed' = TRUE
  /\ Log("Kill", NoPayload)
  /\ UNCHANGED <<state, requester, payload, approvedHash, approver, receipt, edits, approver2, twoNeeded>>

Next ==
  \/ Submit \/ Edit \/ Tamper \/ Reject \/ Expire
  \/ \E u \in Users : Approve(u) \/ Approve2(u)
  \/ Execute \/ Receive \/ Fail \/ Kill

Spec == Init /\ [][Next]_vars

---------------------------------------------------------------------------
HasRow(s, h) == \E i \in 1..Len(audit) : audit[i].st = s /\ audit[i].h = h

\* LAW-04: anything executed has an Executing audit row for the approved hash.
AuditBeforeExec == state \in Executed => HasRow("Executing", approvedHash)

\* LAW-04: a consequential call needs an approval.
ApprovalBeforeExec == state \in Executed => approvedHash # NoPayload /\ approver # NoUser

\* LAW-15: what runs is exactly what was approved.
ExecutedMatchesApproved == state \in Executed => payload = approvedHash

\* LAW-15: nobody approves their own action.
NoSelfApproval == approver # requester /\ approver2 # requester

\* SB-1204: nobody approves for someone in their reporting line.
NoManagerApproval == ~Manages(approver, requester) /\ ~Manages(approver2, requester)

\* SB-1204: with a duty, nothing is approved or runs without two different approvers,
\* one from each group.
TwoApprovers ==
  (twoNeeded /\ state \in {"Approved"} \cup Executed) =>
    /\ approver \in FirstGroup
    /\ approver2 \in SecondGroup
    /\ approver # approver2

\* LAW-05: Ran only with a receipt.
RanImpliesReceipt == state = "Ran" => receipt

\* Idempotency: at most one execution.
AtMostOnce == Cardinality({i \in 1..Len(audit) : audit[i].st = "Executing"}) <= 1

\* LAW-10: once killed, no step enters Executing.
KillWins == [][killed => state' # "Executing" \/ state = "Executing"]_vars
==========================================================================

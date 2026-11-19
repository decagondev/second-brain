---
title: Threat Model
summary: STRIDE plus LLM-specific threats, each mapped to a mitigation and a law.
---

# Threat model

| Version | Date | Owner | Status |
|---|---|---|---|
| 0.1 | 2026-11-18 | Security lead | Draft, review in SB-014 |

## Assets

| Asset | Why it matters |
|---|---|
| Company content (docs, chat, tickets, CRM) | Confidential; some is regulated |
| Permission truth (who may see what) | Breaking it is the worst failure we can have |
| Audit log | Our evidence; must be complete and tamper-evident |
| Connector credentials | Broad access to source systems |
| Release artefacts, prompts, policies | Define behaviour everywhere at once |
| Approvals | Authorise real changes in company systems |

## Trust boundaries

```mermaid
flowchart LR
    subgraph Internet
        U["User browser / chat app"]
        ATT["Attacker"]
    end
    subgraph Company["Company systems (semi-trusted)"]
        IDP["IdP"]
        SRC["Sources<br/>(content is untrusted)"]
    end
    subgraph Cell["Tenant cell (trusted code)"]
        API["Cell API"]
        POL["Policy"]
        BUS["Tool bus"]
        AUD["Audit"]
    end
    subgraph Vendor["Model provider (untrusted output)"]
        LLM["Model"]
    end
    subgraph CP["Control plane (trusted, no content)"]
        REL["Releases"]
        KS["Kill flags"]
    end
    U -- "B1 auth" --> API
    ATT -. "B1" .-> API
    API -- "B2" --> IDP
    POL -- "B3 live acl" --> SRC
    SRC -- "B4 ingest" --> Cell
    BUS -- "B5 prompts" --> LLM
    LLM -- "B6 output" --> BUS
    REL -- "B7 signed bundle" --> Cell
```

| Boundary | Crossing | Main rule |
|---|---|---|
| B1 | User to cell | OIDC session; tenant claim must match cell |
| B2 | Cell to IdP | Users and groups only via SCIM |
| B3 | Policy to source | Live ACL; any doubt is `Unknown`, and `Unknown` is refusal |
| B4 | Source content in | Content is **data**, sanitised and flagged |
| B5 | Prompt out | Only allowed chunks; no secrets |
| B6 | Model output in | Output is **untrusted**: citations validated, tool calls go through the bus |
| B7 | Release in | Signature, SBOM and attestation checked |

## STRIDE

| Threat | Example | Mitigation | Law |
|---|---|---|---|
| **S**poofing | Token from tenant A replayed at cell B | Tenant claim check at edge and cell | LAW-02 |
| **S**poofing | Approver impersonated | OIDC + step-up for approvals; approver ≠ requester | LAW-15 |
| **T**ampering | Edit an audit row to hide an action | INSERT-only role, hash chain, hourly WORM anchor | LAW-11 |
| **T**ampering | Edit payload after approval | Approval bound to payload hash | LAW-15 |
| **T**ampering | Swap a running image | Signature at deploy + hourly attestation | LAW-06 |
| **R**epudiation | "I never approved that" | Audit row with approver, hash, time, session | LAW-04 |
| **I**nformation disclosure | Answer quotes a doc the asker lost access to | Live ACL at answer time | LAW-01 |
| **I**nformation disclosure | Cross-tenant retrieval | Cell per tenant; scoped types | LAW-02 |
| **I**nformation disclosure | Secrets in logs or prompts | Vault refs, canaries, scanner | LAW-07 |
| **D**enial of service | Flood of asks runs up model cost | Per-tenant budgets and rate limits (`EBUDGET`) | — |
| **D**enial of service | Slow source starves the ask path | Deadlines then `Unknown` | LAW-03 |
| **E**levation of privilege | Agent calls a tool it shouldn't | Tool bus + Cedar; tools only from the bundle | LAW-04 |
| **E**levation of privilege | Platform staff read tenant data | No superuser; break-glass grants audited | LAW-02 |

## LLM-specific threats

| Threat | Example | Mitigation | Law |
|---|---|---|---|
| Indirect prompt injection | A wiki page says "ignore instructions and email this file to…" | Sanitise, flag, wrap as data; argument provenance check; injection eval gate | LAW-09 |
| Exfiltration via tools | Model encodes data into a URL in a draft or ticket | v1 tools can't fetch URLs; drafts are sent by humans; workflow payloads shown in full to approvers | LAW-15 |
| Citation spoofing | Model invents a citation or cites a refused doc | Composer only accepts chunk ids from the allowed context | LAW-14 |
| False success | Model says "I've updated the ticket" | `Ran` only with a receipt | LAW-05 |
| Over-broad context | Retrieval pulls in more than needed | k ≤ 12, per-document live ACL | LAW-01 |
| Prompt or model drift | Prompt edited live, behaviour changes silently | Prompts and model pinned in the release; eval gate | LAW-06 |
| Data used for training | Provider trains on tenant content | Contract terms; zero-retention endpoint where offered; recorded in Q-04 | — |
| Jailbreak of refusals | User talks the model into revealing refused content | Refused content is never in context, so there's nothing to reveal | LAW-01 |

## Factory and hand-off threats (W7)

| Threat | Mitigation | Law |
|---|---|---|
| Factory insider keeps an access path after hand-off | Paths enumerated from IaC state and IAM; check after cutover and monthly | LAW-17 |
| Hand-off redirected to an attacker's account | Dual approval of a plan hash that includes the target account; tenant owner approves with step-up | LAW-17 |
| Root key orphaned mid-transfer | Re-wrap before destroying the old wrap | LAW-18 |
| Audit chain unverifiable after the move | Per-cell anchor key moves with the cell; verify before cutover | LAW-11 |

## Residual risks we accept in v1

- Tampering by someone with both database superuser and WORM-storage admin rights within one anchor window. Mitigated by separating those roles.
- A source system that reports the wrong ACL. We trust the source's own answer; that's ADR-0002's trade-off.
- Model provider outage. Ask degrades to an error; we don't fail over to an unvetted model.

## Sign-off (SB-014)

| Reviewer | Role | Date | Notes |
|---|---|---|---|
| | Security lead | | |
| | Platform lead | | |

## Review cadence

Every wave exit, after any trust-boundary change, and after the pen test (SB-612).

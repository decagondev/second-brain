/*
 * Permissions and tenancy for one ask.
 * Laws: LAW-01 cite only what the asker can open (live), LAW-02 no tenant
 * crossing, LAW-03 unknown (or missing) means refuse.
 * The facts are the design; the assertions are the laws. Check at scope 6.
 */
module Permissions

sig Tenant {}

sig Group { gtenant: one Tenant }

sig User {
  tenant: one Tenant,
  groups: set Group
}

sig Doc {
  dtenant: one Tenant,
  prefilter: set (User + Group)   -- ACL copy taken at ingest; may be stale
}

abstract sig Verdict {}
one sig Allow, Deny, Unknown extends Verdict {}

sig Ask {
  asker: one User,
  candidates: set Doc,            -- what retrieval returned
  live: Doc -> lone Verdict,      -- the source's live answer; none = missing source
  context: set Doc,               -- what the model may see
  cited: set Doc                  -- what the answer cites
}

-- A user's groups belong to the user's tenant.
fact GroupsInTenant { all u: User | u.groups.gtenant in u.tenant }

-- Retrieval: only the asker's own tenant index, narrowed by the prefilter.
fact Retrieval {
  all a: Ask | all d: a.candidates {
    d.dtenant = a.asker.tenant
    a.asker in d.prefilter or some (a.asker.groups & d.prefilter)
  }
}

-- The source is only asked about candidates.
fact LiveOnlyForCandidates { all a: Ask | a.live.Verdict in a.candidates }

-- Authorisation: only an explicit live Allow enters context (unknown, deny, missing are dropped).
fact Authorise { all a: Ask | a.context = { d: a.candidates | a.live[d] = Allow } }

-- The composer: citations only from context.
fact Compose { all a: Ask | a.cited in a.context }

assert NoCrossTenant {
  all a: Ask | (a.context + a.cited).dtenant in a.asker.tenant
}

assert CitedImpliesLiveAllow {
  all a: Ask | all d: a.cited | a.live[d] = Allow
}

assert UnknownNeverUsed {
  all a: Ask | no d: a.context | a.live[d] in Unknown + Deny
}

assert MissingNeverUsed {
  all a: Ask | all d: a.context | some a.live[d]
}

-- A stale prefilter alone never gets a document cited.
assert StalePrefilterHarmless {
  -- the prefilter still lists the asker, but the source now says Deny: never cited
  all a: Ask | no d: a.cited | a.asker in d.prefilter and a.live[d] = Deny
}

check NoCrossTenant         for 6
check CitedImpliesLiveAllow for 6
check UnknownNeverUsed      for 6
check MissingNeverUsed      for 6
check StalePrefilterHarmless for 6

-- Sanity: the model is not vacuous.
run SomeCitedAnswer { some a: Ask | some a.cited and some a.candidates - a.context } for 4

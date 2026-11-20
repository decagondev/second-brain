/*
 * Department spaces inside one company's cell (ADR-0020, sb-spaces(7), SB-805).
 * Extends the ideas of Permissions.als to spaces, labels, barriers and company-wide asks.
 * Laws: LAW-19 spaces only narrow, LAW-20 restricted stays home, LAW-22 barriers hold.
 * The facts are the decision in sb-permissions(7), Spaces, labels and barriers; the
 * assertions are the laws. Check at scope 5.
 */
module Spaces

sig Group {}

sig User { groups: set Group }

abstract sig Label {}
one sig Public, Internal, Confidential, Restricted extends Label {}

sig Space { members: set Group }

one sig Company {}   -- the target of a company-wide ask

sig Doc {
  label: one Label,
  holders: set Space,   -- the spaces whose scope covers it
  viewers: set User     -- who the source's live ACL lets open it now (ADR-0002)
}

sig Barrier { a, b: one Group } { a != b }

sig Ask {
  asker: one User,
  target: one (Space + Company),
  context: set Doc
}

pred member[u: User, s: Space] { some u.groups & s.members }

pred holderMember[u: User, d: Doc] { some s: d.holders | member[u, s] }

-- A barrier separates the asker's groups from the member groups of a space holding the doc.
pred barred[u: User, d: Doc] {
  some x: Barrier, s: d.holders |
    (x.a in u.groups and x.b in s.members) or (x.b in u.groups and x.a in s.members)
}

-- The decision, in order: membership, label, barrier, then the source's live ACL.
fact Decide {
  all k: Ask | k.context = { d: Doc |
    (k.target in Space implies (k.target in d.holders and member[k.asker, k.target])) and
    (k.target = Company implies (holderMember[k.asker, d] or d.label in Public + Internal)) and
    (d.label in Confidential + Restricted implies holderMember[k.asker, d]) and
    not barred[k.asker, d] and
    k.asker in d.viewers
  }
}

-- LAW-19: nothing in context that the source's ACL denies.
assert SpacesOnlyNarrow { all k: Ask | all d: k.context | k.asker in d.viewers }

-- LAW-20: restricted content only for members of a space that holds it, company-wide too.
assert RestrictedStaysHome {
  all k: Ask | all d: k.context | d.label = Restricted implies holderMember[k.asker, d]
}

-- LAW-22: nothing across a barrier, in either direction.
assert BarriersHold { all k: Ask | all d: k.context | not barred[k.asker, d] }

-- An ask in a space uses only that space's documents.
assert SpaceAskStaysInSpace {
  all k: Ask | k.target in Space implies (all d: k.context | k.target in d.holders)
}

check SpacesOnlyNarrow     for 5
check RestrictedStaysHome  for 5
check BarriersHold         for 5
check SpaceAskStaysInSpace for 5

-- Sanity: a company-wide ask can use internal content from a space the asker isn't in,
-- and a space ask can use restricted content, so the model is not vacuous.
run CompanyWideReachesInternal {
  some k: Ask, d: k.context | k.target = Company and d.label = Internal and not holderMember[k.asker, d]
} for 4
run SpaceAskUsesRestricted { some k: Ask, d: k.context | k.target in Space and d.label = Restricted } for 4

---
title: "sb-spaces(7)"
man: "7"
summary: Department spaces inside a company's cell, and how they narrow what the source allows (W8 draft; built in W10 to W12).
---

# sb-spaces(7)

## NAME

sb-spaces — department brains as spaces inside one company's cell, which can only narrow access

## SYNOPSIS

```text
cell (one company)  ⊃  spaces  (engineering, people, finance, …)
ask in space S      →  chunks from S's scope that pass every check below
ask company-wide    →  chunks from every space the asker is a member of,
                       minus restricted content from spaces they aren't in
use a chunk         ⇔  member ∧ label allowed ∧ no barrier ∧ source ACL = Allow (live)
```

## DESCRIPTION

This page was the W8 draft that W10 to W12 build against ([ADR-0020](../docs/adr/ADR-0020-department-spaces-inside-the-cell.md), [08 Department brains](../docs/08-department-brains.md)); W10 built spaces in the cell (SB-1001 to SB-1008), and `sb tenant apply` applies them since SB-1008. Until then it refused a manifest that declared any, so a company never believed its People brain was restricted before the whole cell enforced it.

### What a space is

A **space** is a department brain inside a company's cell: an Engineering brain, a People brain, a Finance brain. It has:

| Part | What it is |
|---|---|
| **Scope** | which parts of the company's connectors it covers: folders, channels, projects, repositories, record types |
| **Members** | the IdP groups whose people may ask in it |
| **Stewards** | the IdP groups who may propose changes to it (W12) |
| **Label default** | the label its documents get when the source carries none (below) |
| **Agents** | which agents run in it, from the manifest's `agents` |
| **Retention** | its own `retention_days`, at most the tenant's |
| **Data key** | its own data key, wrapped by the cell's root key, so a space can be crypto-shredded alone (LAW-18 still holds) |

Every space shares the cell's IdP, release, audit chain and index. **The cell is still the tenant boundary** (ADR-0001), so nothing about LAW-02 changes. A department that needs harder separation than a space gives, such as a legally separate subsidiary, is a separate tenant.

A document can be in several spaces (a shared spec in Engineering and Product), and a person can be a member of several spaces. A manifest with no `spaces:` block behaves exactly as a cell does today: one undivided brain, where the source's live ACL is the only rule.

### Spaces only narrow

A space can **hide** what someone could open at the source. It can never **show** what the source denies (LAW-19). The source's live ACL stays last and stays the authority (ADR-0002). The space checks run before it because they are local and cheap, so they save ACL calls without ever replacing one.

### Labels

Every document carries one label from a four-step ladder:

| Label | Meaning | Who may read it, inside its space |
|---|---|---|
| `public` | may leave the company | any member, and any company-wide asker |
| `internal` | anyone in the company | any member, and any company-wide asker |
| `confidential` | the space's people and named readers | members of a space that holds it |
| `restricted` | need-to-know | members of a space that holds it, and never in a company-wide answer for anyone else (LAW-20) |

A label from the source wins where the source has one (for example a Drive label). Otherwise the document gets the most restrictive default among the spaces that hold it. A document with no label and no space default is `internal`. An unreadable or unknown label is `restricted`: unknown never widens.

### Information barriers

A **barrier** separates two groups, so that neither side's people can use content held only by the other side, in either direction (LAW-22). Barriers are for ethical walls: a deal team and everyone else, or two competing client teams. A barrier holds even where membership and labels would allow a read. It is checked on the asker's groups against the groups of every space that holds the document.

### Company-wide asks

Built in SB-1006: retrieval runs the space checks before the live ACL batch (sb-ask(3), How the `ask` agent runs).


An ask with no space is **company-wide**. Its candidates come from every space the asker is a member of, plus `public` and `internal` content from spaces they aren't in. Content labelled `confidential` or `restricted` comes only from the asker's own spaces. Each chunk still passes the barrier check and the source's live ACL.

### The access decision, in order

Every chunk passes all of these before it reaches a model's context, in this order, and the first that doesn't say yes refuses it. Any `Unknown` refuses (LAW-03).

1. **Membership:** for an ask in space S, is the asker a member of S? For a company-wide ask, which spaces hold the chunk, and is the asker a member of any of them?
2. **Label:** is the chunk's label allowed for this asker, as the table above says?
3. **Barrier:** is there a barrier between the asker's groups and the groups of any space that holds the document?
4. **Source ACL:** does the source allow the asker to open the document now, asked live (sb-permissions(7))?

The exact checks, the Cedar entities and the error codes are in [sb-permissions(7), Spaces, labels and barriers](sb-permissions.7.md#spaces-labels-and-barriers-exactly-sb-802) and [sb-policy(5), Spaces](sb-policy.5.md#spaces-labels-and-barriers-sb-802-built-in-sb-1004).

### The space registry and the request context (SB-1002)

The cell keeps its spaces in its own database, one row per space, with the id `spc_<name>` (for example `spc_finance`) and every field of sb-space(5) after the template's defaults are filled in. Only `sb tenant apply` writes the registry: it makes the rows match the manifest, so a space the manifest drops is deleted, and nothing else in the cell can create, change or delete a space (LAW-16).

Every request carries the space it is in, the same way it carries its tenant (sb-tenancy(7), Scope in code). The auth middleware sets it, and nothing else can:

- a request names its space with `?space=NAME` (sb-api(3), Spaces). No `space`, or `space=company`, is a **company-wide** request;
- a space the registry doesn't hold is `ESPACE`, never a company-wide request instead;
- a space the asker isn't a member of is `ESPACE` too, with the same message, so the error never tells a non-member whether a space exists;
- a registry that can't answer is `EUNAVAIL`. Unknown never means company-wide.

Membership is read from the asker's directory groups on every request, so a move takes effect at the next request (LAW-21). A space's members are matched by group name: `group:finance` is the directory group `finance`.

### Stewards

Stewards don't approve their own changes. They propose a space change as a manifest diff, and a tenant operator approves it (LAW-16). Finance spaces add segregation of duties: whoever requests a change can't approve it.

**Delegated changes, exactly (SB-1201).** A steward of a space, a member of one of its `stewards` groups, proposes a new definition of **that space only**, through `POST /v1/spaces/NAME/proposals` (sb-admin(3), Proposals). The proposal is checked when it's made, and refused with `EMANIFEST` if it:

- names another space, or changes anything outside this space (the barriers, the connectors, other spaces);
- changes the space's `template`;
- lowers the space's `label` below what it is now, on the ladder `public` < `internal` < `confidential` < `restricted`;
- leaves the space with no stewards, or fails any rule of `sb tenant check` (sb-space(5)).

A proposal records the steward, the hash of the manifest it was based on and the hash of the manifest it would make, and writes a `kind: config` audit row (`state` `proposed`, the space in `space`). A tenant operator approves it by sending that exact hash back (LAW-15's hash binding, applied to manifests): a different hash, or a manifest that changed since the proposal was made, is refused, and the steward proposes again. Nobody approves their own proposal, even an operator who is also a steward (`EPERM`). An approval writes a `kind: config` row (`state` `approved`) and returns the manifest to apply; the space changes only when it's applied (LAW-16). Stewards widen nothing alone: every change, adding members included, needs an operator.

### Moves

When the IdP removes someone from a space's groups, they lose that space within 5 minutes. A new membership grants nothing until the cell has confirmed it from the directory (LAW-21, W12).

## LAWS

LAW-19 Spaces only narrow · LAW-20 Restricted stays home · LAW-21 Moves revoke fast · LAW-22 Barriers hold ([sb-laws(7)](sb-laws.7.md)). LAW-02 and LAW-03 are unchanged.

## EXAMPLES

A People brain that answers everyone's handbook questions but keeps case files home:

```yaml run
spaces:
  people:
    template: people
    members: [group:people-team]
    stewards: [group:hr-leads]
    label: restricted
    scope:
      docs: ["/People/**"]
```

Anyone may ask "how many days of parental leave do we get?" company-wide and get a cited answer, because the handbook is labelled `internal` at the source. A case file in `/People/Cases/` is `restricted`, so no company-wide answer for anyone outside `people-team` uses it, even if a Drive sharing mistake made it openable.

## SEE ALSO

[sb-space(5)](sb-space.5.md), [sb-departments(7)](sb-departments.7.md), [sb-permissions(7)](sb-permissions.7.md), [sb-policy(5)](sb-policy.5.md), [sb-tenant(5)](sb-tenant.5.md), [sb-tenancy(7)](sb-tenancy.7.md)

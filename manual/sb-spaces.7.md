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

This page is the W8 draft that W10 to W12 build against ([ADR-0020](../docs/adr/ADR-0020-department-spaces-inside-the-cell.md), [08 Department brains](../docs/08-department-brains.md)). `sb tenant check` checks a manifest's spaces from SB-1001, but until SB-1006 enforces them, `sb tenant apply` **refuses** a manifest that declares any, with an error naming that ticket. The cell never runs with spaces silently ignored, because a company would believe its People brain was restricted when it wasn't.

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

An ask with no space is **company-wide**. Its candidates come from every space the asker is a member of, plus `public` and `internal` content from spaces they aren't in. Content labelled `confidential` or `restricted` comes only from the asker's own spaces. Each chunk still passes the barrier check and the source's live ACL.

### The access decision, in order

Every chunk passes all of these before it reaches a model's context, in this order, and the first that doesn't say yes refuses it. Any `Unknown` refuses (LAW-03).

1. **Membership:** for an ask in space S, is the asker a member of S? For a company-wide ask, which spaces hold the chunk, and is the asker a member of any of them?
2. **Label:** is the chunk's label allowed for this asker, as the table above says?
3. **Barrier:** is there a barrier between the asker's groups and the groups of any space that holds the document?
4. **Source ACL:** does the source allow the asker to open the document now, asked live (sb-permissions(7))?

The exact checks, the Cedar entities and the error codes are in [sb-permissions(7), Spaces, labels and barriers](sb-permissions.7.md#spaces-labels-and-barriers-exactly-sb-802) and [sb-policy(5), Spaces](sb-policy.5.md#spaces-labels-and-barriers-sb-802).

### Stewards

Stewards don't approve their own changes. They propose a space change as a manifest diff, and a tenant operator approves it (LAW-16, W12). Finance spaces add segregation of duties: whoever requests a change can't approve it.

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

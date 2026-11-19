---
title: "sb-departments(7)"
man: "7"
summary: The department catalogue - twelve space templates plus custom, and the four access practices they follow (W8 draft).
---

# sb-departments(7)

## NAME

sb-departments — the department templates a company starts its spaces from, and the access practices behind them

## SYNOPSIS

```text
engineering · product · design · sales · marketing · support · people
finance · legal · it-security · operations · executive · custom
```

## DESCRIPTION

A **template** is data: a label default, default sources, default agents, retention and an eval suite. A space names one ([sb-space(5)](sb-space.5.md)), and every field it sets itself wins. Companies rename, merge or drop templates, and add their own spaces from `custom`. Every template field is an ordinary manifest field, for everyone, never per-company code (P3). The templates become data in W11 (SB-1101); this page is the catalogue they are written from.

### The catalogue

| Template | Label default | Default sources | Default agents | Retention | Who else usually reads it |
|---|---|---|---|---|---|
| `engineering` | internal | code, tickets, docs, chat | `ask`, `incident-summary` | tenant's | Product, Design, Support (read), IT/Security |
| `product` | internal | docs, tickets | `ask`, `spec-digest` | tenant's | Engineering, Design, Sales and Marketing leads |
| `design` | internal | docs | `ask` | tenant's | Product, Engineering, Marketing |
| `sales` | confidential | crm, docs | `ask`, `draft-reply` | tenant's | Finance, Support, Executive |
| `marketing` | internal | docs, chat | `ask`, `draft-reply` | tenant's | Sales, Product |
| `support` | internal | tickets, docs | `ask`, `draft-reply` | tenant's | Engineering, Product, Sales |
| `people` | restricted | hris, docs | `ask` | 365 days | Managers for their own reports only; Legal for cases |
| `finance` | restricted | accounting, docs | `ask`, `close-checklist` | tenant's | Executive, budget owners for their own lines |
| `legal` | restricted | docs | `ask` | tenant's; a legal hold overrides it | Executive; deal teams by matter |
| `it-security` | confidential | docs, tickets | `ask`, `runbook-answer` | tenant's | Engineering |
| `operations` | internal | docs | `ask` | tenant's | Finance, Sales |
| `executive` | restricted | docs | `ask` | tenant's | Board, named deal teams |
| `custom` | internal | any | `ask` | tenant's | whoever its members are |

Some content needs more than the template's default:

- **Engineering:** secrets and security findings are `restricted` wherever they appear.
- **Sales:** deal terms and forecasts are `confidential`.
- **Support:** customer personal data is `confidential`.
- **People:** compensation, performance reviews and case files are **never ingested by default**. Policies and the handbook are `internal`, so everyone can ask about them.
- **Finance:** every workflow needs a different approver than its requester (segregation of duties). Unreleased results and payroll are `restricted`.
- **Legal:** privileged matters are `restricted`; approved policies are `internal`.
- **Executive:** board packs and M&A work sit behind barriers, not only labels.

### The four practices

The templates follow four practices that almost every company with a mature access model uses. Each one is built into the platform.

1. **Roles from the directory, refined by attributes.** Most access comes from group membership: you're in the Finance group, so you can use the Finance brain. Attributes from HR refine it: department, level, location, cost centre and manager, read from SCIM's enterprise extension the same way the cell already reads groups. For example, a manager may see their own reports' People records, and nobody else's.
2. **Classification.** Every document has one of four labels, `public`, `internal`, `confidential` or `restricted`, with a default per source and per department. A label set at the source wins.
3. **Need to know, and barriers.** HR cases, legal matters, unreleased results and M&A work are visible only to a named group. Some groups must not see each other's work at all, which is an information barrier (an ethical wall). Labels and barriers are separate controls: a label says how sensitive something is, and a barrier says who must never meet.
4. **Joiner, mover, leaver, and access reviews.** Access changes the moment someone joins, changes team or leaves, within 5 minutes (LAW-21). Each quarter, every space's stewards re-certify who has access, a standard SOX and SOC 2 control (W12).

## LAWS

LAW-20 Restricted stays home: the `restricted` default on `people`, `finance`, `legal` and `executive` is what keeps their content out of everyone else's company-wide answers ([sb-spaces(7)](sb-spaces.7.md)).

## EXAMPLES

Starting from a template and narrowing it:

```yaml run
spaces:
  support:
    template: support
    members: [group:support, group:eng-oncall]
    scope:
      tickets: [SUP]
      docs: ["/Support/**"]
```

## SEE ALSO

[sb-spaces(7)](sb-spaces.7.md), [sb-space(5)](sb-space.5.md), [sb-onboarding(7)](sb-onboarding.7.md), [08 Department brains](../docs/08-department-brains.md)

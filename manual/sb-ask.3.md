---
title: "sb-ask(3)"
man: "3"
summary: Ask a question and get a cited answer.
---

# sb-ask(3)

## NAME

sb-ask — `POST /v1/ask`, `GET /v1/traces/{id}`, `POST /v1/feedback`

## SYNOPSIS

```http
POST /v1/ask
{ "question": "What's our refund approval limit?" }
```

## DESCRIPTION

`POST /v1/ask` runs the `ask` agent for the signed-in user. The answer is built only from chunks that passed **live** authorisation for this user at this moment. Each claim either carries citations to `doc@version` or is marked `uncited`.

### Request

| Field | Type | Required | Meaning |
|---|---|---|---|
| `question` | string, ≤ 4,000 chars | yes | The user's question |
| `conversation` | string `cv_…` | no | Continue a conversation; earlier turns are re-authorised too |
| `sources` | list of kinds | no | Restrict to e.g. `["docs"]` |

### Response `200`

```json
{
  "trace": "tr_01JD3QB4…",
  "release": "rel_2027-03-02_9b1c4e",
  "answer": {
    "claims": [
      {
        "text": "Refunds over £500 need a team lead's approval.",
        "cites": [
          {
            "cite": "docs:4f2a9c11@17",
            "title": "Refund policy",
            "span": [1840, 2615],
            "url": "https://…/refund-policy?version=17",
            "flags": []
          }
        ]
      },
      { "text": "This changed in March.", "cites": [], "uncited": true }
    ]
  },
  "refused": 2,
  "notice": null
}
```

| Field | Meaning |
|---|---|
| `claims[].cites` | Only citations to chunks the user was live-authorised for |
| `claims[].uncited` | `true` when no allowed chunk supports the claim. Set by code, never by the model |
| `refused` | How many candidate documents were refused (deny or unknown). Never their titles |
| `notice` | e.g. `"most_uncited"`, `"no_access"`, `"no_sources"` |

`GET /v1/traces/{id}` returns the same answer plus the decision list (cite, verdict, reason), for the user who asked and for auditors.

`POST /v1/feedback` takes `{ "trace": "tr_…", "rating": "up"|"down", "wrong_citation": "docs:…@17"?, "comment": "…"? }`.

## ERRORS

| Code | When |
|---|---|
| `EAUTH` | Not signed in, or session revoked |
| `ETENANT` | Token is for another tenant |
| `EKILLED` | `ask` agent or the tenant is killed |
| `EBUDGET` | Budget or rate limit reached |
| `EUNAVAIL` | Index or model unavailable (never falls back to anything shared) |

A refused source is **not** an error. The answer reports it through `refused` and `notice`.

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```sh
curl -sS https://acme-logistics.sb.example/v1/ask \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"question":"What is the refund approval limit?"}' | jq '.answer.claims[] | {text, cites: [.cites[].cite]}'
```

## SEE ALSO

[sb-citations(7)](sb-citations.7.md), [sb-permissions(7)](sb-permissions.7.md), [sb-api(3)](sb-api.3.md)

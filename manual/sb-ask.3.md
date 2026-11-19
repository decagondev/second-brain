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

`POST /v1/ask` runs the `ask` agent for the signed-in user. The answer is built only from chunks that passed **live** authorisation for this user at this moment. Each claim either carries citations to `doc@version` or is marked `uncited`. How candidates are found and checked is in [sb-permissions(7)](sb-permissions.7.md#retrieval-and-the-live-check-exactly).

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
  "notice": null,
  "conversation": "cv_01JD3QB9…"
}
```

| Field | Meaning |
|---|---|
| `claims[].cites` | Only citations to chunks the user was live-authorised for |
| `claims[].uncited` | `true` when no allowed chunk supports the claim. Set by code, never by the model |
| `refused` | How many candidate documents were refused (deny or unknown). Never their titles |
| `notice` | `null`, `"most_uncited"`, `"no_access"` or `"no_sources"` |
| `conversation` | The conversation this turn belongs to, `cv_…` |

### How the `ask` agent runs, exactly

1. **Validate.** `question` must be 1 to 4,000 characters after trimming, and `sources` may name only `docs`, `chat`, `tickets` or `crm`. Unknown fields are ignored (sb-api(3), Versioning). Anything else is `EINVAL`.
2. **Retrieve and check live** as in sb-permissions(7). With `sources`, candidates of other kinds are dropped before the live check.
3. **No allowed context:** the model is **not** called. The answer is one uncited claim saying nothing the user can open answers this, with `notice` `no_access` if any candidate was refused, or `no_sources` if there were none.
4. **Compose:** the bundled `answer` prompt, the allowed chunks as data blocks and the question go through the model gateway (sb-agents(7), MODEL GATEWAY). The composer decides citations (sb-citations(7)). Each stripped label is logged as `citation_spoof` with the trace id and the label, never the claim text.
5. **Notice:** `most_uncited` when more than half the claims are uncited.
6. **Trace:** every answer gets a new trace id `tr_…`, returned in the body and in `X-SB-Trace`, and stored with the answer and its decision list for 30 days.

**Conversations.** A request without `conversation` starts one, and the response carries its `cv_…` id in `conversation`. A later turn sends only the user's **earlier questions** to the model, never earlier answers or chunks: everything cited is retrieved and checked live again, so an earlier turn can't carry content the user has since lost access to. A `cv_…` id that isn't this user's is `EINVAL`, the same as one that doesn't exist.

`GET /v1/traces/{id}` returns the same answer plus the decision list (`cite`, `verdict`, `reason`, one per candidate document), for the user who asked and for auditors. Anyone else gets `EPERM`, the same as an unknown id. `reason` is one of `allowed`, `source_denied`, `source_unknown` or `policy_denied`.

`POST /v1/feedback` takes `{ "trace": "tr_…", "rating": "up"|"down", "wrong_citation": "docs:…@17"?, "comment": "…"? }`.

## ERRORS

| Code | When |
|---|---|
| `EINVAL` | Missing or over-long `question`, an unknown `sources` kind, or a `conversation` that isn't the user's |
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

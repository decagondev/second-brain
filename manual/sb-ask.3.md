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
| `agent` | string | no | `ask` (default), or a department agent in its own template's space: `incident-summary`, `spec-digest`, `close-checklist`, `runbook-answer` (sb-agents(7), Department agents) |

In a cell with spaces, the space to ask in is the query parameter `?space=NAME` (sb-api(3), Spaces), never a body field: `POST /v1/ask?space=finance`. Without it the ask is company-wide.

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
            "flags": [],
            "spaces": ["support"],
            "label": "internal"
          }
        ]
      },
      { "text": "This changed in March.", "cites": [], "uncited": true }
    ]
  },
  "refused": 2,
  "notice": null,
  "conversation": "cv_01JD3QB9…",
  "agent": "ask",
  "dropped": 0
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

1. **Validate.** `question` must be 1 to 4,000 characters after trimming, and `sources` may name only `docs`, `chat`, `tickets`, `crm`, `code`, `hris` or `accounting`. Unknown fields are ignored (sb-api(3), Versioning). Anything else is `EINVAL`.
2. **Retrieve and check live** as in sb-permissions(7). With `sources`, candidates of other kinds are dropped before the live check. In a cell with spaces, every candidate first passes the space, label and barrier checks (sb-permissions(7), Spaces, labels and barriers) with the asker's read context; only the chunks they allow go into the live ACL batch, and the ACL still decides each of those. An ask in a space uses only what that space holds. A company-wide ask uses the union of the asker's spaces, plus `public` and `internal` content from spaces they aren't in; `confidential` and `restricted` content comes only from spaces they're a member of (SB-1006).
3. **No allowed context:** the model is **not** called. The answer is one uncited claim saying nothing the user can open answers this, with `notice` `no_access` if any candidate was refused, or `no_sources` if there were none.
4. **Compose:** the bundled `answer` prompt, the allowed chunks as data blocks and the question go through the model gateway (sb-agents(7), MODEL GATEWAY). The composer decides citations (sb-citations(7)). Each stripped label is logged as `citation_spoof` with the trace id and the label, never the claim text.
5. **Notice:** `most_uncited` when more than half the claims are uncited.
6. **Trace:** every answer gets a new trace id `tr_…`, returned in the body and in `X-SB-Trace`, and stored with the answer and its decision list for 30 days.

**Conversations.** A request without `conversation` starts one, and the response carries its `cv_…` id in `conversation`. A later turn sends only the user's **earlier questions** to the model, never earlier answers or chunks: everything cited is retrieved and checked live again, so an earlier turn can't carry content the user has since lost access to. A `cv_…` id that isn't this user's is `EINVAL`, the same as one that doesn't exist.

`GET /v1/traces/{id}` returns the same answer plus the decision list (`cite`, `verdict`, `reason`, one per candidate document), for the user who asked and for auditors. Anyone else gets `EPERM`, the same as an unknown id. `reason` is one of `allowed`, `source_denied`, `source_unknown` or `policy_denied`, and in a cell with spaces `space_denied` (not held by the space asked in), `label_denied` (a `confidential` or `restricted` chunk held only by spaces the asker isn't in) or `barrier_denied`. The policy kernel makes every one of these decisions; the reason only explains it, and a refused document is never named in the answer.

`POST /v1/feedback` takes `{ "trace": "tr_…", "rating": "up"|"down", "wrong_citation": "docs:…@17"?, "comment": "…"? }` (SB-310):

| Rule | Detail |
|---|---|
| Who | Only the user who asked. Anyone else, or an unknown or expired trace, gets `EPERM` |
| `rating` | Required, `up` or `down` |
| `wrong_citation` | Optional. It must be a citation **in that trace's answer**, or the request is `EINVAL` |
| `comment` | Optional, at most 2,000 characters |
| Repeats | One feedback per trace: a later one replaces the earlier one |
| Response | `200 { "trace": "tr_…", "stored": true }` |

Feedback is stored against the trace id with the release, and exported weekly (Monday 00:00 UTC, the previous seven days) as JSON lines for the pilot review. Each line holds the trace, release, rating, wrong citation, comment, time, and the answer's claim, uncited and refused counts, but never the question or the answer text. Reviewers who need those open the trace with auditor rights.

### The web UI

On GCP the web UI is served from a Firebase Hosting site in the cell's own project, which rewrites `/v1/` to that cell's `api` only (sb-runtime(7), Web UI hosting).

The cell serves its web chat UI at `/` on the tenant's own hostname, from the release's built assets (SB-307). It calls only this cell's `/v1` API, with the session cookie, and loads nothing from anywhere else: the page is served with `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'` and no third-party scripts, fonts or analytics.

| Element | Rule |
|---|---|
| Claim with citations | Followed by one chip per citation, showing `cite`. With a `url`, the chip is a link that opens the source at the cited version in a new tab; without one, it links to the answer's trace |
| `uncited` claim | Styled differently **and** labelled with the word "Uncited" (never colour alone), with a tooltip saying no source the user can open supports it |
| `flags` | A chip with `injection_suspect` says "Flagged" next to the cite |
| `notice` | `no_access`: "Some sources you don't have access to may answer this. Ask their owner." `no_sources`: "Nothing indexed answers this yet." `most_uncited`: "Most of this answer isn't backed by your sources." |
| `refused` | "N sources were not shown because you can't open them", never their names |
| Space switcher | In a cell with spaces, a labelled select, "Ask in", lists "Whole company" and the spaces from `GET /v1/spaces`, and nothing else. The chosen space goes on the ask as `?space=`. Changing it starts a new conversation. A cell without spaces shows no switcher |
| Space and label | Each chip also shows the spaces that hold the source and its label, in words (for example "Finance · restricted"), as part of the chip's accessible name |
| Errors | The error's `message` and `remedy`, with the trace id to quote to support |

The UI renders a claim only as cited (with at least one chip) or as uncited, never as plain text (LAW-14). It targets WCAG 2.2 AA: everything works by keyboard, focus is always visible, the answer area is an `aria-live` region, and every chip has an accessible name that includes the cite.

`cites[].url` is set when the source's connector can link to a version; it is omitted otherwise. `agent` names the agent that answered, and `dropped` counts the uncited claims a department agent left out, always 0 for `ask` (sb-agents(7), Department agents). In a cell with spaces, `cites[].spaces` names the spaces that hold the cited document and `cites[].label` its label (sb-spaces(7)); both are omitted in a cell without spaces (SB-1007).

### Spaces

`GET /v1/spaces` lists the spaces the asker is a member of, now, as `{"spaces": [{"name": "finance", "label": "restricted"}]}`, sorted by name. Spaces the asker isn't in are never listed, and a cell without spaces returns an empty list. A registry that can't answer is `EUNAVAIL`.

### The chat app (Slack)

Staff can also ask from Slack with `/ask <question>` (SB-308). It is the same `ask` agent with the same laws. Only the entry point differs: Slack calls `POST /chat/slack/commands` for the command and `POST /chat/slack/interactions` for the Share button.

1. **Verify.** The cell checks Slack's request signature with the signing secret from `vault://tenants/TENANT/connectors/chat` and refuses any request older than 5 minutes. The `team_id` must be the tenant's workspace. Anything else gets HTTP 401 and no reply.
2. **Map the person.** The bot looks up the Slack user's email (`users.info`) and finds the **one active** directory user with that email (SCIM-provisioned, sb-identity(7)). No match, more than one match, or an inactive user gets an ephemeral reply saying the account can't be matched, and **no answer**. The bot never falls back to a shared or service identity.
3. **Answer privately.** Slack gets an immediate ephemeral "Looking that up…". The answer follows through Slack's `response_url` (only `https://hooks.slack.com/`), also **ephemeral**, so only the asker sees it. Claims show as text with their citations as links (`<url|cite>`). Uncited claims start with *Uncited:*. The reply ends with the notice, the refused count and the trace id.
4. **Spaces.** In a cell with spaces, the channel the command was typed in picks the space: a space's `channels` in the manifest (sb-space(5)) list the channels that ask in it. A mapped channel asks in that space, and someone who isn't a member gets an ephemeral reply saying so and **no answer**, never a company-wide one. An unmapped channel, or a direct message, asks company-wide.
5. **Share.** A *Share in channel* button posts the question and the **citation links only, never the claim text**, so each reader opens the sources under their own access. Only the asker can share, and the trace is read again with the asker's own scope.

Errors are ephemeral replies with the error's `message` and trace id. Slack scopes added for the bot: `commands` only. The bot uses no Slack write scopes, because replies go through `response_url`.

## ERRORS

| Code | When |
|---|---|
| `EINVAL` | Missing or over-long `question`, an unknown `sources` kind, or a `conversation` that isn't the user's |
| `ESPACE` | `?space=` names a space the cell doesn't hold or the asker isn't a member of (never asked company-wide instead) |
| `EINVAL` (agent) | A department `agent` asked company-wide, in a space of another template, or not enabled by the manifest or the space |
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

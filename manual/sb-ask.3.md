---
title: "sb-ask(3)"
man: "3"
summary: Ask a question and get a cited answer.
---

# sb-ask(3)

## NAME

sb-ask — `POST /v1/ask`, `GET /v1/search`, `GET /v1/traces/{id}`, `POST /v1/feedback`, `/v1/saved`, `/v1/shares`, `/v1/questions`, `GET /v1/start`, `POST /v1/handover`, `GET /v1/handover/people`, `GET /v1/changes`, `POST /chat/mattermost`, `POST /chat/mattermost/actions`, `/v1/spaces/NAME/evals`, `POST /v1/access-requests`

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
| `language` | `en`, `de` or `fr` | no | (SB-3003) The language to write the answer's claims in. The web UI sends the person's chosen language; the chat apps send the manifest's `language` (sb-tenant(5)). Any other value is ignored, and without it the model writes as it does today |
| `documents` | list of `KIND:ID`, ≤ 5 | no | (SB-3004) Ask about these documents only: retrieval keeps only their chunks (Ask about a document, below). A version (`@N`) is ignored. More than 5, or one that isn't `KIND:ID`, is `EINVAL` |
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
            "section": "Refunds over the limit",
            "owner": { "email": "ann@acme-logistics.example", "title": "Support lead" },
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
  "dropped": 0,
  "earlier_dropped": 0,
  "people": [{ "email": "ann@acme-logistics.example", "title": "Support lead", "cites": 1 }],
  "terms": [{ "term": "PO", "means": "purchase order" }],
  "notice_stale": { "stale": 2, "of": 3, "oldest_days": 412 },
  "notice_sync": [{ "kind": "docs", "last_success": "2027-03-10T14:05:00Z" }],
  "notice_single": false
}
```

| Field | Meaning |
|---|---|
| `claims[].cites` | Only citations to chunks the user was live-authorised for |
| `claims[].uncited` | `true` when no allowed chunk supports the claim. Set by code, never by the model |
| `refused` | How many candidate documents were refused (deny or unknown). Never their titles |
| `notice` | `null`, `"most_uncited"`, `"no_access"` or `"no_sources"` |
| `conversation` | The conversation this turn belongs to, `cv_…` |
| `claims[].cites[].title` | The cited document's title at the source, as indexed |
| `claims[].cites[].section` | (SB-3002) The nearest heading above the cited chunk in its document, at most 120 characters, as indexed (sb-ingest(7), Sections). Omitted for a chunk with no heading above it. The UI shows the cite as "Title › Section" |
| `claims[].cites[].owner` | The cited document's owner at the source: their `email` and job `title` from the directory, only when exactly one **active** directory user has that email. Omitted otherwise (SB-1702) |
| `terms` | The space's glossary entries the question used (Glossary, below; SB-2203). Omitted when none |
| `notice_stale` | (SB-2701) `{ "stale": N, "of": M, "oldest_days": D }` when more than half of the answer's cited documents (each counted once) are `stale`, below; `oldest_days` is the oldest of the stale ones. Only cites in the answer are counted, so a document the asker can't open changes nothing. Omitted otherwise |
| `notice_sync` | (SB-2704) `[{ "kind": "docs", "last_success": "…" }]`, one per connector kind of the answer's cites whose last successful sync is older than the 15-minute freshness target (sb-ingest(7)); `last_success` is `null` when the cell has no record of one or can't read it, which is shown as unknown, never as fresh (LAW-03). It names a connector kind only, never a document. Omitted when every kind is fresh |
| `notice_single` | (SB-3005) `true` when the answer has at least two cited claims and every cite of every claim is one document. Only the cites in the answer count, so a document the asker can't open changes nothing. Omitted otherwise |
| `claims[].cites[].reviewed` | (SB-2702) The date a steward of a space that holds the document marked it **Reviewed, still right** at this version (`YYYY-MM-DD`), the newest such mark. Omitted otherwise. The steward is not named |
| `earlier_dropped` | (SB-2901) How many of the conversation's earlier questions were left out of what the model read: only the last 8 are sent. 0 otherwise. The conversation itself keeps every question |
| `people` | Who to ask: the owners of the answer's cites, most cited first (then by email), at most three, each with how many cites name their documents. Empty when no cite has an owner |

### How the `ask` agent runs, exactly

1. **Validate.** `question` must be 1 to 4,000 characters after trimming, and `sources` may name only `docs`, `chat`, `tickets`, `crm`, `code`, `hris` or `accounting`. Unknown fields are ignored (sb-api(3), Versioning). Anything else is `EINVAL`.
   **Fair share** (SB-2305): each person may ask `asks_per_minute` times in any minute (sb-tenant(5), default 6), counted by the cell in its memory over a sliding minute. One more is `ERATE` (HTTP 429, with `Retry-After` and `retry_after`, the whole seconds until the oldest of the minute's asks is a minute old), with no model call, and is stored as a trace of agent `limited` with no question, so the weekly report counts it. The chat apps reply with the same wait. Search, saved questions, `sb tenant verify`'s sampled asks and everything else are not limited; a restart forgets the minute.
2. **Retrieve and check live** as in sb-permissions(7). With `sources`, candidates of other kinds are dropped before the live check. In a cell with spaces, every candidate first passes the space, label and barrier checks (sb-permissions(7), Spaces, labels and barriers) with the asker's read context; only the chunks they allow go into the live ACL batch, and the ACL still decides each of those. An ask in a space uses only what that space holds. A company-wide ask uses the union of the asker's spaces, plus `public` and `internal` content from spaces they aren't in; `confidential` and `restricted` content comes only from spaces they're a member of (SB-1006).
3. **No allowed context:** the model is **not** called. The answer is one uncited claim saying nothing the user can open answers this, with `notice` `no_access` if any candidate was refused, or `no_sources` if there were none.
4. **Compose:** the bundled `answer` prompt (with `language`, one more line: "Write each claim's text in German. Keep every label exactly as given.", or English or French), the allowed chunks as data blocks and the question go through the model gateway (sb-agents(7), MODEL GATEWAY). The composer decides citations (sb-citations(7)). Each stripped label is logged as `citation_spoof` with the trace id and the label, never the claim text.
5. **Notice:** `most_uncited` when more than half the claims are uncited; `notice_stale` and `notice_sync` as above, beside it.
6. **Trace:** every answer gets a new trace id `tr_…`, returned in the body and in `X-SB-Trace`, and stored with the answer and its decision list for 30 days.

**Conversations.** A request without `conversation` starts one, and the response carries its `cv_…` id in `conversation`. A later turn sends only the user's **earlier questions** to the model, never earlier answers or chunks: everything cited is retrieved and checked live again, so an earlier turn can't carry content the user has since lost access to. A `cv_…` id that isn't this user's is `EINVAL`, the same as one that doesn't exist. **Short conversations** (SB-2901): a later turn sends at most the **last 8** earlier questions, oldest first; any before them are left out of the prompt, and the response says how many in `earlier_dropped`. Nothing is removed from the stored conversation, and the trace keeps the turn's place in it. The web UI then says the earliest questions weren't used and offers **Start fresh**, which starts a new conversation.

**Answers in the asker's language** (SB-3003). `language` changes only the language the claims are written in. Retrieval, the live checks, the citations, the titles, sections and passages are the same as without it, and stay in the source's own language. An uncited claim is marked in every language (LAW-14). The language isn't stored with the question; the trace keeps the answer as written.

**The cited passage** (SB-3001). `GET /v1/traces/{id}/passage?cite=CITE&start=N` returns the text of one cite of that answer: `{ "cite": "docs:4f2a9c11@17", "text": "…", "cut": false }`. Only the person who asked (or an auditor, as for the trace) may call it, only for a cite that is in the trace's answer (`start` picks the span when the answer cites a document more than once; without it, the first), and every call checks the person's access to that document **again, now and as them** (sb-permissions(7)), so an old trace or a share can't show text they have lost access to. `text` is the cited chunk's text at that version, at most **600 characters**, cut after the last full sentence that fits (`cut` is then `true`). It is `EPERM` ("You can't open this source now.") when the live check denies or can't decide (LAW-01, LAW-03), and `200` with `"gone": true` and no `text` when the index no longer holds that version, which the UI shows as "This version is no longer indexed." A cite that isn't in the answer is `EINVAL`. Nothing is stored, and the text is never in a log (LAW-07). The web UI shows it as plain text in a block quote, never as markup (LAW-09).

**Ask about a document** (SB-3004). With `documents`, the candidates are only the chunks of those documents: those the search ranks, in its order, then the rest of each document's chunks in their order in it, so a document is found however the question is worded. They take every check as any candidate does, and the checks run first: the filter only ever narrows. A chosen document the asker can't open is refused like any other, so the answer is `no_access`; `refused` counts only the chosen documents. The three-chunk cap below doesn't apply, since every chunk is from a document the asker chose, but the usual budget does. Conversations, `language` and every notice work as without it.

**One document, three chunks** (SB-2902): of the chunks that pass every check and the one-copy rule (Copies, below), at most **3 of one document** go into what the model reads for an answer or a draft, its best-ranked ones, in rank order; the rest of the budget goes to other documents. Search results (`GET /v1/search`) are unchanged.

### Glossary, exactly (SB-2203)

In a space with a `glossary` (sb-space(5)), step 2 first **expands** the question: each term the question uses, as a whole word, adds its meaning to the text that is searched and embedded, so "PO" also finds documents that say "purchase order". A term with a capital letter matches only as written ("IT", never "it"); a term in lower case matches in any case. Only the space's own glossary is used, and only in an ask or search in that space: a company-wide ask, or another space, never expands one (LAW-19). An entry that reads like an instruction (the patterns of sb-ingest(7), Content is data) is never used. The terms used go to the model inside one data block, `<<data cite="glossary">>`, never as instructions (LAW-09), and the response lists them, `"terms": [{ "term": "PO", "means": "purchase order" }]`, which the web UI shows at the foot of the answer or results. The stewards' review list names the flagged entries, `"glossary_flagged": ["TERM"…]`, so they can fix them.

### Who to ask, exactly (SB-1702)

Every chunk carries its document's owner at the source (an email, SB-1203). The owner is looked up in the directory when the answer is built, and shown only if exactly one active user has that email; a deprovisioned, duplicated or unknown owner is left out, and a directory that can't answer leaves every owner out rather than failing the answer. Owners come only from the answer's cites, and a cite is only ever a chunk that passed the live check for this asker, so nobody learns who owns a document they can't open (LAW-01, LAW-12). Ownership at the source is the only signal: the cell builds no activity, chat or expertise profile of anyone.

### Search, exactly (SB-1701)

`GET /v1/search?q=TEXT[&sources=docs,chat][&space=NAME][&type=pdf,sheet][&since=30d]` finds documents without the model. It runs steps 1 and 2 of the `ask` agent exactly (validation, retrieval, the space checks and the live ACL batch) and stops before the composer, so it makes **no model call** and uses no budget or tokens:

```json
{
  "trace": "tr_01JD3QC1…",
  "release": "rel_2027-03-02_9b1c4e",
  "results": [
    {
      "cite": "docs:4f2a9c11@17",
      "title": "Refund policy",
      "url": "https://…/refund-policy?version=17",
      "owner": { "email": "ann@acme-logistics.example", "title": "Support lead" },
      "age_days": 41,
      "snippet": "Refunds over £500 need a team lead's approval…",
      "flags": [],
      "spaces": ["support"],
      "label": "internal"
    }
  ],
  "refused": 1
}
```

| Rule | Detail |
|---|---|
| Results | Up to 20 documents, each once, in the order of their best-ranked allowed chunk. Only documents that passed every check for this asker now; a document whose check is denied, Unknown or late is never returned, only counted in `refused` (LAW-01, LAW-03) |
| `snippet` | The first 300 characters of the document's best allowed chunk: a quote of the source with its cite, never a claim (LAW-14). `flags` are that chunk's flags; in a chunk flagged `personal_data` each IBAN or card number is masked but its last four characters (sb-ingest(7), Personal data; SB-1904) |
| `age_days` | Whole days since the cell first indexed this version; omitted when the index doesn't know |
| `type` | Filters (SB-2404): only documents of these types, comma-separated, from the extension on the document's source id, else on its title: `pdf` (.pdf), `doc` (.docx, .doc, .odt, .rtf), `sheet` (.xlsx, .xls, .ods, .csv), `slides` (.pptx, .ppt, .odp), `web` (.html, .htm), `text` (.txt, .md), and `other` for anything else, chat messages and tickets included. An unknown type is `EINVAL` |
| `since` | Filters (SB-2404): only documents whose version the index first saw in the last `N` days (`Nd`, 1 to 365); one whose age the index doesn't know is left out. Both filters run **after** every check, on documents the asker may already open, so they only ever narrow: a filter never shows a document, and `refused` counts as without them |
| `owner` | As on a cite (Who to ask) |
| `mine` | (SB-3106) `true` when the asker is the document's current owner at the source: its owner is one of the asker's own addresses in the directory now. Omitted otherwise |
| Errors | As for `POST /v1/ask`: `EINVAL` for an empty or over-long `q` or an unknown kind, `EKILLED` when the `ask` agent, the tenant or the space is killed, `EUNAVAIL` when the index or directory can't answer. Never `EBUDGET` |
| Trace | Every search gets a trace `tr_…` with its decision list, like an answer, with `agent` `search`. The trace keeps the SHA-256 of the query, never its text, and the policy kernel writes its decision rows to the audit log as for an ask |

The weekly report counts searches apart from answers.

### Saved and shared questions, exactly (SB-1703)

A person can keep a question and pass it on. Only the question's text is kept, never an answer: opening a saved or shared question asks it again, live, under the access of whoever opens it.

| Endpoint | Does |
|---|---|
| `POST /v1/saved` `{ "question": "…" }` | Saves a question (1 to 4,000 characters) in the space of the request. `200 { "id": "sq_…" }` |
| `GET /v1/saved` | `{ "items": [{ "id", "question", "space", "at", "watch" }] }`: the caller's own, newest first, at most 100 |
| `DELETE /v1/saved/{id}` | Removes one of the caller's own; anyone else's, or an unknown id, is `EPERM` |
| `PUT /v1/saved/{id}/watch` `{ "watch": true }` | Watches one of the caller's own (SB-2104), or stops with `false`; anyone else's is `EPERM`. `200 { "id", "watch" }` |
| `POST /v1/shares` `{ "question": "…" }` | Makes a share `sh_…` with an unguessable id and the link `/q/sh_…`, valid for 30 days. `200 { "id": "sh_…", "url": "/q/sh_…", "expires": "…" }` |
| `GET /v1/shares/{id}` | `{ "id", "question", "space", "expires" }` for any signed-in person of the same tenant. Another tenant's share, an unknown id or an expired one is `EPERM` (LAW-02) |

**Watching** (SB-2104) runs the question as a **search** (no model) as its owner, in its space, and keeps only the SHA-256 of the result set: each result's `cite`, which carries the version, sorted. Each week, with the digest (Notifications, below), the search runs again as the owner, now, and a different hash tells them the sources changed. A source they can no longer open drops out of the set, so it counts as a change, but nothing names it; no answer and no result is kept (LAW-07). A watched question whose owner is no longer active is skipped.

The link `/q/sh_…` opens the web UI on the shared question with an **Ask** button; asking it is an ordinary `POST /v1/ask` by the recipient, so they see only what their own access allows, never the sharer's answer (LAW-01). Creating a share writes an `access` audit row whose subject is the share id and whose `payload_sha256` is the SHA-256 of the question; the text itself never reaches a log or the audit log (LAW-07). Saved questions and shares are cell records kept under the tenant's retention (and a space's shorter one), held by legal holds like conversations, and erased with the person (`POST /v1/admin/subjects/erase`).

### What changed, exactly (SB-1708)

`GET /v1/changes?since=7d[&space=NAME]` lists documents with a version the index first saw in the window (`since` is `Nd`, 1 to 90 days), newest first, at most 50: `{ "since": "…", "items": [...] }`, each item a search result without the snippet. In a space, it lists what that space holds; company-wide, what the person's own spaces hold (in a cell without spaces, everything). Each document is checked as a search checks it, now, as the person, and one they can't open is left out; a version written by a rebuild is never listed (sb-ingest(7), Rebuild). The web UI shows "What changed this week" on the empty answer page, and the weekly report counts changed documents per space. The weekly digest (Notifications, below) sends each person this list, as them, when it is sent.

### Documents to review, exactly (SB-1709)

`GET /v1/spaces/NAME/review` is the stewards' list (the Cedar action `review_space`, as for access reviews): documents the space holds that answers cited at least 5 times in the last 30 days and whose version is older than the space's `review_after_days` (sb-space(5), default 365), and any document someone marked as a `wrong_citation` (POST /v1/feedback) in those 30 days. The counts come from the traces and feedback the cell already keeps. Each document is checked as a search checks it, now, as the steward: `{ "items": [{ …result, "cited": 7, "flagged": 1 }], "hidden": [{ "owner": "ann@…" }], "personal_data": 3 }`, where an item that is a note whose readers include people no longer members of the space carries `"wider": N`, their number (never who), and `"question": "qs_…"`, the note's question to republish it (SB-2004, listed first); where `personal_data` counts the documents the space holds with an IBAN or a card number in them (SB-1904), whether or not the steward can open them, and never says which; and where a document the steward can't open is only an entry in `hidden`, with its owner if there is one so they can ask the owner, never its title or cite.

**Reports and eval cases (SB-2204).** An item marked as a wrong citation carries `"reports": ["tr_…"]`, the traces of the answers reported (at most 10, newest first), so its stewards can turn one into an eval case: `POST /v1/spaces/NAME/evals`, `GET /v1/spaces/NAME/evals` and `DELETE /v1/spaces/NAME/evals/{id}` (sb-evals(7), Cases from wrong citations), with the same `review_space` decision as the list.

**Documents with no owner (SB-2201).** The list also holds documents the space holds whose owner at the source is no active user in the directory (someone who left, or was deprovisioned), each checked as the steward like the rest and carrying `"owner_gone": true`, with no `owner`: a deprovisioned owner is never shown as someone to ask, here or on any cite. A document with no owner at the source isn't listed for that alone. The directory is read when the list is asked for, so a deprovisioned owner's documents are on it at once, and in any case within the next sync. Those the steward can't open are entries in `hidden` with `"owner_gone": true` and nothing else. At most 50 are listed; a directory that can't answer is `EUNAVAIL`.

**Copies (SB-2202).** Documents the space holds whose normalised text is the same (sb-ingest(7), Provenance) are listed once, as the newest copy the steward can open, with `"copies": [result…]`, the other copies they can open (title, cite and link; no snippet). Copies they can't open are entries in `hidden`. Answers, searches and drafts cite **one copy**: of documents with the same text that the asker can open, only the newest (the one the index saw last) is cited. Delete it, and the next answer cites the next newest.

**Reviewed, still right (SB-2501).** `POST /v1/spaces/NAME/review/marks` `{ "cite": "docs:…@17" }` marks a document on the list as reviewed at its **current version**: only a steward of the space, only for a document they can open now (otherwise `EPERM`, the same answer as for one that doesn't exist), and the cite's version must be the current one (`EINVAL` otherwise). The cell keeps `{ space, document, version, by, at }` and writes an audit row (`config`, state `reviewed`). A marked document leaves the list while its version is that one and the mark is younger than `review_after_days`; a new version, the mark's age, or a wrong-citation report **after** the mark puts it back. Reports before the mark no longer list it. The mark is the document's version, not the index's, so a rebuild neither clears nor renews it (LAW-13). Notes read wider than the space, documents with no owner and copies are listed whatever the mark, since a mark says the text is right, not that its readers or owner are. The answer is `{ "document": "docs:…", "version": 17 }`.

Every cite in an answer also carries `age_days` (whole days since the index first saw its version), and `stale: true` once that is past the `review_after_days` of a space that holds it (365 for a document no space holds). The web UI shows "N days old" on such a chip, in words, as part of its accessible name. A cite with a **Reviewed, still right** mark at its version (SB-2702) carries `reviewed`, and is `stale` only once `review_after_days` have passed since the mark, so a document a steward checked stops warning askers; its chip says "Reviewed 2026-11-20" instead. A new version is judged by its own age again. A document whose own `review-by` or `expires` date has come (sb-ingest(7), Review-by dates; SB-2705) is `stale` from that date, mark or not, and is on the review list once answers cite it at least 5 times, whatever a mark says.

**Reported outdated (SB-2703).** A document someone reported with `outdated_citation` in the last 30 days is on the list too, with `"outdated": N`, whatever its age. A **Reviewed, still right** mark clears the reports made before it, as it does wrong citations; a report after the mark puts it back.

**Retire from answers (SB-2801).** `POST /v1/spaces/NAME/retired` `{ "cite": "docs:…@17", "reason": "…", "replaced_by"?: "docs:…" }` takes a document out of the space's answers: only a steward of the space, only for a document the space holds and they can open now (otherwise `EPERM`, as for one that doesn't exist); `reason` is 1 to 200 characters; `replaced_by`, a document (`KIND:ID`, no version) the space holds and they can open, is optional. The cell keeps `{ space, document, reason, replaced_by, by, at }` and writes an audit row (`config`, state `retired`), and answers `{ "document": "docs:…", "reason": "…", "replaced_by": "docs:…", "at": "…" }`. From then on, answers, searches and drafts asked **in that space** skip every version of the document; a company-wide ask skips it only when every space that holds it retired it. The document stays at the source, in the index and in every other space, and a rebuild keeps it out, because the retirement is on the document's id, not the index (LAW-13). Like the cell's other records it is kept for `retention_days` (sb-tenant(5)); after that the document is back in answers unless a steward retires it again. A cite of the replacement carries `"replaces": "OLD TITLE"` when the asker can open the old document too, checked live. `GET /v1/spaces/NAME/retired` lists the space's retirements, each `{ document, reason, replaced_by, by, at }` with the title of a document the steward can open, and only a count of the others. **Bring back**, `POST /v1/spaces/NAME/retired/restore` `{ "document": "docs:…" }`, ends one, with an audit row (state `restored`), and answers `{ "document": "docs:…", "restored": true }`; one that isn't retired is `EPERM`.

**Same name, different text (SB-2802).** The list's `same_name` holds the documents the space holds whose titles are the same, ignoring case and spacing, and whose text differs, as groups, newest first: `[{ "title": "Leave policy", "items": [result…], "hidden": N }]`. Each group is checked as the steward like the rest: `items` are the ones they can open, `hidden` how many they can't. A group with fewer than two documents in all isn't listed; copies with the same text are listed once (Copies, below).

**Never cited (SB-2803).** `GET /v1/spaces/NAME/review?section=uncited` lists the documents the space holds that no answer cited in the last 180 days and that the index has held for more than 90, oldest first, at most 100: `{ "items": [result…], "hidden": [{ "owner": … }] }`, checked as the steward like the list. These are candidates to retire, nothing more: the section only reads the traces the cell keeps, and never shows a question. It also carries `"unused_terms": ["TERM"…]` (SB-2805): the space's usable glossary terms (sb-space(5)) that no answer asked in the space used in those 180 days, read from each trace's `terms`; a term used once isn't listed. Only the terms, never a question; a term is removed in the manifest, as any glossary change is.

**Eval cases to check (SB-2904).** The list carries `"evals_to_check": [{ "id": "ev_…", "question": "…", "check": "new_version" }]`, the space's eval cases whose expected document has a newer version or is gone (sb-evals(7), Cases that went stale). Omitted when none.

**Start here gone stale (SB-2907).** The list carries `"start_here": [{ "item": "docs:…", "result": {…}, "why": ["gone" | "stale" | "unreadable"], "members_without": N, "members": M }]`, the space's `start_here` items (sb-space(5)) that are **gone** from the index, **stale** by the rule in Answers, above, or that more than a quarter of the space's active members can't open now (**unreadable**: the live ACL for each member, where Unknown or late counts as can't, LAW-01), each with why. A folder item is judged by the documents under it: gone when there are none. `result` is the document as the steward can open it, checked as them; an item the steward can't open has no `result` and only counts, in `"start_here_hidden": N`. Members are checked in one ACL call each, at most 200 members; a bigger space counts the first 200. Omitted when every item is fine.

**Freshness (SB-2807). The list carries `"freshness": { "this_month": { "answers": N, "stale": N }, "last_month": { … } }`: of the answers asked in the space that cite anything, this calendar month and last (UTC), how many carried `notice_stale`, so rested mostly on stale cites (Answers, above). Counts from the traces only; the steward's screen shows each as a share.

**Notes whose sources moved (SB-2804).** A note (SB-1705) is a reply, which keeps the versions it cites. When one of them has a newer version in the index, or is gone from it, the note is on the list with `"sources_moved": N`, how many, until a steward republishes it or marks it **Reviewed, still right** after the change. The changed documents aren't named on the note's entry; the steward opens the note to see its cites, each checked as them.

**Questions that found nothing (SB-3101).** The list carries `"found_nothing": { "this_week": N, "last_week": N, "terms": ["PO", …] }`: of the answers asked in the space, in the last 7 days and the 7 before, how many ended with `notice` `no_sources` or `most_uncited`, and the space's glossary terms those questions used (Glossary, exactly), most used first, at most 10. These are counts and the space's own terms, read from the traces, which keep no question text for this: no question text is shown, stored or sent (LAW-07). The stewards' line in the weekly digest (SB-2507) adds "N questions found nothing" when N > 0.

**Feedback for the stewards (SB-3104).** The list carries `"feedback": { "this_month": { "helpful": N, "not_helpful": N }, "last_month": { … }, "documents": [result…], "documents_hidden": N }`: the ratings given this calendar month and last (UTC) on answers asked in the space, and the documents cited most often in answers rated **Not helpful** this month, at most 10, most cited first, each checked as the steward like the rest of the list; `documents_hidden` counts the ones they can't open. Comments stay in the admins' queue (`GET /v1/admin/feedback`) and are never on this list.

**Several at once (SB-3103).** `POST /v1/spaces/NAME/review/marks` and `POST /v1/spaces/NAME/retired` also take `"cites": ["docs:…@17", …]` (at most 50) instead of `cite`, with one `reason` and no `replaced_by` for a retirement. Each cite is checked, decided and audited on its own, exactly as when it is sent alone, and the response is `{ "items": [{ "cite": "…", "ok": true } | { "cite": "…", "error": { "code": "EPERM", "message": "…" } }] }`, in the order sent: one that fails doesn't stop the others. More than 50 is `EINVAL`. The review list has a checkbox on each document and **Mark reviewed** and **Retire** for the selected ones.

**Coming due (SB-3105).** `GET /v1/spaces/NAME/review?section=due` lists the documents the space holds that answers cited at least 5 times in the last 30 days, that aren't stale yet, and that turn stale in the next 14 days: `review_after_days` after their version was first seen, or after a **Reviewed, still right** mark at that version, or on their own review-by date (SB-2705), whichever is first. Soonest first, at most 100: `{ "items": [{ …result, "due_in_days": N }], "hidden": [{ "owner": … }] }`, checked as the steward like the list. **Reviewed, still right** works on them as on stale ones, and a mark moves the date on, except a document's own review-by date, which a mark doesn't lift. A document due in 10 days is listed; one due in 20 isn't.

**Owners mark their own (SB-3106).** `POST /v1/documents/reviewed` `{ "cite": "docs:…@17" }` lets a document's **current owner at the source** mark it **Reviewed, still right** at its current version: the document's owner must be one of the caller's own addresses in the directory now, and the caller must be able to open it now (the live check, LAW-01). Anyone else gets `EPERM`, the same answer as for a document that doesn't exist; someone deprovisioned can't sign in, and an address the directory no longer gives them isn't theirs (LAW-12). A version that isn't current is `EINVAL`. The mark is the stewards' own record, one for each space that holds the document, with the owner as `by`, so it shows on cites and takes the document off the review list exactly as a steward's does; each writes an audit row (`config`, state `reviewed`, reason `reviewed by its owner`) whose actor is the owner (LAW-11). The answer is `{ "document": "docs:…", "version": 17, "spaces": ["ops"] }`. In the web UI, a search result with `mine` has **Reviewed, still right** on the Documents tab.

### Start here and handover (SB-1706, SB-1707)

`GET /v1/start?space=NAME` is the space's reading list for a member (sb-spaces(7), Start here): `{ "items": [...], "hidden": 1 }`, each item a search result without the snippet. `POST /v1/handover` `{ "person": "usr_…" }` is the handover brief (sb-agents(7), Handover): `{ "trace", "person", "answer": { "claims": [...] }, "dropped", "hidden", "locations": { CITE: LOCATION } }`, every claim cited; `locations` gives where each briefed document lives, for documents the asker can open only (SB-2002). A refusal is `EPERM` with the policy's reason: only the person, their manager or a steward of a space they are in can have the brief, and an unknown person gets the same answer. `GET /v1/handover/people?q=TEXT` (SB-2106) finds the people the caller may brief: `{ "items": [{ "id", "name", "email", "title" }] }`, at most 20, sorted by name, whose user name or email contains `TEXT` (2 to 100 characters, case ignored). Candidates are the caller, the people whose directory manager is the caller, and the active members of spaces the caller stewards, and each one is kept only if the same policy decision as `POST /v1/handover` allows it now, so the list names no one the caller couldn't brief (LAW-01); a colleague with no relation isn't found whatever is typed. In the web UI the **Handover** tab finds a person by name or email with that list, or takes the person's user id, shows the brief grouped by location, and shows `hidden` as a count with the advice to have access moved to whoever takes the work over before the person leaves, since deprovisioning revokes their access (LAW-12).

### Questions to the stewards, exactly (SB-1704)

When nothing the asker can open answers (`notice` `no_sources`, `no_access` or `most_uncited`), they can send the question to the people who know: the stewards of the space they asked in (sb-space(5), `stewards`), or, for a company-wide ask, the manifest's `questions_to` group (sb-tenant(5)).

| Endpoint | Does |
|---|---|
| `POST /v1/questions` `{ "question": "…", "trace": "tr_…" }` | Sends the question to the stewards of the request's space (`?space=`), or to `questions_to` company-wide. `trace` must be the asker's own answer in that space with one of the three notices; anything else is `EINVAL`, and so is a space with no stewards or a cell with no `questions_to`. Only the question's text is sent: never the asker's name, the answer or its cites. `200 { "id": "qs_…", "to": "finance" }` |
| `GET /v1/questions` | `{ "items": [...] }`: the caller's own questions, newest first, each `{ "id", "question", "space", "at", "state": "open"|"answered", "reply"? }` |
| `GET /v1/spaces/NAME/questions` | The queue of space `NAME`, or `company`, for its stewards: `{ "items": [{ "id", "question", "at", "state", "reply"? }] }`, open ones first, never the asker. Anyone else is `EPERM` |
| `POST /v1/spaces/NAME/questions/{id}/reply` `{ "text": "…", "cites": ["docs:…@3"] }` | A steward's reply: text a person writes, 1 to 4,000 characters, and at most 10 cites the steward adds from their own search. Each cite must be one the steward can open now, checked live as a search is, or the reply is `EINVAL`. One reply a question; a second is `EINVAL` |
| `POST /v1/spaces/NAME/questions/{id}/note` `{ "title": "…", "readers"?: [EMAIL…], "confirm"?: N }` | With the manifest's `notes: true`, publishes the reply as a note (sb-spaces(7); SB-1705). Without `confirm` it only previews: `{ "path", "readers", "narrower" }`, the note's path, how many readers it will have (the space's active members, or the listed subset of them; never more) and how many of the reply's cites some of them can't open. With `confirm` equal to that `narrower` count it publishes through the tool bus and returns `{ …, "action", "state", "receipt" }`; a different count is `EINVAL`. On a reply that is a note already it **republishes** it narrowed (SB-2004): the same text, under the title it was published with when `title` is empty, to the readers it has now who are still members of the space (or a listed subset of them), owned by the steward who first published it; it can only ever drop readers |
| `POST /v1/spaces/NAME/questions/{id}/claim`, `DELETE /v1/spaces/NAME/questions/{id}/claim` | (SB-2502) "I'll answer this": the caller claims an open question for an hour, or lets their claim go. The queue then shows the question with `"claim": { "by": "ann@…", "mine": false, "until": "…" }` to the other stewards. A claim only warns: anyone can still reply, and a reply ends it. Claiming a question someone else claims answers with their claim, unchanged. A claim is shown only while it is live **and** its holder is an active member of the space's stewards now (LAW-21): someone removed from the group, or deprovisioned, loses every claim at once, and a cell that can't check shows none |
| `GET /v1/spaces/NAME/templates`, `POST /v1/spaces/NAME/templates`, `PUT /v1/spaces/NAME/templates/{id}`, `DELETE /v1/spaces/NAME/templates/{id}` | (SB-2503) The space's reply templates, for its stewards only: `{ "items": [{ "id", "name", "text" }] }`, at most 20, each a name of 1 to 60 characters and a text of 1 to 2,000. **Use a template** on the Questions screen puts its text in the reply box, still editable; a template is the stewards' own text, like a reply, never sent to the model. A space's templates are offered only in that space (LAW-19); the stewards of another space get `EPERM` |

A reply is shown to the asker with its cites **checked live for the asker** each time they look, exactly as a search checks them: a cite they can't open now is dropped and counted (`reply.dropped`), never named, so a steward can't widen what the asker reads by quoting it (LAW-01). `reply.cites` has the same shape as a search result without the snippet. Who is a steward is decided by policy (`answer_questions` on the space, sb-policy(5)) with the caller's groups at that request, so a steward who leaves the stewards group stops seeing the queue at their next request (LAW-21). Sending writes an `access` audit row (`state` `asked`, the question's SHA-256) before the question is stored, and a reply one (`state` `answered`, the reply's SHA-256) before it is stored (LAW-04); neither row holds text (LAW-07). Questions are cell records under the tenant's retention, or the space's shorter one, so an unanswered question ages out with it; they are held by legal holds on their space and erased with the asker. The weekly report counts questions asked and answered per space, never their text.

### Asking for access, exactly (SB-2205)

Where an answer or a search counts sources the asker can't open, `POST /v1/access-requests { "trace": "tr_…" }` asks their owners to share them. The trace must be the asker's own, from the last 7 days. Only sources **the source itself refused** (`source_denied` in the trace's decision list) are asked about: one refused by a space, label or barrier check isn't, since sharing it at the source wouldn't open it, and a request across a barrier would tell the other side what was asked (LAW-22). For each such document, at most 20, the cell finds its owner at the source as for who to ask (exactly one active directory user with the email, else nobody), and sends that owner one `access` notification (Notifications, below): the asker's email and the document's title, which the owner can open, and never the question or the answer (LAW-07). It sends **one request per asker, document and week**; asking again that week sends nothing more. The response is counts only: `{ "sent": 2, "already": 1, "no_owner": 1 }`, never which documents or who, so the asker learns nothing they didn't know (LAW-01); someone already named in the answer's `people` is still named there. An owner who has turned `access` off gets nothing, and the asker isn't told. A cell with no chat bot and no `email` to send from is `EUNAVAIL`. The web UI shows **Ask the owners for access** under "N sources were not shown", and then the counts.

### Notifications, exactly (SB-2101, SB-2102)

The cell tells people when something waits for them, and nothing it tells them says more than they could open. The `notify` job (sb-runtime(7), Schedules) runs every five minutes over what happened since its last run:

| Kind | To | Says |
|---|---|---|
| `questions` | Each steward of a space that got new questions: the active members, **at the time of sending**, of the space's `stewards` groups, or of `questions_to` for company-wide questions (LAW-21) | "2 new questions for the finance space" (or "sent company-wide"), and a link to that space's steward screen (`/?steward=finance`, or `company`) |
| `replies` | Each asker whose questions got a steward's reply, or whose reply was published as a note | "1 of your questions has a reply" (or "…was published as a note"), and a link to the Questions tab |
| `digest` | Each active person who asked anything in the last 90 days, weekly: Mondays at 08:00 in the manifest's `digest` time zone (sb-tenant(5)) | "4 documents changed in your spaces this week: Refund policy; Carrier contracts; …" with up to 10 titles, then "and N more", and a link to the Documents tab. The documents are `GET /v1/changes?since=7d` company-wide **as that person when the digest is sent** (What changed, above): each checked live, so one whose ACL dropped them since it changed isn't named (LAW-01), and a version written by a rebuild is never a change (LAW-13). A week with nothing sends nothing. A steward's digest adds **one line per space they steward when it is sent** (SB-2507): "ops: 3 answered this week, 2 still open, the oldest 4 days old", counts only; someone removed from the stewards gets no line for that space, and a steward with lines but no changed documents still gets the digest |
| `watch` | The owner of a watched saved question whose sources changed since the last weekly check (Saved and shared questions, above) | "The sources of 1 of your saved questions changed", and a link to the Saved tab, which asks it again live. Never the question, an answer or which source |
| `tokens` | (SB-2606) The owner of a read-only API token, once, seven days before it expires (the job's window, moved seven days on, so each token is told once) | "Your API token nightly report expires in 7 days", and a link to the conversation, where the person makes a new one. The token's name only, never the token; never for a revoked token or an owner who is no longer active |
| `owned` | (SB-2806) Each owner, monthly, on the first weekly run of the month (the first Monday, at the digest's hour in the `digest` time zone, or UTC): the documents they own that answers cited at least 5 times in the last 30 days and that are past review now (stale by the rule in Answers, above) | "3 of your documents that answers often cite are past review", and a link to the Documents tab. A count only, and only of the documents the owner can open, checked as them when it is sent; an owner who is not one active user is told nothing (LAW-12). On by default; the person can turn it off |
| `access` | The owner of a document an asker couldn't open, when the asker asks for access (Asking for access, below) | "ann@… asked a question that your document Carrier contracts answers. Consider sharing it with them.", and a link to the document at its source. Never the question or the answer |

A message holds a kind, a count, a space's name, for the digest the titles of documents the person can open when it is sent, for an access request the asker's email and the title of the owner's own document, and a link to the cell's own web UI (sb-cell(5), `base_url`), and **never** a question's text, a reply, a cite, a document's title or who asked (LAW-07); the person reads those on the page, where the cell checks again what they may see. Someone who is no longer a steward, or no longer active (LAW-12), when the job runs gets nothing. Each event is told once: the job keeps the time it last ran (`state/notify/last`) and only a run that read everything moves it.

Messages go as a **direct message from the chat bot** the cell already has: Mattermost's when sb-cell(5) sets `mattermost`, otherwise Slack's when the cell has a `chat` connector. The person is found by their directory email (Slack's `users.lookupByEmail`; Mattermost's `GET /api/v4/users/email/EMAIL`); someone the chat doesn't know by that email is skipped. A cell with neither sends them **by email** when sb-cell(5) sets `email` (SB-2307): one plain-text message to the person's directory email, from `email.from`, through the company's mail server over `STARTTLS` (a server that doesn't offer it is refused, never used in the clear). The subject names the kind only ("Second Brain: new questions"), and the body is the same text and link as the chat message, then a line saying where to turn the kind off: the same content rule, so a mail server, a mailbox or a forward never holds more than a chat would (LAW-07). An address the server refuses (`550`) is skipped like someone the chat doesn't know. A cell with none of the three sends nothing, and the job still runs and finishes. Slack's bot needs `chat:write`, `im:write` and `users:read.email` for this; Mattermost's bot needs to be able to create direct channels.

| Endpoint | Does |
|---|---|
| `GET /v1/me/notifications` | The caller's settings: `{ "off": ["replies"] }`, the kinds they've turned off; none is every kind on |
| `PUT /v1/me/notifications` `{ "off": [KIND…] }` | Replaces them. A kind that isn't one of `questions`, `replies`, `digest`, `watch`, `access`, `tokens` or `owned` is `EINVAL`. Kept until the person changes them or is erased; never expired by retention |
| `PUT /v1/me/notifications` `{ "off": [...], "pause_days": N }` | (SB-2506) Pauses every kind for N days, 1 to 30 (`EINVAL` otherwise), from now: the settings then carry `"paused_until": "…"`, and the job sends that person nothing, the digest included, until then; the pause ends by itself. `"pause_days": 0` ends it now. A body without `pause_days` keeps the pause as it is, and a `paused_until` in the body is ignored. Only the paused person is skipped: new questions still reach the space's other stewards |
| `POST /v1/me/notifications/test` | (SB-2602) **Send me a test**: the fixed text "This is a test from Second Brain. Its notifications reach you here." through the cell's real channel, to the caller: `{ "sent": true, "via": "mattermost" }` (or `slack`, `email`), with `sent: false` when the channel doesn't know them by their email. Once a minute per person (`ERATE`); a cell with no channel is `EUNAVAIL`. Never turned off by the person's settings or a pause, since they asked for it |

A send that fails is counted (`sb_notify_failed_total{kind}`) and the job returns an error, so the watchdog sees it, but the job doesn't send the same event twice.

`GET /v1/traces/{id}` returns the same answer plus the decision list (`cite`, `verdict`, `reason`, one per candidate document), for the user who asked and for auditors. Anyone else gets `EPERM`, the same as an unknown id. `reason` is one of `allowed`, `source_denied`, `source_unknown` or `policy_denied`, and in a cell with spaces `space_denied` (not held by the space asked in), `label_denied` (a `confidential` or `restricted` chunk held only by spaces the asker isn't in) or `barrier_denied`. The policy kernel makes every one of these decisions; the reason only explains it, and a refused document is never named in the answer.

`POST /v1/feedback` takes `{ "trace": "tr_…", "rating": "up"|"down", "wrong_citation": "docs:…@17"?, "outdated_citation": "docs:…@17"?, "comment": "…"? }` (SB-310, SB-2703):

| Rule | Detail |
|---|---|
| Who | Only the user who asked. Anyone else, or an unknown or expired trace, gets `EPERM` |
| `rating` | Required, `up` or `down` |
| `wrong_citation` | Optional. It must be a citation **in that trace's answer**, or the request is `EINVAL` |
| `outdated_citation` | Optional (SB-2703): a citation in that trace's answer that the reader found **out of date** rather than wrong, with the same rule. It puts the document on its spaces' review list, and is never turned into an eval case, because the cite wasn't wrong when it was made |
| `comment` | Optional, at most 2,000 characters |
| Repeats | One feedback per trace: a later one replaces the earlier one |
| Response | `200 { "trace": "tr_…", "stored": true }` |

Feedback is stored against the trace id with the release, and exported weekly (Monday 00:00 UTC, the previous seven days) as JSON lines for the pilot review. Each line holds the trace, release, rating, wrong citation, outdated citation, comment, time, and the answer's claim, uncited and refused counts, but never the question or the answer text. Reviewers who need those open the trace with auditor rights. Operators see every unhelpful answer in the console's queue, `GET /v1/admin/feedback?rating=down` (sb-admin(3); SB-1609), with the same counts, the trace's policy decisions and the asker's comment, and mark each one reviewed. A later feedback on a reviewed trace puts it back in the queue.

### The web UI

On GCP the web UI is served from a Firebase Hosting site in the cell's own project, which rewrites `/v1/` to that cell's `api` only (sb-runtime(7), Web UI hosting).

The cell serves its web chat UI at `/` on the tenant's own hostname, from the release's built assets (SB-307). It calls only this cell's `/v1` API, with the session cookie, and loads nothing from anywhere else. **Sources first** (SB-2304): asking sends `GET /v1/search` with `POST /v1/ask`, and while the answer is written the turn shows the documents search allowed, each by title, owner and age, under "Sources you can open, while the answer is written", never their text, so nothing there reads as a claim; the cited answer replaces them when it comes. They are search's results, checked live as the asker like any search; when search fails, the turn just waits for the answer. Every string it shows is in a catalog in English, German and French (`web/src/i18n/`; SB-2301): it starts in the manifest's `language` (sb-tenant(5)), a person can switch with the language picker at the top of every screen, and their choice is kept in their browser; a test fails on any string outside the catalog, and axe runs in each language. Answers stay in whatever language the model writes, and an uncited claim is marked in every language (LAW-14). When the API answers `EAUTH` (no session, or one that ended), it says the person is signed out and links to `/login`, which signs them in and returns them to `/`: the page is served with `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'` and no third-party scripts, fonts or analytics.

| Element | Rule |
|---|---|
| Claim with citations | Followed by one chip per citation, showing `cite`. With a `url`, the chip is a link that opens the source at the cited version in a new tab; without one, it links to the answer's trace |
| `uncited` claim | Styled differently **and** labelled with the word "Uncited" (never colour alone), with a tooltip saying no source the user can open supports it |
| `flags` | A chip with `injection_suspect` says "Flagged" next to the cite |
| `notice` | `no_access`: "Some sources you don't have access to may answer this. Ask their owner." `no_sources`: "Nothing indexed answers this yet." `most_uncited`: "Most of this answer isn't backed by your sources." |
| `refused` | "N sources were not shown because you can't open them", never their names |
| `notice_stale` | (SB-2701) One line **above** the answer: "Most of the sources for this answer haven't been reviewed in a while (the oldest is N days old). Check with their owner before relying on it." |
| `notice_single` | (SB-3005) One line above the answer: "This answer rests on one document. Check it if it matters." The chat apps put it first too |
| `notice_sync` | (SB-2704) One line above the answer for each kind: "docs last synced at 14:05; newer changes may be missing.", or "docs: when it last synced is unknown; newer changes may be missing." |
| Feedback | Under an answer, **Helpful** and **Not helpful**, a comment, and two choices of the answer's cites: **Wrong citation** and **Out of date** (SB-2703) |
| Space switcher | In a cell with spaces, a labelled select, "Ask in", lists "Whole company" and the spaces from `GET /v1/spaces`, and nothing else. The chosen space goes on the ask as `?space=`. Changing it starts a new conversation. A cell without spaces shows no switcher |
| Ask about a document | (SB-3004) Each citation chip and each Documents result has **Ask about this document**. It adds the document to a filter shown above the ask box as "Asking about: TITLE" (at most 5), with **Clear**; the next questions carry it as `documents` until it is cleared or the conversation starts fresh |
| Passage | (SB-3001) Each citation chip is followed by **Show passage**, a button that opens the cited text under the chip as a quote, with "Hide passage" to close it. A refusal shows "You can't open this source now." and no text |
| Section | (SB-3002) A chip with a `section` shows "Title › Section"; copied and printed cites do the same |
| Space and label | Each chip also shows the spaces that hold the source and its label, in words (for example "Finance · restricted"), as part of the chip's accessible name |
| Errors | The error's `message` and `remedy`, with the trace id to quote to support. An ask that fails with `EUNAVAIL` or `EBUDGET` also offers "Search documents instead", which runs the same text as a search |
| Tabs | One page with tabs: **Answer**, **Documents** (search), **Saved** and **Questions**, a labelled `tablist` usable by keyboard |
| Documents | Each result shows its title (or cite) as a link to the source, its snippet as a quote, its owner and age, and the same space and label words as a chip |
| Who to ask | Under an answer with `people`: "Ask ann@acme-logistics.example (Support lead), owner of *Refund policy*" for each person, naming only documents cited in that answer |
| Ask the stewards | Under an answer with `no_sources`, `no_access` or `most_uncited`: "Ask the stewards of Finance" (or "Ask the people who answer company-wide questions"). It shows exactly the text that will be sent, the question, and sends it only when the asker confirms |
| Questions | A fourth tab lists the asker's questions to stewards with each reply, its cites as chips and how many were dropped because the asker can't open them |
| Notifications | The Questions tab ends with "Tell me in chat about": a checkbox each for replies to the person's questions, new questions for spaces they steward, the weekly digest and watched questions, saved with `PUT /v1/me/notifications`, and **Pause for** 1 to 30 days, or **Resume** while paused (SB-2506), and **Send me a test** (SB-2602). `/?tab=questions`, the link replies are told with, opens the app on that tab |
| Save and share | Each answered turn has **Save** and **Share**. Share shows the `/q/sh_…` link to copy. The Saved tab lists saved questions; opening one asks it again |
| Watch | Each saved question in the Saved tab has a **Watch** checkbox (`PUT /v1/saved/{id}/watch`). `/?tab=saved` and `/?tab=documents` open the app on those tabs |
| Copy | Each answer has **Copy** and **Copy as Markdown** (SB-2401): the question, then each claim followed by the numbers of its cites, then the cites, numbered, with title and link (a cite with no `url` is its `cite`). An uncited claim is copied with the screen's word "Uncited" in front, hidden sources only as the screen's "N sources were not shown" line, and nothing else: no owner, trace or snippet beyond what the answer shows |
| Print | **Print** opens the browser's print dialog, where Save as PDF makes a file (SB-2402). The print stylesheet leaves out the tabs, the ask form and every button, and prints the question, the answer with "Uncited" where the screen has it, and the cites as numbered footnotes with their links and ages; nothing is rendered on the server |
| Keyboard | (SB-2403) A "Skip to the conversation" link comes first. `/` moves focus to the ask box, Ctrl+Enter (Cmd+Enter on a Mac) in the ask box asks, and Esc closes the API tokens panel and returns focus to its button. None fires while focus is in a field, except Ctrl+Enter in the ask box. A line under the ask box lists them |
| Recent questions | (SB-2405) Under the ask box, **Recent questions** lists the person's last 20 questions, newest first, each a button that puts it back in the box, and **Clear**. They are kept **in that browser only** (`localStorage`), never sent to the cell for this, so the cell keeps no more question text than before (LAW-07). They are cleared by **Clear**, by **Sign out everywhere**, and when the cell says the person is signed out (`EAUTH`). A shared question (`/q/sh_…`) asked from its link isn't added |
| No dead ends | (SB-2406) A search with no results, or an answer whose `notice` is `no_sources` or `no_access`, offers next steps: **Search the whole company** (or **Ask the whole company**) when it was in a space, **Clear filters** when a filter was set, **Ask it as a question** from Documents, and **Ask the stewards** under an answer (as above). Which appear depends only on what the person typed and chose, never on the index: a document the person can't open changes none of them (LAW-01). Glossary terms the question used are shown as everywhere (Glossary, exactly) |
| Stewards' tools | On the steward screen (SB-2501 to SB-2503): **Reviewed, still right** on each document to review, **I'll answer this** and **Let it go** on each open question, "ann@… is answering" on a question someone else claims, and **Use a template** with the space's templates over the reply box, which a steward can add to, edit and remove. The Questions and Documents to review headings say how many wait, read on load and every five minutes (SB-2504), and **Download CSV** saves the review list as shown, made in the browser (SB-2505) |
| Retire and tidy | (SB-2801 to SB-2804) Each document to review has **Retire from answers**, which asks for a reason and, optionally, the replacement's `KIND:ID`; **Retired** lists the space's retirements with **Bring back**. The list also shows **Same name, different text** groups, notes whose sources moved, and **Never cited** on request, each with **Retire from answers**, with the glossary terms no question used (SB-2805). Above the list, **Answers resting on stale sources** shows this month's share and last month's (SB-2807). A citation chip of a replacement says **Replaces OLD TITLE** |

The UI renders a claim only as cited (with at least one chip) or as uncited, never as plain text (LAW-14). It targets WCAG 2.2 AA: everything works by keyboard, focus is always visible, the answer area is an `aria-live` region, and every chip has an accessible name that includes the cite.

`cites[].url` is set when the source's connector can link to a version; it is omitted otherwise. `agent` names the agent that answered, and `dropped` counts the uncited claims a department agent left out, always 0 for `ask` (sb-agents(7), Department agents). In a cell with spaces, `cites[].spaces` names the spaces that hold the cited document and `cites[].label` its label (sb-spaces(7)); both are omitted in a cell without spaces (SB-1007).

### Spaces

`GET /v1/spaces` lists the spaces the asker is a member of, now, as `{"spaces": [{"name": "finance", "label": "restricted"}]}`, sorted by name. Spaces the asker isn't in are never listed, and a cell without spaces returns an empty list. A registry that can't answer is `EUNAVAIL`.

### The chat app (Slack)

Staff can also ask from Slack with `/ask <question>` (SB-308). It is the same `ask` agent with the same laws. Only the entry point differs: Slack calls `POST /chat/slack/commands` for the command and `POST /chat/slack/interactions` for the Share button.

1. **Verify.** The cell checks Slack's request signature with the signing secret from `vault://tenants/TENANT/connectors/chat` and refuses any request older than 5 minutes. The `team_id` must be the tenant's workspace. Anything else gets HTTP 401 and no reply.
2. **Map the person.** The bot looks up the Slack user's email (`users.info`) and finds the **one active** directory user with that email (SCIM-provisioned, sb-identity(7)). No match, more than one match, or an inactive user gets an ephemeral reply saying the account can't be matched, and **no answer**. The bot never falls back to a shared or service identity.
3. **Answer privately.** Slack gets an immediate ephemeral "Looking that up…". The answer follows through Slack's `response_url` (only `https://hooks.slack.com/`), also **ephemeral**, so only the asker sees it. The stale and sync warnings (`notice_stale`, `notice_sync`; SB-2701, SB-2704) come first, in the web UI's words. Claims show as text with their citations as links (`<url|cite>`). Uncited claims start with *Uncited:*. The reply ends with who to ask (the answer's `people`, each with the documents of theirs it cites), the notice, the refused count and the trace id.
4. **Spaces.** In a cell with spaces, the channel the command was typed in picks the space: a space's `channels` in the manifest (sb-space(5)) list the channels that ask in it. A mapped channel asks in that space, and someone who isn't a member gets an ephemeral reply saying so and **no answer**, never a company-wide one. An unmapped channel, or a direct message, asks company-wide.
5. **Share.** A *Share in channel* button posts the question and the **citation links only, never the claim text**, so each reader opens the sources under their own access. Only the asker can share, and the trace is read again with the asker's own scope.

Errors are ephemeral replies with the error's `message` and trace id. Slack scopes for the bot: `commands` for `/ask`, which answers through `response_url`, and `chat:write`, `im:write` and `users:read.email` for notifications (Notifications, above), which are direct messages that never carry an answer.

### The chat app (Mattermost)

Sites that run their own Mattermost can ask from it with a slash command (SB-2005), for example `/brain <question>`. The command is a Mattermost **custom slash command** whose request URL is the cell's `POST /chat/mattermost`, method POST; the cell takes it when sb-cell(5) sets `mattermost.base` and `mattermost.team_id`. It follows the Slack app's rules, from the same code:

1. **Verify.** The command's token (the form's `token`, or `Authorization: Token …`) must equal `command_token` in `vault://tenants/TENANT/mattermost`, read at every request, and `team_id` must be the tenant's team. Anything else is HTTP 401 and no reply.
2. **Map the person.** The cell reads the Mattermost user's email with the bot's token (`bot_token` in the same secret; `GET /api/v4/users/ID`), and finds the **one active** directory user with that email. A deleted Mattermost user, a bot, a hidden email, no match, more than one, or a deprovisioned user (LAW-12) gets an ephemeral reply that the account can't be matched, and **no answer**.
3. **Answer privately.** Mattermost gets an immediate ephemeral "Looking that up…", and the answer follows through the command's `response_url`, which must be the server's own `/hooks/commands/` URL, also **ephemeral**. It is Markdown: the question, the stale and sync warnings in the web UI's words (SB-2701, SB-2704), each claim with its citations as links (`` [`cite`](url) ``), uncited claims marked *Uncited:*, who to ask, the notice, the refused count and the trace id. Claim text is escaped, so it can't become a link, a mention or formatting.
4. **Spaces.** A space's `channels` (sb-space(5)) list Mattermost channels by name as they do Slack's: a mapped channel asks in that space, and someone who isn't a member gets an ephemeral refusal, never a company-wide answer.

5. **Share** (SB-2105). The answer carries a *Share in channel* button, a message attachment action whose integration URL is the cell's `POST /chat/mattermost/actions` (sb-cell(5) `base_url`). Its context holds the trace id, the asker's Mattermost user id, the command's `response_url` and an HMAC-SHA256 over the three keyed with `command_token`; Mattermost doesn't sign action requests, so the cell checks that HMAC with the token read at the press, and anything else is HTTP 401. Only the asker can share: the pressing `user_id` must be the one in the context, mapped again to the one active directory user, and the trace is read again with that scope. The channel then gets, through the `response_url` as `in_channel`, the **citation links only, never the claim text**, exactly the Slack rule, so each reader opens the sources under their own access. Anyone else pressing it gets "Only the person who asked can share this answer." `response_url` lasts 30 minutes, so a later press says the answer can't be shared any more.

The bot account needs to read users' emails (the System Console's *Show Email Address* for it) and to create direct channels for notifications (Notifications, above); replies and shares go through `response_url`.

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
| `ERATE` | The asker's `asks_per_minute` is used up; `retry_after` says how many seconds to wait (SB-2305) |
| `EUNAVAIL` | Index or model unavailable (never falls back to anything shared) |

A refused source is **not** an error. The answer reports it through `refused` and `notice`.

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03), [LAW-07](sb-laws.7.md#law-07), [LAW-12](sb-laws.7.md#law-12), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```sh
curl -sS https://acme-logistics.sb.example/v1/ask \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"question":"What is the refund approval limit?"}' | jq '.answer.claims[] | {text, cites: [.cites[].cite]}'
```

## SEE ALSO

[sb-citations(7)](sb-citations.7.md), [sb-permissions(7)](sb-permissions.7.md), [sb-api(3)](sb-api.3.md)

---
title: "sb-ingest(7)"
man: "7"
summary: Sync, sanitising, provenance, freshness and rebuild.
---

# sb-ingest(7)

## NAME

sb-ingest — how source content becomes searchable, and stays data

## SYNOPSIS

```text
Changes → Fetch → normalise → sanitise → chunk → provenance → embed → index
```

## DESCRIPTION

### Read-only by default

Connectors use read-only scopes unless a gated workflow needs write access to that connector. Paths and channels listed in `exclude` are never fetched.

### Freshness

Changes, including deletes and permission changes, are indexed with a p95 of 15 minutes or less. A delete removes all of the document's chunks at the next sync; the source's change feed is the tombstone record, so nothing else is kept. A permission-only change re-ingests the same version, so the prefilter matches the source again. In the gap, the live ACL check still protects every answer (LAW-01): the prefilter only narrows candidates and never grants.

### Sync

Each connector of a cell is synced by a scheduler every 60 seconds, with up to 10 seconds of jitter. One sync reads the connector's changes from its stored cursor, ingests them, and only then stores the new cursor. If anything fails, the old cursor stays, so the next run repeats the work. Ingest is idempotent, so repeating it is harmless. A cell restart resumes from the stored cursors and never from the beginning; a cursor that was never stored, or was lost, starts from the beginning, so nothing is skipped. The ingest service (`sb-ingest`, or `sb-api --with ingest`; sb-runtime(7)) re-routes every connector's documents to the current spaces before its first sync, runs each connector in its own loop, and backs a failing connector off (one minute, doubling, at most 15 minutes) without slowing the others. When a connector's last success, or the service's start if it has none, is more than 15 minutes old, it sends one `ingest_stale` alert (Sev-2) until the connector succeeds again. Secrets and fields dropped at ingest are counted in `sb_ingest_secrets_dropped_total{connector, pattern}` and `sb_ingest_fields_dropped_total{connector, field}`.

On GCP that scheduler is Cloud Scheduler, every minute, and a missed run alerts (sb-runtime(7), Schedules).

Freshness is measured per connector as the age of the last successful sync, exported as `sb_ingest_last_success_age_seconds{tenant, connector}`. It stays well under the 15-minute target while syncs succeed, and an alert fires when it passes 15 minutes. Failures are counted in `sb_ingest_sync_failures_total{tenant, connector}`.

### Content is data

Ingested text is written by people who aren't the asker, and some of it may be hostile. The sanitiser:

1. strips hidden text (zero-width characters, off-screen or white-on-white styling, comments, alt-text payloads);
2. **flags** spans that look like instructions to an AI;
3. at prompt time, wraps every chunk in a quoted data block with its provenance.

Tool arguments derived from flagged spans are refused by the tool bus (LAW-09). Flags show on citations, so users can see when a source contained suspicious text.

The rules, in order (`internal/core/ingest`, `Sanitise`):

| Step | Rule | Recorded as |
|---|---|---|
| Strip | Zero-width characters, bidirectional controls and other invisible format characters; HTML comments; elements styled `display:none`, `visibility:hidden`, `font-size:0` or positioned off-screen; image alt text and link titles longer than 100 characters | flag `hidden_text_removed` |
| Flag | Text addressed to an AI: overrides ("ignore previous instructions", "disregard the above"), new roles ("you are now", "act as"), chat-role markers (`system:`, `assistant:`, `<|im_start|>`), tool-call syntax (`{"tool": …}`, function-call JSON), requests to hide things from the user, and links that carry data in their query string to a domain outside the tenant | flag `instruction_like` |
| Wrap | At prompt time each chunk becomes `<<data cite="…" flags="…">>` … `<</data>>`. Any `<<` or `>>` inside the text is escaped first, so content can't close its own block | — |

Flags are per chunk and appear on its citation. A flagged chunk is still indexed and can be cited; it just can't steer the agent. The corpus in `evals/corpus/injection.json` (visible, hidden and benign look-alike cases) runs in `make check`. The agent-level injection suite (zero tool calls from poisoned sources) runs once the tool bus exists (SB-405, SB-501).

### Fields (SB-1106)

For a document that arrives as fields (sb-connector(3), Fields), ingest builds the text itself, before chunking, from the fields that are **allowed**:

- a declared field that isn't sensitive is allowed;
- a declared sensitive field is allowed only if the `fields` allow-list of a space that holds the document names it (sb-space(5)), and then the document is labelled `restricted`, whatever its source or space would say (LAW-20), and is held **only by the spaces that allow every sensitive field it carries**, so another space holding the same record never reads them. If no holding space allows them all, only the sensitive fields every holder allows go in (SB-1401);
- a field the connector didn't declare is never allowed.

Every other field is **dropped**: its value is never chunked, embedded, indexed or logged. The sync reports the dropped fields by name and count only, as `sb_ingest_fields_dropped_total{tenant, connector, field}`. The text is the document's title, then one `Name: value` line per allowed field, in the order the connector declares them. A document that arrives as fields from a connector that can't declare them isn't ingested at all, and is counted as `undeclared`, so a source that can't say what it holds never feeds a restricted space.

### Secrets never stored (SB-1103)

Every chunk of every connector passes a secret scanner before it's embedded or written, and so does the whole document first, so a secret a chunk boundary splits (a private key longer than a chunk) is found in every chunk it touches (SB-1401). A chunk that holds or touches what looks like a credential is **dropped**: a private key block from header to footer, a cloud access key, a GitHub, GitLab, Slack, Stripe or model-provider key, a JWT or bearer token, credentials in a URL (`postgres://user:pass@host`), or an assignment to a name that says it is a secret, such as `password = "…"`, `aws_secret_access_key=…` or `token: …`, with a long value, quoted or not. A dropped chunk is it is never embedded, indexed or logged, and `sb_ingest_secrets_dropped_total{tenant, connector}` counts it. The rest of the document is indexed as usual. Only the chunk's id and the pattern's name reach the trace of the sync, never the match (LAW-07).

### Personal data (SB-1904)

Ingest also looks for two kinds of personal data in every document: an **IBAN** whose check digits are right (ISO 13616, mod-97 = 1; 15 to 34 characters, grouped in fours or not) and a **payment card number** (13 to 19 digits, spaces or dashes between groups allowed) that passes the Luhn check and starts with a prefix a card network issues (Visa, Mastercard, American Express, Discover, JCB, Diners). A number with wrong check digits, or one inside a longer word or number, isn't one. Each chunk that holds a number, or part of one a chunk boundary cut, is flagged `personal_data`. Unlike a secret, it is indexed and can be cited: a payroll sheet is still a document people need. But nobody reads the whole number through Second Brain: a flagged chunk's search snippet, and the text the model is given to answer from, show each number masked but its last four characters, with separators kept (`•••• •••• •••• •••• ••30 00`), and every answer's text is masked the same way before it is returned or stored. The flag is derived from the text, so a rebuild gives the same flags (LAW-13). The space's stewards see how many documents in the space carry it (sb-ask(3), the review list), as a count, never which. The detectors have a fuzz target (`FuzzMaskPII`): a masked text never still holds a number, and keeps its length.

### Provenance

Every chunk records tenant, `cite` (`kind:id@version`), span, content hash, flags and the release that ingested it, and, from SB-1701 to SB-1708, the document's title and location and when the index **first saw this version** (`seen`). Ingesting the same version again keeps the first `seen`; a new version gets the time it was first ingested. That time is what search shows as a document's age, what "what changed" lists (sb-ask(3)) and what the stewards' review list measures staleness by.

Every chunk also carries the **text hash** of its document (SB-2202): the SHA-256 of the document's normalised text, without its title, the same on every chunk of the version and empty for a document with no text. Two documents with the same text, in two places or under two names, have the same hash, which is how the stewards' review list finds copies and how answers cite one copy (sb-ask(3)). Like every field of a chunk it depends only on the source, so a rebuild gives every document the same hash again (LAW-13). The hash is of the text, never the text itself, and never logged.

### Chunks

A document becomes chunks in a fixed order: normalise (line endings, trailing space), split into paragraphs, then pack paragraphs into chunks of at most 1,200 characters. A paragraph longer than that is cut at the last space before the limit. Boundaries depend only on the text, so the same text always gives the same chunks, and an edit to one paragraph changes only the chunks that contain it.

Each chunk's id is derived from tenant, `cite` and span, so ingesting the same `(source, version)` twice writes exactly the same chunks: ingest is idempotent. A newer version replaces all of the document's chunks in one write. An older or equal version than the one indexed changes nothing.

Embeddings come through the `Embedder` port. By default cells use a deterministic local embedder (feature hashing, 64 dimensions) that needs no network. A cell can instead embed with an OpenAI-compatible embeddings endpoint (`embed` in sb-cell(5); SB-1710), such as the model server on a self-hosted cell's own machine. **Changing the embedder rebuilds the index from the sources:** the ingest service records the embedder's id and dimensions under the cell's state, and when they differ at start it drops the index and starts every connector's sync again from the beginning, so no chunk is ever searched with vectors from two embedders; the Postgres index changes its vector column's size when it is empty (LAW-13). Chunks are written through the `Index` port; the in-memory index serves development and tests, and the Postgres/pgvector adapter comes with the retrieval work.

### Spaces and labels (SB-1005)

In a cell with spaces (sb-spaces(7)), ingest **routes** every document version before it writes a chunk, so nothing is ever indexed without knowing which spaces hold it and how sensitive it is:

1. **Location.** The connector reports where the document lives (sb-connector(3)): a Drive file's folder path and name (`/Finance/Q3/Forecast`), a Slack message's channel (`#eng-platform`), a Linear issue's team key (`ENG`), a HubSpot record's type (`deals`).
2. **Spaces.** A space holds the document if one of its `scope` patterns for the document's connector kind matches the location. In a pattern, `**` matches anything, `*` matches anything but `/`, and every other character matches itself, so `/Finance/**` holds everything under `/Finance/`, and `#eng-*` holds `#eng-platform`. A document can be held by several spaces, or by none.
3. **Label.** The source's label wins where the source has one (a Drive classification label; a private Slack channel is `confidential`). Otherwise the label is the **strictest** default among the spaces that hold it, on the ladder `public` < `internal` < `confidential` < `restricted`, so a document in a restricted space and an internal one is `restricted`. A document no space holds, with no source label, is `internal`. A source label the cell doesn't know is `restricted`: unknown never widens.

Every chunk of the version carries its label, its holders (`spc_NAME` ids) and the **routing fingerprint**: a hash of every space's id, label, scope and field allow-lists, and a routing version that changes when what ingest records about a document changes (version 2 adds the source owner, so every document is routed again once after the upgrade; SB-1401). When a manifest apply changes a scope, a label or a field allow-list, the fingerprint changes, and the next sync re-routes every document whose chunks carry the old one, before it handles new changes. A document the source can't give while it is re-routed (deleted, moved, out of reach) is taken out of the index, counted in `sb_ingest_reroute_dropped_total{connector}`, and the sync goes on: it is never left routed by an old configuration, and one document never stops a connector's sync. It returns when it next changes at the source, or on a rebuild (SB-1401). Routing depends only on the document's location and label and the manifest, so a rebuild routes every document exactly as before (LAW-13). A cell without spaces routes nothing: chunks carry no label and no holders, as today.

### Rebuild

The index is derived data. It can be dropped and rebuilt from sources at any time, and the scheduled drill proves it (LAW-13). Each cell's ingest worker runs the drill weekly (`ingest.Drill`): it snapshots the `(cite, chunk, ACL)` set, drops the index, replays every source from the start of its change feed, and compares. The result and the rebuild time are recorded per tenant. Any difference is a LAW-13 violation and a Sev-1, reported with chunk ids and cites, never content.

**First-seen times across a rebuild (SB-1708).** A rebuild reads every version's `seen` before it drops the index and gives each version the same time again when it is replayed, so a rebuild doesn't make old documents look new. A version the rebuild meets that wasn't indexed before (one that changed while the index was being dropped), or every version when the old index can't be read (a lost index), gets the rebuild's time and is **marked as rebuilt**. "What changed" never lists a rebuilt version, and the weekly report doesn't count one, so a rebuild never shows as a week of changes.

## LAWS

[LAW-09](sb-laws.7.md#law-09), [LAW-13](sb-laws.7.md#law-13), [LAW-01](sb-laws.7.md#law-01).

## EXAMPLES

```yaml
exclude:
  chat: ["#hr-*", "#legal-*", "#board"]
  docs: ["/sites/HR/**"]
```

## SEE ALSO

[sb-connector(3)](sb-connector.3.md), [04 Ingest pipeline](../docs/architecture/04-ingest-pipeline.md)

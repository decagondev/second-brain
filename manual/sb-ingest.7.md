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

Each connector of a cell is synced by a scheduler every 60 seconds, with up to 10 seconds of jitter. One sync reads the connector's changes from its stored cursor, ingests them, and only then stores the new cursor. If anything fails, the old cursor stays, so the next run repeats the work. Ingest is idempotent, so repeating it is harmless. A cell restart resumes from the stored cursors and never from the beginning.

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

### Provenance

Every chunk records tenant, `cite` (`kind:id@version`), span, content hash, flags and the release that ingested it.

### Chunks

A document becomes chunks in a fixed order: normalise (line endings, trailing space), split into paragraphs, then pack paragraphs into chunks of at most 1,200 characters. A paragraph longer than that is cut at the last space before the limit. Boundaries depend only on the text, so the same text always gives the same chunks, and an edit to one paragraph changes only the chunks that contain it.

Each chunk's id is derived from tenant, `cite` and span, so ingesting the same `(source, version)` twice writes exactly the same chunks: ingest is idempotent. A newer version replaces all of the document's chunks in one write. An older or equal version than the one indexed changes nothing.

Embeddings come through the `Embedder` port. Until the model provider is chosen (Q-04), cells use a deterministic local embedder (feature hashing, 64 dimensions) that needs no network. Chunks are written through the `Index` port; the in-memory index serves development and tests, and the Postgres/pgvector adapter comes with the retrieval work.

### Rebuild

The index is derived data. It can be dropped and rebuilt from sources at any time, and the scheduled drill proves it (LAW-13). Each cell's ingest worker runs the drill weekly (`ingest.Drill`): it snapshots the `(cite, chunk, ACL)` set, drops the index, replays every source from the start of its change feed, and compares. The result and the rebuild time are recorded per tenant. Any difference is a LAW-13 violation and a Sev-1, reported with chunk ids and cites, never content.

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

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

Changes, including deletes and permission changes, are indexed with a p95 of 15 minutes or less. In the gap, the live ACL check still protects every answer (LAW-01).

### Content is data

Ingested text is written by people who aren't the asker, and some of it may be hostile. The sanitiser:

1. strips hidden text (zero-width characters, off-screen or white-on-white styling, comments, alt-text payloads);
2. **flags** spans that look like instructions to an AI;
3. at prompt time, wraps every chunk in a quoted data block with its provenance.

Tool arguments derived from flagged spans are refused by the tool bus (LAW-09). Flags show on citations, so users can see when a source contained suspicious text.

### Provenance

Every chunk records tenant, `cite` (`kind:id@version`), span, content hash, flags and the release that ingested it.

### Rebuild

The index is derived data. It can be dropped and rebuilt from sources at any time, and the scheduled drill proves it (LAW-13).

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

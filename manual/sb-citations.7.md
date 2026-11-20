---
title: "sb-citations(7)"
man: "7"
summary: doc@version, claims, and the uncited label.
---

# sb-citations(7)

## NAME

sb-citations — what a citation promises, and what uncited means

## SYNOPSIS

```text
cite   = kind ":" source-id "@" version        e.g.  docs:4f2a9c11@17
answer = claim*      claim = text (cite+ | uncited)
```

## DESCRIPTION

### What a citation promises

A citation `docs:4f2a9c11@17` promises three things:

1. the asker could open that document **when the answer was built** (LAW-01);
2. the cited text came from **version 17** of it;
3. the chunk was in the context the model was given.

Whether the chunk really *supports* the claim is measured by the citation-faithfulness eval, not promised per answer.

### Uncited

A claim with no valid citation is labelled `uncited` **by code** (LAW-14). It may be correct general knowledge or reasoning, but the platform can't point to a source, so it says so. UIs always style `uncited` claims differently.

### What can't become a citation

- A chunk id the model made up.
- A chunk that wasn't in the allowed context.
- A document that was refused (deny or unknown).
- A chunk a space, label or barrier check refused (SB-1006): it never reaches the context, so it can't be cited.

These are stripped and logged as `citation_spoof`.

### How the composer decides (code, not the model)

The model sees each allowed chunk as a data block labelled `C1`, `C2`, … and must answer in JSON: `{"claims": [{"text": "…", "cites": ["C1", "C3"]}]}`. The composer then, in code:

1. maps each label to its chunk; a label that isn't one of this answer's allowed chunks is **stripped** and counted as `citation_spoof` (by label, never by text);
2. marks a claim `uncited` if none of its labels survived;
3. turns surviving labels into citations: `cite`, title, span and the chunk's flags;
4. treats output that isn't valid JSON of that shape as one claim, the whole text, `uncited`.

A claim never carries a citation the asker wasn't live-authorised for, and the model can't mark anything as cited by itself (LAW-01, LAW-14).

### Versions

Citations keep the version they used. If the source has moved on, the UI shows "newer version available" next to the citation.

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```text
Refunds over £500 need a team lead's approval. [1]
This changed in March. (uncited)

[1] Refund policy · docs:4f2a9c11@17 · "Approvals"
```

## SEE ALSO

[sb-ask(3)](sb-ask.3.md), [07 Answer composer](../docs/architecture/07-answer-composer.md)

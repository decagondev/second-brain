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
4. treats output that isn't valid JSON of that shape as one claim, the whole text, `uncited`;
5. keeps a claim's `"disagree": true` (SB-3304), the model's word that its sources say different things, only when its surviving citations name at least two different documents, and drops it otherwise.

A claim never carries a citation the asker wasn't live-authorised for, and the model can't mark anything as cited by itself (LAW-01, LAW-14).

### Numbers from tables (SB-4307)

When an ask uses the table tool (sb-ask(3), Numbers from tables), its result is one more passage, first among the sources, with the spreadsheet's own cite: the operation, the filters, the range of cells it read and the value, all computed in code from the chunks the asker may open. A claim cites it as any passage. Then one more check, in code: a claim that cites a spreadsheet and states a number found neither in the text of the passages it cites nor as the tool's value is **dropped**, and counted in `dropped`, because a number the model worked out itself is not one a source says (LAW-14). A number is a run of digits, with `,` between thousands and `.` before decimals; it matches the tool's value at the precision it is written in, so 33.33 matches 33.3333.

### The passage (SB-3001)

A citation can be opened as its passage: the cited chunk's own text at the cited version, at most 600 characters, cut at a sentence (sb-ask(3), The cited passage). It is the source's text, never the model's, shown as a quote and never as markup (LAW-09). It is checked live each time it is opened, exactly as a retrieval is (sb-permissions(7)), so a citation someone could open yesterday can't show them its text today if they have lost access (LAW-01). A citation also names its **section**, the nearest heading above it (sb-ingest(7), Sections; SB-3002).

### Sources that disagree (SB-3304)

A claim marked `disagree` cites every source it rests on, so the asker can open each one and judge. The mark is the model's word, kept by code only when two different documents survived the live check for this asker; a document the asker can't open never makes a disagreement and is never hinted at. The citation suite has a case whose documents disagree and one whose documents agree (sb-evals(7)).

### Replaced decisions (SB-4505)

A decision can name the one it replaces (`supersedes:`, sb-ask(3), Work across departments). A cite of the older decision is still a cite of what that document says: it is kept, and carries `replaced_by` with the newer decision's cite and title only when the asker can open the newer one now, checked live like any cite (LAW-01). The composer never swaps one for the other, and a newer decision the asker can't open is neither named nor hinted at. Nothing is linked by the model: a link comes from a decision's own front matter, or from a steward confirming it when publishing a decisions draft.
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

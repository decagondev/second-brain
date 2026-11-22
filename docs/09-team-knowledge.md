---
title: Team Knowledge
summary: Research and concept for W16 (proposed): ten small features that make the brain a place where a team hands knowledge to each other, on one sovereign machine.
---

# Team knowledge

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-22 | Platform lead | Proposed, for the user's review | [Principles](01-principles.md) · [ADR-0002](adr/ADR-0002-source-acl-is-authority.md) · [ADR-0022](adr/ADR-0022-self-hosted-cell.md) · [PRD waves](03-prd.md#waves) · [E17 plan](../epics/E17-team-knowledge/README.md) |

## Summary

After W15 a company can run its whole brain on one machine it owns: it asks, gets cited answers from a file share, and its operators see health and unhelpful answers. What it can't yet do well is the thing a team most wants from a second brain: **pass knowledge from the person who has it to the person who needs it**, and keep it from going stale.

Today an answer ends at the answer. When nothing answers, the asker is told so and left there. Nobody knows who to ask, a good answer can't be kept or handed on, a new joiner has nowhere to start, a leaver's knowledge goes with them, and nobody sees which documents everyone relies on but nobody has touched in a year.

This plan proposes **W16 Team Knowledge** (epic E17, SB-1701 to SB-1710): ten small features, each a few endpoints and a console or UI section on records the cell already keeps. None adds a service the cell must run, a store, a third-party Go module or a per-company code path. Every one keeps the source ACL as the authority (ADR-0002), counts what a person can't see instead of naming it, and writes its audit row first. The tenth puts a local model on the box, so a cell can answer with nothing leaving the machine at all.

## What blocks knowledge transfer in a team

Research on how teams share knowledge keeps landing on the same few points, and each one is a gap in the product today.

| What teams need | Why it matters | What the cell does today | Gap |
|---|---|---|---|
| **Find it fast** | Most "where is that doc" questions are lookups, not essays. A slow or costly answer for a lookup is friction | Every question goes through the model; with the model down or the budget spent, nothing answers (`EUNAVAIL`, `EBUDGET`) | No way to find documents without the model |
| **Know who knows** | Teams work best when members know *who knows what* and route questions there (transactive memory). The source of most tacit knowledge is a person, not a page | Chunks carry their owner (SB-1203) for barriers, but answers never show it | Owners are known and never shown |
| **Ask a person when the docs don't say** | The answers people need most are the ones nobody wrote down | `no_sources` and `most_uncited` say so, and stop. Unhelpful answers reach an operator's queue (SB-1609), never the person who knows | No route from an unanswered question to an owner |
| **Capture the answer once** | Knowledge-Centered Service's solve loop: capture the answer while solving it, so the next person finds it. Answers that live only in a chat thread are found once | A steward's answer, if any, lives in chat and is never indexed with an ACL | No way for an answer to become a document |
| **Keep and pass on a good answer** | People re-find the same answer weekly and forward answers to colleagues | Conversations exist; the Slack Share button posts citations only. The web UI can't save or share | No save, no share in the web UI |
| **Onboard** | New joiners lose weeks to "where do I start". Every team has a short reading list in someone's head | Nothing per space | No "start here" |
| **Hand over** | When someone leaves or changes role, their documents and the context around them are the knowledge most often lost | Deprovisioning revokes in 5 minutes (LAW-12); nothing helps a successor first | No handover brief |
| **Stay aware** | Teams need to know what changed this week in the things they work on | Ingest knows every new version; nobody is told | No "what changed" |
| **Trust that it's current** | The most damaging answer is a confident one from a document that's out of date | Every cite is `doc@version`, with no age, and nobody sees which old documents answer the most questions | No freshness signal, no review list |
| **Keep it on the box** | A sovereign cell that sends every question to a hosted model isn't fully sovereign | `model.provider: openaicompat` can already point anywhere; embeddings are the 64-dimension hashing embedder | No supported local model or embedder in the layout |

## The ten features

| # | Ticket | Feature | For | In one line |
|---|---|---|---|---|
| 1 | SB-1701 | Search without the model | Asker | Find documents you can open, ranked, with a snippet, in under a second and at no cost |
| 2 | SB-1702 | Who to ask | Asker | Each cite shows its document's owner; an answer lists the people who own its sources |
| 3 | SB-1703 | Save and share a question | Asker | Keep a question and re-ask it live; send a colleague a link that asks it under *their* access |
| 4 | SB-1704 | Ask the stewards | Asker, steward | When nothing answers, send the question, with consent, to the space's stewards; their reply comes back with its cites re-checked |
| 5 | SB-1705 | Answer once: replies become notes | Steward | A steward publishes a reply as a Markdown note with an ACL file in the notes folder, so the next person gets a cited answer |
| 6 | SB-1706 | Start here | New joiner, steward | A short reading list per space in the manifest, each item checked live |
| 7 | SB-1707 | Handover brief | Successor, manager | A cited brief of what a person owns, for whoever takes over, before deprovisioning |
| 8 | SB-1708 | What changed | Asker | Documents that changed this week in your spaces, only the ones you can open |
| 9 | SB-1709 | Documents to review | Steward | Documents cited often but unchanged for a year, or flagged as wrong, for the space's stewards |
| 10 | SB-1710 | A local model on the box | Operator | An optional model container for answers and embeddings, pinned and gated, with no route out |

### 1. Search without the model (SB-1701)

`GET /v1/search?q=…[&space=…]` runs the `ask` path's retrieval, space checks and live ACL batch exactly, and stops before the composer: it returns up to 20 documents, each once, with its title, `doc@version`, link, owner (with SB-1702), age and the best chunk's first 300 characters. No model call, so no tokens and no budget, and on a CPU-only box it answers in well under a second. Every search gets a trace and audit row like an ask. The web UI gets a "Documents" tab beside "Answer", and when an ask fails with `EUNAVAIL` or `EBUDGET` the UI offers the same query as a search.

*Why it's cheap:* retrieval and the live check are already one call (SB-301, SB-302); this is a second exit from it. *Laws:* LAW-01, LAW-03 (the same live check), LAW-14 (a snippet is a quote with its cite, never a claim).

### 2. Who to ask (SB-1702)

Every chunk already carries its document's owner. Each cite in an answer gains `owner` (the owner's email and job `title` from the directory, only for an active user), and the answer gains `people`: the owners of its allowed cites, most cited first, at most three. The UI shows "Ask ann@acme.example (Support lead), owner of *Refund policy*". An owner is shown only on a cite the asker was allowed, so nobody learns who owns a document they can't open. No activity, chat or "expertise" profile is built: ownership at the source is the only signal, which keeps it clear of employee monitoring and works-council questions (Q-12).

*Laws:* LAW-01 (owners only from allowed cites), LAW-12 (a deprovisioned owner is never suggested), LAW-22 (barriers already check the owner's groups).

### 3. Save and share a question (SB-1703)

A person can save a question (their own text, nothing of the answer) and see their saved list; opening one asks it again, live, so a saved question never shows something they've since lost access to. Sharing makes a link, `/q/SHARE`, that shows the recipient the sharer's question and asks it under the **recipient's** access, the same rule as the Slack Share button: never the sharer's answer. Saved and shared questions are cell records under the tenant's retention, erased with the person (`POST /v1/admin/subjects/erase`), and creating a share writes an audit row.

*Laws:* LAW-01 (the recipient's own live check), LAW-02 (a share link from another tenant is `EPERM`), LAW-07 (question text never in logs or audit, only its hash).

### 4. Ask the stewards (SB-1704)

When an answer's notice is `no_sources`, `no_access` or `most_uncited`, the UI offers "Ask the stewards of *Finance*" (or, for a company-wide ask, the manifest's `questions_to` group). The asker sees exactly what will be sent, their question and nothing else, and sends it. Stewards get a queue on their screen; a reply is text a person writes, with cites they add from their own search. The reply reaches the asker in a "Questions" list, and every cite is checked live **for the asker**: a cite they can't open is dropped and counted, so a steward can't widen access by quoting. Unanswered questions age out at the space's retention. The weekly report counts questions asked and answered per space, never their text.

*Why it fits:* it turns the operator's unhelpful-answer queue (SB-1609) into a loop that reaches the person who knows, and the question text travels only because the asker chose to send it. *Laws:* LAW-01 (cites re-checked for the asker), LAW-04 (audit rows for send and reply), LAW-07, LAW-21 (a steward who moves out of the space stops seeing its queue at the next decision).

### 5. Answer once: replies become notes (SB-1705)

A steward's reply can be published as a **note**: a Markdown file with an ACL file beside it, written into a notes directory the `docs: folder` connector reads. The next ingest indexes it, so the next person who asks gets a cited answer instead of a question to the stewards. This is the solve loop of Knowledge-Centered Service on the cell's own rules:

- The note is written by a person and published through the tool bus as the tool `note-publish` on the folder connector's `act`: kill check, authorise, audit row first, then the write (LAW-04). The receipt is the file's path and SHA-256 read back after the write, and the note is `Ran` only with it (LAW-05).
- The note's readers default to the space's members and can only be narrowed. Before publishing, the steward sees how many of the note's cites are readable by fewer people than the note will be, and must confirm, because the note is new text and the source ACL can't follow it.
- Only `act` writes, only under `notes/` in the share, only `.md` and `.acl`, opened as an `os.Root` so it can't leave. The rest of the share stays read-only, as mounted today.
- No model writes a note. A note is the steward's own words, with the steward as `owner`.

*Laws:* LAW-04, LAW-05, LAW-09 (content can't cause the write; only the steward's click), LAW-13 (notes are source documents, so a rebuild keeps them), LAW-15 (the publish binds the note's hash).

### 6. Start here (SB-1706)

A space gains `start_here`: up to ten cites or folder paths, in order, in the manifest (so a change is a proposal a steward makes and an operator approves, SB-1201). Members see "Start here" when they open the space; each item is checked live and an item they can't open is counted, not named. `sb tenant verify` reports any item no member of the space can open, so a broken list is found before a new joiner finds it.

*Laws:* LAW-01, LAW-16 (the list is in the manifest), LAW-03 (an item that can't be checked isn't shown).

### 7. Handover brief (SB-1707)

`handover` is a read-only agent in the department-agent mould (sb-agents(7)): asked with a person's directory id, it builds a brief of the documents that person owns that **the asker** can open, grouped by location, newest first, every line cited, uncited lines dropped. It adds how many more documents the person owns that the asker can't open, as a count, so a manager knows to ask for access to be moved before deprovisioning revokes it. Who may ask for whom is policy: the person's manager (from SCIM's `manager`), a steward of a space the person is in, or the person themselves.

*Why it's cheap:* it is the `ask` path with a filter on `owner` and a prompt in the bundle. *Laws:* LAW-01, LAW-14, LAW-12 (it works on the leaver's documents, never their access), LAW-09 (no tools).

### 8. What changed (SB-1708)

`GET /v1/changes?since=7d[&space=…]` lists documents with a new version in the window, in the spaces the person is a member of, newest first, each checked live, with title, cite, owner and age. The index records when it first saw each version; a rebuild's versions are marked so a rebuild doesn't look like a week of changes. The UI shows "What changed this week" on the empty answer page. The weekly report gains a count per space. No email: a cell has no mail adapter, and a feed people open beats a digest they filter.

*Laws:* LAW-01, LAW-13 (first-seen times are derived; a rebuild marks, never invents, changes).

### 9. Documents to review (SB-1709)

A steward's screen gains "Documents to review": documents held by their space that were cited in answers at least five times in the last 30 days and whose version is older than the space's `review_after_days` (default 365), plus any document marked as a wrong citation (SB-310). Counts come from the traces, which already hold each answer's cites. Each item is checked live **as the steward**: one they can't open is counted, not named, with its owner (if any) so they can ask the owner. A cite in an answer gains `age_days`, and the UI shows a document's age on its chip once it passes the space's threshold.

*Laws:* LAW-01 (the steward's own live check), LAW-14 (age is shown, never hidden), LAW-16.

### 10. A local model on the box (SB-1710)

An optional compose profile, `model`, runs one OpenAI-compatible model server (llama.cpp's server, pinned by digest) on the `data` network, which has no route out. `model.provider: openaicompat` with `model.base: http://model:8080/v1` already works; this adds an `openaicompat` embedder behind the `Embedder` port, the weights' SHA-256 pinned in the release bundle with the model id (P5), the eval gate run on that exact model, weights in the offline install bundle, and `make selfhost-model` to try it. Changing the embedder rebuilds the index from the sources (LAW-13). Price is 0, so the budget guard only counts.

*Honest limits:* a small quantised model on CPU answers in tens of seconds, so NFR-01's 6-second p95 isn't met without a GPU; search (SB-1701) stays the fast path. A model that fails the gate doesn't run, whatever the operator wants. *Laws:* LAW-06 (the model is part of the release), LAW-07 (the model server holds no secret), LAW-02 (one model server per cell, never shared).

## Keeping it small

Each feature has to pass five rules, or it doesn't go in.

1. **No new service the cell must run.** The model container is a profile, off by default.
2. **No new store.** Saved questions, shares, steward questions and replies are `cell_docs` kinds; notes are files in the share; first-seen times are a column in the index.
3. **No new third-party Go module.** Everything is the standard library and the adapters that exist.
4. **At most one manifest field a feature, for everyone** (P3): `start_here`, `review_after_days`, `questions_to` and the notes directory, and nothing else.
5. **One page.** The web UI stays one page with tabs (Answer, Documents, Saved, Questions); the steward screen gains two sections.

Indicative footprint: a cell without the model runs as today on a 4 vCPU, 16 GB VM; with the `model` profile and a 7 to 8 billion parameter quantised model, 8 vCPU and 32 GB, or a small GPU.

## Considered and left out

| Idea | Why not now |
|---|---|
| Model-written wiki pages or summaries stored as documents | The model would become an author whose text others trust; notes stay a person's words (P5, LAW-14) |
| Expertise profiles from chat activity | Monitoring people, and a works-council question (Q-12); ownership at the source is enough |
| A glossary that expands queries | Useful, but it changes retrieval for everyone from one document; revisit after search lands |
| Email digests | No mail adapter in the cell; a feed is enough for W16 |
| A self-hosted chat entry point (Mattermost or Matrix) | A good W17 candidate for sovereign sites; the Slack entry point's pattern carries over |
| Meeting recordings and transcription | A new kind of source with its own consent questions; out of scope |
| Leaderboards and badges | Rewards volume, not answers |
| Private personal notes | A second store of personal content with its own retention and erasure rules; saved questions cover the need |

## Decisions for the user

1. **Accept W16 as proposed**, or pick a subset. The recommended order is the slices in the [plan](../epics/E17-team-knowledge/README.md).
2. **Notes approval (SB-1705).** Recommended: the steward's publish is the approval, bound to the note's hash, with segregation of duties off for notes. The alternative is a second steward approving each note, which is safer and slower.
3. **Model server (SB-1710).** Recommended: llama.cpp's server (one binary, MIT licence, answers and embeddings, CPU or GPU). The alternative is Ollama, easier to try and heavier to pin.
4. **Who may ask for a handover brief (SB-1707).** Recommended: the person's manager, a steward of one of their spaces, or the person.

## Sources

- Consortium for Service Innovation, [KCS v6 Practices Guide: the Solve Loop](https://library.serviceinnovation.org/KCS/Knowledge-Centered_Success_Practices_Guide/201-Solve_Loop): capture knowledge in the workflow, reuse it, improve it as it is used.
- Atlassian, [Knowledge-Centered Service](https://www.atlassian.com/itsm/knowledge-management/kcs): the same loop, as practised by service teams.
- Ren and Argote, [Transactive Memory Systems 1985–2010](https://pubsonline.informs.org/doi/references/10.1287/orsc.1110.0647) (Organization Science): teams perform better when members know who knows what.
- Lewis, Lange and Gillis, [Transactive Memory Systems, Learning, and Learning Transfer](https://pubsonline.informs.org/doi/references/10.1287/orsc.1050.0143) (Organization Science): knowing who knows what helps a team carry learning to new tasks.
- Ollama, [OpenAI compatibility](https://ollama.readthedocs.io/en/openai/), and llama.cpp, [the server's embeddings endpoint](https://www.mintlify.com/ggml-org/llama.cpp/api/rest/embeddings): both serve the OpenAI chat and embeddings API locally, which the `openaicompat` adapter already speaks.

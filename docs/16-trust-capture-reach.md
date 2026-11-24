---
title: Trust Each Answer, Capture Conversations, Work Where People Are
summary: Concept for W35 to W37 (accepted 2026-11-24): twenty small features drawn from what other company knowledge tools do well, kept to the bloat rules. No new service, store or Go module.
---

# Trust each answer, capture conversations, work where people are

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-24 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [15-ask-capture-trust.md](15-ask-capture-trust.md) · [PRD waves](03-prd.md#waves) |
| 0.2 | 2026-11-24 | Platform lead | Plan accepted: every recommended decision (2026-11-24) | [Principles](01-principles.md) · [15-ask-capture-trust.md](15-ask-capture-trust.md) · [PRD waves](03-prd.md#waves) |

## Summary

W32 to W34 are built and pushed (`51af6eb`). This time the plan starts from other products, not just from the code. We read what company knowledge tools and personal second brains offer in 2026, and asked of each feature: does it fit a brain that cites everything, checks every source live as the asker, and runs in the company's own cell?

**What the others do well**

- **Guru** gives every card a named verifier and a review interval (30, 60 or 90 days, 6 months, a year), lets answers search verified content only, and drops a verification when people rate it unhelpful or a near-duplicate appears. Its health skills find gaps, stale and conflicting content, and send a weekly digest of question trends. It turns a Slack thread into a draft card, and serves knowledge to other AI tools through an MCP server.
- **Slite** lets Ask use verified documents only, and verifies, deprecates and reassigns in bulk.
- **Tettra** turns repeated questions into reusable answers and sends questions to subject-matter experts.
- **Glean** has verified answers, agent governance and usage controls.
- **Atlassian Rovo** keeps a list of every agent and integration, with who built it, where it runs and how much it's used. It has adoption dashboards, and an MCP server for outside tools.
- **Notion AI** and **Tana** capture meetings and turn them into searchable notes, and Tana keeps decisions as typed records. Tana holds every AI write for a person to approve.

**What we already have.** Review marks (Guru's verification, without the interval), found-nothing gaps, copies (duplicates), notes from replies, suggested questions, owners and caretakers, watched questions and digests.

**What's missing.**

- **Trust.** Pinned answers, a verified-only mode, a review interval for each document, conflicts collected in one place, and a way for an asker to report a bad answer.
- **Capture.** Meeting transcripts, decisions, chat threads, question trends, and sending a question to a document's owner.
- **Reach.** Other AI tools through MCP, asking about a file, a list of every token, adoption, a daily brief and browser search.

Three waves, twenty tickets. Each reuses something the cell already has.

**What we leave out.** Autonomous agents that act across tools (Rovo Max, Glean agents), since actions here already go through approvals (sb-approvals(7)). A browser extension, which would be a new client to ship. An editor AI that rewrites documents, since the source stays the truth. Bot-free meeting recording, since the brain reads transcripts but never records anyone.

## Keeping it small

The bloat rules still apply:

1. **No new service the cell must run.** Everything runs in `sb-api`, `sb-jobs` or the existing scripts.
2. **No new store.** New records are `cell_docs` kinds or small files under `state/`.
3. **No new third-party Go module.** MCP is a small JSON-RPC handler written in the repo.
4. **At most one setting per feature.**
5. **No new page.** Features go in as chips, lines, list sections and buttons.
6. **Nothing widens what anyone sees.** Every title, count and list is checked live as the person.

One more rule for these waves: **a person approves every write.** A transcript, a chat thread or a pinned answer reaches the index only after a steward's click, as notes already do (SB-1705, LAW-16). Nothing the brain writes is published by itself.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line | From |
|---|---|---|---|---|---|---|
| 1 | SB-3601 | W35 | Verified answers | Steward, asker | A steward pins an answer to a question. A close match shows it first, with each cite checked again as the asker | Guru, Glean, Tettra |
| 2 | SB-3602 | W35 | Verified only | Asker | A **Verified only** toggle answers and searches from documents with a live review mark, and notes | Guru, Slite |
| 3 | SB-3603 | W35 | Review every N days | Steward, owner | A review mark takes an interval of 30, 90, 180 or 365 days instead of the space's `review_after_days` | Guru |
| 4 | SB-3604 | W35 | Unhelpful clears the mark | Steward | 3 unhelpful ratings of answers citing a marked document since its mark put it back on the review list, with why | Guru |
| 5 | SB-3605 | W35 | Conflicts | Steward | The review list names pairs of documents that answers marked "Sources disagree" this month, with a count | Guru |
| 6 | SB-3606 | W35 | Report this answer | Asker, steward | On a bad answer the asker can **Send to the stewards**. The stewards see the question, the answer and what it was drawn from, checked as them | Guru (audit a wrong answer) |
| 7 | SB-3607 | W35 | How sure the answer is | Asker | A line under the answer says how many of its documents are verified, like "2 of 3 sources verified" | Slite, Guru |
| 8 | SB-3701 | W36 | Meeting transcripts | Everyone | The folder connector reads `.vtt` and `.srt` transcripts, with speaker and time, and cites them by time | Notion, Tana |
| 9 | SB-3702 | W36 | Decisions | Asker | A section headed "Decision" is typed `decision`. `type=decision` and a **Decisions** filter find them | Tana |
| 10 | SB-3703 | W36 | Save a chat thread | Everyone, steward | A Slack or Mattermost message action sends a thread to the space's stewards as a draft note. Their click publishes it | Guru |
| 11 | SB-3704 | W36 | Question trends | Steward | The weekly digest lists the space's top 5 asked terms, each with at least 5 asks, as counts | Guru |
| 12 | SB-3705 | W36 | Write the missing document | Steward | A found-nothing term on the review list gets **Draft a note**, filled with the term and the count | Slite, Guru |
| 13 | SB-3706 | W36 | Ask the owner | Asker, owner | From "Who to ask", the asker can send the question to the document's owner. Replies use the steward question flow | Tettra |
| 14 | SB-3707 | W36 | Already written? | Steward | Before a note is published, the steward sees up to 3 documents in the space that read the same, which they can open | Guru |
| 15 | SB-3801 | W37 | Use the brain from other AI tools | Everyone | `POST /v1/mcp` gives read-only `search`, `ask` and `passage` tools to an MCP client with the person's `sbt_` token | Guru, Rovo, Tana |
| 16 | SB-3802 | W37 | Ask about a file | Asker | Attach one file (up to 2 MB, any type the connectors read) to one ask. It's read once and never kept | Notion, Glean |
| 17 | SB-3803 | W37 | Every token in one place | Admin | `GET /v1/admin/tokens` lists every API token: owner, name, created, last used and expiry, never the token. **Revoke** ends one | Rovo |
| 18 | SB-3804 | W37 | Adoption | Admin | The weekly report says how many of the seats asked anything, and by space with at least 5 people | Rovo, Guru |
| 19 | SB-3805 | W37 | Daily brief | Everyone | An opt-in morning message: what changed in your spaces and your watched questions, in place of the weekly digest | Glean, Notion |
| 20 | SB-3806 | W37 | Search from the address bar | Everyone | `/opensearch.xml` lets the browser search the brain from its address bar | Glean (extension, lighter) |

## W35 Trust each answer (epic E36, SB-3601 to SB-3607)

The goal is that an asker knows how far to trust an answer, and a steward can fix one that was wrong.

### S35.1 Answers people can rely on

**SB-3601 Verified answers.** A steward of a space can pin a reply or a good answer to its question, with its cites. It's one `cell_docs` record per pin, audited, and kept to 50 a space. When a new ask in the space matches a pinned question closely (the embedder, as in SB-3402), the answer shows the pinned one first as "Verified by the stewards on DATE". Each cite is checked again as the asker, and a pin with a cite they can't open isn't shown to them. The model's answer follows as usual. A pin comes due after the space's `review_after_days`.
*Laws:* LAW-01, LAW-14, LAW-19. *Acceptance:* a close question gets the pin first. An asker who can't open a cite doesn't get the pin. A pin from another space never shows.

**SB-3602 Verified only.** `POST /v1/ask` and `GET /v1/search` take `"verified": true`. Retrieval then keeps only chunks of documents with a live review mark at their current version (SB-2501), and notes. It only ever narrows. The UI gets a **Verified only** toggle, kept in the browser. If nothing verified matches, the answer says so, as found-nothing does.
*Laws:* LAW-01, LAW-13 (a mark at an old version doesn't count). *Acceptance:* an unmarked document is never cited with the toggle on. A new version drops out until it's marked again.

**SB-3603 Review every N days.** **Reviewed, still right** takes `"days": 30 | 90 | 180 | 365`. The mark then stays good for that long instead of the space's `review_after_days`, and the document comes due at its end. Without `days` nothing changes. Owners marking their own documents (SB-3106) get the same choice.
*Laws:* LAW-13. *Acceptance:* a 30-day mark comes due on day 30. Another value gives `EINVAL`.

**SB-3604 Unhelpful clears the mark.** When answers citing a marked document get 3 unhelpful ratings after its mark, it goes back on the review list with "rated unhelpful 3 times since its review". Ratings before the mark don't count, the same rule as wrong-citation reports (SB-2501). The ratings are counted, and no asker is named.
*Laws:* LAW-07. *Acceptance:* 3 unhelpful ratings list it again, and 2 don't. Ratings before the mark don't count.

### S35.2 When an answer goes wrong

**SB-3605 Conflicts.** The review list gains **Conflicts**: pairs of the space's documents that answers marked "Sources disagree" (SB-3304) this month, each with its count, most first. Only pairs where the steward can open both documents are named. The rest are counted.
*Laws:* LAW-01, LAW-07. *Acceptance:* two answers that disagree on A and B list "A and B, 2". A pair with a document the steward can't open is only counted.

**SB-3606 Report this answer.** Under a rating of unhelpful, the asker can choose **Send to the stewards**, with an optional note. The answer goes to the space's question queue as a report. It holds the question, the answer and the documents it was drawn from. The steward sees each document by title only if they can open it, and the others as a count. The asker is named, since they chose to send it. Nothing is sent without that click.
*Laws:* LAW-01, LAW-07 (only by the asker's choice), LAW-11. *Acceptance:* a report reaches the stewards with its documents checked as the steward. An unhelpful rating alone sends nothing.

**SB-3607 How sure the answer is.** The response gains `verified: { "of": 3, "marked": 2 }`, counting the cited documents with a live review mark. The UI shows "2 of 3 sources verified" under the answer, and the chat apps say it in words. With no cites there's no line.
*Laws:* LAW-14. *Acceptance:* the count matches the marks at the cited versions. A stale mark doesn't count.

## W36 Capture conversations (epic E37, SB-3701 to SB-3707)

The goal is that what people decide in meetings and chat reaches the brain, after a steward's click.

### S36.1 From meetings and chat

**SB-3701 Meeting transcripts.** The folder connector reads `.vtt` and `.srt` files. Each cue's speaker and text become the document's lines, and its start time becomes the cite's place (`docs:Ops/standup.vtt@3#t=00:12:40`). Speaker names stay as text in the document, and personal data masking (SB-1906) applies as it does to any document. The brain never records a meeting. It reads only the transcripts people put in the share.
*Laws:* LAW-09 (a cue is text), LAW-01. *Acceptance:* a transcript is found by what was said, and cited at its time. A malformed cue is skipped and counted.

**SB-3702 Decisions.** A section headed "Decision" or "Decisions", or a note with front matter `type: decision`, is typed `decision` (`ask.TypeOf`). `GET /v1/search?type=decision` lists them, newest first, and the Documents tab gets a **Decisions** filter. Nothing else changes.
*Laws:* LAW-01. *Acceptance:* a transcript with a "Decision:" section shows under Decisions. Other documents don't.

**SB-3703 Save a chat thread.** A Slack or Mattermost message action, **Save to the brain**, sends the thread's messages to the space's stewards as a draft note, with who sent it and a link to the thread. It joins the question queue as a draft. **Publish as a note** runs the note flow (SB-1705) and **Discard** drops it. Only people the cell knows can send one, and only a steward's click publishes.
*Laws:* LAW-16, LAW-11, LAW-12. *Acceptance:* a saved thread waits for a steward. A discarded one never reaches the index. Someone the cell doesn't know gets a refusal.

**SB-3704 Question trends.** The stewards' weekly digest gains one line: the space's 5 most asked terms this week, by the same term counting as found-nothing (SB-3101). Each term needs at least 5 asks, and the line shows counts only, like "Asked most: cutoff 14, PO 9". Below 5 asks there's no line.
*Laws:* LAW-07. *Acceptance:* the line lists only terms with at least 5 asks. A space with fewer asks gets no line.

### S36.2 Fill it in once

**SB-3705 Write the missing document.** Each found-nothing term on the review list (SB-3101) gets **Draft a note**. It opens the note form with the term as the title and "Asked N times this month, found nothing" as a comment. The steward writes the text, and publishing runs the note flow.
*Laws:* LAW-16. *Acceptance:* the draft opens with the term. Nothing is published without the steward's click.

**SB-3706 Ask the owner.** In "Who to ask", **Ask them** sends the question to that person, for an owner or caretaker who is one active user. It uses the steward question flow with the owner as the only answerer: their reply, the notification and the claim all work the same. It's audited, and the owner sees only the question.
*Laws:* LAW-12, LAW-07, LAW-11. *Acceptance:* the owner gets the question and their reply reaches the asker. A deprovisioned owner can't be asked.

**SB-3707 Already written?** Before **Publish as a note**, the steward sees up to 3 documents in the space whose text reads the same as the note (the embedder, as in SB-3402), each checked as the steward. They can still publish. It's a warning only.
*Laws:* LAW-01. *Acceptance:* a note that repeats a document shows it. A document the steward can't open isn't named.

## W37 Work where people are (epic E38, SB-3801 to SB-3806)

The goal is that the brain is in reach from wherever people work, and admins see how it's used.

### S37.1 From other tools

**SB-3801 Use the brain from other AI tools.** `POST /v1/mcp` speaks MCP (JSON-RPC over streamable HTTP) with three read-only tools: `search`, `ask` and `passage`. It takes the person's own `sbt_` token (SB-2306), so every result is checked live as them, at the same rate limit (SB-2305). The tools return cites and passages, never more than the UI shows. It's in-repo code with no new module, and it's off unless sb-cell(5) `mcp: true`.
*Laws:* LAW-01, LAW-12 (a revoked token stops at once), LAW-11. *Acceptance:* an MCP client lists the three tools and gets cited answers. A revoked token is refused. No tool writes.

**SB-3802 Ask about a file.** The ask box takes one attached file, up to 2 MB, of any type the folder connector reads (text, Markdown, Office, sheets). The cell extracts its text for this one ask and keeps no copy: not in the index, not in the trace (which keeps only "1 file"). The answer cites it as "your file". Instruction-like text in it is flagged as in ingest (sb-ingest(7)).
*Laws:* LAW-07 (nothing kept), LAW-09, LAW-14. *Acceptance:* an answer cites the file. The trace and the index hold none of its text. A 3 MB file gives `EINVAL`.

**SB-3803 Every token in one place.** `GET /v1/admin/tokens` lists every live API token in the tenant: owner, name, created, last used day and expiry, never the token. **Revoke** (`DELETE /v1/admin/tokens/{id}`) ends one at once, audited with a reason. The admin console lists them, oldest unused first.
*Laws:* LAW-12, LAW-11. *Acceptance:* the list shows every token and no secret. A revoked token stops working.

### S37.2 Know it's used

**SB-3804 Adoption.** The weekly report adds "N of M seats asked this week (last week N)". By space, it adds people who asked, for spaces with at least 5 of them. It names no one.
*Laws:* LAW-07. *Acceptance:* the counts match the traces. A space with 3 askers shows "fewer than 5".

**SB-3805 Daily brief.** Notification settings gain `digest: daily`. Instead of Monday's digest, the person then gets a short message each working morning at 08:00: what changed in their spaces and their watched questions, from the same parts as the weekly digest. Nothing is sent on a day with nothing to say.
*Laws:* LAW-07, LAW-12. *Acceptance:* a daily person gets a message on a day with changes and none on a quiet day. A weekly person is unchanged.

**SB-3806 Search from the address bar.** The cell serves `/opensearch.xml`, pointing at the Documents tab with the words as `q`. A browser can then add the brain as a search engine. It's a static file. The search itself runs signed in, as always.
*Laws:* none new. *Acceptance:* the file is valid OpenSearch and its URL opens the Documents tab with the words.

## Decisions for the user

Decided by the user on 2026-11-24, with every recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W35, W36, W37. *Other:* W37 first, so MCP and the file ask land first.
2. **Verified answers (SB-3601).** *Recommended:* a pinned answer shows first, above the model's, with every cite checked again as the asker. *Other:* only suggest the pin to stewards, never to askers.
3. **MCP (SB-3801).** *Recommended:* read-only tools with the person's own API token, off until the admin turns it on. *Other:* leave it out for now.
4. **Ask about a file (SB-3802).** *Recommended:* one file of up to 2 MB per ask, read once and never kept. *Other:* leave it out, since files may hold things people shouldn't paste.
5. **Report this answer (SB-3606).** *Recommended:* the asker chooses to send it and is named. *Other:* send it anonymously, without the question's text.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Atlassian: Rovo at Team '26](https://www.atlassian.com/blog/company-news/rovo-team-26)
- [Guru: How content is verified](https://help.getguru.com/docs/verifying-and-unverifying-cards)
- [Glean: verification feature](https://www.glean.com/resources/product-videos/manage-your-companys-knowledge-with-gleans-verification-feature)
- [Slite: knowledge base on autopilot](https://slite.com/nl/blog/knowledge-base-on-autopilot)
- [Notion AI for Work](https://www.notion.com/en-US/product/notion-ai-for-work)
- [Tettra vs Slite](https://tettra.com/article/tettra-vs-slite/)
- [Tana: best second brain apps 2026](https://tana.inc/blog/best-second-brain-apps-2026)

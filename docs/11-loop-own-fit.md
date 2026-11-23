---
title: Keep People in the Loop, Keep Knowledge Owned, Fit the Site
summary: Concept for W20 to W22 (accepted 2026-11-23): twenty small features that tell people when something needs them, keep documents owned and right after people move, and fit the cell into the site's own languages, logs and storage.
---

# Keep people in the loop, keep knowledge owned, fit the site

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-23 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [10-run-protect-close.md](10-run-protect-close.md) · [09-team-knowledge.md](09-team-knowledge.md) · [selfhost runbook](runbooks/selfhost.md) · [PRD waves](03-prd.md#waves) |
| 0.2 | 2026-11-23 | Platform lead | Plan accepted: every recommended decision (2026-11-23) | [Principles](01-principles.md) · [10-run-protect-close.md](10-run-protect-close.md) · [09-team-knowledge.md](09-team-knowledge.md) · [selfhost runbook](runbooks/selfhost.md) · [PRD waves](03-prd.md#waves) |

## Summary

After W19 a company runs its brain on its own machine, protects its data on and off the box, and asks from the web, Slack or Mattermost. Reading the code at `e082478`, what's left is mostly *between* people and the brain:

- **Nobody is told anything.** A steward finds a new question only by opening the steward screen; an asker finds a reply only by opening their questions; "what changed" waits for someone to look. Every loop W16 built is pull-only.
- **Ownership decays.** When someone leaves, their documents keep answering with no owner to ask; the review list doesn't say so. Copies of the same document in two places are both cited. A wrong-citation report fixes nothing for next time: it's counted, not learned from.
- **Hidden means stuck.** "3 sources you can't open" tells the asker to ask the owner, but not who, and gives them no way to ask.
- **The cell doesn't fit the site yet.** The UI is English only, though sovereign sites are mostly in the EU. Audit rows can be exported but not streamed to the company's SIEM. Off-box copies need a mounted NAS, not the S3 bucket many sites already have. Spreadsheets on file shares aren't read. Personal data flags know IBANs and cards, not national ID numbers. Scripts can't ask without a browser session. A box on one CPU can be held up by one person asking twenty questions.

Three waves, twenty tickets. Same five bloat rules as W16 to W19.

## Keeping it small

1. **No new service the cell must run.** Notifications are a job in `sb-jobs`; the SIEM stream and S3 copies are adapters behind existing ports.
2. **No new store.** New records are `cell_docs` kinds (notifications sent, glossary entries, eval cases, access requests, tokens).
3. **No new third-party Go module.** Spreadsheets are `archive/zip` + `encoding/xml`; syslog over TLS is `crypto/tls`; S3 reuses `internal/adapters/backup/s3`.
4. **At most one setting a feature.**
5. **No new page.** Sections and tabs, not screens.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line |
|---|---|---|---|---|---|
| 1 | SB-2101 | W20 | Stewards hear about new questions | Steward | A chat message (or email) to a space's stewards when a question arrives: the space and a link, never the text |
| 2 | SB-2102 | W20 | Askers hear about replies | Asker | A message when a steward replies or publishes a note; opening it shows the reply under the asker's access |
| 3 | SB-2103 | W20 | Weekly "what changed" digest | Everyone | Once a week, titles of changed documents the person can open now, in their spaces; opt out per person |
| 4 | SB-2104 | W20 | Watch a saved question | Asker | A saved question is re-asked weekly; the person is told when its cited sources change, never the answer |
| 5 | SB-2105 | W20 | Share in Mattermost | Asker | The Slack Share button's rule in Mattermost: citation links only, never claim text |
| 6 | SB-2106 | W20 | Handover by name | Manager | Pick the person by name or email from the people you may brief, instead of a user id |
| 7 | SB-2201 | W21 | Orphaned documents | Steward | Documents whose owner is deprovisioned are listed for review, with a count, so someone takes them over |
| 8 | SB-2202 | W21 | Duplicate documents | Steward | The same text in two places is listed once for review, and answers cite the newest copy |
| 9 | SB-2203 | W21 | Space glossary | Steward, asker | Stewards keep terms and acronyms; search expands them, and answers show the ones they use |
| 10 | SB-2204 | W21 | Wrong citations become eval cases | Steward, factory | A steward turns a wrong-citation report into a tenant eval case the gate runs from then on |
| 11 | SB-2205 | W21 | Ask for access | Asker, owner | From a hidden count, send the documents' owners a request; the owner sees which document, the asker never does |
| 12 | SB-2206 | W21 | Spreadsheets and CSV on file shares | Everyone | `.xlsx`, `.ods` and `.csv` text in the folder connector, sheet by sheet, bounded |
| 13 | SB-2207 | W21 | National ID numbers as personal data | DPO | UK NINO, US SSN, Dutch BSN and German tax ids, by checksum, flagged and masked like IBANs |
| 14 | SB-2301 | W22 | The UI in the company's language | Everyone | A string catalog with English, German and French; the manifest picks the default, a person can switch |
| 15 | SB-2302 | W22 | Audit stream to the SIEM | Security | Audit rows (ids, hashes, decisions; no content) forwarded as syslog over TLS as they're written |
| 16 | SB-2303 | W22 | Off-box copies to S3 | IT | `SB_OFFSITE=s3://…`: sealed copies to the company's bucket instead of a mounted NAS |
| 17 | SB-2304 | W22 | Search results while the answer is written | Asker (CPU boxes) | The web UI shows the allowed sources at once, then the cited answer when the local model finishes |
| 18 | SB-2305 | W22 | Fair share on a small box | Operator | `asks_per_minute` per person, so one person can't hold up a one-CPU box; refused politely, counted |
| 19 | SB-2306 | W22 | Read-only API tokens | Integrators | A person makes an expiring token that can ask and search as them, nothing else; revocable and audited |
| 20 | SB-2307 | W22 | Notifications by email | Sites without chat | An SMTP adapter for SB-2101 to SB-2104, for sites that use neither Slack nor Mattermost |

## W20 Keep people in the loop (epic E21, SB-2101 to SB-2106)

The goal: when something needs a person, they hear about it, and nothing they hear tells them more than they could open.

### S20.1 Questions and replies

**SB-2101 Stewards hear about new questions.** A `notify` job in `sb-jobs` runs every five minutes over steward questions created since its last run. For each space it sends one message to each steward who works its queue now (their groups at send time, LAW-21): "N new questions in Support" and a link to the steward screen. Never the question's text: the steward reads it on the screen, where the cell checks they're still a steward. Chat goes through the Slack or Mattermost bot as a direct message (the Slack bot gains `chat:write` and `im:write`); a site with neither uses email (SB-2307). A person can turn it off.
*Laws:* LAW-21 (sent only to stewards now), LAW-07 (no question text, no answer in any message or log). *Acceptance:* a steward removed from the group between question and send gets nothing; a message with question text fails a test.

**SB-2102 Askers hear about replies.** The same job tells an asker "Your question has a reply" with a link to their questions, when a steward replies or publishes a note. The link opens the reply under the asker's access (SB-1704 already drops cites they can't open).
*Laws:* LAW-01, LAW-07. *Acceptance:* the message carries no reply text and no cite.

### S20.2 What changed, for you

**SB-2103 Weekly digest.** Monday 08:00 in the tenant's time zone, each person who asked anything in the last 90 days gets the titles of up to ten documents that changed in their spaces, each one checked live as them (the same check as `GET /v1/changes`), and a count of the rest. Opt out per person; `digest: false` in the manifest turns it off.
*Laws:* LAW-01 (only what they can open now), LAW-13 (a rebuild is never a change). *Acceptance:* a document whose ACL dropped the person between Friday and Monday isn't named.

**SB-2104 Watch a saved question.** A saved question (SB-1703) gains "Watch". Weekly, the cell re-asks it as the person, compares the set of cited documents and their versions with last week's, and if they changed sends "A source of your saved question changed" and a link that asks it again live. No answer is stored, only the cite set's hash.
*Laws:* LAW-01, LAW-07 (no answer kept). *Acceptance:* a changed cite the person can no longer open counts as a change but isn't named.

### S20.3 Chat and handover parity

**SB-2105 Share in Mattermost.** The Mattermost answer gains a Share action through an interactive message button; it posts the question and the citation links only to the channel, exactly the Slack rule (SB-308), and only for the asker, with their scope read again.
*Laws:* LAW-01. *Acceptance:* the Slack Share test, run against the Mattermost app.

**SB-2106 Handover by name.** The Handover tab gains a person search: it lists, by name and email, only the people the asker may brief (themselves, people who report to them, members of spaces they steward), from the directory. Typing a user id still works.
*Laws:* LAW-01 (the list reveals no one the asker couldn't brief). *Acceptance:* a colleague with no relation isn't in the list.

## W21 Keep knowledge owned and right (epic E22, SB-2201 to SB-2207)

The goal: documents keep an owner after people move, the brain stops repeating the same mistake, and an asker who hits a wall can ask to get through it.

### S21.1 Owned

**SB-2201 Orphaned documents.** The stewards' review list (SB-1709) gains documents in the space whose owner at the source is no active directory user: deprovisioned, left, or never known. Each is checked live as the steward and shown with "owner gone"; ones the steward can't open are counted. The handover brief's "hidden" advice links here.
*Laws:* LAW-12 (a deprovisioned owner is never shown as someone to ask), LAW-01. *Acceptance:* deprovision an owner, and their document moves to the list within the next sync.

**SB-2202 Duplicate documents.** Ingest keeps a hash of each document's normalised text. Two or more documents with the same hash in a space are listed once on the review list with their locations (those the steward can open). Answers that would cite two copies cite the newest only.
*Laws:* LAW-13 (the hash is derived, rebuilt with the index), LAW-01 (a copy the asker can't open is never the one cited). *Acceptance:* copy a file, and the answer cites one; delete the newer, and it cites the older.

### S21.2 Right

**SB-2203 Space glossary.** A space's stewards keep up to 200 terms (`term`, `means`), proposed like any space change (SB-1201). Search expands a query's terms with their meanings; an answer that uses a term shows it at the foot. Terms are the stewards' own text, never instructions to the model (they go in the data block, LAW-09).
*Laws:* LAW-09, LAW-19 (a space's glossary is used only in that space). *Acceptance:* "PO" finds documents that say "purchase order"; a glossary entry that reads like an instruction is flagged and not used.

**SB-2204 Wrong citations become eval cases.** On the review list, a wrong-citation report gains "Add to evals": the steward writes what the answer should cite, and the case (question, expected cite, space) joins the tenant's eval set (SB-309). The gate runs it on every release; the question is stored, never an answer.
*Laws:* LAW-06 (a release that fails a tenant case isn't admitted to that tenant). *Acceptance:* a release that regresses the case is held for the tenant.

### S21.3 Through the wall

**SB-2205 Ask for access.** Where an answer says "N sources were not shown because you can't open them", a button sends each of their owners one request: "ann@… asked a question that one of your documents answers. Consider sharing it." The owner's message names the document (they own it); the asker never learns which, or who unless they were already named in `people`. One request per asker, document and week.
*Laws:* LAW-01 (the asker learns nothing new), LAW-07. *Acceptance:* the asker's view of the request is a count; the owner's names the document.

### S21.4 Read more, flag more

**SB-2206 Spreadsheets and CSV.** The folder connector reads `.xlsx` (`xl/sharedStrings.xml`, `xl/worksheets/sheetN.xml`), `.ods` (`content.xml`) and `.csv`, as "Sheet: name" then one line a row, cells tab-separated, up to 5,000 rows and the same 16 MiB bound as SB-2003. Formulas are read as their cached values, never evaluated; macros are never opened.
*Laws:* LAW-09, LAW-13. *Acceptance:* fixtures per format; a fuzz target; a sheet past the bound is skipped whole.

**SB-2207 National ID numbers.** The personal-data scanner (SB-1904) gains UK National Insurance numbers (format and prefix rules), US SSNs (area, group and serial rules), Dutch BSNs (11-check) and German tax ids (ISO 7064 check), flagged `personal_data` and masked but the last four.
*Laws:* LAW-07. *Acceptance:* each format's valid and near-miss fixtures; the masking law test covers them.

## W22 Fit the site (epic E23, SB-2301 to SB-2307)

The goal: the cell speaks the company's language, logs where the company logs, stores where it stores, and stays fair on small hardware.

### S22.1 Language and logs

**SB-2301 The UI in the company's language.** Every string the web UI shows moves into a catalog (`web/src/i18n/`), with English, German and French. The manifest's `language` sets the default; a person can switch, kept in their browser. Cell error messages carry a code the UI translates. Answers stay in whatever language the model writes.
*Laws:* LAW-14 (an uncited claim is marked in every language). *Acceptance:* a test fails on a string outside the catalog; axe in each language.

**SB-2302 Audit stream to the SIEM.** `audit.stream: syslog+tls://siem.example:6514` forwards each audit row as it's written, RFC 5424, the row as the export has it (ids, hashes, decisions, no content). A SIEM that's down never blocks a write: rows queue on disk up to a day, a Sev-2 alert fires after an hour, and the stream resumes from the last row the SIEM took.
*Laws:* LAW-11 (the stream is the chain's rows, verifiable with the export's tooling), LAW-07. *Acceptance:* a fake syslog receiver gets every row in order across a restart.

**SB-2303 Off-box copies to S3.** `SB_OFFSITE=s3://bucket/prefix` with a key pair in the vault: the sealed dumps and backups (SB-1901) go to the company's bucket through the existing S3 adapter. `--doctor` checks the bucket; restore reads from it.
*Laws:* LAW-07 (nothing unsealed leaves the box). *Acceptance:* the restore drill passes from versitygw on another port.

### S22.2 Small boxes

**SB-2304 Sources first, then the answer.** On a CPU box answers take 7 to 32 seconds. The web UI asks `GET /v1/search` and `POST /v1/ask` together and shows the allowed sources (title, owner, age) at once, then replaces them with the cited answer. Search needs no model.
*Laws:* LAW-01 (the sources shown are search's, checked as the asker), LAW-14 (sources are never shown as claims). *Acceptance:* a slow fake model; the sources render first; axe.

**SB-2305 Fair share.** `asks_per_minute` (default 6) per person, counted in the cell. Over it, a polite `ERATE` with the seconds to wait; search isn't limited. The weekly report counts refusals.
*Laws:* none new. *Acceptance:* a burst of ten asks gets six answers and four `ERATE`s.

### S22.3 Integrations

**SB-2306 Read-only API tokens.** A person makes a token on the web UI that can `POST /v1/ask` and `GET /v1/search` as them, nothing else, for up to 90 days. The cell keeps its hash; every use is audited with the token's id; revoking it, signing out everywhere (SB-1906) or deprovisioning (LAW-12) ends it. The person's groups are read at each use, never from the token.
*Laws:* LAW-12, LAW-01, LAW-02. *Acceptance:* a token can't approve, act, administer or read another tenant; a deprovisioned person's token fails at once.

**SB-2307 Notifications by email.** An SMTP adapter (STARTTLS required, credentials in the vault) for the W20 messages, for sites with no chat. Plain text, the same content rule: links and counts, no question, answer or document text.
*Laws:* LAW-07. *Acceptance:* a fake SMTP server receives each W20 message; a message with document text fails a test.

## Decisions for the user

Decided by the user on 2026-11-23, with every recommended option:

1. **All three waves**, all twenty tickets, in the order W20, W21, W22.
2. **How people are told (W20):** direct messages through the Slack or Mattermost bot the site already has, with email (SB-2307) as the fallback for sites without chat. The Slack bot gains `chat:write` and `im:write` on top of `commands`.
3. **UI languages (SB-2301):** English, German and French.
4. **API tokens (SB-2306):** yes, read-only (ask and search), 90 days at most.

## Still open from earlier waves

- **SB-1605** still needs its run on a real Linux VM, and W17's `--doctor`, restore and offline upgrade would be best checked on the same VM.
- **SB-903 and SB-908** need a real GCP project; the human reviews of W0 to W7 are unchanged.
- **The Mattermost command (SB-2005)** has been tested against a fake server only; a run against a real Mattermost would close it out.

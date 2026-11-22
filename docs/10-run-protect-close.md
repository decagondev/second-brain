---
title: Run It, Protect It, Close the Loops
summary: Concept for W17 to W19 (accepted): twenty small features that make a sovereign cell easy to run on one machine or two VMs, protect its data on and off the box, and close the gaps W16 left.
---

# Run it, protect it, close the loops

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.2 | 2026-11-22 | Platform lead | Plan accepted (the user, 2026-11-22), with every recommended decision | [Principles](01-principles.md) · [ADR-0022](adr/ADR-0022-self-hosted-cell.md) · [09-team-knowledge.md](09-team-knowledge.md) · [selfhost runbook](runbooks/selfhost.md) · [PRD waves](03-prd.md#waves) |

## Summary

After W16 a company can run its whole brain on one machine it owns, with a local model, and a team can hand knowledge to each other through it. Reading the [selfhost runbook](runbooks/selfhost.md) and the code at `b6ed548`, what's left is mostly the *operator's* day, not the asker's:

- **Running it is still a set of manual steps.** Restore is five commands. An air-gapped upgrade is `docker load` of images *before* anything checks their signature, then a reinstall. There's no rollback on the box, no outbound proxy setting, no way to use the company's existing Postgres, and the health panel doesn't know the disk is full or the certificate expires on Friday.
- **The off-box copy isn't protected.** `dbdump.sh` writes a plaintext `pg_dump` of the whole database to `SB_OFFSITE`, a NAS that more people can read than the cell. The keys that make a restore possible live only in `build/selfhost`, and the runbook says "back up with the machine's own encrypted backup", which many sites don't have.
- **Data-protection duties stop halfway.** A subject can be erased but not given a copy of their data. Retention deletes on schedule with no preview. A person can't sign out everywhere. Ingest flags secrets but not personal data. Retiring a box has no proof the data is gone.
- **W16 left two known gaps.** Notes are refused in cells with information barriers, and the handover brief has no UI. The folder connector reads `.txt`, `.md` and `.pdf` only, and most file shares are mostly Word.

This plan adds three waves, twenty tickets, each a script flag, a few endpoints or a console section on records the cell already keeps. The bloat rules are the same as W16's.

## Keeping it small

Each ticket has to pass all five, or it doesn't go in.

1. **No new service the cell must run.** The Mattermost entry point (SB-2005) is a handler in `sb-api`, like Slack's.
2. **No new store.** New records are `cell_docs` kinds or files under `build/selfhost`.
3. **No new third-party Go module.** Office text is `archive/zip` + `encoding/xml`; sealing is `crypto/ecdh`, `crypto/hkdf` and AES-GCM, all in Go 1.24's standard library; Shamir shares are ~100 lines over GF(256).
4. **At most one setting a feature**, in the manifest or `install.env`.
5. **No new page.** Console and UI gain sections, not screens.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line |
|---|---|---|---|---|---|
| 1 | SB-1801 | W17 | `install.sh --doctor` | IT | One command that checks the running box and prints each problem with the line that fixes it |
| 2 | SB-1802 | W17 | Host health in the console | Operator | Disk, certificate days left, last dump and off-box copy age, beside jobs and connectors |
| 3 | SB-1803 | W17 | One-command restore and a monthly restore drill | IT | `restore.sh` does the five steps; a monthly job proves the newest dump restores |
| 4 | SB-1804 | W17 | Offline upgrade, verified before it's loaded | IT (air-gapped) | `update.sh --bundle FILE` checks signature and gate first, then loads and applies |
| 5 | SB-1805 | W17 | Roll back the box in one command | IT | `update.sh --rollback` starts the previous admitted release's digests |
| 6 | SB-1806 | W17 | Outbound proxy | IT | `SB_HTTPS_PROXY` and `SB_NO_PROXY` for release pulls, connectors and a hosted model |
| 7 | SB-1807 | W17 | Use the company's Postgres: two VMs | IT | `SB_DATABASE` points at an existing Postgres server; `--check` proves TLS and grants |
| 8 | SB-1808 | W17 | A support bundle with no content | IT, factory | Versions, health, redacted logs and config in one tar, canary-scanned before it's written |
| 9 | SB-1901 | W18 | Off-box copies sealed | DPO, IT | Dumps and backups are encrypted to a recovery key before they leave the box |
| 10 | SB-1902 | W18 | Recovery kit in shares | Custodians | The box's keys split into 3-of-5 printable shares; a restore needs any three |
| 11 | SB-1903 | W18 | Give a person their data | DPO | `POST /v1/admin/subjects/export`, the twin of erase: everything the cell keeps about a person |
| 12 | SB-1904 | W18 | Personal data flags at ingest | Steward | IBANs and card numbers flagged and masked in snippets, counted per space |
| 13 | SB-1905 | W18 | Retention preview | Operator, DPO | `sb tenant retention --plan`: what the nightly job deletes in the next 7 days, by kind and space |
| 14 | SB-1906 | W18 | Sign out everywhere | Everyone | A person ends all their sessions; an operator can for anyone, audited |
| 15 | SB-1907 | W18 | Retire a box with proof | DPO, auditor | `install.sh --retire` exports, shreds the keys and leaves a signed statement |
| 16 | SB-2001 | W19 | Notes in cells with barriers | Steward | A note is placed by its space before it exists, so barrier cells can publish notes |
| 17 | SB-2002 | W19 | Handover brief in the UI | Manager, successor | The brief W16 built, on the steward screen, with the hidden-count prompt |
| 18 | SB-2003 | W19 | Word, OpenDocument and HTML on file shares | Everyone | `.docx`, `.odt`, `.pptx` and `.html` text in the folder connector |
| 19 | SB-2004 | W19 | Notes stay narrow | Steward | A note whose readers are wider than its space after a membership change is flagged for review |
| 20 | SB-2005 | W19 | Mattermost entry point | Sovereign sites | Ask from a self-hosted chat, with the Slack entry point's rules |

## W17 Easy to run (epic E18, SB-1801 to SB-1808)

The goal: a company's IT team runs, upgrades, restores and gets help for a cell without reading the source, on one machine or two VMs.

### S17.1 Know it's healthy

**SB-1801 `install.sh --doctor`.** `--check` today checks the machine *before* an install. `--doctor` checks the running box: every service up, both timers enabled and their last run green, free disk on the data and dump directories, TLS certificate expiry, `SB_OFFSITE` mounted, writable by uid 65532 and *not* on the root disk, clock skew against the IdP, `/readyz`, and whether `build/selfhost` sits on an encrypted disk (LUKS via `lsblk`, a warning, not a refusal). Each line is `pass`, `warn` or `fail` with the command that fixes it. Exit code is the worst line.
*Laws:* LAW-07 (it reads no secret, prints no path's contents). *Acceptance:* each check has a test that makes it fail; the runbook's Check steps call it.

**SB-1802 Host health in the console.** The health panel (SB-1607) gains a "Box" section from a small file the `jobs` service writes hourly: disk used per volume, certificate days left, age of the newest dump and of the newest off-box copy. Warn at 80% disk, 21 days, 26 hours; Sev-2 alerts through the existing alert port at 90%, 7 days, 50 hours. GCP cells show "managed" for the section.
*Laws:* LAW-03 (a file that can't be read shows Unknown, never green). *Acceptance:* a test per threshold; the panel test with axe.

### S17.2 Upgrade and recover

**SB-1803 One-command restore and a monthly restore drill.** `tools/selfhost/restore.sh [--dump FILE]` does the runbook's steps: check `build/selfhost` is the one the dump was made with (by the anchor key's id), stop the writers, `pg_restore`, start, `sb audit verify`, `sb tenant verify`. A monthly `restore-drill` job restores the newest dump into a scratch database inside the postgres container, counts rows per table against the live one, drops it, and records the result (SB-613's game day, on a schedule).
*Laws:* LAW-11 (verify the chain after restore), LAW-13 (the index rebuilds, never restored). *Acceptance:* `tests/gameday` restores with the script; the drill fails on a truncated dump.

**SB-1804 Offline upgrade, verified before it's loaded.** Today an air-gapped site runs `docker load` on images nobody has checked, then reinstalls. `update.sh --bundle FILE` unpacks to a staging directory, checks the release signature against `SB_RELEASE_TRUSTED` and the gate record, checks every image tar's digest against the release, and only then loads and applies, exactly as the online path. A bundle that fails changes nothing.
*Laws:* LAW-06. *Acceptance:* a tampered image tar and an unsigned release are refused before `docker load` runs.

**SB-1805 Roll back the box in one command.** The updater already records the digests it admitted. `update.sh --rollback` starts the previous admitted release's digests (never one that wasn't admitted), records it, and holds the timer until `--resume`, so the next hourly run doesn't upgrade again. The console shows "held at REL, rolled back from REL".
*Laws:* LAW-06, LAW-08 (at most one behind stable). *Acceptance:* rollback twice in a row refuses to go two back.

### S17.3 Fit the company's network

**SB-1806 Outbound proxy.** `SB_HTTPS_PROXY` and `SB_NO_PROXY` in `install.env`, passed to `api`, `ingest`, `jobs` and `updater` only (never `postgres`, `model` or the `data` network, which stay without a route out). The existing CA setting already trusts a proxy's certificate. `--check` and `--doctor` test the release store through the proxy.
*Laws:* LAW-07 (proxy credentials, if any, are a file, redacted in logs). *Acceptance:* the smoke runs through a local proxy; the model profile still has no route.

**SB-1807 Use the company's Postgres: two VMs.** `SB_DATABASE=external` with a connection file (host, database, user, `sslmode=verify-full`, CA). The compose layout then doesn't start `postgres`; `dbdump.sh` dumps over the network. `--check` proves TLS, that the role can create its schema and can't read other databases. The write-once store and keys stay on the cell's VM. This is the supported "more than one machine" layout without Kubernetes (ADR-0011).
*Laws:* LAW-02 (one database per tenant; `--check` refuses a database with another tenant's schema). *Acceptance:* `make selfhost` with a second Postgres container standing in for the other VM.

**SB-1808 A support bundle with no content.** `install.sh --support` writes one tar: versions and digests, `--doctor` output, the health report, the last 1,000 log lines per service through the redactor, and `install.env` with every `*_FILE` path's contents left out. Before writing, the canary scanner (SB-411) runs over it and a hit aborts. No documents, questions, answers or audit payloads, ever.
*Laws:* LAW-07. *Acceptance:* a planted canary in a log aborts the bundle.

**Exit gate (W17):** a fresh VM installs, upgrades offline from a bundle, rolls back, restores from yesterday's dump with one command each; `--doctor` is green; a two-VM layout passes `make selfhost-smoke`; every law test green; no new store, service or Go module.

## W18 Protect the data (epic E19, SB-1901 to SB-1907)

The goal: nothing readable leaves the box, the keys can't be lost with the box, and the company can meet a data-protection request without a developer.

### S18.1 Off the box

**SB-1901 Off-box copies sealed.** A recovery key pair (X25519) is made at install; the private half goes only into the recovery kit (SB-1902). Every file written to `SB_OFFSITE`, the database dump and the audit and approval backups, is sealed first: a fresh AES-256-GCM key per file, wrapped to the recovery public key (ECDH + HKDF), with the file's SHA-256 in the header. The local copies on the box are unchanged, so a normal restore needs no kit. The NAS sees only ciphertext. A new `sb backup open` decrypts for SB-1803.
*Laws:* LAW-07, LAW-11 (sealed backups still verify the chain once opened). *Acceptance:* no plaintext byte pattern from a test document appears in `SB_OFFSITE`; a flipped bit fails to open.

**SB-1902 Recovery kit in shares.** `install.sh --recovery-kit` splits the recovery private key and the root key's wrap into five Shamir shares, any three of which rebuild them, each printed as a page with a QR-free text block and a checksum, for five named custodians. `restore.sh --from-kit` takes three. Who holds a share is recorded (names, never the shares) next to the custody record, so LAW-18 has more than one custodian on a box.
*Laws:* LAW-18. *Acceptance:* any three of five rebuild the key; any two fail; a mistyped share fails its checksum.

### S18.2 Duties to people

**SB-1903 Give a person their data.** `POST /v1/admin/subjects/export` mirrors erase: saved and shared questions, steward questions and replies, and feedback: every record erase removes, in one JSON file. An operator action like erase, policy-checked and audited before it runs, with only the export's hash in the audit row.
*Laws:* LAW-04, LAW-07. *Acceptance:* export then erase then export gives an empty file.

**SB-1904 Personal data flags at ingest.** The ingest sanitiser already flags secrets. It gains `pii` flags for IBANs (checksum) and payment cards (Luhn). National id formats differ by country and are left for a later field. A flagged chunk's snippet in search and its quote in an answer are masked (`•••• 4242`); stewards see per space how many documents carry personal data, so a People share that slipped into an all-staff space is found. Nothing is deleted or blocked.
*Laws:* LAW-07, LAW-13 (flags are derived). *Acceptance:* a fuzz target over the detectors; the space-leak eval gains a PII case.

**SB-1905 Retention preview.** `sb tenant retention --plan [--days 7]` and a console line: what the nightly retention job will delete in the window, counted by kind and space, with legal holds (SB-1207) shown as "kept by hold". It runs the job's own selection with no delete.
*Laws:* LAW-16. *Acceptance:* the plan's counts equal what the job then deletes in the test.

**SB-1906 Sign out everywhere.** A person sees their active sessions (device, last used) and ends them all; an operator can end anyone's, audited. Uses the sessions store deprovisioning already revokes.
*Laws:* LAW-04, LAW-12. *Acceptance:* a session ended is refused at the next request.

### S18.3 End of life

**SB-1907 Retire a box with proof.** `install.sh --retire` runs the detach export (SB-708) if asked, stops the cell, removes the database, index and share-cache volumes, overwrites and deletes the keys in `build/selfhost` (so every sealed off-box copy is unreadable without the kit), and writes a signed statement: what was destroyed, when, by whom, with the last anchor's hash. The write-once anchors keep their seven-year lock; the statement says so.
*Laws:* LAW-11, LAW-17, LAW-18 (the statement records the custodian's release). *Acceptance:* after retire, no key file is readable and a sealed backup can't be opened without the kit.

**Exit gate (W18):** nothing in `SB_OFFSITE` is readable without the kit; three of five custodians restore a lost box; an export and erase of one person round-trip; retention plans match deletions; every law test green.

## W19 Close the loops (epic E20, SB-2001 to SB-2005)

The goal: W16's features work everywhere, and the brain reads the documents companies actually keep.

**SB-2001 Notes in cells with barriers.** The bus places a call's target before it runs, and a note that doesn't exist can't be placed, so barrier cells refuse notes. A note's target is placed by its space and folder (`notes/SPACE/`), which is known before the write; the barrier check then runs on the steward and the space as for any document in it.
*Laws:* LAW-22, LAW-04. *Acceptance:* the W16 barrier refusal test becomes an allow-for-member, deny-across-barrier test.

**SB-2002 Handover brief in the UI.** A "Handover" section on the steward screen and for managers: pick a person you may ask for (the same policy), see the cited brief and "N more documents you can't open; ask for access to be moved before they leave".
*Laws:* LAW-01, LAW-12. *Acceptance:* a UI test with axe; a person you may not ask for isn't offered.

**SB-2003 Word, OpenDocument and HTML on file shares.** The folder connector's `Document` accepts `.docx`, `.pptx`, `.odt` and `.html`. Text comes from the XML parts with `archive/zip` and `encoding/xml` (headings kept as Markdown headings for chunking), with a zip-bomb guard (entry count, total uncompressed size under `MaxSize`). HTML drops scripts and styles. Macros and embedded objects are never read.
*Laws:* LAW-09 (text is data), LAW-13. *Acceptance:* fixtures per format; a fuzz target; a zip bomb is skipped and counted.

**SB-2004 Notes stay narrow.** A note's readers are a snapshot of the space's members at publish. When the space narrows (a member leaves, a group is removed), the next sync lists notes whose ACL file includes people no longer in the space, on the steward's "Documents to review" (SB-1709) with "readers wider than the space"; the steward re-publishes narrowed with one click (a new note version, the same approval as SB-1705).
*Laws:* LAW-19, LAW-21. *Acceptance:* removing a member flags the note within one sync.

**SB-2005 Mattermost entry point.** A slash command and outgoing webhook handler in `sb-api`, reusing the Slack entry point's code path: the Mattermost user is matched by email to the directory, answers are posted ephemeral to the asker, Share posts the question (never the answer), and the bot token is a vault reference. Only for self-hosted Mattermost the company runs; nothing goes to a vendor.
*Laws:* LAW-01, LAW-07, LAW-12. *Acceptance:* a fake Mattermost in the integration test; the Slack tests run against the shared path.

**Exit gate (W19):** a barrier cell publishes a note; a manager reads a handover brief in the UI; a share of Word files answers with cites; a narrowed space flags its notes; Mattermost asks answer under the asker's access; every law test green.

## Considered and left out

| Idea | Why not now |
|---|---|
| Kubernetes or a Helm chart | ADR-0011; two VMs with an external Postgres (SB-1807) covers "more than one machine" |
| High availability on the box (two cells, failover) | A second running cell is a new service and a replication store; restore in minutes (SB-1803) is the honest target |
| Rootless Podman as a supported runtime | Likely works with `compose.sh`, but supporting it means a second CI matrix; try it after W17 |
| OCR of scanned PDFs | Needs Tesseract or a model; a new dependency for a minority of files |
| A glossary that expands queries | Still changes retrieval for everyone from one document; no new evidence since W16 |
| Matrix entry point | Mattermost first; Matrix's end-to-end encryption makes a bot that reads rooms a separate decision |
| Automatic deletion of documents flagged with personal data | The source is the authority (ADR-0002); flag and tell, never delete someone else's file |
| Hardware keys (HSM/TPM) for the box's root key | Worth it, but per-vendor; the share kit (SB-1902) is the vendor-free first step |

## Decisions for the user

All four decided by the user on 2026-11-22, each with the recommended option.

1. **Accept all three waves**, a subset, or a different order. **Decided:** all twenty, in the order W17, W18, W19.
2. **Recovery kit shape (SB-1902).** **Decided:** 3-of-5 Shamir shares on paper. The alternative was one passphrase-sealed file, simpler and a single point of loss.
3. **Off-box sealing default (SB-1901).** **Decided:** on for every new install; existing boxes switch at their next upgrade with a warning. The alternative was opt-in.
4. **Second chat entry point (SB-2005).** **Decided:** Mattermost. The alternatives were Matrix, or dropping it.

## Still open from earlier waves

- **SB-1605** needs its run on a real Linux VM. W17's `--doctor`, restore and offline-upgrade tickets would be best tested on that same VM, so doing both together saves a trip.
- Human reviews in W0 to W7 and the real-GCP tickets (SB-903, SB-908) are unchanged by this plan.

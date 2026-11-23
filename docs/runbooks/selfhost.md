---
title: Self-hosted install
summary: Install, people, upgrades, backup and restore for a cell on one machine the company owns (SB-1605).
---

# Self-hosted install

A self-hosted cell is a whole cell on one Linux VM or on-prem server that the company owns ([sb-selfhost(7)](../../manual/sb-selfhost.7.md)). This runbook is for the company's IT team. Every step says what to check before moving on.

## Before you start

- A Linux VM or server (Debian 12 or Ubuntu 24.04 tested; 4 vCPU, 16 GB RAM, 200 GB disk for a company of a few hundred), with systemd.
- Docker Engine and the compose plugin, or let the install add them from the distribution with `--install-docker`.
- A DNS name for the cell, such as `brain.acme.example`, and either a certificate and key for it (`SB_TLS=files`) or ports 80 and 443 reachable from the internet so Caddy can get one (`SB_TLS=acme`).
- Sign-in: the company's OIDC IdP, set up with a client whose redirect is `https://HOST/callback`, a `tenant` claim equal to the tenant, `sub` equal to the directory id, and SCIM provisioning to `https://HOST/scim/v2` ([sb-identity(7)](../../manual/sb-identity.7.md)). Or the bundled IdP, [sb-idp(1)](../../manual/sb-idp.1.md), for a site with none.
- A directory on **another machine** for the off-box backup copy, mounted on this one (NFS or SMB) and writable by uid 65532.
- The tenant's manifest ([sb-tenant(5)](../../manual/sb-tenant.5.md)) and, from the factory, the release store's address, its pull token and the release identity to trust. An air-gapped site gets an install bundle and a directory of releases instead.
- Who may log in to the machine. Shell and Docker access is the whole machine, so keep it to the company's own administrators; the factory gets none.

## Install

1. Copy the install bundle (`make selfhost-bundle` makes it) or a checkout of the repository to `/opt/second-brain`, and unpack it there.
2. Copy `deploy/selfhost/install.env.example` to `/etc/second-brain/install.env` and fill it in. Put the files it names (manifest, certificate, key, OIDC client secret, pull token) under `/etc/second-brain`, owned by root, mode 0600.
3. Check the machine and the file without changing anything:

   ```text
   sudo /opt/second-brain/tools/selfhost/install.sh --check /etc/second-brain/install.env
   ```

   Fix anything it reports. A warning that `SB_OFFSITE` is on the root disk means the off-box copy isn't off the box: mount the other machine's directory first.

   It also prints the outbound proxy and the database it will use. Behind a proxy, set `SB_HTTPS_PROXY=http://HOST:PORT`, or, when the proxy needs a user and password, put the whole URL in a root-only file and set `SB_HTTPS_PROXY_FILE` to it; a URL with credentials in `install.env` itself is refused. Add the company's own hosts the cell talks to (an internal IdP, the share's server) to `SB_NO_PROXY`. To use the company's Postgres on another machine, set `SB_DATABASE=external` and put its connection string, with `sslmode=verify-full`, in a root-only file named by `SB_DATABASE_DSN_FILE` (and the server's CA in `SB_DATABASE_CA` if it isn't a public one); the database must be Postgres 17 with the `vector` extension available to the cell's user.
4. Install:

   ```text
   sudo /opt/second-brain/tools/selfhost/install.sh /etc/second-brain/install.env
   ```

   It makes every key and secret on this machine, brings the layout up, makes and proves the write-once bucket, applies the first release, and enables `sb-selfhost.service`, `sb-selfhost-update.timer` and `sb-selfhost-dbdump.timer`.
5. Check: `sudo /opt/second-brain/tools/selfhost/install.sh --doctor` prints `doctor: pass` (or only warnings you accept). It checks every service, both timers, disk, the certificate, the off-box directory, the backups and `/readyz`, and gives the command that fixes each problem. The first dump and off-box copy are `unknown` until the first night.

## People

- **Company IdP:** people arrive over SCIM from the IdP. Assign the app to the pilot group first ([sb-onboarding(7)](../../manual/sb-onboarding.7.md)).
- **Bundled IdP:** add each person, with a password of at least 12 characters given on stdin:

  ```text
  printf '%s' "$PASSWORD" | tools/selfhost/compose.sh run --rm -T idp user add --data /var/lib/sb-idp \
      --email ann@acme.example --scim http://api:8080/scim/v2 --scim-token-file /run/sb/scim
  ```

  `user disable --email …` (same options) removes a person: the IdP refuses them at once and the cell ends their sessions within 5 minutes (LAW-12). `user list --data /var/lib/sb-idp` lists who exists.
- Check: the person signs in at `https://HOST/login` and an ask answers with citations.

## Mattermost

To ask from the site's own Mattermost (sb-ask(3), The chat app (Mattermost)):

1. In Mattermost, make a bot account whose email address it may read, and a personal access token for it. Under *Integrations*, add a **custom slash command**, e.g. `/brain`, with request URL `https://SB_HOST/chat/mattermost`, method POST. Note its token.
2. On the box: `printf '%s' '{"command_token":"…","bot_token":"…"}' > build/selfhost/vault/api/vault/tenants/TENANT/mattermost && chmod 600 …`, and set `"mattermost": {"base": "https://chat.example", "team_id": "…"}` in the cell's configuration; restart `api`.
3. Type `/brain What is our refund approval limit?` as a provisioned person. Someone the directory doesn't know, or who has been deprovisioned, is told their account can't be matched.

## Check the laws

From any machine that reaches the cell, with the control plane's token from `build/selfhost/vault/api/vault/tenants/TENANT/control-plane`:

```text
SB_TOKEN=… sb tenant verify /etc/second-brain/tenant.yaml --cell https://HOST
```

Every line should say `pass`. Run it after the install, after each upgrade you watch, and monthly.

## Upgrades

The `sb-selfhost-update.timer` runs `tools/selfhost/update.sh` hourly. The updater admits the channel's release (signature, gate, the four checks of [sb-release(7)](../../manual/sb-release.7.md)) and records the digests; the host step starts exactly those and records what runs. A release the cell doesn't admit changes nothing, fails the unit and raises a Sev-1 `release_unsigned` ([LAW-06](LAW-06.md)).

- Check after an upgrade: `systemctl status sb-selfhost-update.service` succeeded; `/v1/verifier/attestation` names the new release.
- To hold upgrades, `systemctl stop sb-selfhost-update.timer`; `start` resumes them ([pausing release pulls](pause-pulls.md)).
- Air-gapped: bring the factory's bundle (made with `SB_BUNDLE_RELEASES`) on removable media and run `sudo /opt/second-brain/tools/selfhost/update.sh --bundle FILE`. The installed `sb` checks the release's signature and every image against it **before** anything is loaded; a bundle that fails changes nothing. Never `docker load` a bundle's images by hand.
- Something wrong after an upgrade: `sudo /opt/second-brain/tools/selfhost/update.sh --rollback` starts the release that ran before and holds upgrades. Fix or report, then `update.sh --resume`. Only one step back is kept.

## Backup

Three copies, two kinds of media, one off the box ([sb-audit(7)](../../manual/sb-audit.7.md)):

- The nightly `backup` job writes the audit log and approvals to `state/backups`, the write-once store, and `SB_OFFSITE`. It refuses to run with fewer than three copies.
- `sb-selfhost-dbdump.timer` dumps the database nightly to `build/selfhost/db-dumps` and, sealed to the recovery key, to `SB_OFFSITE/db`, fourteen nights each.
- Anchors go hourly to the write-once store, locked for seven years.
- Everything in `SB_OFFSITE` is sealed to the recovery key: whoever reads the NAS reads nothing.

**The recovery kit, once, right after the install.** The keys that open the off-box copies and run the cell live in `build/selfhost`. Take them off the box so losing it loses nothing:

1. Choose five custodians (for example two IT administrators, the DPO, finance, and the office safe). Any three of them together can rebuild the keys; two can't.
2. `sudo /opt/second-brain/tools/selfhost/install.sh --recovery-kit --custodian "1=NAME" … --custodian "5=NAME"`. It seals an archive of the keys, copies it to `SB_OFFSITE/kit/`, deletes the recovery key's private half from the box, and writes five pages to `build/selfhost/kit/`.
3. Print each page on a printer that keeps no copy, put it in an envelope with the custodian's name, and hand it over. Never photograph, scan or email a page.
4. Delete the pages: `sudo rm /opt/second-brain/build/selfhost/kit/kit_*-share-*.txt`. The doctor passes "recovery kit" once the key and the pages are gone.
5. Once a year, check each custodian still has their page. If one is lost, make a new kit: `install.sh --recovery-kit --new-key`, and give out the new pages.

Check weekly: `install.sh --doctor` passes (the console's Box section shows the same checks, hourly, and a failing one raises a `box_check` alert once a day); today's files in `SB_OFFSITE/TENANT/DATE/` and `SB_OFFSITE/db/`; `sb audit verify --tenant TENANT` is green.

## Restore

On the same machine (a bad change, a broken database): `sudo /opt/second-brain/tools/selfhost/restore.sh` restores the newest dump, or `--dump FILE`, and checks the audit chain afterwards. The monthly drill (`restore.sh --drill`, run by `sb-selfhost-drill.timer`) proves the newest dump restores; the doctor shows when it last passed.

To a new machine, after losing the old one:

1. Copy the install bundle to `/opt/second-brain` and mount the off-box directory at the same `SB_OFFSITE`. Gather three custodians with their pages.
2. Rebuild the keys: `sudo tools/selfhost/restore.sh --from-kit SB_OFFSITE/kit/kit_ID.sealed page-A.txt page-B.txt page-C.txt` (the pages typed back in, or scanned and then deleted). A mistyped page fails its checksum; fix the character and run it again. Without the kit, put the old `build/selfhost` back under `/opt/second-brain` instead. With neither, the cell gets new keys, and the audit chain's earlier anchors verify only against the old anchor key.
3. Install as above with the same `install.env`. It keeps the rebuilt keys.
4. Restore the newest dump from off the box: `sudo tools/selfhost/restore.sh --dump SB_OFFSITE/db/sb-DATE.dump.sealed`. It opens it with the rebuilt recovery key, refuses a dump made under other keys, starts the services again, and checks the audit chain.
5. With the bundled IdP, add people again (`sb-idp user add`); their accounts in the cell come back with the database.
6. Check `sb tenant verify`, then make a new recovery kit (`install.sh --recovery-kit`) so the recovery key leaves this box too.
7. The index rebuilds from the sources on its own if the dump is older than the last sync ([LAW-13](../../manual/sb-laws.7.md#law-13)); nothing else is lost but what changed after the dump.

## Retire a box

When the company stops using the box (SB-1907):

1. Make sure the recovery kit's custodians still hold their shares, if the off-box copies must stay readable; otherwise the copies become noise once the shares are destroyed.
2. See what goes: `sudo /opt/second-brain/tools/selfhost/install.sh --retire`. It lists the units, the volumes and the files, and stops.
3. Destroy it: `sudo /opt/second-brain/tools/selfhost/install.sh --retire --yes-destroy`. If the audit export can't be taken or doesn't verify, it stops before anything is destroyed; fix the cell (the doctor helps) and run it again.
4. Print `build/retired/TENANT-TIME/statement.json` together with the key it printed (`ed25519:…`); the key is kept nowhere else. File both with the company's records. Anyone can check them: `sb selfhost verify-retirement statement.json --key ed25519:…`.
5. With `SB_DATABASE=external`, drop the company's database: the statement lists it under `kept`.
6. Wipe or return the machine as the company's policy says; the box itself holds no key any more.

## LAW-17 on one machine

There is no IAM to enumerate, so the cell keeps its own list of what the factory can reach (`state/iam/TENANT.json`): at most the release store's read-only pull token. The monthly `access` job checks it and leaves a signed statement in `state/access/statement-DATE.json`, whose hash is in the audit log and so in the next anchor. An unlisted path is a Sev-1 `access_path` ([LAW-17](LAW-17.md)). To show an auditor: the statement, `sb audit verify`, and the anchor covering the statement's audit row.

## When something breaks

1. Run `sudo /opt/second-brain/tools/selfhost/install.sh --doctor` and follow the fixes it prints.
2. If you need help, run `sudo /opt/second-brain/tools/selfhost/install.sh --support`. It writes `build/selfhost/support-TIME.tar.gz`: versions, digests, the layout's settings, the doctor's verdicts, `ps` and the last 1,000 log lines per service. It has no documents, questions or answers in it, and it is scanned for every secret the box stores before it is written; if the scan finds one it writes nothing and names the file and line, so tell the maintainers what it named rather than sending the logs some other way.
3. Send that one file. Nothing else from the box needs to leave it.

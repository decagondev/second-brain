---
title: Microsoft 365 setup
summary: What sb selfhost check-microsoft checks, and how to fix each failure (SB-6003).
---

# Microsoft 365 setup

`sb selfhost check-microsoft --config FILE` checks the cell's Microsoft 365 setup before the first sync and after any change in Entra ([sb(1)](../../manual/sb.1.md), [sb-selfhost(7)](../../manual/sb-selfhost.7.md)). It reads only, prints no secret, and names a section below for each failure. The cell reaches Microsoft 365 read-only through Graph ([ADR-0023](../adr/ADR-0023-microsoft-365.md)).

## app-registration

The app registration's credentials at `vault://tenants/TENANT/msgraph` got no token, its permissions couldn't be read, or it holds none.

1. In the Entra admin center, open **App registrations** and check the cell's app exists and its client secret hasn't expired.
2. Put `{"tenant_id": "…", "client_id": "…", "client_secret": "…"}` in the vault under `msgraph`, owner-only.
3. Under **API permissions**, add the read-only application permissions the cell needs (below) and grant admin consent.

## write-permission

The app registration holds an application permission that doesn't only read, such as `Files.ReadWrite.All`, `Sites.Manage.All` or `Mail.Send`. The cell never writes to Microsoft 365, and a permission it doesn't need is a risk if its secret leaks.

1. Under **API permissions**, remove every permission the check names.
2. Keep only these: `Sites.Read.All` or `Sites.Selected`, `Files.Read.All`, `Group.Read.All` or `GroupMember.Read.All`, `User.Read.All`, `Calendars.Read`, `OnlineMeetingTranscript.Read.All`, `Mail.Read` (with the policy below) and `InformationProtectionPolicy.Read.All`.
3. Run the check again.

## shared-mailbox

A shared mailbox a space names (sb-space(5) `mailbox`) can't be read.

1. Check the address in the manifest is the shared mailbox's primary address.
2. Check the mailbox is in the application access policy below.

## mail-access-policy

With `Mail.Read`, an app can read every mailbox unless an Exchange application access policy limits it. The cell must read only the shared mailboxes the manifest names, never a person's own (SB-5907). The check reads a `--probe` mailbox, such as your own: it must be refused.

1. Make a mail-enabled security group holding only the named shared mailboxes.
2. In Exchange Online PowerShell: `New-ApplicationAccessPolicy -AppId CLIENT_ID -PolicyScopeGroupId GROUP -AccessRight RestrictAccess`.
3. Wait up to an hour for it to apply, and run the check again with `--probe`.

## sites-and-drives

A drive in sb-cell(5) `microsoft.drives` doesn't answer, or none is named.

1. Find each library's drive id with Graph Explorer (`GET /sites/{site}/drives`).
2. With `Sites.Selected`, grant the app read on each site.
3. Put the ids in `microsoft.drives` and restart the cell.

## label-mapping

The tenant defines sensitivity labels that sb-cell(5) `labels.microsoft` doesn't map. Until an admin maps a label, every document carrying it counts as `restricted` (SB-5902). Or the labels couldn't be read at all.

1. Grant `InformationProtectionPolicy.Read.All` if the labels couldn't be read.
2. Map each label id the check names to `public`, `internal`, `confidential` or `restricted` in `labels.microsoft`.
3. Restart the cell; the next sync relabels.

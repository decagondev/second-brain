---
title: "sb-identity(7)"
man: "7"
summary: Sign-in, SCIM, sessions and deprovisioning.
---

# sb-identity(7)

## NAME

sb-identity — how users get in, and how fast they get out

## SYNOPSIS

```text
sign-in:  OIDC (your IdP)  →  session (re-validated every 60 s)
users:    SCIM only        →  deactivate revokes in ≤ 5 min
```

## DESCRIPTION

### Sign-in

Users sign in with the company's own OIDC IdP, named by `idp` in the manifest. The cell checks the token's issuer, audience and `tenant` claim. MFA and conditional access are the IdP's job, and we inherit them.

Sign-in is the OIDC authorization-code flow with PKCE (`S256`), a `state` and a `nonce`. `GET /login` starts it and redirects to the IdP; the IdP returns to `GET /callback` at the cell's `base_url` (sb-cell(5)), which checks the token, sets the session cookie (`HttpOnly`, `Secure`, `SameSite=Lax`) and redirects to `/`. A `state` is single use and lives 10 minutes. The cell's OIDC client secret is read from `vault://tenants/TENANT/oidc` at call time (LAW-07). The ID token is accepted only if **every** check below passes. Any failure, including an IdP or key set that can't be reached, is `EAUTH`; there is no fallback.

| Check | Rule |
|---|---|
| Issuer | `iss` equals the manifest's `idp`, written as `https://` (`oidc://login.acme.example` means `https://login.acme.example`) |
| Signature | `RS256` with a key from the issuer's published key set, matched by `kid`; `none` and every other algorithm are refused. An unknown `kid` refreshes the key set at most once a minute |
| Audience | `aud` contains the cell's client id |
| Time | `exp` in the future and `iat` not in the future, with 60 seconds of leeway |
| Nonce | `nonce` equals the one sent with this sign-in |
| Tenant | `tenant` equals the cell's tenant. A token from another tenant's IdP, or with another tenant's claim, never signs in here (LAW-02) |
| Subject | `sub` is present; it becomes the user id |

A self-hosted cell at a site with no IdP it can reach, such as an air-gapped one, signs in with the bundled IdP, sb-idp(1), instead: the same flow and the same token rules, with passwords kept as PBKDF2 hashes and users provisioned in the cell over SCIM as they are added (sb-selfhost(7), Install).

### Users and groups

Users and groups exist **only** if SCIM created them. Nobody can add a user by hand, so access always follows the IdP. Groups drive policy (`group:NAME` in the manifest and Cedar) and are read from the directory at decision time, never from a cached token, so a membership change applies to the very next decision.

Each cell serves SCIM 2.0 at `/scim/v2` for its own tenant only. The IdP authenticates with a bearer token, stored at `vault://tenants/TENANT/scim`. Any other token is `401`.

| Request | Effect |
|---|---|
| `POST /Users` | Create a user (`userName`, optional `externalId`, `active`, `emails`, and the enterprise extension's `manager.value`, the manager's user id, for segregation of duties, sb-approvals(7)). A duplicate `userName` is `409` |
| `GET /Users/ID` | Read a user |
| `PUT /Users/ID` | Replace a user |
| `PATCH /Users/ID` | `replace` of `active` (and other simple attributes, and `manager` by its full enterprise path) |
| `DELETE /Users/ID` | Same as deactivate; the user is kept for the audit trail |
| `POST /Groups`, `PUT /Groups/ID`, `PATCH /Groups/ID`, `DELETE /Groups/ID` | Groups and their members (`add`/`remove` of `members`) |

Setting `active` to false (by `PUT`, `PATCH` or `DELETE`) revokes every session of that user in the cell **before** the response is sent. If revocation fails, the request fails with `503` and the IdP retries. The 60-second re-validation then catches anything a lost revocation missed, so no request from a deactivated user succeeds more than 5 minutes after the deactivate (LAW-12).

### Identity Platform, exactly (SB-907)

A company without its own OIDC IdP signs in through **Identity Platform**, Firebase Authentication's enterprise tier. Each company gets **its own Identity Platform tenant**, which federates the company's SAML or OIDC provider (for example, a Google Workspace or Microsoft 365 account). The manifest names it as `idp: identity-platform://PROJECT/IDP_TENANT`, where `PROJECT` is the GCP project that holds Identity Platform and `IDP_TENANT` is the company's Identity Platform tenant id.

The cell verifies these tokens through the same `oidc` port as any other IdP, with these rules:

- the issuer is `https://securetoken.google.com/PROJECT`, found by discovery, and the audience is `PROJECT`;
- the token's `firebase.tenant` claim must equal `IDP_TENANT` exactly. A token from **another** company's Identity Platform tenant has a valid signature, the same issuer and the same audience, so this claim is what tells them apart. A missing or different value is `ETENANT` (LAW-02);
- the token must carry an `email` with `email_verified: true`, and the directory user is found by that email (Identity Platform's own user id is never trusted across tenants);
- every other rule (algorithm, expiry, nonce, subject) is as above.

### Attributes and moves, exactly (SB-1205)

SCIM gives each user an id `usr_` and 24 hex characters, and each group `grp_` and 24, the ids legal holds, erasure and the audit trail name them by (BUG-0025). The cell keeps five more facts about each user from SCIM: the core `title`, and the enterprise extension's (`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`) `department`, `division`, `costCenter` and `manager.value`. `POST` and `PUT` take them in the usual SCIM shape; `PATCH` takes `title` by name and the others by their full enterprise path, such as `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department`. The manager may be `{"value": "<id>"}` or the id as a plain string, and `remove` on its path clears it (SB-1401). They are read from the directory on every request, with the user's groups, and passed to policy as the principal's attributes (sb-policy(5)), so a rule can say "finance's cost centres only". A directory that can't give them refuses the request with `EUNAVAIL`; it never runs with yesterday's attributes.

**Moves.** When someone changes team, the IdP moves them between groups with SCIM `PATCH /Groups/ID` (`remove` from one, `add` to another). The cell reads groups from the directory on every request, and never caches them across requests, so leaving a space's group refuses that space at the next request, well within LAW-21's 5 minutes. Joining grants a space only once the membership is in the cell's directory: a group in a sign-in token, or an `add` the directory didn't commit (the IdP gets `503` and retries), grants nothing. The law test checks both on a fake clock, through SCIM and the request path (LAW-21).

**Groups still come from SCIM.** Identity Platform signs people in; it doesn't say who is in which group. A company that can't run SCIM may instead list its groups in the manifest with `groups:` (sb-tenant(5)), which its stewards change by manifest diff like any other field. `groups:` is accepted **only** with an `identity-platform://` idp. Someone who signs in but isn't in any listed group, or in the directory, has no groups and can use nothing, so the sign-in is never open. Removing someone from `groups:` and applying the manifest revokes them within 5 minutes, the same bound as a SCIM deactivate (LAW-12).

### Owners shown to askers (SB-1702)

An answer or a search names the owner of a cited document, so the asker knows who to ask (sb-ask(3), Who to ask). The owner is the document's owner at the source, an email, matched to the directory by that email: it is shown, with the user's SCIM `title`, only when exactly one active user has it. A user SCIM deactivates stops being shown at once, since the lookup is made for every answer, never cached (LAW-12). No other fact about a person is shown, and nothing about what anyone does in the cell is recorded to decide who knows what.

### Sessions

Sessions last at most 12 hours from sign-in and are never extended: after 12 hours the user signs in again. Each session is re-validated against the directory every 60 seconds. A user the directory doesn't know, or a directory that can't answer, ends the session with `EAUTH`. A SCIM deactivate triggers immediate revocation as well, so the 5-minute bound (LAW-12) has plenty of margin. Sessions live in the cell's own session store; no session is valid in any other cell.

**Signing out everywhere (SB-1906).** A person who has lost a laptop ends every session of theirs, this one included, with **Sign out everywhere** at the foot of the web UI, which is `POST /v1/sessions/revoke` (sb-api(3)). An operator ends a named person's sessions with `POST /v1/admin/sessions/revoke` (sb-admin(3)), policy-checked like every admin endpoint. Both write a `kind: access` audit row (subject `sessions:USER`, state `revoke_requested`) **before** any session ends, and a second row (`revoked`, with `sessions=N`) after; if the first row can't be written, nothing is revoked. A revoked session is refused with `EAUTH` at its next request, because every request reads the session from the store. Revoking ends sessions, not the account: the person can sign in again at once, as long as the directory says they are active. To keep them out, deactivate them in the IdP (SCIM), which also revokes every session.

### Approvals

Approving a consequential action requires a session less than 15 minutes old, or step-up re-authentication at the IdP.

### Service identities

The chat-app integration and CI use workload identity with short-lived tokens. There are no static API keys.

### Chat users

A chat-app request acts as a person only after it is mapped to the directory. The workspace must be the tenant's, the chat user's email (from the chat app's own user profile) must match exactly **one active** directory user, and the scope is issued for that user with their directory groups, the same as a web session. Without a single match there is no answer and no fallback identity (sb-ask(3), The chat app). The mapping is checked on every request and never cached, so a SCIM deactivation stops chat answers at once.

## LAWS

[LAW-02](sb-laws.7.md#law-02), [LAW-12](sb-laws.7.md#law-12).

## EXAMPLES

```text
09:00:00  SCIM PATCH /Users/… active=false      (from IdP)
09:00:01  sessions revoked: 2                    audit kind=access
09:00:03  request from usr_…: 401 EAUTH
```

## SEE ALSO

[sb-api(3)](sb-api.3.md), [01 Identity and tenancy](../docs/architecture/01-identity-and-tenancy.md)

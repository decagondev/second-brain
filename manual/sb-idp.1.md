---
title: "sb-idp(1)"
man: "1"
summary: The bundled sign-in for a self-hosted cell with no IdP of its own, for demos and air-gapped sites (SB-1605).
---

# sb-idp(1)

## NAME

sb-idp — the bundled OIDC identity provider for a self-hosted cell

## SYNOPSIS

```text
sb-idp serve --issuer URL --data DIR --tenant TENANT --client ID --secret-file FILE --redirect URL [--listen ADDR] [--behind-proxy]
sb-idp user add --data DIR --email EMAIL --scim URL --scim-token-file FILE    # the password is read from stdin
sb-idp user disable --data DIR --email EMAIL --scim URL --scim-token-file FILE
sb-idp user list --data DIR
```

## DESCRIPTION

A company that runs its own cell signs in with its own OIDC IdP (sb-identity(7)). A site with no IdP the cell can reach, such as an air-gapped one, or a demo, uses **sb-idp** instead: a small OIDC provider with passwords that serves one tenant's one client, behind the cell's TLS proxy at `https://HOST/idp` (sb-selfhost(7), Install). It is not the development IdP (`tools/devidp`), which has no passwords and refuses any address but loopback.

**Serving.** `serve` answers discovery, the key set (`/jwks`), `/authorize` and `/token`, exactly the flow sb-identity(7) describes: authorization code with PKCE (S256 only), the client secret over HTTP Basic, the token's `tenant` claim and `sub` set to the user's directory id. `/authorize` refuses any `client_id` but its own and any `redirect_uri` but the one `--redirect` names, before showing anything, so it can't be used to send a person elsewhere. It then shows a sign-in form. A sign-in succeeds only for an enabled user whose password matches; the code it issues is single-use, bound to the PKCE challenge, the redirect and the nonce, and expires after 60 seconds. ID tokens expire after 5 minutes; the cell keeps its own session (sb-identity(7), Sessions).

**Passwords** are kept only as PBKDF2-SHA-256 hashes with 600,000 iterations and a 16-byte random salt each (`pbkdf2-sha256$600000$SALT$HASH`), compared in constant time. A password shorter than 12 characters is refused. Five failed sign-ins for one address, or twenty from one client address, within 15 minutes refuse further tries from it for the rest of that window, with the same answer as a wrong password, so the form says nothing about which addresses exist. Behind the proxy (`--behind-proxy`, as the self-hosted layout runs it) the client address is the last `X-Forwarded-For` entry, the one the proxy added, so one person's failures never lock out everyone else; nothing but the proxy can reach the IdP there.

**The form** is served with `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'`, `Cache-Control: no-store` and `Referrer-Policy: no-referrer`, and carries a token, an HMAC over the request's parameters and the time with a key made at start, that a submitted form must return within 10 minutes.

**Users** live in `DIR/users.json` (mode 0600), read at each sign-in, so a change takes effect at once. `user add` reads the password from stdin, provisions the user in the cell over SCIM (sb-identity(7), SCIM) to get its directory id, and stores the id, the address and the hash. `user disable` marks the user disabled here and deactivates them over SCIM, so the cell ends their sessions within 5 minutes (LAW-12). The signing key, RSA 2048, is made on first start in `DIR/key.pem` (mode 0600) and never leaves the box.

## OPTIONS

| Option | Meaning |
|---|---|
| `--issuer URL` | The issuer, `https://HOST/idp`; the cell's `oidc.issuer` names the same (sb-cell(5)) |
| `--data DIR` | The users file and signing key |
| `--tenant`, `--client`, `--secret-file`, `--redirect` | The one tenant, the cell's client id and secret, and the cell's `BASE_URL/callback` |
| `--listen ADDR` | Where to listen, default `0.0.0.0:8082`; only the proxy reaches it |
| `--behind-proxy` | Count failures by the proxy's `X-Forwarded-For` client address |
| `--scim URL`, `--scim-token-file FILE` | The cell's SCIM base (`http://api:8080/scim/v2` inside the layout) and its bearer token |

## EXIT STATUS

0 on success; 1 on an error; 2 on bad usage, a refused password, or an address already present.

## LAWS

[LAW-02](sb-laws.7.md#law-02): a token carries its tenant and is refused by any other cell. [LAW-03](sb-laws.7.md#law-03): an unknown address, a disabled user or an unreadable users file never signs in. [LAW-07](sb-laws.7.md#law-07): passwords exist only as salted hashes, and the client secret and signing key are files only this service reads. [LAW-12](sb-laws.7.md#law-12): disabling a user deprovisions them in the cell.

## EXAMPLES

```text
$ printf '%s' "$PASSWORD" | docker compose run --rm -T idp user add --data /var/lib/sb-idp --email ann@acme.example \
    --scim http://api:8080/scim/v2 --scim-token-file /run/sb/scim
sb-idp: added ann@acme.example as usr_3f9c0e1d2b7a4c5e
```

## SEE ALSO

[sb-selfhost(7)](sb-selfhost.7.md), [sb-identity(7)](sb-identity.7.md), [sb-cell(5)](sb-cell.5.md)

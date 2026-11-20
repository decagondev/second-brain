---
title: "sb-errors(7)"
man: "7"
summary: Every error code, its HTTP status, CLI exit code and remedy.
---

# sb-errors(7)

## NAME

sb-errors — the error registry, shared by the CLI, API and UI

## SYNOPSIS

```json
{ "error": { "code": "EPERM", "message": "…", "trace": "tr_…", "remedy": "…" } }
```

## DESCRIPTION

There is one list of errors. `internal/errs/codes.go` is generated from the table below (SB-006), so this page can't drift from the code. Codes are stable forever. Messages may improve.

## REGISTRY

| Code | HTTP | Exit | Meaning | Remedy |
|---|---|---|---|---|
| `EUSAGE` | — | 2 | CLI usage error | See `sb COMMAND --help` |
| `EINVAL` | 400 | 2 | Malformed request | Fix the request body |
| `EAUTH` | 401 | 3 | Not signed in, or session revoked | Sign in again |
| `EPERM` | 403 | 3 | Refused: policy deny, source deny, **unknown**, or missing source | Ask the owner for access; if frequent, check the connector's unknown rate |
| `ETENANT` | 403 | 3 | Request crossed a tenant boundary | Use the cell for your own tenant |
| `ESPACE` | 403 | 3 | Asked in a space the asker isn't a member of | Ask the space's stewards for membership, or ask company-wide |
| `ELABEL` | 403 | 3 | A chunk's label isn't allowed for this asker (a refusal reason, never shown with the chunk) | Nothing to do as a user; operators see it in refusals |
| `EBARRIER` | 403 | 3 | An information barrier separates the asker from a chunk's space (a refusal reason) | Nothing to do as a user; barriers are set in the manifest |
| `ENOTSUP` | 501 | 2 | Operation not supported by this connector | Choose a workflow its connector supports |
| `ENOTIMPL` | 501 | 2 | Command or endpoint is specified in this manual but not built in this release | See the ticket named in the message |
| `EMANIFEST` | 422 | 1 | Manifest invalid | Fix the lines reported |
| `EDRIFT` | 409 | 1 | Live config differs from the applied manifest | Re-apply the manifest; investigate the change |
| `EHOLD` | 409 | 1 | A legal hold covers what would be deleted | Ask a Legal steward; deletion waits for the hold's release |
| `ELAW` | — | 1 | A law check failed or returned unknown in `verify` | See the per-law output; do not go live |
| `ECHAIN` | — | 1 | Audit chain or anchor verification failed | Sev-1: preserve evidence, page security |
| `EEVAL` | — | 1 | Eval threshold missed | Fix the regression; thresholds aren't overridable |
| `EKILLED` | 503 | 4 | A kill switch is active for this scope | Wait for the operator to `sb unkill` |
| `ENOAPPROVAL` | 409 | 5 | Approval missing, expired, self-approved, or hash mismatch | Request a fresh approval of the current payload |
| `ENORECEIPT` | 502 | 6 | Target returned no receipt; action **did not run** | Check the target; re-propose if needed |
| `EUNAVAIL` | 503 | 7 | A dependency is unavailable or timed out | Retry later |
| `EBUDGET` | 429 | 8 | Budget or rate limit reached | Wait for reset, or raise `budget_usd_month` |
| `EUNSIGNED` | — | 1 | Image or bundle signature didn't verify | Sev-1: the release is blocked from starting |

## NOTES

- `EPERM` deliberately covers deny **and** unknown. Telling a user "unknown" invites retry games, and telling them "deny" may leak that a document exists. Operators see the breakdown in `/v1/admin/refusals`.
- `ELABEL` and `EBARRIER` follow the same rule as `EPERM`: they are reasons in traces and refusal counts, and a user is never told which chunk a label or barrier refused (sb-permissions(7), Spaces, labels and barriers).
- A refused source inside an answer is not an error; it's reported through `refused` and `notice` ([sb-ask(3)](sb-ask.3.md)).

## SEE ALSO

[sb(1)](sb.1.md), [sb-api(3)](sb-api.3.md)

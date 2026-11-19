---
title: Laws
summary: The law tests, the world generator, the red ratchet and the self-tests.
---

# Laws

Executable checks for the laws in [sb-laws(7)](../manual/sb-laws.7.md) (SB-010). Run them with `make laws`.

| File | What it is |
|---|---|
| [prop.go](prop.go) | `Check`: runs a property over seeds 1..300; `SB_SEED=n` reproduces a failure (ADR-0016) |
| [world.go](world.go) | The world generator: tenants, users, groups and docs, with stale prefilters, unknown verdicts, missing and public docs, timeouts, and targets with no receipt |
| [system.go](system.go) | The surface the laws test. `NewSystem` returns a pending system until SB-302, SB-405 and SB-408 wire the real one in |
| [checks.go](checks.go) | One check per law: LAW-01 to LAW-05 |
| [laws_test.go](laws_test.go) | The law tests, run against `NewSystem` |
| [selftest_test.go](selftest_test.go) | Tests of the tests: a reference system passes every law; six broken systems are each caught; the generator covers every hard case |
| `expected_red.txt` | The ratchet: laws that must still be red, and the ticket that turns each green |

## Turning a law green

1. Wire the real system into `NewSystem` (in the ticket that builds it).
2. Run `make laws`. The ratchet reports `LAW-nn is green`.
3. Remove that line from `expected_red.txt`, and move the law to *Enforced* in the [assurance ledger](../living/ASSURANCE_LEDGER.md).

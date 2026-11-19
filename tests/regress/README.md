---
title: Regression tests
summary: One test per bug, kept forever.
---

# Regression tests

Every entry in [BUGS_AND_MITIGATIONS](../../living/BUGS_AND_MITIGATIONS.md) that can be tested gets a regression test here, named for the bug, and it's never deleted (SB-016).

| Language | File name | Run by |
|---|---|---|
| Go | `bug_nnnn_short_name_test.go` | `make regress` |
| Python | `test_bug_nnnn_short_name.py` | `make regress` |

A bug that a law self-test or another permanent test already covers names that test in its entry instead of duplicating it.

| Bug | Test |
|---|---|
| BUG-0001 | Design: covered by `Rollout.tla` and `proofs/crosscheck.py` (`NoRevokedRunsUndrained`) |
| BUG-0002 | Design: covered by `Approvals.tla` `KillWins` and `proofs/crosscheck.py` |
| BUG-0003 | [`test_bug_0003_cedar_reserved_type.py`](test_bug_0003_cedar_reserved_type.py) |
| BUG-0004 | `laws/selftest_test.go`: `TestSelf_EachMutationIsCaught` ("ignores tenant boundary") and `TestSelf_GeneratorCoversHardCases` ("public doc") |

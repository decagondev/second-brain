---
title: Smoke tests
summary: Black-box checks of the real binary, and later of every deployed cell.
---

# Smoke tests

`make smoke` builds the real `sb` binary and checks its exit-status contract from [sb(1)](../../manual/sb.1.md). From W3, the same suite gains a post-deploy check against each cell: sign in as a synthetic user, ask a known question, expect the known citation, and expect a refused document to stay refused.

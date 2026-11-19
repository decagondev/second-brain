---
title: Patch-release playbook
summary: From a CVE advisory to a patch release on testing within 24 hours (SB-505).
---

# Patch-release playbook

When the advisory watcher fails ([sb-release(7)](../manual/sb-release.7.md#cves)), or an advisory arrives any other way, follow these steps. The clock starts when the advisory is seen. The target is a patch release on `testing` within **24 hours**.

| Step | What | Check |
|---|---|---|
| 1. Triage (1 h) | Read the watcher's output: advisory id, package, affected and fixed versions. For Go, `govulncheck` reports only code that is actually called. Decide whether we are affected | Write it in the incident channel: affected, or not affected and why |
| 2. Bump (2 h) | Change only that dependency to the fixed version. For a Go module, also update `tools/deps/allowed.txt` and the ADR that admitted it (ADR-0017), because `make deps` enforces the list | The diff touches one dependency, and its lockfile lines |
| 3. Lockfiles | `make lockfiles` | `git diff` shows only the bumped package |
| 4. Check | `make check` and `make integration`; for a web package also `make web` | All green |
| 5. Release | Push to `main`. The release workflow rebuilds twice, writes the SBOM, bundles and signs. Gate it as usual | The new release id; the SBOM lists the fixed version |
| 6. Testing channel | Promote to `testing` (ring 1) through the normal path (`sb release promote`, sb-release(1)) | Ring 1 cells run the new release (attested digests, SB-509) |
| 7. Record | Add the advisory id and the fixed version to `living/CHANGELOG.md` under Security, and close the incident | The changelog line |

Nothing skips the eval gate or the signature, even for a CVE. Because no tenant on the release train is more than one release behind stable (LAW-08), everyone has the fix after the next promotion to `stable`.

If the fixed version breaks a test and can't be adapted within the window, record why, and mitigate (a kill switch on the affected connector or agent, sb-kill-switches(7)) until the release ships.

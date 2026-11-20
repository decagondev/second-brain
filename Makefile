# One command per job. `make check` is what CI runs, and it must pass on a clean clone.
include tools/versions.env
export

GO      ?= go
PY      ?= python3
LDFLAGS := -X example.com/second-brain/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: policy integration ci-rules check build test lint deps gen gen-check laws smoke regress evals examples crosscheck proofs docs diagrams web release-build lockfiles tools clean

check: gen-check lint deps build test laws smoke regress evals examples policy crosscheck ci-rules docs  ## everything that runs offline
	@echo "make check: all green"

build:                 ## compile everything; the sb binary goes to bin/
	$(GO) build ./...
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/sb ./cmd/sb
test:                  ## unit tests (Go and Python)
	$(GO) test ./internal/... ./cmd/...
	$(PY) -m unittest discover -s evals/tests -t . -q
lint:                  ## gofmt + golangci-lint (includes depguard, SB-008)
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt: files above need formatting"; exit 1; }
	golangci-lint run ./...
	$(PY) tools/check/decisions.py
	$(PY) tools/check/custody.py
	$(PY) tools/check/toolbus.py
deps:                  ## the dependency rule, independent of the linter (SB-008)
	$(PY) tools/check/deps.py
gen:                   ## regenerate code from the manual
	$(PY) tools/gen/errs.py
gen-check:             ## generated code matches the manual (SB-006)
	$(PY) tools/gen/errs.py --check
laws:                  ## law tests; expected-red laws must fail for the right reason (SB-010)
	$(PY) tools/check/laws.py
smoke:                 ## black-box smoke tests of the sb binary (SB-016)
	$(GO) test -count=1 ./tests/smoke/...   # never cached: it runs the freshly built bin/sb (BUG-0008)
	$(GO) test ./tests/canary/...   # the CI secret scan with planted canaries (SB-411)
	$(GO) test ./tests/backup/...   # backup 3-2-1 and the restore drill (SB-412)
	$(GO) test ./tests/gameday/...  # the DR game day rehearsed on local adapters (SB-613)
	$(GO) test ./tests/handoff/...  # the hand-off game day rehearsed end to end (SB-710)
regress:               ## one test per bug, forever (SB-016)
	@if ls tests/regress/*_test.go >/dev/null 2>&1; then $(GO) test ./tests/regress/...; fi
	$(PY) -m unittest discover -s tests/regress -p "test_*.py" -q
evals:                 ## eval harness: mock smoke, citation suite on the local cell, tenant suites (SB-015, SB-309)
	$(PY) -m evals run evals/suites/smoke.yaml --model mock
	@rm -rf build/evals
	$(PY) -m evals run evals/suites/permission.yaml evals/suites/citation.yaml evals/suites/injection.yaml evals/suites/refusal.yaml evals/suites/regression.yaml evals/suites/space-leak.yaml evals/suites/agent-incident-summary.yaml evals/suites/agent-spec-digest.yaml evals/suites/agent-close-checklist.yaml evals/suites/agent-runbook-answer.yaml evals/suites/template-*.yaml --model extractive --results build/evals --release rel_local
	$(PY) -m evals gate build/evals/rel_local
	@for t in tenants/*/; do $(PY) -m evals run --tenant-dir $$t --model extractive || exit 1; done
	@! $(PY) -m evals run evals/suites/citation.yaml --model misattributing >/dev/null 2>&1 || { echo "evals: a misattributing model passed the citation suite"; exit 1; }
	@echo "evals: the misattributing model fails the citation suite, as it must"
	$(PY) tools/check/evals_red.py   # suites red on purpose fail only as expected (SB-807)
examples:              ## manual and example files are valid (SB-009)
	$(PY) tools/check/examples.py
	$(PY) tools/check/runbook.py
	$(PY) tools/check/findings.py   # no open critical or high pen-test finding (SB-612)
policy:                ## policies validate, every permit is tenant-scoped (SB-112)
	$(PY) tools/check/policy.py
crosscheck:            ## Python mirror of the TLA+ models (not a substitute for TLC)
	$(PY) proofs/crosscheck.py
proofs: tools          ## TLC + Alloy, then mutation checks (SB-011..013, SB-705)
	tools/ci/proofs.sh
	$(PY) tools/ci/mutations.py
docs:                  ## build the site and check every link (SB-002, SB-009)
	$(PY) tools/site/build.py
	$(PY) tools/check/links.py site
diagrams:              ## render every Mermaid diagram headless (SB-009; needs playwright + npm)
	$(PY) tools/check/diagrams.py site
web:                   ## web UI: typecheck, tests (LAW-14 rendering, axe) and build; needs Node 22 (SB-307)
	npm ci --prefix web --no-audit --no-fund
	npm run --prefix web build
	npm test --prefix web
lockfiles:             ## regenerate go.sum, vendor/ and web/package-lock.json (SB-505); review the diff
	$(GO) mod tidy
	$(GO) mod vendor
	npm install --prefix web --package-lock-only --no-audit --no-fund
COMMIT_TIME := $(shell git show -s --format=%cI HEAD 2>/dev/null || echo 1970-01-01T00:00:00Z)
COMMIT_DAY  := $(shell git show -s --format=%cs HEAD 2>/dev/null || echo 1970-01-01)
release-build:         ## reproducible build twice, SBOM and bundle into build/release (SB-504)
	tools/ci/repro.sh build/release
	$(GO) run ./tools/release/sbom --binary build/release/a/sb --npm web/package-lock.json --created $(COMMIT_TIME) --out build/release/sbom.spdx.json
	$(GO) run ./tools/release/bundle --artefact sb=build/release/a/sb --artefact sbom=build/release/sbom.spdx.json --created $(COMMIT_DAY) --out build/release/bundle.json --digest-out build/release/sign.txt
integration:           ## adapter tests against real services; needs SB_TEST_POSTGRES_DSN (SB-105)
	@test -n "$$SB_TEST_POSTGRES_DSN" || { echo "integration: set SB_TEST_POSTGRES_DSN"; exit 1; }
	$(GO) test -count=1 ./internal/adapters/postgres/...
tools:                 ## fetch pinned tool jars and verify checksums
	tools/ci/fetch_tools.sh
clean:
	rm -rf bin tools/vendor
ci-rules:              ## the manual-diff rule behaves as specified (SB-007)
	tools/ci/test_manual_diff.sh

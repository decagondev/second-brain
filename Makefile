# One command per job. `make check` is what CI runs, and it must pass on a clean clone.
include tools/versions.env
export

GO      ?= go
PY      ?= python3
LDFLAGS := -X example.com/second-brain/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: policy integration ci-rules check build test lint deps gen gen-check laws smoke regress evals examples crosscheck proofs docs diagrams web tools clean

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
deps:                  ## the dependency rule, independent of the linter (SB-008)
	$(PY) tools/check/deps.py
gen:                   ## regenerate code from the manual
	$(PY) tools/gen/errs.py
gen-check:             ## generated code matches the manual (SB-006)
	$(PY) tools/gen/errs.py --check
laws:                  ## law tests; expected-red laws must fail for the right reason (SB-010)
	$(PY) tools/check/laws.py
smoke:                 ## black-box smoke tests of the sb binary (SB-016)
	$(GO) test ./tests/smoke/...
regress:               ## one test per bug, forever (SB-016)
	@if ls tests/regress/*_test.go >/dev/null 2>&1; then $(GO) test ./tests/regress/...; fi
	$(PY) -m unittest discover -s tests/regress -p "test_*.py" -q
evals:                 ## eval harness against the mock model (SB-015)
	$(PY) -m evals run evals/suites/smoke.yaml --model mock
examples:              ## manual and example files are valid (SB-009)
	$(PY) tools/check/examples.py
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
integration:           ## adapter tests against real services; needs SB_TEST_POSTGRES_DSN (SB-105)
	@test -n "$$SB_TEST_POSTGRES_DSN" || { echo "integration: set SB_TEST_POSTGRES_DSN"; exit 1; }
	$(GO) test -count=1 ./internal/adapters/postgres/...
tools:                 ## fetch pinned tool jars and verify checksums
	tools/ci/fetch_tools.sh
clean:
	rm -rf bin tools/vendor
ci-rules:              ## the manual-diff rule behaves as specified (SB-007)
	tools/ci/test_manual_diff.sh

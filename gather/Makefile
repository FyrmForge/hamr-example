.PHONY: build test lint templint templ-check seed db-sh docker-up docker-down docker-delete docker-reload clean install check-templ generate check-node-modules css-build e2e e2e-local e2e-run e2e-run-local

# Force bash so the ENV_LOAD eval below works cross-shell (sh on Debian/Ubuntu
# is dash, which doesn't grok `eval "$(...)"` quoting consistently).
SHELL := bash
.SHELLFLAGS := -ec

HAMR_VERSION  := $(shell grep 'github.com/FyrmForge/hamr ' go.mod | awk '{print $$2}')
TEMPL_VERSION := $(shell grep 'github.com/a-h/templ ' go.mod | awk '{print $$2}')

# ENV_LOAD: prefix targets that need hamr dev's port-walked .env values
# (DATABASE_URL, S3_ENDPOINT, etc.). `hamr env --export` reads
# .hamr/walks.json (written by `hamr dev` after walking) + .env, and emits
# `export KEY=VALUE` lines for the values that were rewritten. When hamr
# dev isn't running or no ports walked, output is empty and eval is a
# no-op — the target falls through to whatever .env it already had.
# `|| true` survives the case where `hamr` isn't on PATH (e.g. during
# `make install`); plain .env still loads normally inside the binary.
ENV_LOAD := eval "$$(hamr env --export 2>/dev/null || true)";

## install: Install development dependencies
install:
	go install github.com/FyrmForge/hamr/cmd/hamr@$(HAMR_VERSION)
	go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)
	cd frontend && npm install

## check-templ: Verify templ is installed
check-templ:
	@command -v templ >/dev/null 2>&1 || { echo "templ not found. Run: make install" >&2; exit 1; }

## check-node-modules: Auto-install npm deps if missing
check-node-modules:
	@[ -d frontend/node_modules ] || (cd frontend && npm install)

## css-build: Build Tailwind CSS for production (minified)
css-build: check-node-modules
	cd frontend && npm run css:build

VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")

## fmt: Format all Go source files
fmt:
	go fmt ./...

## build: Build the site binary
build: check-templ check-node-modules
	templ generate
	cd frontend && npm run css:build
	hamr gen static
	go build -ldflags "-X main.version=$(VERSION)" -o bin/site ./cmd/site
	$(MAKE) generate

## generate: Generate static pages
generate:
	$(ENV_LOAD) ./bin/site --generate

## test: Run all tests
test: check-templ
	templ generate
	go test ./...

## seed: Load the dev seed (test accounts + the hamr_disposable marker that unlocks wipe/reseed). Safe to re-run.
seed:
	$(ENV_LOAD) go run ./cmd/site --seed

## db-sh: Open an interactive shell to the local dev database
db-sh:
	$(ENV_LOAD) ./scripts/db-shell.sh

## lint: Run linters
lint:
	golangci-lint run

## templ-check: Fail if generated *_templ.go files are stale
templ-check: check-templ
	templ generate -check

## templint: Lint .templ files for common issues
templint:
	hamr lint templ

# `hamr compose` merges the port-walk override hamr dev generates when a host
# port is busy. Plain `docker compose` sees only the base file, decides the
# running stack has drifted, and recreates it on the original — busy — ports.
# Fall back to `docker compose -f docker/docker-compose.yaml ...` in
# environments without the hamr binary (CI images that only need the stack up).

## docker-up: Start Docker services
docker-up:
	hamr compose up -d

## docker-down: Stop Docker services
docker-down:
	hamr compose down

## docker-delete: Delete Docker containers and volumes
docker-delete:
	hamr compose down -v

## docker-reload: Delete and recreate Docker services
docker-reload: docker-delete docker-up

## e2e: Run E2E tests (containerized)
e2e:
	go test -v -tags=e2e ./e2e -timeout 10m

## e2e-local: Build, then run E2E tests against a spawned bin/site (free port from 8080) and the local DB
e2e-local: build
	$(ENV_LOAD) go test -v -tags=e2e ./e2e -local -timeout 10m

## e2e-run: Run specific E2E test: make e2e-run T=TestName
e2e-run:
	go test -v -tags=e2e ./e2e -run "$(T)" -timeout 5m

## e2e-run-local: Run specific E2E test locally: make e2e-run-local T=TestName
e2e-run-local: build
	$(ENV_LOAD) go test -v -tags=e2e ./e2e -local -run "$(T)" -timeout 2m

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

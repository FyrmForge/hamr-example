# gather

A Go web application built with the [HAMR framework](https://github.com/FyrmForge/hamr).

## Prerequisites

- Go 1.27.2+
- Docker (for PostgreSQL)
- [templ](https://templ.guide) CLI (run `make install` to get the version pinned in go.mod; the CLI must match it)
- [hamr](https://github.com/FyrmForge/hamr) CLI

## Quick Start

```bash
# Install dev tools (templ)
make install
# Start local services
make docker-up

# Run the dev server (builds, live reload)
hamr dev
```

The server starts at [http://localhost:3000](http://localhost:3000) (proxied from `:8080`). If those ports are busy on your machine, hamr walks +1 (3001/8081 etc.) and prints the actual URL in its startup banner.

## Development

```bash
hamr dev            # Run dev server (live reload)
make build          # Build binary to bin/site
make test           # Run tests
make lint           # Run golangci-lint
make templint       # Lint .templ files
make db-sh          # Open pgcli shell to local dev DB
make docker-up      # Start services
make docker-down    # Stop services
```

> **Note:** Migrations run automatically when the server starts.

## E2E Testing

```bash
make e2e            # Run containerized E2E tests
make e2e-local      # Run against local server
make e2e-run T=Test # Run specific test
```

## Project Structure

```
cmd/site/              Application entry point
internal/
  config/                Environment configuration
  db/                    Database + migrations
  repo/                  Data access layer
  web/                   HTTP handlers + components
frontend/                Static assets, CSS source, npm config, dist/
docker/                  Docker Compose
docs/                    Documentation
```

## Stack

- **Go** 1.27.2 + Echo v4
- **Templ** + HTMX
- **PostgreSQL**
- **HAMR Framework** — server, middleware, validation, responses

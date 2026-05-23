# Trove

Trove is a self-hosted registry for agent-consumable engineering artifacts: `AGENTS.md` files, skills, commands, prompts, context packs, schemas, runbooks, templates, and related project assets.

It runs as a single Go binary that serves JSON APIs, raw artifact endpoints, health checks, metrics, and an embedded React/Vite web UI. PostgreSQL is the MVP source of truth for metadata and artifact bytes.

> [!WARNING]
> Trove is pre-v1.0 software. It is highly unstable and subject to heavy changes, including additions, removals, and breaking changes to functionality.
>
> The API contract is not guaranteed at this stage. Use at your own risk.

- Website: [nowhereworks.github.io/trove](https://nowhereworks.github.io/trove/)
- Docs: [nowhereworks.github.io/trove/docs](https://nowhereworks.github.io/trove/docs/)
- Releases: [github.com/nowhereworks/trove/releases](https://github.com/nowhereworks/trove/releases)
- Container image: [nowhereworks/trove](https://hub.docker.com/r/nowhereworks/trove)

## What Trove Provides

- Immutable published package versions using strict SemVer.
- One-level package references: `org/namespace/package@selector`.
- Server-side selectors for `latest`, `stable`, exact versions, and digests.
- Public, internal, and private visibility rules.
- OIDC-oriented human auth, scoped bearer tokens for agents and CI, and dev auth for local use.
- Upload, review, approval, publish, deprecate, and yank workflows.
- Raw artifact, manifest, archive, resolve, search, update check, and compatibility APIs.
- CLI workflows for `resolve`, `fetch`, `install`, `check`, and dry-run `update`.
- React SPA for package discovery, package detail, upload/publish, reviews, and adoption reporting.

## Quick Start

### Run With Seeded In-Memory Data

This starts the server without PostgreSQL. It is useful for browsing the API and embedded UI with seeded sample data.

```bash
go run ./cmd/trove
```

Open [http://localhost:8080](http://localhost:8080).

When `TROVE_DATABASE_URL` is unset, write APIs are not backed by PostgreSQL and the process logs that it is using the seeded in-memory store.

### Run With PostgreSQL

Start PostgreSQL:

```bash
docker run --rm --name trove-postgres \
  -e POSTGRES_USER=trove \
  -e POSTGRES_PASSWORD=trove \
  -e POSTGRES_DB=trove \
  -p 5432:5432 \
  postgres:16
```

In another shell, run Trove with startup migrations enabled for local development:

```bash
export TROVE_DATABASE_URL='postgres://trove:trove@localhost:5432/trove?sslmode=disable'
export TROVE_DATABASE_MIGRATE_ON_STARTUP=true
export TROVE_AUTH_MODE=dev
export TROVE_AUTH_DEV_MODE_ENABLED=true

go run ./cmd/trove
```

Open [http://localhost:8080](http://localhost:8080).

### Use The CLI Locally

The same binary exposes CLI subcommands when the first argument is a CLI command:

```bash
go run ./cmd/trove help
go run ./cmd/trove resolve nwks/platform/agent-backend@stable
go run ./cmd/trove resolve nwks/platform/agent-backend@stable --json
```

Configure the CLI with environment variables:

```bash
export TROVE_SERVER_URL=http://localhost:8080
export TROVE_TOKEN='<bearer-token-if-required>'
```

## Install Released CLI Builds

Trove publishes CLI binaries to GitHub Releases for Linux, macOS, and Windows on `amd64` and `arm64`.

- Rolling prerelease: [`latest`](https://github.com/nowhereworks/trove/releases/tag/latest), replaced when CLI-related files change on `main`.
- Tagged releases: `vMAJOR.MINOR.PATCH`, published from matching Git tags.
- Release assets are named like `trove-cli-linux-amd64.tar.gz`, `trove-cli-darwin-arm64.tar.gz`, and `trove-cli-windows-amd64.zip`.
- Each release includes `checksums.txt`.

Example Linux install from a tagged release:

```bash
version=v0.1.0
curl -L -o trove-cli-linux-amd64.tar.gz \
  "https://github.com/nowhereworks/trove/releases/download/${version}/trove-cli-linux-amd64.tar.gz"
tar -xzf trove-cli-linux-amd64.tar.gz
./trove version
```

## Run The Container Image

Trove publishes a multi-architecture Docker Hub image for `linux/amd64` and `linux/arm64`.

```bash
docker pull nowhereworks/trove:latest
docker run --rm -p 8080:8080 nowhereworks/trove:latest
```

Run the image with PostgreSQL-backed storage:

```bash
docker network create trove-dev

docker run --rm --name trove-postgres --network trove-dev \
  -e POSTGRES_USER=trove \
  -e POSTGRES_PASSWORD=trove \
  -e POSTGRES_DB=trove \
  postgres:16
```

In another shell:

```bash
docker run --rm --network trove-dev -p 8080:8080 \
  -e TROVE_DATABASE_URL='postgres://trove:trove@trove-postgres:5432/trove?sslmode=disable' \
  -e TROVE_DATABASE_MIGRATE_ON_STARTUP=true \
  -e TROVE_AUTH_MODE=dev \
  -e TROVE_AUTH_DEV_MODE_ENABLED=true \
  nowhereworks/trove:latest
```

Image tags are published by the release workflow as follows:

| Source | Tags |
|---|---|
| `main` branch | `latest` |
| `vMAJOR.MINOR.PATCH` tag | `latest`, `MAJOR.MINOR.PATCH`, `MAJOR.MINOR`, `MAJOR` |

Use an exact version tag in production instead of `latest`.

## Development

Prerequisites:

- Go 1.26.1 or newer compatible 1.26.x toolchain.
- Node.js 20+ for the React UI and VitePress docs.
- PostgreSQL 15+ for database-backed local development.

Build the embedded web UI:

```bash
cd web
npm ci
npm run build
```

Run Go tests and builds:

```bash
go test ./...
go build ./...
```

Build the docs site locally:

```bash
cd docs-site
npm ci
npm run docs:build
```

Install `sqlc` only when regenerating database access code:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

The committed schema and query sources live under `migrations/` and `internal/db/queries/`; generated Go code is committed under `internal/db/sqlc/`.

## Documentation

The user-facing documentation is maintained in `docs-site/` and published with GitHub Pages:

- Concepts: package references, selectors, visibility, lifecycle states, artifact types, and immutability.
- Publishing: manifests, upload/publish flow, review workflow, and security scanning.
- CLI: `resolve`, `fetch`, `install`, `check`, `update`, and lockfiles.
- API: public APIs, management APIs, raw artifacts, and archives.
- Operations: configuration, deployment, and compatibility.

Start with [What is Trove?](https://nowhereworks.github.io/trove/docs/concepts/what-is-trove.html).
